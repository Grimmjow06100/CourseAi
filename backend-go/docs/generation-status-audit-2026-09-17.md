# Audit des incoherences de statut de generation

Date : 2026-09-17
Revision locale auditee : `024575900a78f1a62c6ca01732250db67b068292`
Statut : diagnostic termine ; correctifs proposes, non appliques.
Perimetre : frontend React, API Go, persistance PostgreSQL, workers et reprises.

Suivi du 2026-09-18 : les correctifs sont documentes dans
[[generation-status-fixes-2026-09-18]]. Le present document conserve les constats
de la revision auditee, avant correction.

## 1. Conclusion

**Le probleme ne se limite pas a une mauvaise etiquette frontend.** Le backend
peut conserver une formation complete tout en marquant sa demande de generation
en echec. Le frontend affiche ensuite cet echec sans indiquer que le contenu reste
disponible ; sa politique de cache peut aussi prolonger un statut devenu obsolete.

Deux variantes exactes du symptome ont ete reproduites avec les services actuels :

- Toutes les lecons sont presentes, mais `course.status=failed` et `pipelineStatus=failed`.
- La formation est `completed`, mais sa demande reste `failed` et le finaliseur refuse de la reprendre.

Deux autres anomalies backend et quatre comportements frontend ont ete reproduits.
Une course d'ecriture supplementaire est etablie par inspection du code, sans
reproduction concurrente PostgreSQL dans cet audit.

**Ne pas corriger en remplacant simplement le badge par "terminee" des qu'un
`courseId` existe.** Une formation est creee des l'architecture et peut etre
partielle. Il faut corriger les transitions backend et expliquer separement la
disponibilite du contenu et l'etat technique du traitement.

## 2. Incident Signale Et Limites

