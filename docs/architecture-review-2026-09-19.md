# Revue d'architecture Course AI — 19 septembre 2026

## Objectif et périmètre

La revue couvre les frontières et les parcours principaux du backend Go et du frontend React : domaine, services, contrats, adaptateurs HTTP/PostgreSQL/OpenAI, workers, composition, authentification, transport, cache, génération, catalogue, lecteur et composants UI. Elle conserve l'architecture existante et recherche des améliorations concrètes de lisibilité, cohésion, couplage et testabilité.

Il s'agit d'une revue de code et de vérifications locales, pas d'un audit d'intrusion ni d'une validation des services de production. Les contrats HTTP, migrations, modèles de données et bibliothèques UI restent compatibles.

## Constats et corrections

| Constat | Correction | Effet attendu |
| --- | --- | --- |
| Le fichier central du générateur mélange construction, lectures, écritures, validation et admission ; le fichier des jobs contient aussi leur planification. | Regroupement des méthodes existantes dans des fichiers dédiés aux commandes, requêtes, contexte, persistance, admission, brief, planification et état du cours. | Retrouver une responsabilité sans ajouter de services intermédiaires ni déplacer les règles hors du domaine. |
| Le handler découvre implicitement le suivi par conversion de son port de lecture. | Injection explicite des trois ports : commande, requête, suivi. | Dépendances visibles dans le constructeur et dans `cmd/api`. |
| Une ancienne préparation de reprise n'est plus appelée en production. | Suppression du chemin mort ; test de reprise à travers `EnqueueStructureRetry`. | Une seule orchestration de reprise à maintenir. |
| Les lectures de suivi ne vérifient pas les dépendances du service comme les autres méthodes publiques. | Garde commune et test des services absents/incomplets. | Erreur applicative explicite au lieu d'une panique. |
| L'idempotence compare le prompt brut alors que le domaine en normalise les espaces. | Comparaison après la même normalisation, en conservant la limite de longueur sur l'entrée initiale. | Une même demande avec espaces multiples peut être rejouée sans faux conflit ; un prompt différent reste refusé. |
| Les clés du cache sont réparties entre des features qui s'importent mutuellement. | Clés de ressources dans `shared/api/query-keys.ts`, typées à partir du contrat OpenAPI. | Suppression de la dépendance circulaire catalogue/génération. |
| Un hook de mutation décide de la navigation de l'écran. | Navigation après acceptation dans le formulaire, invalidation dans le hook API. | Le hook peut servir une autre interaction sans imposer une route. |
| Les helpers d'état sont mêlés aux hooks et les petits widgets de suivi sont exportés depuis le rendu complet des modules. | Fonctions pures `tracking-state.ts`, `catalog/polling.ts` et composants `tracking-operation.tsx`. | Tests ciblés et dépendances plus précises pour le lecteur. |
| Le catalogue considère `partial` comme une génération encore active. | Arrêt du polling des résultats partiels stabilisés ; conservation de la récupération bornée des échecs. | Pas de rafraîchissement perpétuel sans travail en cours. |
| La suppression d'une génération laisse ses projections de suivi en cache. | Retrait des snapshots et événements ; invalidations indépendantes concurrentes. | Cache cohérent avec la suppression. |
| Le frontend conserve un ancien suivi par jobs et deux mutations de contenu sans consommateur dans les écrans. | Suppression des hooks et helpers morts ; remplacement du test d'invalidation par le parcours actuel de tracking. | Une seule stratégie de suivi frontend ; endpoints backend conservés. |
| Les frontières Go sont documentées mais non vérifiées ; le contrôle frontend ne couvre que certains alias. | Test AST des imports Go et règle ESLint testée pour alias, imports relatifs, réexports et imports dynamiques littéraux. | Détection automatique des dépendances inversées lors des vérifications et en CI. |
| La CI backend ne lance pas `staticcheck`, pourtant demandé aux contributeurs. | Installation de la version déjà prévue par le Makefile et exécution en CI. | Même exigence statique localement et dans la pipeline de livraison. |

