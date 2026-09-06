import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { mockApi } from './mock-api'

test('creates, clarifies, reads, and reveals a generated course', async ({ page }) => {
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  const api = await mockApi(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /apprendre aujourd/i })).toBeVisible()
  await page
    .getByLabel(/objectif de formation/i)
    .fill('Je veux apprendre Linux pour administrer des serveurs en production.')
  await page.getByRole('button', { name: /générer la formation/i }).click()
  await expect(page.getByRole('heading', { name: /précisions/i })).toBeVisible()
  await page.getByText('Débutant', { exact: true }).click()
  await page.getByRole('button', { name: /confirmer et continuer/i }).click()
  await expect(page.getByRole('heading', { name: /formation est prête/i })).toBeVisible()
  await page.getByRole('link', { name: /commencer la formation/i }).click()
  await expect(page.getByRole('heading', { name: 'Linux de zéro à autonome' })).toBeVisible()
  const startLesson = page.getByRole('link', { name: 'Commencer', exact: true })
  await expect(startLesson).toHaveAttribute('href', /\/lessons\//)
  await startLesson.click()
  await expect(page).toHaveURL(/\/lessons\//)
  await expect(page.getByText(/pas encore été généré/i)).toBeVisible()
  expect(api.getSolutionRequests()).toBe(0)
  await page.getByRole('button', { name: /générer cette leçon/i }).click()
  await expect(page.getByRole('heading', { name: 'Le shell' })).toBeVisible()
  expect(api.getSolutionRequests()).toBe(0)
  await page.getByRole('button', { name: /voir la correction/i }).click()
  await expect(page.getByText(/Utilisez.*pwd/)).toBeVisible()
  expect(api.getSolutionRequests()).toBe(1)
  // Audit a settled screen, not a toast partway through its exit animation.
  await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(
    accessibility.violations.filter(
      (violation) => violation.impact === 'critical' || violation.impact === 'serious',
    ),
  ).toEqual([])
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
  )
  expect(overflow).toBe(false)
  expect(consoleErrors).toEqual([])
})

test('supports keyboard navigation', async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /apprendre aujourd/i })).toBeVisible()
  await page.keyboard.press('Tab')
  const focused = await page.evaluate(() => document.activeElement?.tagName)
  expect(focused).not.toBe('BODY')
})
