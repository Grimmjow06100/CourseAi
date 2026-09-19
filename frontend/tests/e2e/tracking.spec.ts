import { expect, test } from '@playwright/test'
import { mockApi, tracking, requestId, moduleId, lessonId } from './mock-api'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
})

test('a partial generation retries one versioned task and preserves the same key after a lost response', async ({
  page,
}) => {
  await mockApi(page)
  let resumed = false
  const calls: { key: string | undefined; body: unknown }[] = []
  await page.route(`**/api/generations/${requestId}/tracking`, (route) =>
    route.fulfill({ json: tracking(resumed ? 'running' : 'failed', { revision: resumed ? '22' : '21' }) }),
  )
  await page.route(`**/api/generations/${requestId}/retry-failed`, async (route) => {
    calls.push({
      key: route.request().headers()['idempotency-key'],
      body: JSON.parse(route.request().postData() ?? '{}') as unknown,
    })
    if (calls.length === 1) return route.abort('failed')
    resumed = true
    return route.fulfill({
      status: 202,
      json: {
        requestId,
        generationAttempt: 1,
        revision: '22',
        replacements: [
          {
            previousJobId: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee',
            jobId: 'ffffffff-ffff-4fff-8fff-ffffffffffff',
            operationVersion: 2,
          },
        ],
      },
    })
  })
  await page.goto(`/generations/${requestId}`)
  const retry = page.getByRole('button', { name: 'Relancer la tâche arrêtée' })
  await retry.click()
  await expect(page.getByRole('alert').filter({ hasText: /réseau|connexion/i })).toBeVisible()
  await retry.click()
  await expect(retry).toHaveCount(0)
  expect(calls).toHaveLength(2)
  expect(calls[1]).toEqual(calls[0])
  expect(calls[0]?.key).toBeTruthy()
  expect(calls[0]?.body).toEqual({
    operations: [{ jobId: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee', operationVersion: 1 }],
  })
  await page.reload()
  await expect(page.getByRole('button', { name: 'Relancer la tâche arrêtée' })).toHaveCount(0)
})

test('500 lessons stay collapsed, retain exact counts, and do not issue per-lesson requests', async ({
  page,
}, testInfo) => {
  await mockApi(page)
  const snapshot = tracking('partial', {
    operations: [],
    contentAvailability: 'partial',
    hasActiveWork: false,
    counts: {
      modules: 20,
      plansReady: 20,
      lessonsExpected: 500,
      lessonsAvailable: 450,
      jobsActive: 0,
      jobsFailed: 50,
      jobsCompleted: 450,
    },
  })
  snapshot.modules = Array.from({ length: 20 }, (_, m) => ({
    id: moduleId.slice(0, -4) + String(m).padStart(4, '0'),
    title: `Module ${m + 1} — Administration`,
    order: m + 1,
    lessons: Array.from({ length: 25 }, (_, l) => ({
      id: lessonId.slice(0, -4) + String(m * 25 + l).padStart(4, '0'),
      title: `Leçon ${m * 25 + l + 1}`,
      order: l + 1,
      hasContent: m < 18,
    })),
  }))
  const template = tracking('failed').operations[0]
  if (!template) throw new Error('Missing operation fixture')
  snapshot.operations = snapshot.modules.flatMap((module) =>
    module.lessons.map((lesson) => ({
      ...template,
      id: lesson.id,
      targetId: lesson.id,
      status: lesson.hasContent ? 'completed' : 'failed',
      retryable: !lesson.hasContent,
    })),
  )
  let trackingReads = 0
  let lessonReads = 0
  page.on('request', (request) => {
    if (request.url().includes('/api/lessons/')) lessonReads++
  })
  await page.route(`**/api/generations/${requestId}/tracking`, async (route) => {
    trackingReads++
    await route.fulfill({ json: snapshot })
  })
  await page.goto(`/generations/${requestId}`)
  await expect(page.getByText('450 / 500 leçons disponibles', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Relancer les 50 tâches arrêtées' })).toBeVisible()
  await expect(page.getByText('Leçon 500', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: /Module 20 — Administration/ }).click()
  await expect(page.getByText('Leçon 500', { exact: false }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Relancer', exact: true })).toHaveCount(25)
  expect(lessonReads).toBe(0)
  expect(trackingReads).toBe(1)
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
  await page.screenshot({ path: testInfo.outputPath('tracking-500-lessons.png'), fullPage: true })
  await page.getByRole('tab', { name: 'Historique', exact: true }).click()
  await expect(page).toHaveURL(/tab=history/)
  await page.reload()
  await expect(page.getByRole('tab', { name: 'Historique', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  )
})
