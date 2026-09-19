# Architecture du frontend

## Objectifs

Le frontend traite l'API Go comme seule source de vérité. Il ne persiste ni JWT, ni génération, ni progression pédagogique dans le navigateur. L'état local est réservé aux interactions éphémères ; la langue et le thème sont des préférences client persistées.

## Carte des packages

```text
src/
  app/                    composition Clerk, Query, Router et shell
  routes/                 fichiers TanStack Router, sans logique réutilisable
  features/auth/          écran Clerk et garde d'authentification
  features/generation/    création, clarification, statut et historique
  features/catalog/       catalogue, cours et accès aux contenus
  features/course-reader/ lecteur, activités et révélation des solutions
  components/ui/          primitives shadcn/ui et variantes communes
  shared/api/             transport OpenAPI, cache keys, erreurs et types générés
  shared/config/          validation Zod de l'environnement
  shared/i18n/            ressources FR/EN
  shared/ui/              composants visuels génériques
```

`shared` et les primitives UI ne dépendent d'aucune feature. Les routes composent des features, et les features n'importent ni les routes ni `app`. Les dépendances entre features suivent `course-reader -> catalog -> generation` ; le lecteur peut aussi dépendre directement de `generation`. `auth` et `generation` restent indépendantes des autres features. La règle locale ESLint `architecture` vérifie ces frontières, y compris les chemins relatifs, réexports et imports dynamiques littéraux.

Les hooks API gèrent transport et cache. Les formulaires ou écrans gèrent la navigation après une mutation. Les fonctions pures comme `tracking-state.ts` et `catalog/polling.ts` expriment les politiques d'affichage et de rafraîchissement ; elles sont testables sans monter React. `tracking-operation.tsx` contient les petits composants métier réutilisés par le suivi et le lecteur. La taille d'un fichier ne justifie pas à elle seule une nouvelle abstraction.

## Authentification

`ClerkProvider` encadre l'application. Une fois Clerk chargé, `useAuth().getToken(options)` est injecté dans l'adaptateur `openapi-fetch`. Le transport demande un jeton actuel à chaque appel et ajoute `Authorization: Bearer`; aucun jeton n'est écrit dans `localStorage`.

Après un premier HTTP 401, le transport demande `getToken({ skipCache: true })` et rejoue la requête une seule fois, avec le même corps et la même clé d'idempotence. Les requêtes simultanées partagent le renouvellement en cours au sein de leur client de session. Les autres erreurs HTTP et les erreurs réseau ne déclenchent pas cette reprise. L'annulation et le délai global de 30 secondes couvrent aussi l'attente de Clerk et la seconde tentative.

Un jeton absent renvoie le code client `session_expired` et invite à se connecter. Un 401 persistant de l'API affiche un refus de validation avec son identifiant de requête et propose aussi de réessayer lorsque l'écran dispose de cette action. Un 401 seul ne prouve pas que la session Clerk a expiré : vérifier également l'origine autorisée, les clés et l'horloge serveur. Le middleware Go tolère un décalage horaire de 15 secondes ; la machine doit rester synchronisée.

Le contexte du routeur reçoit seulement `isSignedIn`. Le layout `_authenticated` applique sa garde dans `beforeLoad` et redirige vers `/sign-in`. Les routes Clerk dédiées gèrent connexion et inscription.

Chaque couple `userId` / `sessionId` possède son propre `QueryClient`, son routeur et son `SessionLifetime`. Un changement de compte démonte ce runtime, annule les requêtes et mutations HTTP en cours, puis vide son cache. Aucune donnée privée de la session précédente n'est réutilisée. Le client refuse les appels sans jeton et combine annulation de session, annulation TanStack Query et délai maximal de 30 secondes. Ce délai ne coupe pas les jobs durables du backend : les commandes HTTP retournent déjà `202` rapidement.

## État serveur et API

TanStack Query possède tout l'état distant. Les clés sont centralisées dans `shared/api/query-keys.ts`, incluent les filtres et servent de cible aux invalidations. Ce module dépend du contrat API, jamais des types d'une feature. Les invalidations indépendantes sont concurrentes ; supprimer une génération retire aussi ses snapshots et événements du cache. `openapi-fetch` compile chaque route, paramètre et payload contre `schema.gen.ts`. `ApiError` normalise le statut HTTP, le code public et `X-Request-ID`.

