import { expect, test } from '@playwright/test'
import { courseId, lessonId, mockApi, requestId, tracking } from './mock-api'

test.beforeEach(async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
})

test('refreshing a long event history returns to the newest page', async ({ page }) => {
  const cursors: string[] = []
  await page.route(`**/api/generations/${requestId}/tracking`, (route) =>
    route.fulfill({ json: tracking('completed') }),
  )
  await page.route(`**/api/generations/${requestId}/events?*`, (route) => {
    const cursor = new URL(route.request().url()).searchParams.get('cursor') ?? '0'
    cursors.push(cursor)
    const id = cursor === '0' ? 600 : Number(cursor) - 1
    return route.fulfill({
      json: {
        items: [
          {
            id: String(id),
            generationAttempt: 1,
            jobId: null,
            kind: 'pipeline',
            status: 'completed',
            targetId: null,
            operationVersion: null,
            attemptCount: null,
            occurredAt: '2026-09-19T12:00:00Z',
          },
        ],
        nextCursor: String(id),
      },
    })
  })
  await page.goto(`/generations/${requestId}?tab=history`)
  const more = page.getByRole('button', { name: 'Afficher les événements précédents' })
  for (let pageIndex = 0; pageIndex < 6; pageIndex++) {
    await expect(more).toBeEnabled()
    await more.click()
    await expect.poll(() => cursors.length).toBe(pageIndex + 2)
  }
  await page.getByRole('button', { name: 'Vérifier les événements' }).click()
  await expect.poll(() => cursors.at(-1)).toBe('0')
})

test('history actions retain keyboard focus after cancelling deletion', async ({ page }) => {
  await page.route('**/api/generations?*', (route) =>
    route.fulfill({
      json: {
        items: [
          {
            requestId,
            title: 'Administration Linux',
            initialUserPrompt: 'Apprendre Linux',
            pipelineStatus: 'completed',
            courseId,
            courseStatus: 'completed',
            contentComplete: true,
            createdAt: '2026-09-19T12:00:00Z',
          },
        ],
        page: 1,
        pageSize: 20,
        totalItems: 1,
        totalPages: 1,
        hasNext: false,
        hasPrevious: false,
      },
    }),
  )
  await page.goto('/generations')
  const actions = page.getByRole('button', { name: /^Actions pour/ }).first()
  await actions.click()
  const remove = page.getByRole('menuitem', { name: 'Supprimer' })
  await remove.click()
  const dialog = page.getByRole('alertdialog')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Annuler' }).click()
  await expect(dialog).toBeHidden()
  await expect(remove).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(actions).toBeFocused()
})

test('desktop reader can hide and restore its curriculum', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'Desktop curriculum; mobile uses the tested Sheet')
  await page.goto(`/courses/${courseId}/lessons/${lessonId}`)
  await page.getByRole('button', { name: 'Masquer le programme' }).click()
  await expect(page.locator('#lesson-curriculum')).toBeHidden()
  await page.getByRole('button', { name: 'Afficher le programme' }).click()
  await expect(page.locator('#lesson-curriculum')).toBeVisible()
})

test('Course AI favicon loads at small sizes on both backgrounds', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'One asset check is sufficient')
  await page.goto('/')
  const href = await page.locator('link[rel="icon"]').getAttribute('href')
  expect(href).toContain('course-ai-mark-v2.svg')
  if (!href) throw new Error('Missing Course AI favicon')
  const response = await page.request.get(href)
  expect(response.ok()).toBe(true)
  expect(response.headers()['content-type']).toContain('image/svg+xml')
  await page.setContent(
    `<main style="display:flex;gap:24px;padding:24px">${['white', '#18181b'].map((background) => `<section style="background:${background};padding:24px;display:flex;gap:16px;align-items:center">${[16, 32].map((size) => `<img alt="Course AI ${size}px" src="${href}" width="${size}" height="${size}">`).join('')}</section>`).join('')}</main>`,
  )
  for (const img of await page.locator('img').all()) {
    await expect(img).toBeVisible()
    await expect
      .poll(() => img.evaluate((node) => (node as HTMLImageElement).naturalWidth))
      .toBeGreaterThan(0)
  }
  await page.screenshot({ path: testInfo.outputPath('favicon-light-dark-16-32.png') })
})
