# Audit et amélioration du frontend Course AI

Date : 14 septembre 2026.

## Périmètre et méthode

Audit du client React/Vite : navigation, tableau de bord, formulaire de génération, clarification, suivi des générations, bibliothèque, programme, lecteur, exercices, authentification et pages de secours. Lecture des composants, des styles, des contrats et des tests existants, puis vérification dans Chromium avec API simulée, captures et axe-core.

Le skill Figma a été consulté. Aucun fichier de référence ni connecteur MCP Figma n'était disponible dans cette session. Le travail est une amélioration du produit existant dans le code ; aucune maquette Figma n'a été créée ou utilisée comme référence de conformité.

## Constats et corrections

| Priorité | Constat                                                                                                                                                                    | Correction                                                                                                                                                         |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Haute    | La règle CSS globale `a { color: inherit }`, hors des couches Tailwind, écrasait les couleurs des liens. Le bouton « Commencer » présentait un contraste mesuré de 2,02:1. | Suppression de cette règle : les utilitaires et tokens reprennent leur rôle. Contrôle des liens d'action avec axe.                                                 |
| Haute    | Les enfants de la grille de l'accueil pouvaient imposer une largeur minimale supérieure à celle du mobile.                                                                 | Colonnes `minmax(0, 1fr)`, panneaux réductibles et titres pouvant se couper. Vérification par `scrollWidth` contre `clientWidth`, et inspection des captures.      |
| Haute    | La pagination utilisait deux boutons d'icônes sans nom accessible.                                                                                                         | Noms localisés « Précédent » et « Suivant ».                                                                                                                       |
| Haute    | Aides et erreurs des champs non associées aux contrôles.                                                                                                                   | Association des descriptions et erreurs avec `aria-describedby`, via le composant Field ; état invalide transmis aux contrôles.                                    |
| Moyenne  | Tableau de bord uniforme, peu de repères entre intention, saisie et activités récentes.                                                                                    | En-tête hiérarchisé, formulaire principal, explication des trois étapes et listes récentes en panneaux. Les compteurs utilisent exclusivement les totaux de l'API. |
| Moyenne  | Navigation et focus peu visibles.                                                                                                                                          | Navigation active plus contrastée, `aria-current`, lien d'évitement et focus visible global ; menu mobile à défilement conservant le comportement Radix.           |
| Moyenne  | Thème seulement dicté par le système.                                                                                                                                      | Choix système/clair/sombre, persistance locale, application avant le montage React et fonctionnement même si le stockage est bloqué.                               |
| Moyenne  | Suggestions longues et difficilement lisibles sur mobile.                                                                                                                  | Libellés courts avec icônes existantes ; clic remplissant le texte complet et recentrant le focus sur le champ.                                                    |
| Moyenne  | Nouvelle formation sans guide pour formuler un objectif.                                                                                                                   | Guide sur le sujet, les acquis et le résultat souhaité ; message expliquant l'étape suivante.                                                                      |
| Moyenne  | Bibliothèque vide confondue avec une recherche sans résultat.                                                                                                              | Messages distincts, création de formation et remise à zéro des filtres selon le contexte.                                                                          |
| Moyenne  | Plusieurs statuts de cours ne pouvaient pas être sélectionnés dans les filtres.                                                                                            | Liste alignée sur le schéma de recherche existant et prise en charge de l'ordre par statut déjà accepté dans l'URL.                                                |
| Moyenne  | Cartes de formation sans hiérarchie et pied de carte trop proche du résumé.                                                                                                | Bandeau, titre cliquable, niveau cible, statut, date et actions regroupées ; espacement et titres longs traités.                                                   |
| Moyenne  | Clarification : erreur générique de réponse serveur et état sélectionné discret.                                                                                           | Message orienté vers les champs à vérifier ; options sélectionnées mises en évidence.                                                                              |
| Moyenne  | Suivi, réussite et erreurs visuellement disparates.                                                                                                                        | Panneaux cohérents, barre de progression nommée, étape active identifiée et action claire après réussite.                                                          |
| Moyenne  | Programme : génération du contenu et disponibilité peu explicites.                                                                                                         | Modules distincts et libellés « Contenu disponible » / « À générer » ; niveau et contraintes du projet final exposés.                                              |
| Moyenne  | Lecture longue : programme non fixe, lignes très larges, fin de parcours sans issue.                                                                                       | Sommaire fixe et défilant sur ordinateur, largeur de lecture limitée, objectif mis en évidence, position de la leçon et retour au programme.                       |
| Moyenne  | Les listes Markdown perdaient leurs puces avec le reset CSS.                                                                                                               | Restauration des listes ordonnées et non ordonnées.                                                                                                                |
| Basse    | Écrans d'authentification et page 404 incomplets visuellement.                                                                                                             | Tokens de thème dans Clerk, voile lisible sur la photographie existante, titre mobile, choix du thème et description de la page introuvable.                       |

