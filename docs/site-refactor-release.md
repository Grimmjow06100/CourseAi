# Livraison de la refonte — 2026-09-19

## Ordre de livraison

Le frontend dépend des nouveaux endpoints de suivi. Livrer le backend et ses
migrations avant d’activer ce frontend en production. Un push Git ne prouve pas
que Railway a migré la base ni que Vercel a terminé son déploiement.

1. Sauvegarder la base et inventorier les demandes actives/échouées et leurs
   contenus. Ne pas requalifier globalement les échecs.
2. Arrêter les anciens workers pendant la bascule des versions d’opérations.
3. Appliquer les migrations Goose forward 00011–00013. La 00011 ajoute `partial`.
   La 00012 ajoute opérations courantes, versions, événements, révisions et reçus ;
   elle annule les anciennes opérations remplacées et rouvre les échecs qui ont
   encore du travail actif. La 00013 indexe la rétention.
4. Déployer l’API et les workers de ce commit. Vérifier `/health/ready`, Clerk,
   CORS, les routes de suivi et les métriques de queue.
5. La maintenance réconcilie les anciens échecs à partir des contenus et du travail
   courant. Utiliser `cmd/reconcile-generations` selon son aide pour une inspection
   contrôlée. L’historique manquant n’est pas reconstruit artificiellement.
6. Déployer Vercel avec Root Directory `frontend`, Node 24, `npm ci`, `npm run build`
   et les variables publiques de production existantes. `VITE_E2E_MODE=false`.
7. Sur `https://www.courseai.site`, tester connexion/déconnexion, changement de
   compte, liens profonds, thème, favicon après rechargement, génération complète,
   résultat partiel et reprise. Les corrections restent chargées à la demande.

Les tests locaux utilisent uniquement la base isolée
`course_ai_refactor_20260919`. Ils ne migrent pas la production.

## Rétention et exploitation

Événements et reçus utilisent la rétention des jobs terminés (30 jours par défaut)
et des suppressions bornées. Les opérations courantes sont conservées ; les jobs
historiques remplacés deviennent purgeables. La suppression d’événements marque
l’historique incomplet. Après expiration d’un reçu, la version courante interdit
toujours une seconde reprise d’une opération remplacée.

Surveiller résultats partiels, reprises acceptées, conflits 409, saturation, retard
de finalisation et fraîcheur du suivi. Conserver les diagnostics sensibles dans
les logs internes. Une erreur de configuration permanente exige une intervention.

## Repli

Ne pas exécuter de Down destructif pendant que les nouveaux workers fonctionnent.
Conserver colonnes, contraintes et événements ; privilégier un correctif forward.
Une UI de repli doit comprendre `partial`. Un rollback Vercel ne restaure ni la
base ni les tentatives exécutées. Le Down de 00011 conserve les valeurs d’enum.
Réserver l’annulation de 00012/00013 aux environnements jetables ou à un plan
explicite de restauration.

## Contrôles reproductibles

Backend : `gofmt`, `go vet ./...`, `staticcheck ./...`, `go test -count=1 ./...`,
`go test -race ./...`, `sqlc vet`, `make test-integration` avec une base dédiée.
Frontend : `npm run api:generate`, `npm run typecheck`, `npm run format:check`,
`npm run lint`, `npm test`, `npm run build`, `npm run test:e2e`.
`PLAYWRIGHT_REUSE_SERVER=true` sert uniquement à réutiliser volontairement le
serveur E2E local ; la CI lance son serveur dédié par défaut.
