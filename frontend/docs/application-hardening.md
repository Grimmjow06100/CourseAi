# Durcissement applicatif du frontend

## Authentification et confidentialité

- Le cache et le routeur appartiennent désormais à une seule session Clerk. Une déconnexion ou un changement de session annule ses requêtes HTTP et vide ses données.
- Le jeton est demandé pour chaque appel, sans stockage persistant. Son absence produit une erreur authentification au lieu d'une requête anonyme.
- Les appels acceptent les signaux d'annulation des queries et un timeout de 30 secondes. Les jobs backend restent indépendants des requêtes HTTP.
- Les solutions restent masquées au retour sur une leçon, même lorsque la query possède déjà des réponses. Chaque activité nécessite sa propre révélation.

## Générations et cache

- Ajout de la lecture protégée des jobs par demande dans les contrats Go, le service, le handler, le routeur et l'OpenAPI ; les types TypeScript ont été régénérés.
- Les IDs des jobs ne dépendent plus d'un état React perdu au rechargement. Le backend est la source du suivi.
- Les jobs enfants d'un module sont suivis jusqu'à leur état terminal. Une ancienne tentative échouée ne masque pas un nouvel essai sur la même cible.
- Les contenus ne sont plus repollés indéfiniment sur `hasContent=false`. Les queries lourdes sont invalidées lors des transitions terminales des jobs.
- Les mutations partielles, les relances, les suppressions et les changements de statut invalident les listes concernées.
- Les erreurs du suivi peuvent être relancées explicitement. Un job en échec affiche un accès au suivi de la demande, où la relance remet la pipeline dans un état admissible.
- La progression affichée correspond aux seuils réels du backend : analyse, architecture, plans, contenus, finalisation.

## Interactions et erreurs

- La clé d'idempotence suit le prompt normalisé et reste stable lors d'une tentative identique dans le même formulaire. Les rejets des mutations sont traités sans promesse rejetée non gérée.
- Une suppression attend la réponse du serveur. Le dialogue reste ouvert avec son erreur si le serveur refuse ou ne répond pas.
- Le tableau de bord distingue une panne API d'une collection vide.
- Les erreurs communes sont traduites et les détails internes des erreurs serveur ne sont pas affichés. Les notices API peuvent présenter l'identifiant de corrélation.
- Le backend expose `X-Request-ID` et `Retry-After` via CORS pour les clients sur une autre origine.
- Une erreur de rendu ou un chargement de chunk devenu invalide affiche une page récupérable par rechargement, au lieu d'un écran vide.
- La recherche temporisée est déclenchée par la saisie uniquement ; un retour dans l'historique restaure l'URL sans être écrasé par une ancienne valeur locale. Le tri par titre utilise l'ordre ascendant.

## Rendu et accessibilité

- Le menu mobile possède un titre de dialogue ; les boutons de fermeture et groupes radio ont un nom accessible.
- Les erreurs, statuts de cours, navigation et états de chargement utilisent les traductions.
- Le renderer Markdown ne crée plus de balises `pre` imbriquées. Les tableaux et le code long disposent d'un débordement interne.
- Une erreur de chargement ou de syntaxe Mermaid retourne au texte source. Le HTML brut demeure désactivé et Mermaid reste en mode strict.

## Vérification et limites

Les tests unitaires couvrent notamment l'environnement de production, le client HTTP, la durée de vie des sessions, l'idempotence, la sélection des jobs, la recherche temporisée, les suppressions refusées, le Markdown et la révélation des corrections. Les tests navigateur couvrent trois tailles d'écran, le parcours principal, les erreurs API, la reprise des jobs, les enfants d'un module, l'historique de recherche et les dialogues mobiles. Les nouveaux tests Go vérifient la propriété de la demande avant toute lecture des jobs et l'absence de payload privé dans la réponse HTTP.

Restent nécessaires avant publication : configuration et test réels Clerk/Vercel/Railway, origines CORS explicites, collecte distante des erreurs frontend, vérification de la CSP contre les domaines Clerk réels. Les mocks navigateur ne remplacent pas un parcours connecté avec une vraie formation.

Les optimisations plus larges, telles qu'un DTO de curriculum léger évitant le téléchargement du graphe complet et un budget de bundle Mermaid mesuré sur des formations représentatives, restent des travaux distincts. La mémoire d'idempotence n'est pas persistée lors d'un rechargement du formulaire ; l'historique serveur permet de retrouver une demande déjà acceptée.

## Résultats de vérification

- TypeScript, ESLint et formatage : validés.
- Vitest : 26 tests réussis dans 12 fichiers.
- Playwright : 23 tests réussis ; un test de tiroir mobile volontairement ignoré sur desktop. Les contrôles couvrent les largeurs 390, 768 et 1440 px et les violations d'accessibilité critiques et sérieuses des écrans testés.
- Build de production : validé avec une configuration publique fictive de vérification, sans déploiement. Un essai avec une clé Clerk vide est correctement refusé avant génération du bundle.
- Go : suite unitaire complète, `go vet`, `staticcheck`, race detector et `sqlc vet` validés.
- Intégration PostgreSQL : impossible sur cette session, Docker Desktop étant arrêté et le port local 5433 inaccessible. Relancer `make test-integration` une fois la base disponible.
- Le build signale encore des chunks supérieurs à 500 ko : ce n'est pas bloquant pour la compilation, mais leur réduction reste une optimisation distincte. Aucun avertissement n'a été masqué par une augmentation artificielle du seuil.
- Aucun commit, push ou déploiement effectué.
