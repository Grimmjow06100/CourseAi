# Rapport de refonte du site — 2026-09-19

Source : cahier des charges du vault `Technical/course-ai-site-refonte-cahier-des-charges-2026-09-19.md`.
Référence Git avant refonte : `66503f7` sur `main` et `origin/main`.

## Périmètre livré

- Statut partiel persisté, échecs locaux isolés, réconciliation des anciens échecs.
- Opérations courantes versionnées, protection des anciens workers, reprises unitaires/groupées atomiques et idempotentes.
- Instantané léger cohérent, révisions et événements publics paginés persistés ; OpenAPI et DTO régénérés.
- Toutes les pages migrées vers shadcn/ui, thèmes neutres clair/sombre/système, Geist local, accents verts sobres.
- Navigation responsive, historique et disponibilité, suivi détaillé, curriculum repliable, activités et corrections à la demande.
- Logo Course AI commun et favicon SVG versionné ; suppression des assets de template.
- Guide de migration/repli et note de livraison dans le vault.

Les modifications préexistantes à AGENTS.md et TASKS.md, qui décrivaient le cahier des charges et les règles shadcn/ui, sont conservées.

## Décisions structurantes

La barrière existante entre tous les plans de modules et les contenus est conservée.
Une opération de contenu produit texte et activités. La reprise ciblée conserve
la tentative globale et remplace une opération versionnée ; la reprise globale
crée une nouvelle tentative. Les contenus acquis sont conservés.

Le statut du pipeline, la disponibilité du contenu et les opérations sont séparés.
Les GET ne déclenchent pas de génération. Le suivi ne transporte ni Markdown,
ni corrections, ni prompt, ni diagnostic fournisseur.

## Validation backend

`gofmt`, `go vet ./...`, `staticcheck ./...`, `go test -count=1 ./...`,
`go test -race ./...`, `sqlc vet` et `make test-integration` passent.
Le dernier ajustement legacy a également été revérifié par tests de service/race,
analyses statiques et tests PostgreSQL ciblés.

Les migrations 00001–00013 sont appliquées uniquement à la base isolée
`course_ai_refactor_20260919` sur PostgreSQL local, port 5433.

Les tests couvrent notamment : échec local avec tâches sœurs actives, issue partielle,
ancien échec sans jobs conservés, propriétaire distinct, confidentialité, pagination,
huit reprises groupées concurrentes avec replay, conflit de version, rollback de
capacité, quota sur une demande active, callback obsolète, contenu déjà acquis,
plans incomplets et erreur permanente de configuration. Les routes métier restent
protégées par Clerk ; les tests HTTP vérifient aussi absence de cache, révision en
chaîne, paramètres invalides et transmission de la version/clé de reprise.

Le scénario PostgreSQL de 500 leçons produit environ 50 ko de métadonnées, sans
corps de contenu ni requêtes par leçon. Le navigateur vérifie également 20 modules
et 500 leçons avec ouverture progressive et compteurs exacts.

## Validation frontend

TypeScript, ESLint, Prettier, génération OpenAPI et build de production passent.
Les 63 tests unitaires applicatifs passent : 47 dans la suite finale puis 16 dans
le fichier relancé après un timeout de démarrage du worker Vitest sous Windows.
Cette reprise concernait le worker, sans assertion applicative en échec.

La recette Playwright principale compte 104 cas applicables réussis, sur les
thèmes clair/sombre et les formats 390, 768 et 1440 px, avec un contrôle à 320 px.
Elle couvre création, clarification, authentification expirée, isolation de session,
solutions à la demande, statuts partiels, perte réseau/reconnexion, reprise avec
réponse perdue, URL, clavier et absence de violations Axe sérieuses/critiques.
Les cinq échecs du premier passage ont été corrigés et repassés avec succès.

Huit contrôles navigateur supplémentaires passent pour le menu d’actions et son
focus, la reprise de l’historique à sa page la plus récente après pagination,
le programme repliable et le favicon. Au total : **112 scénarios applicables
validés**, avec huit exclusions intentionnelles pour les contrôles réservés à
un format d’écran. Le premier passage des nouveaux tests d’historique utilisait
une fixture vide ; la fixture a été corrigée puis les trois formats revérifiés.

Le contrôle visuel du favicon vérifie son chargement et sa lisibilité en 16/32 px,
sur fond blanc et sombre. Les captures de recette sont conservées dans les
artefacts locaux Codex et les sorties Playwright, pas dans le bundle.

## Mesure reproductible du JavaScript initial

Construire les deux révisions avec leur lockfile, Node 24, Vite 8.2.2 et les mêmes
variables publiques, avec `vite build --manifest`. Puis lancer :

```powershell
node scripts/compare-initial-bundles.mjs <baseline-dist> dist
```

Le script compte une fois chaque dépendance statique de l’entrée et des routes
nécessaires à l’accueil authentifié, y compris le shell chargé par TanStack Router.
Les imports des autres pages et Mermaid restent différés. Les scripts Clerk
externes, les polices et le CSS ne font pas partie de ce budget JavaScript.
Le seuil de 10 % sur le premier écran est vérifié par le script.

| JavaScript chargé | Référence minifiée / gzip | Refonte minifiée / gzip | Évolution minifiée / gzip |
| --- | --- | --- | --- |
| Amorçage | 801 108 / 236 584 octets | 831 872 / 249 176 octets | +3,84 % / +5,32 % |
| Accueil authentifié et shell | 931 007 / 287 339 octets | 999 450 / 313 421 octets | +7,35 % / +9,08 % |

Le budget est respecté. Le supplément correspond notamment à la navigation et
aux composants accessibles shadcn/Radix. Sonner est chargé uniquement sur le
suivi. Le chunk principal de 624,35 ko et certains chunks Mermaid différés
déclenchent encore l’avertissement Vite de 500 ko ; leur taille est documentée,
sans masquer cet avertissement ni charger Mermaid au démarrage.

## Limites de la recette locale

Playwright utilise une API simulée et le mode E2E exclusivement de développement.
La recette ne valide pas OAuth réel, les secrets/CORS distants, un appel IA réel,
le cache favicon distant ni le succès des déploiements Railway/Vercel.
Aucune migration de production n’a été exécutée pendant ce travail.

Suivre [la procédure de livraison](site-refactor-release.md) : migrations et backend
avant activation du frontend, puis recette réelle sur `https://www.courseai.site`.