## Direction visuelle

- Palette existante conservée : vert profond, fonds neutres et accents ambre pour les avertissements.
- Manrope pour l'interface, JetBrains Mono pour le code et les repères techniques ; polices locales existantes.
- Panneaux de 16 px de rayon, contrôles de 12 px, espacements cohérents et contraste des états explicite.
- Présentation des étapes avec CSS et Lucide déjà présent ; aucune dépendance ou image externe ajoutée.
- Thèmes clair/sombre cohérents et respect de `prefers-reduced-motion`.
- Textes ajoutés disponibles en français et en anglais.

## Comportement préservé

Les routes TanStack Router, le cache TanStack Query, la validation Zod/React Hook Form, l'authentification Clerk et les DTO OpenAPI restent les fondations de l'application. Les actions de génération conservent leurs clés d'idempotence et le suivi des jobs. Les confirmations de suppression restent explicites. Les solutions des exercices restent chargées et révélées à la demande.

Le frontend n'affiche pas une progression d'apprentissage fictive : l'indicateur du lecteur représente la position dans le programme, pas des leçons validées. La disponibilité du contenu et le statut de génération restent distincts de l'acquisition de compétences.

## Vérification

Le fichier `tests/e2e/design.spec.ts` couvre les écrans principaux dans les deux thèmes, la navigation depuis un état vide, les filtres, les suggestions, la langue, la persistance du thème et les erreurs accessibles. Un contrôle dédié exerce aussi les titres longs à 320 px et les états de génération en cours, échouée et terminée. Les captures sont produites dans `frontend/test-results` et `frontend/test-results-final` (ignorés par Git).

- Suite unitaire : 26 tests réussis. Le lancement initial avec threads a rencontré des expirations de démarrage de workers Windows ; le lancement complet avec `--pool=forks --maxWorkers=1` a réussi.
- Suite navigateur initiale : 80 tests réussis, 1 ignoré car le menu mobile n'existe pas sur ordinateur ; Chromium, formats 390 × 844, 768 × 1024 et 1440 × 1000.
- TypeScript, ESLint global et ciblé, et formatage Prettier : réussis.
- Compilation Vite de production : réussie avec les variables temporaires de contrôle décrites ci-dessous. Le build signale des chunks de plus de 500 kB (entrée principale : environ 602 kB, 174 kB gzip ; un chunk de diagrammes : environ 662 kB). Le chargement de Mermaid reste différé. Une optimisation supplémentaire du poids du JavaScript est un travail distinct à suivre.
- Après les derniers changements, les deux tests unitaires ciblés sur les exercices et la confirmation passent également.
- Vérification navigateur finale ciblée : 19 cas applicables réussis, dont les états de génération et le format 320 px. Trois de ces cas avaient initialement expiré sur l'attente du premier titre pendant la compilation simultanée ; ils passent tous lors de la relance isolée, avec les mêmes assertions et délais. Deux variantes du test à 320 px sont ignorées car ce test est dédié au projet mobile.
- Au total, 84 scénarios navigateur distincts applicables ont été exercés avec succès, plus les répétitions de contrôle après ajustement. Aucune violation axe critique ou grave sur les écrans audités ; captures inspectées sur mobile, tablette et ordinateur, en clair et en sombre.

Commandes de vérification depuis `frontend` :

```powershell
npm run typecheck
npm run lint
npm run format:check
npm run test -- --pool=forks --maxWorkers=1
npm run test:e2e
# Vérification ciblée après les derniers ajustements :
npm run test:e2e -- --grep 'small screens|generation progress|dashboard is accessible|lesson is accessible|keeps a failed deletion'
# Variables API/Clerk requises pour ce build :
npm run build
```

Les tests de mise en page comparent la largeur du document à `documentElement.clientWidth`. `window.innerWidth` peut s'élargir avec le viewport de mise en page sur mobile et masquer un débordement réel. Les captures complètent ces assertions automatiques.

## Limites de l'audit

- Les tests navigateur utilisent le mode E2E de développement et une API simulée. Ils vérifient le client, ses interactions HTTP et ses états ; ils ne prouvent pas la disponibilité d'un backend déployé, de PostgreSQL ou d'OpenAI.
- Le widget Clerk réel et ses parcours hébergés nécessitent une vérification avec un compte de test connecté. Son intégration est revue dans le code et contrôlée par TypeScript ; les tests E2E contournent l'authentification conformément au dispositif existant.
- Les variables de déploiement API et Clerk sont absentes du poste. La compilation de contrôle utilise temporairement une URL HTTPS `.test` et une clé publique factice valide pour le schéma ; ces valeurs ne sont ni enregistrées dans un fichier d'environnement ni destinées au déploiement.
- Axe détecte certaines classes de défauts d'accessibilité. Il ne remplace pas une évaluation avec lecteur d'écran ou par des utilisateurs.
- Aucune publication et aucune modification de fichier Figma n'ont été effectuées.
