# Correctifs de coherence des generations

Date : 2026-09-18. Suite a `generation-status-audit-2026-09-17.md` et son annexe de reproduction.

## Perimetre

Les huit constats de l'audit sont traites dans le code backend, le contrat API et le frontend.
La cause historique exacte de la demande de production
`90bc4e76-7f7e-48ac-893f-0f50b5cd1993` reste non confirmee : aucun acces authentifie
aux donnees ou aux logs de cette demande n'a ete obtenu. Aucun deploiement ni
aucune reparation de donnees de production n'a ete effectue pendant ce travail.

## Correspondance Avec L'Audit

| Constat | Correction |
| --- | --- |
| Contenu persiste puis echec d'enqueue/finalisation | Le gestionnaire d'echec verifie la completude avant de propager un echec. Le reconciliateur periodique recupere les finalisations interrompues sans appel IA. |
| Completion formation/demande dans deux transactions | Une seule transaction verrouille la demande, verifie le contenu, puis complete les deux agregats. Une erreur annule les deux ecritures. |
| Callback d'une ancienne reprise | `generation_attempt` sur demandes et jobs ; une reprise incremente la tentative et annule les anciens jobs encore en attente. Les callbacks et transactions de workers obsoletes sont ignores. |
| Ecritures concurrentes et reponses IA tardives | Verrou de demande avant lecture/mutation ; relecture de la formation ; progression monotone ; controle du claim et de son expiration avant et apres chaque transaction de worker. |
| Contenu accessible masque par un echec technique | Projection `contentComplete`, presentation commune du suivi/historique/dashboard et lien vers la formation, meme partielle. Un contenu complet n'est plus presente comme une generation integralement perdue. |
| Caches terminaux figes | Revalidation au focus/reconnexion, invalidation du detail a la fin d'un job, polling des listes actives et polling borne de reconciliation. Les jobs d'anciennes tentatives ne contaminent plus le suivi courant. |
| Commandes partielles acceptees mais inexecutables | Refus avant enqueue lorsque la demande est en echec, sous le verrou de demande ; HTTP 409 via l'erreur applicative existante. La reprise globale reste la voie executable. |
| Message anglais generique | Codes publics `generation_failed` / `finalization_failed`, textes FR/EN, identifiant copiable et bouton Actualiser distinct d'une relance IA. Les messages internes restent masques dans les DTO. |

## Backend Et Invariants

- `internal/service/generation_transaction.go` applique l'ordre demande puis claim aux transactions des workers. Les appels IA restent hors transaction.
- Le claim est controle par ID du job, worker, compteur d'execution, statut et expiration SQL avec `clock_timestamp()`. Une perte de claim provoque un rollback.
- `internal/service/generation_completion.go` centralise la finalisation et la reconciliation idempotente. Chaque candidat est traite dans sa propre transaction ; une erreur n'annule pas les autres candidats.
- `internal/infrastructure/jobs/maintenance.go` lance la reconciliation au demarrage puis a chaque passage de maintenance.
- La tentative metier (`generationAttempt`) est distincte du compteur d'execution du job (`attemptCount`) et de la version des clarifications.
- Les erreurs historiques des jobs sont conservees. Reparer une demande ne supprime pas son historique et ne cree pas de nouvelle tentative IA.
- La completude garde la definition existante : au moins un module, au moins une lecon par module, et Markdown non vide ou exercices/quizzes pour chaque lecon. Ce n'est pas une nouvelle evaluation de qualite pedagogique.

## Migration Et Contrat

`migrations/00010_generation_consistency.sql` ajoute les tentatives, un index de
recherche des jobs par tentative et la vue `course_content_states`.
Les requetes sqlc ont ete regenerees a partir des sources SQL, sans edition manuelle
du code genere. OpenAPI et les types TypeScript derives sont synchronises.

Les DTO ajoutent `generationAttempt`, `contentComplete` et `failureCode` au suivi
et a l'historique ; les jobs exposent leur tentative. Ces ajouts restent optionnels
dans le schema pour la compatibilite de lecture pendant le deploiement. Le frontend
accepte l'ancien contrat, mais une valeur explicite `contentComplete: false` reste
prioritaire sur un ancien statut de formation `completed`.

Les lignes historiques recoivent la tentative initiale 1. La migration ne peut pas
reconstruire de facon fiable la tentative historique de chaque vieux job. Il faut
donc arreter les anciennes instances avant migration et inventorier/drainer le
travail historique actif avant remise en service. Ne pas laisser deux versions de
workers ecrire simultanement. La protection entre reprises est garantie pour les
tentatives creees avec le nouveau code, pas pour une chronologie ancienne absente
de la base.

## Reparation Controlee

La commande `cmd/reconcile-generations` utilise les memes invariants que le worker.
Sans `-apply`, elle effectue uniquement des lectures. Elle ne demande aucun appel IA.
Avec `-apply`, elle ecrit un instantane de statut avant de tenter la reparation,
reverifie le contenu et ignore les demandes avec travail actif dans la tentative
courante. Aucun UPDATE global `failed -> completed` n'est utilise.

Depuis `backend-go`, avec une connexion autorisee configuree sans exposer ses secrets :

```powershell
# Inventaire en lecture seule. Conserver la sortie avant toute application.
go run ./cmd/reconcile-generations -limit 100

# Inspection de l'exemple signale, sans mutation.
go run ./cmd/reconcile-generations -request-id 90bc4e76-7f7e-48ac-893f-0f50b5cd1993

# Apres revue de l'inventaire et sauvegarde de la base.
go run ./cmd/reconcile-generations -request-id 90bc4e76-7f7e-48ac-893f-0f50b5cd1993 -apply
```

