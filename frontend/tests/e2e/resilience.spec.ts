import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { course, courseId, lessonId, mockApi, requestId, tracking } from './mock-api'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
})

test('restores running work after reload and stops polling after terminal failure', async ({ page }) => {
  await mockApi(page)
  let failed = false
  let reads = 0
  await page.route(`**/api/generations/${requestId}/tracking`, async (route) => {
    reads++
    await route.fulfill({ json: tracking(failed ? 'failed' : 'running', { revision: failed ? '21' : '20' }) })
  })
  await page.goto(`/courses/${courseId}/lessons/${lessonId}`)
  await expect(page.getByText('En cours', { exact: true })).toBeVisible()
  await page.reload()
  await expect(page.getByText('En cours', { exact: true })).toBeVisible()
  failed = true
  await expect(page.getByText('Échec', { exact: true })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Voir le suivi de génération' })).toBeVisible()
  const terminalReads = reads
  await page.waitForTimeout(2_500)
  expect(reads).toBe(terminalReads)
})

test('refreshes the course when generated lessons become available', async ({ page }) => {
  const api = await mockApi(page)
  let finished = false
  await page.route(`**/api/generations/${requestId}/tracking`, (route) =>
    route.fulfill({
      json: tracking(finished ? 'completed' : 'running', { revision: finished ? '21' : '20' }),
    }),
  )
  await page.goto(`/courses/${courseId}`)
  await expect(page.getByRole('heading', { name: 'Linux de zéro à autonome' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Commencer', exact: true })).toHaveCount(0)
  api.setContentGenerated(true)
  finished = true
  await expect(page.getByRole('link', { name: 'Commencer', exact: true })).toBeVisible()
})
test('shows dashboard failures and allows retry instead of displaying empty data', async ({ page }) => {
  await mockApi(page)
  let unavailable = true
  await page.route('**/api/generations?*', async (route) => {
    if (!unavailable) return route.fallback()
    return route.fulfill({ status: 503, json: { code: 'service_unavailable', message: 'Unavailable' } })
  })
  await page.goto('/')
  await expect(page.getByRole('alert')).toContainText('indisponible')
  await expect(page.getByText('Aucune génération en cours.')).toHaveCount(0)
  unavailable = false
  await page.getByRole('button', { name: 'Réessayer' }).click()
  await expect(page.getByText('Aucune génération en cours.')).toBeVisible()
})

test('keeps a failed deletion confirmation open', async ({ page }) => {
  await mockApi(page)
  await page.route('**/api/courses?*', (route) =>
    route.fulfill({
      json: {
        items: [course(false)],
        page: 1,
        pageSize: 12,
        totalItems: 1,
        totalPages: 1,
        hasNext: false,
        hasPrevious: false,
      },
    }),
  )
  await page.route(`**/api/courses/${courseId}`, async (route) => {
    if (route.request().method() !== 'DELETE') return route.fallback()
    return route.fulfill({ status: 503, json: { code: 'unavailable', message: 'Unavailable' } })
  })
  await page.goto('/courses')
  await page.getByRole('button', { name: 'Supprimer', exact: true }).click()
  const dialog = page.getByRole('alertdialog')
  await dialog.getByRole('button', { name: 'Supprimer', exact: true }).click()
  await expect(dialog.getByRole('alert')).toBeVisible()
  await expect(dialog).toBeVisible()
})

test('respects browser history when a debounced catalog search changes', async ({ page }) => {
  await mockApi(page)
  await page.goto('/courses?search=linux')
  const search = page.getByRole('textbox', { name: 'Rechercher une formation' })
  await search.fill('react')
  await expect(page).toHaveURL(/search=react/)
  await expect(search).toBeFocused()
  await page.getByRole('combobox', { name: 'Toutes les langues' }).selectOption('en')
  await expect(page).toHaveURL(/language=en/)
  await search.fill('go')
  await expect(page).toHaveURL(/search=go/)
  await page.goBack()
  await expect(search).toHaveValue('react')
  await page.waitForTimeout(500)
  await expect(page).toHaveURL(/search=react/)
})

test('mobile navigation has a named dialog and restores focus when closed', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'Desktop uses permanent navigation')
  await mockApi(page)
  await page.goto('/')
  const menu = page.getByRole('button', { name: 'Menu', exact: true })
  await menu.click()
  await expect(page.getByRole('dialog', { name: 'Menu' })).toBeVisible()
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')).toEqual(
    [],
  )
  await page.keyboard.press('Escape')
  await expect(menu).toBeFocused()
})
