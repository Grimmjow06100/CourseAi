# Refonte du site et suivi de génération

Les primitives shadcn/ui (Radix) résident exclusivement dans `src/components/ui`.
Les composants métier les composent ; les anciennes primitives de `shared/ui`
ont été supprimées. `components.json` configure le registre, les alias et les
tokens. Ajouter une primitive avec `npx shadcn@latest add <composant>`, puis
vérifier TypeScript, les conventions locales et son accessibilité.

`src/index.css` centralise les paires fond/texte, les états sémantiques, la sidebar,
les rayons et les polices Geist locales. Les surfaces restent neutres ; le vert
sert aux actions `brand`, au focus et aux états de réussite. Le thème est appliqué
avant React et suit les changements système et inter-onglets. Le favicon versionné
et `CourseAiLogo` utilisent le même SVG.

La sidebar utilise Sheet sur mobile, le mode icône sur tablette et le mode étendu
à partir de 1024 px. La fermeture du menu mobile restitue le focus au déclencheur.
Les onglets du suivi sont dans l’URL. Les listes sont paginées par le serveur et
le curriculum s’ouvre progressivement. Le lecteur conserve le Markdown sécurisé,
le code défilant et les corrections chargées après une action explicite.

## Contrat de suivi

`GET /api/generations/{requestID}/tracking` retourne un instantané cohérent :
révision décimale en chaîne, tentative globale, disponibilité du contenu,
compteurs, cinq phases réelles, modules/leçons sans corps de contenu et opérations
courantes versionnées. Le nombre attendu de leçons reste inconnu tant qu’un plan
de module manque. Texte et activités appartiennent à une même opération.

`GET /api/generations/{requestID}/events` parcourt les événements persistés par
curseur exclusif, du plus récent au plus ancien. Aucun prompt, contenu, solution
ou diagnostic fournisseur. Les historiques anciens ou purgés sont explicitement
incomplets.

Après consultation des pages anciennes, « Vérifier les événements » recharge
explicitement la page la plus récente, même lorsque la limite de cache est atteinte.

`POST /api/generations/{requestID}/jobs/{jobID}/retry` et
`POST /api/generations/{requestID}/retry-failed` exigent une clé d’idempotence et la
version observée. Une transaction verrouille la demande, vérifie propriétaire,
versions, quotas et capacité, remplace toute la sélection et sauvegarde le reçu.
Le replay restitue le même résultat ; un conflit 409 n’entraîne aucune écriture
partielle. La reprise ciblée conserve la tentative globale. La reprise globale
historique en crée une nouvelle.

Un échec local laisse continuer les tâches indépendantes. Sans travail utile
actif, un résultat incomplet avec une leçon utilisable devient `partial`, sinon
`failed`. Un coordinateur terminé ne signifie pas que ses enfants sont disponibles.
Les contenus acquis sont conservés. Une erreur de configuration exige une
intervention technique ; les messages publics n’exposent pas le fournisseur.
Claims, versions et tentatives empêchent un ancien callback d’écraser une reprise.

Le suivi est interrogé toutes les deux secondes pendant l’activité, suspendu en
arrière-plan et revalidé au focus/reconnexion. Une erreur réseau conserve le
dernier instantané et ralentit les lectures. Les révisions anciennes sont ignorées.
La réconciliation sans travail actif est bornée à 30 observations ; une file
incohérente est expliquée. « Vérifier maintenant » effectue uniquement un GET.

## Recette

Les tests couvrent création/clarification, résultats partiels, finalisation en
incident avec contenu disponible, perte de réponse, relances concurrentes, quotas,
propriétaires distincts, versions obsolètes, barrière des plans et 500 leçons.
Playwright utilise une API simulée et vérifie les deux thèmes, 390/768/1440 px,
320 px et les violations Axe sérieuses/critiques. Il ne valide pas l’instance
Clerk réelle ni son OAuth de production.

Voir le [guide de livraison](../../docs/site-refactor-release.md) et le
[rapport de vérification](../../docs/site-refactor-progress.md).
