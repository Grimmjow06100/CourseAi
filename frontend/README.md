# Course AI Frontend

Client React/Vite de Course AI. L'application permet de lancer une génération asynchrone, répondre aux clarifications, consulter l'historique, parcourir ses formations et lire chaque leçon.

## Stack

- React 19, TypeScript strict et Vite
- TanStack Router et TanStack Query
- Clerk React
- Tailwind CSS v4 et shadcn/ui (Radix), polices Geist locales
- React Hook Form et Zod
- i18next (français et anglais)
- React Markdown, GFM, Prism et Mermaid
- Vitest, Testing Library, MSW et Playwright

## Prérequis

- Node.js 24.x (version alignée entre `.nvmrc`, `package.json` et la CI)
- le backend Course AI sur `http://localhost:8080`
- une application Clerk avec une publishable key

## Installation

```powershell
cd frontend
Copy-Item .env.example .env.local
npm ci
npm run api:generate
npm run dev
```

Configurer `.env.local` :

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_CLERK_PUBLISHABLE_KEY=pk_test_xxx
VITE_E2E_MODE=false
```

Ne jamais activer `VITE_E2E_MODE` hors des tests Playwright locaux. Ce mode est aussi conditionné à `import.meta.env.DEV` et n'est pas utilisable dans un build de production.

Dans Clerk, autoriser `http://localhost:5173`. Dans le backend, inclure cette origine dans `CLERK_AUTHORIZED_PARTIES` et `CORS_ALLOWED_ORIGINS`.

## Commandes

```powershell
npm run dev
npm run typecheck
npm run lint
npm run format:check
npm test
npm run build
npm run test:e2e
npm run api:generate
npm run api:check
```

`api:generate` produit `src/shared/api/schema.gen.ts` depuis l'OpenAPI Go. Ce fichier est généré et ne doit pas être modifié à la main.

## Parcours de génération

1. `POST /api/generations` retourne immédiatement un `requestId` et le frontend ouvre la page de suivi.
2. Le suivi utilise un instantané léger, avec polling pendant le travail actif et conservation des données en cas de perte réseau.
3. Sur `awaiting_clarification`, le polling s'arrête. Le formulaire renvoie exclusivement les `value` des options proposées.
4. Après validation, les jobs durables poursuivent le travail jusqu’au résultat terminé, partiel ou échoué.
5. La page de suivi permet des reprises ciblées et groupées idempotentes, en conservant les contenus disponibles.
6. Les corrections sont chargées via `/solutions` uniquement après une action explicite.

## Déploiement Vercel

Suivre le [guide de mise en production](docs/vercel-deployment.md) : import du dépôt avec **Root Directory = `frontend`**, configuration Vercel, instance Clerk de production, API HTTPS, CORS et vérification des parcours réels. Le fichier `vercel.json` fixe Vite, `npm ci`, `npm run build`, `dist` et le routage SPA. Le modèle `.env.production.example` contient les trois variables publiques à renseigner.

### Contrôles avant publication

- Choisir `frontend` comme **Root Directory**, `npm run build` comme commande et `dist` comme dossier de sortie.
- Les variables sont publiques et intégrées au bundle : toute modification nécessite un nouveau déploiement. Ne jamais ajouter une clé secrète Clerk ou OpenAI dans une variable `VITE_*`.
- Le build valide la configuration avec Zod et refuse une API non HTTPS, une URL contenant `/api`, une clé Clerk factice ou `VITE_E2E_MODE=true`. Sur `VERCEL_ENV=production`, une clé `pk_live_*` est obligatoire. Les previews peuvent utiliser une instance Clerk de test avec les origines correspondantes. La validation de syntaxe ne vérifie pas l'existence de l'instance Clerk.
- `vercel.json` fournit le rewrite SPA, `nosniff`, une politique de référent et une CSP minimale interdisant l'intégration du frontend dans une iframe. Une CSP restrictive complète nécessite d'inventorier les domaines de l'instance Clerk réelle.
- Déployer d’abord le backend et les migrations 00011–00013 ; suivre le [guide de livraison](../docs/site-refactor-release.md).
- Les tests Playwright utilisent un serveur dédié sur `4180`, deux workers et une API simulée. Ils ne valident pas les paramètres de votre instance Clerk, Railway ou Vercel.
- Avant ouverture publique : tester connexion/déconnexion et changement de compte avec Clerk réel, les liens profonds après rechargement, CORS, une génération complète et une relance après échec. Configurer aussi le suivi des erreurs frontend dans votre outil d'observabilité.

Voir [`docs/application-hardening.md`](docs/application-hardening.md) pour les corrections applicatives et les limites restantes.

Architecture et fonctionnement : [refonte shadcn/ui et suivi](docs/site-refactor.md).
