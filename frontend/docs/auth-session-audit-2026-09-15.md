# Audit des erreurs de session Course AI

Date : 2026-09-15. Incident signale : `3e20e17a-63d3-49b0-8f67-6ebd273fa49f`.

## Conclusion

Le refus de connexion a une cause reproductible dans la validation temporelle du backend et deux facteurs aggravants dans le frontend. L'horloge Windows retarde d'environ 13 secondes sur Clerk. Le middleware Go utilisait une tolerance nulle : un jeton signe, tout juste emis, peut donc etre refuse car ses dates `iat` et `nbf` sont dans le futur pour cette machine. Le frontend traduisait tout HTTP 401 par "Votre session a expire" et ne tentait pas de renouvellement force du jeton.

La configuration des deux instances Clerk a ete verifiee : les cles de signature publiques correspondent, la cle backend est acceptee par Clerk et l'origine autorisee correspond a `http://localhost:5173`.

La cause temporelle est reproduite par un test signe avant correction (401), puis corrigee (204). Le jeton exact et les logs associes a l'identifiant de l'incident n'ont pas ete accessibles : l'attribution de cette requete particuliere reste une inference appuyee par les mesures et la reproduction, et non une analyse de son JWT.

## Constats prioritaires

| Priorite | Constat                                                                           | Consequence                                                                       | Correction                                                                                          |
| -------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| P1       | Validation Clerk Go sans tolerance horaire, avec retard local mesure de 13 s      | Un jeton recent peut etre refuse sur toutes les routes protegees                  | Tolerance fixe et bornee de 15 s, tests avant/apres                                                 |
| P1       | Pas de renouvellement force apres un 401 dans le transport frontend               | Echec immediat du lancement, des listes recentes et du polling                    | `getToken({ skipCache: true })`, une seule reprise, renouvellement simultane partage                |
| P2       | Tout 401 presente comme une session expiree                                       | Diagnostic trompeur et reconnexion parfois sans effet                             | Distinguer absence de session et rejet persistant par l'API, conserver l'ID et proposer une relance |
| P2       | Delai HTTP et signal de query appliques seulement apres `getToken()`              | Attente non bornee si Clerk ne repond pas                                         | Annulation et delai de 30 s couvrant toute la requete, renouvellement compris                       |
| P2       | Tests navigateur avec un jeton fixe et tests backend avec dates largement valides | Les tests existants ne reproduisaient ni le decalage horaire ni le renouvellement | Tests temporels signes, tests du transport et parcours navigateur avec 401 transitoire/persistant   |

## Preuves recueillies

- Frontend accessible sur `http://localhost:5173`, API sur `http://localhost:8080`.
- `VITE_API_BASE_URL=http://localhost:8080` ; `CLERK_AUTHORIZED_PARTIES=http://localhost:5173`.
- Lecture des JWKS publics du frontend Clerk et des JWKS du backend Clerk : HTTP 200 dans les deux cas, identifiant de cle de signature identique. Aucune cle secrete ni aucun jeton utilisateur dans ce rapport.
- Trois mesures rapprochees le 15 septembre vers 16:14 UTC : temps local moins en-tete HTTP `Date` de Clerk, environ **-13 s**. L'estimation prend le milieu de l'aller-retour HTTP (298 a 491 ms) ; `Date` a une precision d'une seconde.
- `w32tm /query /status` : service non demarre, erreur `0x80070426`.
- Un nouveau controle vers 21:58 UTC confirme un retard d'environ 13 secondes : le probleme d'horloge persiste apres la validation du correctif.
- Le SDK installe `clerk-sdk-go/v2 v2.7.0` appelle `claims.ValidateWithLeeway(clock.Now().UTC(), params.Leeway)`. Sans option, `Leeway` vaut zero.
- Le test `TestClerkAuthenticationClockSkew` echoue avant correction pour le decalage de 13 s (HTTP 401 au lieu de 204) ; les tests Go passent apres correction.

## Parcours audite

1. `frontend/src/app/application.tsx` attend `useAuth().isLoaded`. Chaque couple utilisateur/session dispose de son runtime, de son cache Query et de ses signaux d'annulation.
2. `frontend/src/shared/api/client.ts` obtient le jeton Clerk et envoie le bearer. Il n'y a pas de JWT persiste par le code applicatif.
3. Le dashboard utilise `useGenerationList` et `useCourses`, respectivement `GET /api/generations` et `GET /api/courses`.
4. Le formulaire utilise `POST /api/generations` avec une cle d'idempotence. Le suivi interroge le statut toutes les deux secondes pendant `queued` ou `running`.
5. `backend-go/internal/infrastructure/http/middlewares/auth.go` verifie Clerk avant les handlers metier. Une erreur 401 d'authentification n'a donc pas encore cree de generation.
6. `frontend/src/app/query-client.ts` ne repete pas les erreurs 4xx et ne relance pas les mutations. Le polling s'arrete sur une erreur. Cette politique reste adaptee apres ajout de la reprise d'authentification bornee dans le transport.
7. `error-message.ts` et `ApiErrorNotice` decident du texte et des actions. Le code `session_expired` signifie maintenant que Clerk n'a fourni aucun jeton ; un refus de l'API conserve son propre code.

