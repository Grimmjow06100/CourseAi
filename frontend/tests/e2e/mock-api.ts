import type { GenerationTracking } from '../../src/shared/api/types'
import type { Page } from '@playwright/test'

export const requestId = 'd8c1571f-380e-44da-be06-9de1f8ba2aff'
export const courseId = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
export const moduleId = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
export const lessonId = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
const exerciseId = 'dddddddd-dddd-4ddd-8ddd-dddddddddddd'

const lessonBase = {
  id: lessonId,
  moduleId,
  order: 1,
  title: 'Comprendre le système Linux',
  type: 'mixed',
  estimatedDurationMinutes: 35,
  learningGoal: 'Comprendre les composants essentiels et utiliser le shell.',
  requiresDiagram: false,
  technicalKeywords: ['shell', 'filesystem'],
  createdAt: '2026-09-04T10:00:00Z',
  updatedAt: '2026-09-04T10:00:00Z',
}

function lesson(hasContent: boolean) {
  return {
    ...lessonBase,
    hasContent,
    hasStructuredActivities: hasContent,
    contentMarkdown: hasContent
      ? '## Le shell\n\nLinux expose ses outils à travers un **shell**.\n\n```bash\npwd\nls -la\n```'
      : null,
    exercises: hasContent
      ? [
          {
            id: exerciseId,
            lessonId,
            type: 'command_line',
            difficulty: 'beginner',
            title: 'Explorer le système',
            objective: 'Manipuler les commandes de navigation.',
            instructionsMarkdown: 'Exécutez les commandes dans un terminal.',
            contentMarkdown: '',
            payload: {
              tasks: ['Afficher le dossier courant', 'Lister les fichiers cachés'],
              resources: [],
              starterCode: 'pwd',
              expectedOutput: '/home/student',
              hints: ['Utilisez ls avec une option.'],
            },
            createdAt: '2026-09-04T10:00:00Z',
            updatedAt: '2026-09-04T10:00:00Z',
          },
        ]
      : [],
    quizzes: [],
  }
}

export function course(hasContent: boolean) {
  return {
    id: courseId,
    requestId,
    language: 'fr',
    status: hasContent ? 'completed' : 'lessons_generated',
    initialUserPrompt: 'Je veux apprendre Linux',
    title: 'Linux de zéro à autonome',
    synopsis: 'Un parcours pratique pour comprendre Linux et administrer un poste.',
    targetAudience: 'Développeurs débutants',
    currentLevel: 'beginner',
    targetLevel: 'intermediate',
    prerequisites: ['Aucun prérequis'],
    goals: ['Utiliser le terminal'],
    acquiredSkills: ['Naviguer dans Linux'],
    finalProjectTitle: 'Serveur Linux',
    finalProjectDescription: 'Configurer un serveur.',
    finalProjectConstraints: ['Documenter les commandes'],
    totalDurationMinutes: 35,
    modules: [
      {
        id: moduleId,
        courseId,
        order: 1,
        title: 'Fondations Linux',
        description: 'Les concepts essentiels.',
        keyLearningPoints: ['Shell'],
        totalDurationMinutes: 35,
        lessons: [lesson(hasContent)],
        createdAt: '2026-09-04T10:00:00Z',
        updatedAt: '2026-09-04T10:00:00Z',
      },
    ],
    createdAt: '2026-09-04T10:00:00Z',
    updatedAt: '2026-09-04T10:00:00Z',
  }
}

export function tracking(
  state: GenerationTracking['pipelineStatus'] = 'running',
  overrides: Partial<GenerationTracking> = {},
): GenerationTracking {
  const complete = state === 'completed'
  const ready = complete ? 1 : 0
  return {
    requestId,
    courseId: state === 'awaiting_clarification' ? null : courseId,
    generationAttempt: 1,
    revision: '20',
    observedAt: '2026-09-19T12:00:00Z',
    createdAt: '2026-09-19T11:00:00Z',
    completedAt: complete ? '2026-09-19T12:00:00Z' : null,
    title: 'Linux de zéro à autonome',
    pipelineStatus: state,
    isOutOfScope: false,
    historyComplete: true,
    contentComplete: complete,
    contentAvailability: complete ? 'complete' : 'none',
    hasActiveWork: state === 'running',
    reconciliation: 'none',
    counts: {
      modules: 1,
      plansReady: 1,
      lessonsExpected: 1,
      lessonsAvailable: ready,
      jobsActive: state === 'running' ? 1 : 0,
      jobsFailed: state === 'failed' ? 1 : 0,
      jobsCompleted: complete ? 1 : 0,
    },
    phases: (['analysis', 'architecture', 'lesson_plan', 'lesson_content', 'finalize_course'] as const).map(
      (kind, index) => ({
        kind,
        status:
          state === 'awaiting_clarification'
            ? index === 0
              ? 'awaiting_clarification'
              : 'pending'
            : complete || index < 3
              ? 'completed'
              : index === 3
                ? state === 'partial'
                  ? 'failed'
                  : state
                : 'pending',
      }),
    ),
    modules: [
      {
        id: moduleId,
        title: 'Fondations Linux',
        order: 1,
        lessons: [{ id: lessonId, title: 'Comprendre le système Linux', order: 1, hasContent: complete }],
      },
    ],
    operations: [
      {
        id: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee',
        targetId: lessonId,
        parentJobId: null,
        kind: 'lesson_content',
        status: state === 'failed' ? 'failed' : complete ? 'completed' : 'running',
        operationVersion: 1,
        supersedesJobId: null,
        attemptCount: 1,
        maxAttempts: 3,
        availableAt: '2026-09-19T11:00:00Z',
        startedAt: null,
        completedAt: null,
        retryable: state === 'failed',
        failureCode: null,
      },
    ],
    ...overrides,
  }
}

