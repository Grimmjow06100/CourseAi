# Déployer le frontend Course AI sur Vercel

Le frontend est une SPA React/Vite. Vercel sert `frontend/dist`, tandis que l'API Go et ses workers restent sur leur hébergement backend. Les appels `/api/...` partent directement vers `VITE_API_BASE_URL` ; il n'y a pas de proxy API dans Vercel.

## 1. Préparer les services et le domaine

- Disposer de l'origine HTTPS de l'API déployée, par exemple `https://api.example.com`, sans `/api`, chemin, query string ou identifiants.
- Choisir le domaine frontend définitif, par exemple `https://app.example.com`, et le rattacher au projet Vercel lors de sa création.
- Créer l'instance **Production** de Clerk pour le domaine possédé, ajouter les enregistrements DNS demandés et terminer l'activation des certificats. Configurer les identifiants OAuth propres à la production si les connexions sociales sont activées. Copier la clé publique `pk_live_...` depuis cette instance. Ces étapes sont détaillées dans le [guide de production Clerk](https://clerk.com/docs/guides/development/deployment/production).
- L'application gère `/sign-in` et `/sign-up`, ainsi que leurs sous-routes. Configurer ces chemins dans Clerk si nécessaire et vérifier les retours OAuth sur le domaine définitif.

Une clé bien formée ne prouve pas qu'une instance Clerk existe ou que son domaine est opérationnel. Les exemples de CI sont des fixtures, pas des identifiants utilisables.

## 2. Importer le dépôt

Dans Vercel, importer le dépôt Git puis utiliser les paramètres suivants avant de lancer le premier déploiement :

| Paramètre         | Valeur                                          |
| ----------------- | ----------------------------------------------- |
| Root Directory    | `frontend`                                      |
| Framework Preset  | Vite                                            |
| Node.js           | 24.x                                            |
| Install Command   | `npm ci`                                        |
| Build Command     | `npm run build`                                 |
| Output Directory  | `dist`                                          |
| Production Branch | La branche de publication choisie pour le dépôt |

Les commandes et le framework sont déclarés dans `frontend/vercel.json`. La racine se règle dans le projet Vercel. La version majeure Node est fixée par `package.json` et partagée avec la CI via `.nvmrc` ; Vercel prend en charge [Node 24.x](https://vercel.com/docs/functions/runtimes/node-js/node-js-versions).

Le rewrite vers `index.html` permet le rechargement des liens profonds TanStack Router, selon la [configuration SPA Vite de Vercel](https://vercel.com/docs/frameworks/frontend/vite). Les en-têtes existants appliquent `nosniff`, une politique de référent et une CSP minimale interdisant les iframes, les objets intégrés et les bases externes.

## 3. Définir les variables Vercel

Dans **Settings → Environment Variables**, définir les valeurs réelles pour **Production** :

| Variable                     | Valeur attendue                                               |
| ---------------------------- | ------------------------------------------------------------- |
| `VITE_API_BASE_URL`          | Origine HTTPS de l'API Go, sans `/api`                        |
| `VITE_CLERK_PUBLISHABLE_KEY` | Clé publique de l'instance Clerk de production, `pk_live_...` |
| `VITE_E2E_MODE`              | `false`                                                       |

Laisser activée l'exposition automatique des variables système de Vercel : le build utilise `VERCEL_ENV=production` pour imposer la clé Clerk live. Vercel fournit cette variable ; ne pas lui substituer une valeur de preview. Voir les [variables système Vercel](https://vercel.com/docs/environment-variables/system-environment-variables).

Les variables `VITE_*` sont publiques et intégrées au JavaScript au moment du build. Après une modification, reconstruire et redéployer. Ne jamais ajouter `CLERK_SECRET_KEY`, `OPENAI_API_KEY` ou `DATABASE_URL` au frontend. Les fichiers `.env*` réels et le dossier `.vercel` sont ignorés par Git ; seuls les modèles restent versionnés.

Pour **Preview**, configurer séparément une API HTTPS de staging et une clé `pk_test_...` réelle, avec `VITE_E2E_MODE=false`. Utiliser une origine de preview stable autorisée côté backend ; une URL de déploiement aléatoire ne sera pas automatiquement autorisée. Ne pas ouvrir CORS avec `*` pour contourner cette configuration.

## 4. Aligner le backend et Clerk

Sur l'hébergement de l'API, configurer les valeurs suivantes pour l'environnement concerné :

```env
APP_ENV=production
CLERK_SECRET_KEY=<clé secrète de la même instance Clerk que le frontend>
CLERK_AUTHORIZED_PARTIES=https://app.example.com
CORS_ALLOWED_ORIGINS=https://app.example.com
```

Les origines incluent le protocole, sans chemin ni slash final. Si plusieurs origines sont réellement nécessaires, les séparer par des virgules. Pour une preview, aligner l'API de staging sur l'instance Clerk de test et l'origine de preview. Déployer les migrations et la version API compatibles avec le client ; le backend doit notamment exposer `GET /api/generations/:requestID/jobs`.

La clé secrète reste exclusivement côté backend. L'authentification API utilise le bearer token Clerk et la vérification du propriétaire ; cette application n'a pas besoin d'un webhook de synchronisation des profils.

## 5. Vérifier avant publication

Depuis `frontend`, avec Node 24 et de vraies valeurs dans `.env.production.local` (copié depuis `.env.production.example`) :

```powershell
npm ci
npm run api:check
npm run typecheck
npm run format:check
npm run lint
npm test
$env:VERCEL_ENV = 'production'
npm run build
Remove-Item Env:VERCEL_ENV
npx playwright install chromium
npm run test:e2e
```

Le build échoue avant la génération du bundle si les variables sont absentes, la clé Clerk est mal formée, l'API contient un chemin ou utilise HTTP, ou le mode E2E est activé. En production Vercel, une clé Clerk de test est aussi refusée.

La CI GitHub vérifie les types OpenAPI, TypeScript, le formatage, ESLint, les tests unitaires, le build avec les exigences Vercel Production et les tests navigateur sur trois tailles d'écran. Elle conserve les rapports et traces en cas d'échec. Le build CI utilise une clé live de syntaxe valide mais factice ; les tests navigateur utilisent une API simulée et l'identité de test réservée au serveur de développement.

Exiger la réussite du job `Frontend CI / verify` dans les règles de fusion de la branche de publication. La présence de ce workflow ne bloque pas à elle seule une publication automatique Vercel : configurer la stratégie de publication dans Vercel et le dépôt selon le flux retenu.

## 6. Recette sur le domaine déployé

1. Charger `/sign-in`, `/sign-up` puis recharger directement `/courses` et une URL de leçon. Vérifier le chargement des fichiers JS/CSS, l'absence de 404 et la redirection de l'utilisateur déconnecté.
2. Avec Clerk réel, tester inscription, connexion, éventuel retour OAuth, déconnexion puis connexion avec un second compte. Vérifier que les formations du premier compte ne restent pas visibles.
3. Dans le réseau du navigateur, vérifier que les requêtes partent vers l'API HTTPS et incluent le bearer token. Une requête API sans token doit être refusée ; une requête authentifiée depuis l'origine autorisée doit passer CORS.
4. Générer une formation courte : création, clarification éventuelle, suivi, rechargement pendant la génération et lecture. Cette étape consomme les ressources normales du backend et du fournisseur IA.
5. Vérifier mobile et desktop, thème, changement de langue, solutions et messages d'erreur. Contrôler les en-têtes de réponse Vercel et les erreurs réseau/console.

Ces contrôles distants restent nécessaires après le premier déploiement : les tests locaux ne valident ni DNS, ni certificats, ni les clés réelles, ni CORS entre les services déployés.

## Vérification locale du 14 septembre 2026

- TypeScript, ESLint, Prettier et régénération OpenAPI sans différence : réussis.
- Compatibilité manifeste/lockfile : `npm ci --dry-run --ignore-scripts --no-audit --no-fund` réussi. Les versions des dépendances restent inchangées ; aucune installation Linux Vercel n'a été exécutée localement.
- Build Vite avec `VERCEL_ENV=production`, API HTTPS factice et clé live de test syntaxique : réussi. Deux builds négatifs ont bien refusé une clé `pk_test_...` et `VITE_E2E_MODE=true`.
- 37 tests unitaires distincts réussis : 36 lors de la suite complète, puis le test de confirmation lors d'une reprise ciblée après expiration du délai de démarrage de son worker Vitest local.
- 23 tests navigateur fonctionnels et de résilience réussis sur mobile, tablette et desktop ; le test de navigation mobile est volontairement ignoré sur desktop. Le rapport CI HTML est désormais activé.
- Aucun identifiant E2E recherché ni fichier source map dans `dist`. Le bundle principal reste à environ 603 Ko minifiés (174 Ko gzip) ; Vite signale aussi un gros chunk Mermaid chargé à la demande. Cette optimisation reste une piste de performance.

Le compte Vercel connecté ne présentait aucun projet au moment de l'inspection. Aucun projet n'a été créé ni publié. Le domaine frontend, l'API distante et les clés Clerk réelles restent à renseigner avant la recette de production décrite ci-dessus.

## Retour arrière et diagnostic

En cas de régression frontend, restaurer le dernier déploiement validé depuis Vercel. Les variables étant compilées dans les assets, un changement de clé ou d'API nécessite un nouveau build ; une restauration frontend ne restaure pas les migrations ni les données backend.

| Symptôme                                | Vérification                                                            |
| --------------------------------------- | ----------------------------------------------------------------------- |
| `Invalid frontend environment` au build | Valeurs réelles, format de la clé publique et origine API sans `/api`   |
| `live Clerk publishable key`            | Clé `pk_live_...` dans l'environnement Production                       |
| Connexion Clerk indisponible            | Instance, domaine, DNS, certificats et configuration OAuth              |
| API 401 après connexion                 | Même instance Clerk des deux côtés, token et `CLERK_AUTHORIZED_PARTIES` |
| Erreur CORS                             | Origine frontend exacte dans `CORS_ALLOWED_ORIGINS` côté API            |
| 404 après rechargement                  | Root Directory `frontend` et prise en compte de `vercel.json`           |