Ces commandes de production ne sont pas executees automatiquement par ce rapport.
L'instantane JSON de la commande n'est pas une sauvegarde complete de la base.
La maintenance automatique du nouveau worker peut egalement reparer les candidats :
realiser l'inventaire avant de redemarrer les workers si une validation operateur
de chaque candidat est necessaire.

## Ordre De Mise En Service

1. Sauvegarder la base, relever les jobs actifs et les demandes concernees ; arreter/drainer les anciennes instances API/workers.
2. Appliquer la migration Goose 00010. Ne pas demarrer le nouveau binaire contre l'ancien schema.
3. Avec le nouveau code, effectuer l'inventaire via la commande en lecture seule avant de demarrer l'API/les workers, puis valider la reconciliation. Sur Railway, la migration pre-deploy ne remplace pas l'arret prealable des anciens writers.
4. Redemarrer les workers corriges et deployer le frontend compatible avec les nouvelles projections.
5. Verifier API de statut, historique, dashboard, catalogue et lecteur apres actualisation. Suivre `generation_completion_reconciliation_failed` et les evenements de claim/reprise existants.
6. Ne pas appliquer le Down de 00010 tant que le nouveau code tourne. Un rollback vers l'ancien code reintroduit les risques de concurrence de l'audit.

## Verification

Verification locale du correctif. Aucun appel IA reel ni deploiement de production.

- Suite Go complete, `go test -race -p 1 -count=1 ./...`, `go vet`, `staticcheck` et `sqlc vet` : reussis apres les correctifs applicatifs.
- `make test-integration` : reussi dans une base locale isolee avec migration 00010, y compris rollback atomique, concurrence finalisation/echec, expiration de lease, deux dernieres lecons simultanees et rejeu apres acquittement perdu.
- Commande de reconciliation en lecture seule : demarrage valide sur la base de test (aucun candidat residuel apres nettoyage des fixtures).
- Frontend : TypeScript, ESLint, formatage des fichiers du correctif et build reussis ; 60 tests unitaires reussis. Le build utilise uniquement une URL HTTPS de validation dans l'environnement du processus, sans modification des fichiers `.env`.
- Navigateur : suite complete sur mobile, tablette et desktop, 99 scenarios reussis et 3 exclusions conditionnelles prevues pour les tailles d'ecran non concernees. API simulee, pas de validation Clerk/OpenAI de production.
- Avertissement du build : certains chunks depassent 500 Ko apres minification ; ce chantier de decoupage des bundles n'est pas inclus dans les correctifs de statut.
- Reserve de formatage : `npm run format:check` signale 10 fichiers preexistants hors correctif, verifies sans diff Git : `docs/vercel-deployment.md`, `README.md`, `src/app/application.tsx`, `src/shared/config/environment.test.ts`, `src/shared/config/environment.ts`, `src/vite-env.d.ts`, `tsconfig.app.json`, `tsconfig.node.json`, `vercel.json`, `vite.config.ts`. Ils ne sont pas reformates dans ce changement.
- Base de test : `course_ai_status_test_20260917`. Les donnees de developpement existantes et de production n'ont pas ete reparees ni migrees.

## Tests De Non-Regression Ajoutes

| Couche | Fichier | Scenarios |
| --- | --- | --- |
| Service | `internal/service/generation_consistency_test.go` | Echec de coordination apres contenu complet ; callback de tentative obsolete ; commandes partielles refusees avant enqueue ; progression et etat terminal non regressifs ; execution obsolete sans persistance ; reconciliation historique sans nouvelle tentative. |
| PostgreSQL reel | `tests/integration/postgres/generation_consistency_test.go` | Rollback de la paire de statuts ; finaliseurs et callbacks d'echec concurrents ; expiration de lease pendant transaction ; attente des jobs actifs ; deux dernieres lecons simultanees, finaliseur unique et rejeu apres acquittement perdu. |
| DTO | `internal/infrastructure/http/dto/generation_test.go` | Code d'erreur selon completude, tentative exposee, absence de details internes. |
| Frontend unitaire | `src/features/generation/status-consistency.test.tsx` et `presentation.test.ts` | Acces au contenu ; revalidation au focus ; invalidation detail/historique ; polling de liste ; completude explicite ; limite de polling ; exclusion des anciennes tentatives. |
| Polling | `src/shared/api/polling.test.ts` | Fenetre de reconciliation independante de la duree de generation ; reinitialisation apres reprise ; arret sur etat terminal stable et isolation entre requetes. |
| Navigateur | `tests/e2e/status-consistency.spec.ts` | Avertissement localise, lien vers la formation, absence du message anglais et de relance inutile, actualisation GET sans POST. |

## Limites Connues

- Les tests navigateur utilisent l'API simulee du projet ; ils ne valident pas les identifiants Clerk/OpenAI de production.
- Le bouton Actualiser relit les projections, il ne force ni reparation de base ni nouvelle generation payante.
- Le polling de reconciliation est borne a 30 observations apres l'entree dans l'etat a reconcilier, et non depuis le debut de la generation. Cela couvre aussi une generation longue. La reprise d'un travail actif reinitialise ce compteur ; focus, reconnexion et actualisation permettent toujours une nouvelle lecture.
- Une formation dont le contenu est incomplet reste un echec reel possible et conserve son action de reprise.
