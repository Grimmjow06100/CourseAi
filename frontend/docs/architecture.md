# Architecture du frontend

## Objectifs

Le frontend traite l'API Go comme seule source de vérité. Il ne persiste ni JWT, ni génération, ni progression pédagogique dans le navigateur. L'état local est réservé aux interactions éphémères ; seule la langue est une préférence client persistée.

## Carte des packages

```text
src/
  app/                    composition Clerk, Query, Router et shell
  routes/                 fichiers TanStack Router, sans logique réutilisable
  features/auth/          écran Clerk et garde d'authentification
  features/generation/    création, clarification, statut et historique
  features/catalog/       catalogue, cours et accès aux contenus
  features/course-reader/ lecteur, activités et révélation des solutions
  shared/api/             client OpenAPI, erreurs et types générés
  shared/config/          validation Zod de l'environnement
  shared/i18n/            ressources FR/EN
  shared/ui/              composants visuels génériques
```

`shared` ne dépend d'aucune feature. Les routes composent des features, et les features n'importent ni les routes ni `app`. ESLint vérifie ces frontières.

## Authentification

`ClerkProvider` encadre l'application. Une fois Clerk chargé, `useAuth().getToken()` est injecté dans l'adaptateur `openapi-fetch`. Le middleware demande ainsi un jeton actuel à chaque appel et ajoute `Authorization: Bearer`; aucun jeton n'est écrit dans `localStorage`.

Le contexte du routeur reçoit seulement `isSignedIn`. Le layout `_authenticated` applique sa garde dans `beforeLoad` et redirige vers `/sign-in`. Les routes Clerk dédiées gèrent connexion et inscription.

Chaque couple `userId` / `sessionId` possède son propre `QueryClient`, son routeur et son `SessionLifetime`. Un changement de compte démonte ce runtime, annule les requêtes et mutations HTTP en cours, puis vide son cache. Aucune donnée privée de la session précédente n'est réutilisée. Le client refuse les appels sans jeton et combine annulation de session, annulation TanStack Query et délai maximal de 30 secondes. Ce délai ne coupe pas les jobs durables du backend : les commandes HTTP retournent déjà `202` rapidement.

## État serveur et API

TanStack Query possède tout l'état distant. Les clés sont centralisées dans `query-keys.ts`, incluent les filtres et servent de cible aux invalidations. `openapi-fetch` compile chaque route, paramètre et payload contre `schema.gen.ts`. `ApiError` normalise le statut HTTP, le code public et `X-Request-ID`.

Le polling de génération retourne `2000` uniquement pour `queued` et `running`, sinon `false`. Une erreur de lecture arrête le polling automatique après les tentatives configurées et propose une relance explicite.

Les générations partielles utilisent `GET /api/generations/:requestID/jobs`, dont le service vérifie le propriétaire Clerk avant de lire les jobs. Cette réponse ne contient ni payload privé, ni sortie brute IA. La liste est partagée dans le cache et réinterrogée toutes les deux secondes tant qu'un job est `queued`, `running` ou `retry_scheduled`, avec reprise à la remise au premier plan. Elle restaure le suivi après un rechargement sans conserver d'identifiant de job dans le navigateur.

Un job de module terminé signifie que ses jobs enfants ont été créés, pas que les leçons sont prêtes. Le lecteur et le curriculum tiennent donc compte des jobs `lesson_content` ciblant les leçons du module. Le dernier job de chaque couple type/cible remplace ses anciens essais. Une transition terminale invalide les contenus et les listes ; les gros documents Markdown ne sont plus téléchargés à chaque tick de polling. Un échec propose le suivi de la demande et sa relance, au lieu de laisser un spinner sans fin.

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

Vitest couvre les schémas Zod, la garde, les query keys, l'ajout du bearer token et les erreurs API avec MSW. Playwright intercepte le backend et vérifie le parcours création, clarification, lecture, contenu partiel et révélation. Les projets tournent à 390, 768 et 1440 pixels et contrôlent débordement horizontal, erreurs console, accessibilité critique et navigation clavier.

Le mode `VITE_E2E_MODE` fournit une identité uniquement lorsque Vite est en mode développement. La condition `import.meta.env.DEV` rend ce mécanisme inactif dans les builds de production.

## Déploiement

Vercel sert le SPA et Railway l'API. Pour chaque domaine frontend :

1. enregistrer l'origine dans Clerk ;
2. l'ajouter à `CLERK_AUTHORIZED_PARTIES` ;
3. l'ajouter à `CORS_ALLOWED_ORIGINS` ;
4. vérifier que `VITE_API_BASE_URL` utilise HTTPS ;
5. exécuter le parcours réel avec un utilisateur Clerk après déploiement.