Le suivi, la vue de cours et le lecteur utilisent `GET /api/generations/:requestID/tracking`. Ce snapshot contient les opérations courantes, phases, modules, leçons, compteurs et autorisations de reprise. `newestTracking` rejette les réponses d'une tentative ou révision antérieure ; les révisions sont comparées avec `BigInt`. Le backend reste responsable du statut global et des opérations relançables. Un incident local ne transforme pas automatiquement la génération en échec.

Le suivi interroge le snapshot toutes les deux secondes pendant les travaux actifs. Les situations de réconciliation bénéficient d'un polling borné ; les erreurs transitoires conservent le dernier snapshot et espacent les lectures à dix secondes. Les refus 401/403/404 arrêtent cette boucle. Focus et reconnexion permettent une nouvelle lecture. Le catalogue ne considère pas un résultat `partial` comme un travail actif. Un échec historique bénéficie seulement de la fenêtre de récupération bornée définie dans `shared/api/polling.ts`.

Un changement du statut, de la disponibilité des leçons ou de la tentative invalide cours et historique. Les gros documents Markdown ne sont donc pas téléchargés à chaque tick. Les événements sont paginés séparément, à la demande. Les reprises ciblées utilisent les identifiants et versions des opérations, avec une clé d'idempotence conservée après une réponse incertaine. L'ancien suivi frontend fondé sur la liste brute des jobs et les mutations de contenu sans consommateur ont été retirés ; les endpoints backend restent disponibles.

La lecture ordinaire n'appelle jamais `/solutions`. Le bouton d'une activité active cette query et n'affiche que la correction choisie. Revenir sur une leçon n'affiche pas automatiquement des réponses déjà présentes dans le cache.

Une clé d'idempotence est conservée en mémoire par prompt normalisé tant que le formulaire reste monté. Une nouvelle tentative identique réutilise cette clé après une réponse incertaine ; changer le prompt crée une autre clé. Un rechargement du formulaire réinitialise cette mémoire : consulter l'historique avant de resoumettre une demande dont la réponse a été perdue.

## Routage et URL

- `/` : tableau de bord
- `/generate` : nouvelle formation
- `/generations` : historique paginé
- `/generations/$requestId` : suivi et clarifications
- `/courses` : catalogue
- `/courses/$courseId` : vue d'ensemble
- `/courses/$courseId/lessons/$lessonId` : lecteur

Recherche, filtres, tri et page du catalogue sont validés avec Zod et conservés dans l'URL. Ils peuvent donc être partagés, rechargés et parcourus avec l'historique du navigateur.

## Rendu pédagogique

Le Markdown accepte GFM mais pas le HTML brut. `rehype-sanitize` reste actif. Les blocs Mermaid chargent la dépendance à la demande et utilisent `securityLevel: strict`. Le code est rendu par Prism. Le lecteur présente une leçon à la fois, une navigation précédent/suivant et un curriculum permanent sur desktop ou dans un tiroir sur mobile.

## Tests

Vitest couvre les schémas Zod, la garde, les query keys, les politiques de polling, la cohérence des snapshots, la navigation du formulaire, l'ajout du bearer token et les erreurs API avec MSW. `npm run lint` teste la règle d'architecture avec ESLint puis l'applique aux sources. Playwright intercepte le backend et vérifie le parcours création, clarification, lecture, contenu partiel et révélation. Les projets tournent à 390, 768 et 1440 pixels et contrôlent débordement horizontal, erreurs console, accessibilité critique et navigation clavier.

Le mode `VITE_E2E_MODE` fournit une identité uniquement lorsque Vite est en mode développement. La condition `import.meta.env.DEV` rend ce mécanisme inactif dans les builds de production.

## Déploiement

Vercel sert le SPA et Railway l'API. Pour chaque domaine frontend :

1. enregistrer l'origine dans Clerk ;
2. l'ajouter à `CLERK_AUTHORIZED_PARTIES` ;
3. l'ajouter à `CORS_ALLOWED_ORIGINS` ;
4. vérifier que `VITE_API_BASE_URL` utilise HTTPS ;
5. exécuter le parcours réel avec un utilisateur Clerk après déploiement.