## Correctifs et garanties

- Tolere 15 secondes d'ecart pour les controles temporels Clerk. Les tests refusent un jeton futur ou expire au-dela de cette borne, et conservent les controles d'origine et de sujet existants.
- Transmet les options du fournisseur de jeton jusqu'au vrai `useAuth().getToken(options)`.
- Renouvelle seulement apres un HTTP 401. Deux tentatives HTTP au maximum par appel ; pas de boucle de reconnexion.
- Preserve le corps, les parametres et `Idempotency-Key` lors de la reprise. Une erreur reseau ou un statut 403/429/5xx n'entraine pas de repetition automatique d'une mutation par ce transport.
- Partage uniquement le renouvellement en cours dans le client de la session, sans ajouter de stockage persistant du jeton.
- Annule aussi l'attente du jeton si la session se termine, si la query est annulee ou si le delai global expire.
- Distingue en francais et en anglais la session absente du refus persistant de l'API. Le dashboard peut etre relance sans passer obligatoirement par l'ecran de connexion.

La tolerance de 15 secondes prolonge aussi au maximum de 15 secondes l'acceptation d'un jeton apres `exp`. C'est un compromis volontaire et borne pour la derive mesuree ; il ne remplace pas la synchronisation de l'horloge. Ne pas augmenter arbitrairement cette valeur si le decalage continue de croitre.

## Verification

| Verification                                            | Resultat                                                                                                                                                           |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Tests unitaires frontend, suite complete                | 51 tests reussis dans 13 fichiers                                                                                                                                  |
| Playwright : authentification, generation et resilience | 35 scenarios reussis sur mobile, tablette et desktop ; 1 test de menu mobile ignore sur desktop                                                                    |
| TypeScript et ESLint                                    | Reussis ; controle cible du transport relance apres la derniere correction                                                                                         |
| Prettier sur les fichiers modifies                      | Reussi                                                                                                                                                             |
| Tests Go `go test -count=1 ./...`                       | Reussis sur tous les packages                                                                                                                                      |
| `go vet ./...`, `staticcheck ./...`, `sqlc vet`         | Reussis                                                                                                                                                            |
| Build frontend `npm run build`                          | Reussi avec `VITE_API_BASE_URL=https://api.example.com` et `VITE_E2E_MODE=false` definis uniquement pour le processus de validation ; aucun fichier `.env` modifie |
| Detecteur de courses Go `go test -race -p 1 ./...`      | Reussi sur tous les packages, avec `GOMAXPROCS=2` pour limiter la memoire                                                                                          |
| Integration PostgreSQL `make test-integration`          | Reussie contre PostgreSQL local sur le port 5433                                                                                                                   |

Les premieres executions paralleles ont subi des delais de demarrage et de chargement, avec environ 370 Mo de RAM libre mesures. La validation frontend finale a ete executee avec un seul worker, sans allonger les delais ni desactiver les assertions. Deux ajustements issus des tests ont ete faits : conserver les `DOMException` de delai pendant l'annulation, et exposer `X-Request-ID` dans la simulation CORS comme le fait le vrai backend.

La compilation instrumentee Go a egalement ete reprise avec une concurrence reduite, puis la suite globale a reussi. Le build frontend signale encore des bundles de plus de 500 Ko ; cet avertissement de taille ne bloque pas la compilation et ne concerne pas le refus de session.

Les tests navigateur utilisent le mode E2E existant et une API interceptee. Ils prouvent les reprises du transport et le comportement des ecrans, pas une connexion Clerk reelle ni une generation OpenAI facturee. L'outil d'acces a l'onglet connecte a echoue a son initialisation ; aucun cookie ni profil de navigateur n'a ete extrait.

## Application locale

Le frontend Vite recharge les changements. Le backend Go doit etre relance pour charger le middleware modifie. Synchroniser durablement l'heure de Windows dans les parametres de date/heure et verifier que le service Windows Time fonctionne, puis recharger la vue d'ensemble.

Un controle final dans la session reelle consiste a ouvrir la vue d'ensemble, verifier les deux listes recentes, lancer une formation et observer le passage a la clarification puis la reprise de generation. Si un refus persiste, conserver le nouvel `X-Request-ID`, verifier l'heure et la configuration du processus effectivement lance.

## Sources officielles

- [Clerk : structure des jetons de session](https://clerk.com/docs/guides/sessions/session-tokens), en particulier `azp`, `iat`, `nbf` et `exp`.
- [Clerk : forcer le renouvellement du jeton](https://clerk.com/docs/guides/sessions/force-token-refresh), `getToken({ skipCache: true })`.
- [Clerk : verifier une session en Go](https://clerk.com/docs/guides/sessions/verifying).
- Code effectivement installe : `github.com/clerk/clerk-sdk-go/v2@v2.7.0/http/middleware.go`, `jwt/jwt.go` et `frontend/node_modules/openapi-fetch/src/index.js`.