export async function mockApi(page: Page) {
  let clarified = false
  let contentGenerated = false
  let solutionRequests = 0
  await page.route('http://api.course-ai.test/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const json = (body: unknown, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    if (request.headers().authorization !== 'Bearer e2e-test-token')
      return json({ code: 'unauthorized', message: 'Missing token' }, 401)
    if (url.pathname === '/api/generations' && request.method() === 'GET')
      return json({
        items: [],
        page: 1,
        pageSize: 20,
        totalItems: 0,
        totalPages: 0,
        hasNext: false,
        hasPrevious: false,
      })
    if (url.pathname === '/api/courses' && request.method() === 'GET')
      return json({
        items: [],
        page: 1,
        pageSize: 12,
        totalItems: 0,
        totalPages: 0,
        hasNext: false,
        hasPrevious: false,
      })
    if (url.pathname === '/api/generations' && request.method() === 'POST')
      return json(
        {
          jobId: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee',
          requestId,
          status: 'queued',
          jobStatus: 'queued',
          statusUrl: `/api/generations/${requestId}/status`,
          jobStatusUrl: '/api/generation-jobs/job',
          resultUrl: `/api/generations/${requestId}/result`,
        },
        202,
      )
    if (url.pathname === `/api/generations/${requestId}/tracking`)
      return json(
        tracking(
          clarified ? 'completed' : 'awaiting_clarification',
          clarified
            ? {}
            : {
                modules: [],
                operations: [],
                counts: {
                  modules: 0,
                  plansReady: 0,
                  lessonsAvailable: 0,
                  lessonsExpected: null,
                  jobsActive: 0,
                  jobsFailed: 0,
                  jobsCompleted: 1,
                },
              },
        ),
      )
    if (url.pathname === `/api/generations/${requestId}/events`) return json({ items: [], nextCursor: null })
    if (url.pathname === `/api/generations/${requestId}/status`)
      return json(
        clarified
          ? {
              requestId,
              courseId,
              pipelineStatus: 'completed',
              courseStatus: 'completed',
              currentStep: 'completed',
              progressPercent: 100,
              failureMessage: null,
              isOutOfScope: false,
              errorMessage: null,
              warningMessage: null,
              suggestedTitle: 'Linux de zéro à autonome',
              shortSynopsis: 'Un parcours Linux.',
              detectedCurrentLevel: 'beginner',
              detectedTargetLevel: 'intermediate',
              detectedGoal: 'Administrer Linux',
              detectedLanguage: 'fr',
              clarificationQuestions: [],
              actionRequired: null,
            }
          : {
              requestId,
              courseId: null,
              pipelineStatus: 'awaiting_clarification',
              courseStatus: null,
              currentStep: 'awaiting_clarification',
              progressPercent: 25,
              failureMessage: null,
              isOutOfScope: false,
              errorMessage: null,
              warningMessage: null,
              suggestedTitle: 'Linux de zéro à autonome',
              shortSynopsis: 'Un parcours Linux pratique.',
              detectedCurrentLevel: 'unknown',
              detectedTargetLevel: 'intermediate',
              detectedGoal: 'Administrer Linux',
              detectedLanguage: 'fr',
              clarificationQuestions: [
                {
                  id: 'currentLevel',
                  question: 'Quel est votre niveau actuel ?',
                  options: [
                    { value: 'beginner', label: 'Débutant' },
                    { value: 'intermediate', label: 'Intermédiaire' },
                  ],
                  allowMultiple: false,
                },
              ],
              actionRequired: {
                type: 'submit_clarifications',
                url: `/api/generations/${requestId}/clarifications`,
              },
            },
      )
    if (url.pathname === `/api/generations/${requestId}/clarifications` && request.method() === 'POST') {
      clarified = true
      contentGenerated = true
      return json(
        {
          jobId: 'ffffffff-ffff-4fff-8fff-ffffffffffff',
          requestId,
          status: 'running',
          jobStatus: 'queued',
          statusUrl: `/api/generations/${requestId}/status`,
          jobStatusUrl: '/api/generation-jobs/job',
          resultUrl: `/api/generations/${requestId}/result`,
        },
        202,
      )
    }
    if (url.pathname === `/api/generations/${requestId}/jobs`) return json([])
    if (url.pathname === `/api/courses/${courseId}`) return json(course(contentGenerated))
    if (url.pathname === `/api/lessons/${lessonId}` && request.method() === 'GET')
      return json(lesson(contentGenerated))
    if (url.pathname === `/api/generations/lessons/${lessonId}/content` && request.method() === 'POST') {
      contentGenerated = true
      return json(
        {
          jobId: '11111111-1111-4111-8111-111111111111',
          requestId,
          status: 'running',
          jobStatus: 'queued',
          statusUrl: '',
          jobStatusUrl: '',
          resultUrl: '',
        },
        202,
      )
    }
    if (url.pathname === `/api/lessons/${lessonId}/solutions`) {
      solutionRequests += 1
      return json({
        lessonId,
        exercises: [{ exerciseId, correctionMarkdown: 'Utilisez `pwd` puis `ls -la`.' }],
        quizzes: [],
      })
    }
    return json({ code: 'not_found', message: url.pathname }, 404)
  })
  return {
    getSolutionRequests: () => solutionRequests,
    setContentGenerated: (value: boolean) => {
      contentGenerated = value
    },
  }
}