## Structure conservée après revue

- **Domaine Go** : invariants et transitions restent dans les entités ; les agrégats cohérents ne sont pas fractionnés pour respecter un seuil de lignes.
- **Services Go** : autorisation, orchestration et transactions restent derrière les ports `contract`. Le même service implémente les interfaces existantes ; aucun framework supplémentaire.
- **Persistance et workers** : sqlc reste généré, chargements groupés préservés, appels IA hors transaction, contrôle des tentatives, claims et leases conservé. Les nouveaux fichiers déplacent les méthodes sans modifier leur ordre d'exécution.
- **API et sécurité de session** : DTO publics, contrôle de propriété Clerk, erreurs assainies et cache frontend isolé par session conservés. Les tests HTTP et PostgreSQL couvrent ces garanties localement.
- **React** : routes de composition, features responsables de leurs interactions, transport et clés de cache partagés. Le backend reste la source des états métier ; les fonctions frontend expliquent ces états sans reconstruire la pipeline.
- **UI** : fondation shadcn/ui, tokens centralisés, Markdown assaini, chargement différé de Mermaid et navigation responsive conservés. Aucun changement de direction visuelle.

## Règles pour les prochaines évolutions

[`AGENTS.md`](../AGENTS.md) explicite les huit principes demandés et leur application aux deux applications. Les cartes de responsabilités et les parcours sont actualisés dans [l'architecture backend](../backend-go/docs/architecture.md) et [l'architecture frontend](../frontend/docs/architecture.md).

Les extractions doivent répondre à une responsabilité réelle. Un helper privé ne nécessite pas automatiquement une interface ; une fonction pure ne nécessite pas un hook ; deux lignes semblables ne justifient pas automatiquement une abstraction partagée. Toute nouvelle dépendance entre features ou couche technique doit être expliquée et compatible avec les contrôles d'architecture.

## Vérification

| Vérification | Résultat |
| --- | --- |
| Go : formatage, `go vet ./...`, `staticcheck ./...`, `sqlc vet` | Réussis |
| Go : `go test -count=1 ./...` | Réussi, y compris le contrôle d'architecture |
| Go : `go test -race ./...` | Réussi sur l'ensemble des packages |
| PostgreSQL : migrations 00001 à 00013 depuis une base vide, puis suite `-tags=integration` | Réussies sur PostgreSQL 18, instance locale isolée |
| Frontend : TypeScript, ESLint, Prettier | Réussis |
| Règle ESLint d'architecture | 3 tests réussis |
| Vitest | 64 tests validés sur 18 fichiers, avec relances ciblées après deux échecs de démarrage de workers Windows |
| Playwright : mobile, tablette, desktop, thèmes clair/sombre, clavier et accessibilité | 112 réussis, 8 exclusions conditionnelles prévues selon le viewport, aucun échec |
| Build Vite avec validation Vercel production | Réussi avec les variables publiques HTTPS de la CI |

Les contrôles Go sont exécutés avec une compilation à la fois (`-p=1`, `GOMAXPROCS=1`) et des caches temporaires dédiés : le cache utilisateur était partiellement inaccessible et les compilations simultanées avaient saturé la mémoire. Les sous-processus Vite/Playwright ont nécessité une exécution hors du bac à sable. Ces adaptations concernent l'environnement d'exécution des vérifications.

Le build signale encore des chunks de plus de 500 kB ; le chunk d'entrée mesure environ 624 kB minifié, 184 kB gzip. Le découpage des dépendances lourdes reste une piste d'optimisation distincte ; l'avertissement n'est pas masqué par une augmentation du seuil.

Les parcours navigateur utilisent les fixtures API/Clerk du projet. Ils ne valident pas les clés Clerk, les appels OpenAI réels ni la configuration d'un déploiement distant.