Exemple fourni :
[Generation de production](https://www.courseai.site/generations/90bc4e76-7f7e-48ac-893f-0f50b5cd1993).

Identifiant : `90bc4e76-7f7e-48ac-893f-0f50b5cd1993`.

L'ouverture dans le navigateur de diagnostic redirige vers `/sign-in`.
Aucune session authentifiee de production n'etait disponible dans ce navigateur.
Ni les reponses API authentifiees, ni les logs Railway, ni la base de production
n'ont donc ete consultes. Le commit reellement deploye reste a verifier.

La base configuree par le backend local est distincte de cet incident. Consultation
en transaction `READ ONLY`, sans exposition des identifiants de connexion :

| Observation locale                        | Resultat                           |
| ----------------------------------------- | ---------------------------------- |
| Premier releve                            | 2026-09-17 16:19:41 UTC            |
| Demandes terminees / formations terminees | 4 / 4                              |
| Autres demandes                           | 1 queued, 1 awaiting_clarification |
| Demandes failed                           | 0                                  |
| Jobs presents                             | 127, tous completed                |
| Releve cible                              | 2026-09-17 16:27:03 UTC            |
| Presence de l'identifiant signale         | 0                                  |
| Lecons des quatre formations locales      | 219, toutes avec Markdown          |

Ces releves n'invalident pas le signalement en production. Ils interdisent seulement
d'attribuer avec certitude cet incident particulier a l'un des scenarios ci-dessous.
Aucune generation payante, modification de donnees, relance de job ni operation
de deploiement n'a ete effectuee.

## 3. Les Etats Affiches

| Surface                            | Source effective                        | Consequence                                                  |
| ---------------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| Historique et Generations recentes | `GenerationSummary.pipelineStatus`      | Badge de la demande, pas du contenu                          |
| Suivi /generations/:requestId      | `GenerationStatus.pipelineStatus`       | La branche failed domine meme si courseStatus vaut completed |
| Carte du catalogue                 | `Course.status`                         | Peut differer du badge de generation                         |
| Vue formation / lecteur            | Modules, lecons, hasContent et Markdown | Affiche le contenu meme si le pipeline a echoue              |
| Formation recente du dashboard     | Titre, synopsis et lien                 | Cette ligne ne montre pas de statut global                   |
| Generation partielle               | Derniers jobs par kind/targetId         | Une troisieme lecture, distincte du statut global            |

Sources : [liste](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/generation-list.tsx:27),
[suivi](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/generation-tracker.tsx:46),
[carte](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/catalog/course-card.tsx:39),
[lecteur](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/course-reader/lesson-reader.tsx:141).

Le GET de statut fait une jointure directe entre demande et formation ; il ne
reconcilie pas leurs etats. Le catalogue autorise la lecture du contenu apres
verification du proprietaire, sans exiger un pipeline termine. Ce choix de conserver
l'acces au travail deja produit est utile et ne doit pas etre supprime.

Sources : [projection SQL](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/db/queries/generation_requests.sql:122),
[lecture catalogue](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_catalog_service.go:24).

## 4. Constats Priorises

### A. P1 : contenu sauvegarde, puis echec technique considere comme echec total

**Preuve : reproduction backend 1.**

`runLessonContentJob` enregistre le contenu via `persistLessonContent`, puis appelle
`enqueueFinalizeIfReady`. Ce sont des transactions differentes. Une erreur lors de
la creation du finaliseur n'annule donc pas le contenu deja enregistre.

Si cette erreur est classee definitive, ou si ses retries sont epuises, le worker
appelle `handleTerminalJobFailure`. Ce dernier marque toute la demande en echec
tant qu'elle n'est pas deja terminee. `markPipelineFailed` marque aussi la formation
non terminale en echec, **sans verifier si toutes les lecons sont deja completes**.

Scenario reproduit :

1. Generer normalement analyse, architecture et plan.
2. Enregistrer le contenu de la derniere lecon.
3. Injecter une erreur avant l'enqueue du finaliseur.
4. Traiter cette erreur comme terminale.
5. Constater : contenu complet, demande failed, formation failed.

Les erreurs injectees simulent un echec technique ; elles ne constituent pas la
preuve d'une panne PostgreSQL ou d'un timeout dans l'incident signale.

Sources : [persistance puis finalisation](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:154),
[persistance du contenu](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:320),
[propagation de l'echec](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:559).

**Correctif :** separer echec de production du contenu et echec de coordination.
Une fois le contenu valide complet, preferer une finalisation/reconciliation
idempotente plutot que transformer la formation en echec global. Garantir qu'un
finaliseur ne peut pas etre perdu : transaction/outbox avec serialisation par
demande, ou reconciliation des formations completes sans finalisation aboutie.
Attention aux deux dernieres lecons concurrentes : une simple transaction par lecon,
sans coordination, ne garantit pas que l'une verra toutes les autres ecritures.

### B. P1 : finalisation de la formation et de la demande non atomique

**Preuve : reproduction backend 2.**

`runFinalizeCourseJob` enchaine :

1. `completeCourseIfReady` : transaction qui passe la formation a completed.
2. Relecture de la formation.
3. `updateRequestProgress(..., 95)` : autre transaction.
4. `completeRequest` : encore une autre transaction.

Si une erreur terminale intervient apres l'etape 1, la formation reste completed.
Le gestionnaire d'echec peut passer la demande encore running a failed, tout en
preservant la formation completed grace a son garde terminal.

Resultat reproduit : **course=completed, request=failed**.
Une nouvelle execution normale du finaliseur est ensuite rejetee par
`loadRunnableJobRequest`, qui refuse les demandes failed. Il n'existe pas de
reconciliation de succes equivalente a celle des echecs.

Sources : [finaliseur](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:213),
[completion formation](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:501),
[completion demande](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:348),
[garde failed](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:463).

**Correctif prioritaire :** finaliser demande et formation dans une meme transaction,
avec verification du contenu, de la tentative courante et verrouillage des lignes.
Utiliser directement les repositories de cette transaction, pas les helpers actuels
qui ouvrent chacun une autre transaction. Un rejeu apres commit doit etre un no-op
reussi. Ajouter une reconciliation controlee des anciennes incoherences.

L'acquittement technique du job peut rester distinct, a condition qu'un probleme
d'acquittement ne puisse pas degrader un resultat metier deja valide.

### C. P1 : un ancien echec peut contaminer une nouvelle tentative

**Preuve : reproduction backend 3.**

La reprise globale verrouille la demande et reconfirme son brief ; cette confirmation
incremente `ClarificationVersion`. Les nouvelles cles de jobs utilisent cette version.

En revanche, `handleTerminalJobFailure` ne compare pas la version/tentative du job
avec celle de la demande. Il ne verifie pas non plus si ce job a ete remplace par
une tentative plus recente. La maintenance rejoue les echecs dont
`failure_handled_at` est encore null.

Scenario reproduit :

1. Un job de la tentative v1 provoque failed.
2. La reprise globale demarre v2, demande running.
3. Un ancien callback ou une reconciliation tardive traite a nouveau l'echec v1.
4. La demande v2 repasse a failed.

Ce cas peut notamment survenir si la synchronisation initiale a reussi mais que son
marquage `failure_handled_at` a echoue, ou si un ancien job termine tardivement.
Une demande deja completed est protegee dans le traitement sequentiel habituel :
ce constat concerne d'abord une reprise encore active.

Sources : [retry](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/generation_retry.go:21),
[version du brief](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/domain/generation_request.go:592),
[maintenance](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/infrastructure/jobs/maintenance.go:42),
[traitement terminal](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:559).

**Correctif :** porter une identite explicite de tentative sur la demande et les jobs,
et la verifier sous verrou avant toute ecriture metier ou propagation d'echec.
Marquer les callbacks obsoletes comme traites/supplantes, sans affecter la tentative
courante. Annuler les anciens jobs encore en attente si approprie ; proteger les
ecritures de ceux deja en cours. La version du brief et la tentative d'execution
sont deux concepts a separer a terme.

Ne pas baser cette protection uniquement sur createdAt/updatedAt ou sur un parsing
fragile de la cle d'idempotence.

### D. P1 conditionnel : ecritures concurrentes pouvant ecraser un etat plus recent

**Preuve : inspection statique ; pas de test concurrent PostgreSQL execute.**

`updateRequestProgress`, `completeRequest` et `markPipelineFailed` lisent la
demande sans `FOR UPDATE`, puis reecrivent l'objet entier. La requete SQL
`UpdateGenerationRequest` filtre uniquement sur l'id, sans version attendue.
`transitionCourse` modifie de meme une formation chargee avant sa transaction.

Exemple d'entrelacement possible :

1. A lit une demande running pour traiter un echec.
2. B termine et commit la demande completed.
3. A met a jour son ancienne copie en failed.
4. L'UPDATE inconditionnel de A ecrase completed.

Le garde `IsTerminal()` ne suffit pas lorsqu'il porte sur une copie perimee.
De meme, deux fins de plan peuvent reecrire un ancien statut de formation apres
le debut de la generation de contenu.

Sources : [progression](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:224),
[transition formation](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:333),
[echec](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:362),
[UPDATE demande](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/db/queries/generation_requests.sql:74),
[UPDATE formation](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/db/queries/courses.sql:69),
[transactions](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/infrastructure/postgres/repositories.go:78).

**Conditions :** executions concurrentes, plusieurs instances, reaper concurrent,
ou reprise pendant un traitement. Le pool a une concurrence par defaut de 1 :
cela reduit certains entrelacements mais ne prouve pas leur absence en production.

**Correctif :** verrouillage coheremment ordonne demande puis formation, ou
optimistic locking/CAS avec version attendue et gestion des conflits. Mettre a jour
les seuls champs necessaires, rendre la progression monotone dans une tentative,
et verifier la tentative/lease avant de persister les resultats d'un worker obsolete.
Les gardes des requetes d'acquittement des jobs ne protegent pas a eux seuls les
ecritures de contenu et de statut metier.

### E. P2 : le suivi transforme un etat mixte en echec sans nuance

**Preuve : reproduction frontend 1.**

Le composant teste seulement `pipelineStatus === 'failed'` pour afficher le bloc
d'erreur. Il ignore `courseStatus` dans cette branche et n'offre aucun lien vers
la formation, meme lorsqu'elle est completed.

Les listes utilisent elles aussi pipelineStatus directement. En parallele, le
lecteur affiche les lecons presentes. D'ou une experience "echec ici, formation
lisible ailleurs".

Source : [branche d'echec](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/generation-tracker.tsx:46).

**Correctif :** definir une presentation partagee entre suivi, historique et
dashboard, fondee sur une projection API coherente. Si le contenu est disponible,
conserver le lien "Ouvrir la formation" et distinguer "contenu disponible,
finalisation en erreur" d'un echec de contenu. Une formation partielle doit rester
explicitement partielle. Ne pas cacher une incoherence serveur avec un succes
inconditionnel cote client.

### F. P2 : cache et polling peuvent conserver une ancienne etiquette

**Preuve : reproductions frontend 2, 3 et 4.**

- Le suivi ne poll que queued/running ; failed/completed arretent la boucle.
- Le QueryClient desactive `refetchOnWindowFocus`.
- `staleTime: 30_000` marque les donnees perimees, mais ne declenche pas seul un fetch.
- Les listes de generations et de formations ne pollent pas.
- Lors d'un nouveau job terminal, `useRequestJobs` invalide cours et listes de
  generations, **mais pas le detail de generation**.
- Les mutations de generation partielle invalident les jobs et le contenu, pas
  systematiquement le detail global.
- Apres une erreur de query, le polling est aussi arrete. L'ecran d'erreur de
  transport reste distinct du vrai pipelineStatus failed.

Test concret : l'API simulee passe de failed a completed, 60 secondes passent et
le focus revient ; le suivi conserve failed et n'effectue qu'une seule requete.

Sources : [polling du suivi](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/api.ts:31),
[jobs et invalidations](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/jobs.ts:20),
[configuration du cache](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/app/query-client.ts:8),
[mutations partielles](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/catalog/api.ts:71).

**Nuance importante :** ce cache n'est pas une preuve d'erreur persistante en base.
Un rechargement complet ou un remontage perime peut recuperer le bon etat.
L'absence de polling terminal est normalement raisonnable pour des etats immuables,
mais pas avec des reprises ou reconciliations hors de la page.

**Correctif :** invalider aussi `generationKeys.detail(requestId)` aux changements
de jobs et reprises ; activer une revalidation au focus/reconnexion pour les ecrans
de suivi ; proposer "Actualiser le statut" sans relancer une generation.
Poller les listes tant qu'elles contiennent du travail actif, avec cadence adaptee.
Pour les incoherences terminales, utiliser une reconciliation bornee ou une
projection serveur indiquant explicitement qu'une synchronisation reste necessaire.

### G. P2 : une reprise partielle peut etre acceptee mais inexecutable

**Preuve : reproduction backend 4.**

`enqueueTargetedContentJob` accepte un job sur une demande failed sans la reprendre.
Le worker appelle ensuite `loadRunnableJobRequest`, qui refuse immediatement
cette meme demande. On peut donc recevoir une commande acceptee/queued qui ne peut
pas produire de contenu.

Le frontend desactive generalement le bouton lorsqu'un job cible est failed et
renvoie vers la reprise globale. Ce garde ne remplace pas une validation serveur :
une autre cible ou un autre client peut toujours soumettre la commande.

Sources : [commande partielle](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/generation_commands.go:325),
[garde worker](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_jobs.go:463),
[bouton module](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/catalog/course-curriculum.tsx:34).

**Correctif :** choisir un contrat explicite. Soit retourner un conflit metier avant
l'enqueue avec une action de reprise globale, soit reprendre atomiquement la demande,
la formation et la bonne tentative avant de creer le job cible. Les anciens echecs
de parents ou de cibles supplantees doivent rester historiques, pas bloquer
indefiniment une reprise valide.

### H. P2 UX : le message anglais masque toutes les causes

**Preuve : code et rendu.**

`failureMessage(err)` retourne toujours le texte signale, sans utiliser l'erreur.
Le DTO remplace aussi chaque message d'echec non null par ce meme texte. Le frontend
l'affiche directement au lieu de la traduction de secours.

Ce texte **ne prouve donc ni un echec OpenAI ni une session expiree**. Il peut
correspondre a une erreur de persistance, d'etat, de lease, de timeout ou autre.

Sources : [message service](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/service/course_generator_service.go:562),
[nettoyage DTO](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/infrastructure/http/dto/generation.go:296),
[affichage](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/frontend/src/features/generation/generation-tracker.tsx:52).

**Correctif :** conserver la protection contre la fuite de details internes ;
exposer un code public stable, l'etape et une action autorisee, puis traduire le
message cote client. Afficher un identifiant copiable et conserver les details
techniques uniquement dans les logs serveur correles.

## 5. Diagnostic Cible De La Production

Pour l'identifiant signale, commencer par des lectures, sans cliquer "Reessayer" :
une reprise change l'historique et peut declencher des appels factures.

1. Verifier le SHA deploye frontend et backend.
2. Dans une session autorisee, lire GET /api/generations/{id}/status et GET /api/generations/{id}/jobs.
3. Lire GET /api/courses/{courseId} si courseId existe ; compter modules et lecons disponibles.
4. Comparer les reponses au badge avant et apres une simple actualisation.
5. Consulter les logs du backend par request_id puis job_id, sans publier de jetons ou de payloads prives.
6. Comparer versions/tentatives et chronologie des failures, retries, completions et reconciliations.

Interpretation :

| Etat observe                                                 | Orientation                                         |
| ------------------------------------------------------------ | --------------------------------------------------- |
| API completed, UI failed                                     | Cache, ancienne reponse ou ancien frontend          |
| Demande failed, formation completed                          | Finalisation non atomique ou ecriture obsolete      |
| Demande/formation failed, contenu complet                    | Echec apres persistance, propagation trop large     |
| Echec d'un ancien job apres lancement d'une nouvelle version | Contamination de reprise                            |
| Contenu incomplet                                            | Echec reel possible ; ne pas afficher terminee      |
| HTTP 401                                                     | Incident auth distinct, pas le statut metier failed |

Evenements utiles : `generation_job_failed`, `generation_job_completion_failed`,
`generation_job_retry_scheduled`, `generation_job_failure_sync_failed`,
`generation_job_failure_mark_failed`, `generation_job_failure_reconciled`,
`generation_job_claim_lost`, `generation_job_leases_reaped`.

### Requetes SQL En Lecture Seule

A executer uniquement sur la base de production autorisee, avec une connexion
securisee. Ne pas recopier ses secrets dans un rapport. Ces requetes n'ont pas ete
executees en production pendant cet audit.

```sql
BEGIN TRANSACTION READ ONLY;

SELECT gr.id, gr.pipeline_status, gr.current_step, gr.progress_percent,
       gr.clarification_version, gr.started_at, gr.completed_at, gr.updated_at,
       c.id AS course_id, c.status AS course_status, c.updated_at AS course_updated_at
FROM generation_requests gr
LEFT JOIN courses c ON c.request_id = gr.id
WHERE gr.id = '90bc4e76-7f7e-48ac-893f-0f50b5cd1993';

SELECT id, kind, status, target_id, parent_job_id, attempt_count, max_attempts,
       last_error_code, created_at, started_at, completed_at,
       updated_at, failure_handled_at
FROM generation_jobs
WHERE request_id = '90bc4e76-7f7e-48ac-893f-0f50b5cd1993'
ORDER BY created_at, id;

SELECT m.id AS module_id, count(l.id) AS lesson_count,
       count(l.id) FILTER (
         WHERE nullif(btrim(l.content_markdown), '') IS NOT NULL
       ) AS markdown_count,
       count(l.id) FILTER (
         WHERE nullif(btrim(l.content_markdown), '') IS NOT NULL
           OR EXISTS (SELECT 1 FROM lesson_exercises e WHERE e.lesson_id = l.id)
           OR EXISTS (SELECT 1 FROM lesson_quizzes q WHERE q.lesson_id = l.id)
       ) AS available_content_count
FROM courses c
JOIN modules m ON m.course_id = c.id
LEFT JOIN lessons l ON l.module_id = m.id
WHERE c.request_id = '90bc4e76-7f7e-48ac-893f-0f50b5cd1993'
GROUP BY m.id
ORDER BY m.id;

COMMIT;
```

La definition actuelle de completude accepte du Markdown ou des exercices/quizzes
par lecon, avec au moins un module et au moins une lecon par module. Un beau sommaire
ou le seul compteur de lecons ne suffit pas. Les details internes de
`last_error_message` peuvent etre consultes en environnement protege si necessaire.
Les logs d'epoque peuvent manquer apres retention.

Reference : [predicat SQL de completude](C:/Users/samy0/OneDrive/Bureau/info/web/projects/course-ai/backend-go/internal/db/queries/courses.sql:166).

## 6. Plan De Correction

### Lot 1 : integrite backend

- Creer une finalisation atomique et idempotente de la demande et de la formation.
- Relire/verrouiller les etats avant mutation ; utiliser un ordre de verrouillage unique.
- Introduire une tentative explicite et proteger execution, persistance et echec contre les anciennes tentatives.
- Ne pas propager un echec de coordination a un contenu complet sans reconciliation.
- Garantir la presence d'un finaliseur/reconciliateur pour tout contenu devenu complet.
- Valider les commandes partielles avant d'accepter un job inexecutable.

Les invariants appartiennent au domaine, l'orchestration transactionnelle aux services,
les contrats aux ports et les requetes SQL aux adapters. Toute evolution de schema
passe par une nouvelle migration Goose ; regenerer sqlc, jamais modifier ses fichiers
generes manuellement.

### Lot 2 : contrat de suivi et frontend

- Exposer de facon coherente etat de traitement, disponibilite du contenu, progression et tentative.
- Partager le calcul de presentation entre suivi, historique et dashboard.
- Conserver l'acces au contenu disponible en cas d'echec technique.
- Corriger invalidations et revalidations sans polling infini des vrais etats terminaux.
- Distinguer "Actualiser" de "Relancer" ; traduire les codes d'erreur publics.
- Regenerer les types OpenAPI si le contrat change.

### Lot 3 : reparation des donnees existantes

1. Deployer les protections avant toute reparation, sinon un vieux worker peut recrasser les etats.
2. Inventorier les incoherences en mode dry-run et sauvegarder les etats concernes.
3. Verifier completude, tentative courante et absence de travail pertinent encore actif.
4. Reparer par le service transactionnel idempotent, sans effacer le contenu ni l'historique d'echec des jobs.
5. Verifier API, historique, suivi, catalogue et lecteur apres revalidation du cache.

Ne pas lancer d'UPDATE global `failed -> completed`. Ne pas regenerer toutes les
lecons deja valides. Une formation partielle doit rester partielle.

## 7. Tests A Ajouter Pour Valider Le Fix

| Scenario                                             | Invariant attendu apres correction                                                 |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Enqueue finaliseur indisponible apres derniere lecon | Contenu conserve, coordination recuperable, pas de faux echec definitif du contenu |
| Erreur entre completion formation/demande            | Pas de paire metier contradictoire commitee                                        |
| Acquittement de job perdu apres succes               | Rejeu idempotent sans degradation metier                                           |
| Callback v1 apres reprise v2                         | Aucun changement des etats v2                                                      |
| Worker ayant perdu son lease                         | Aucun commit metier obsolete                                                       |
| Finalisation concurrente avec echec/progression      | Etat terminal coherent, aucune regression                                          |
| Deux dernieres lecons paralleles                     | Finalisation assuree une fois la completude atteinte                               |
| Commande partielle sur demande failed                | Conflit clair ou reprise atomique executable                                       |
| Formation disponible et pipeline en erreur           | Avertissement exact et lien d'acces au contenu                                     |
| Job terminal observe                                 | Detail, listes et contenu coheremment revalides                                    |
| Mise a jour hors onglet                              | Focus/actualisation recupere la nouvelle projection                                |
| Historique monte pendant generation                  | Mise a jour jusqu'au resultat final                                                |
| Formations partielles                                | Jamais annoncees terminees sur simple presence de courseId                         |
| Erreur de suivi HTTP                                 | Ne devient pas un echec metier de generation                                       |
| Message public                                       | Francais/anglais localises, pas de details provider/SQL                            |

Executer les tests de concurrence avec PostgreSQL reel et synchronisation controlee :
les doubles en memoire ne prouvent pas la correction du verrouillage.

Apres implementation : gofmt, go vet, staticcheck, tests Go unitaires/race,
sqlc vet, integrations PostgreSQL ; puis unitaires frontend, typecheck, lint,
formatage, build et tests navigateur couvrant le parcours complet.

## 8. Verifications Effectuees

- Lecture du frontend, des services, du domaine, des DTO, des workers et des requetes SQL.
- Lecture de la base locale uniquement, avec deux releves identifies plus haut.
- Tentative d'acces a l'exemple de production : redirection vers la connexion.
- Quatre reproductions backend isolees : toutes ont confirme les anomalies.
- Quatre reproductions frontend jsdom : toutes ont confirme les comportements.
- Suites Go completes des packages service et workers : reussies avec
  `go test -p 1 -count=1 ./internal/service ./internal/infrastructure/jobs`.
- Les suites navigateur, le build et la verification globale du depot n'ont pas
  ete relances pour cet audit documentaire sans modification fonctionnelle.
- Aucun correctif fonctionnel, schema, secret, donnee ou deploiement modifie.

Les huit sondes et leurs commandes sont conservees dans
`generation-status-audit-reproductions-2026-09-17.md`, annexe de cet audit.
Elles ont ete retirees des suites actives apres execution pour ne pas imposer les
bugs actuels comme comportement de reference.

## 9. Recommandation Finale

Priorite : **finalisation atomique, protection contre les anciennes tentatives,
puis coherence du suivi frontend**. Le cache et le message generique aggravent
l'experience, mais une correction uniquement visuelle laisserait les defauts de
persistance et de reprise intacts.

La cause precise de la generation de production reste a confirmer avec les lectures
ciblees de la section 5. Les defauts de code et les scenarios reproduits, eux, sont
documentes et exploitables immediatement pour preparer le correctif.
