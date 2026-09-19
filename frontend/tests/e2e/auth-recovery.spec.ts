import { expect, test } from '@playwright/test'
import { mockApi, requestId, tracking } from './mock-api'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await mockApi(page)
})

test('recovers both recent dashboard lists after an authentication rejection', async ({ page }) => {
  const attempts = new Map<string, number>()
  await page.route(/\/api\/(courses|generations)\?/, async (route) => {
    const path = new URL(route.request().url()).pathname
    const count = (attempts.get(path) ?? 0) + 1
    attempts.set(path, count)
    if (count > 1) return route.fallback()
    return route.fulfill({ status: 401, json: { code: 'unauthenticated', message: 'Invalid token' } })
  })
  await page.goto('/')
  await expect(page.getByText('Aucune génération en cours.')).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(attempts.get('/api/generations')).toBe(2)
  expect(attempts.get('/api/courses')).toBe(2)
})

test('retries the generation with the original prompt and idempotency key after a 401', async ({ page }) => {
  const attempts: { body: string | null; key: string | undefined }[] = []
  await page.route('**/api/generations', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    attempts.push({ body: route.request().postData(), key: route.request().headers()['idempotency-key'] })
    if (attempts.length > 1) return route.fallback()
    return route.fulfill({ status: 401, json: { code: 'unauthenticated', message: 'Invalid token' } })
  })
  await page.goto('/')
  await page
    .getByLabel(/objectif de formation/i)
    .fill('Je veux apprendre Linux pour administrer des serveurs.')
  await page.getByRole('button', { name: /générer la formation/i }).click()
  await expect(page.getByRole('heading', { name: /précisions/i })).toBeVisible()
  expect(attempts).toHaveLength(2)
  expect(attempts[0]?.key).toBeTruthy()
  expect(attempts[1]).toEqual(attempts[0])
  await expect(page.getByRole('alert')).toHaveCount(0)
})

test('continues generation polling after refreshing a rejected token', async ({ page }) => {
  let reads = 0
  await page.route(`**/api/generations/${requestId}/tracking`, async (route) => {
    reads++
    if (reads === 1) return route.fulfill({ json: tracking('running') })
    if (reads === 2)
      return route.fulfill({ status: 401, json: { code: 'unauthenticated', message: 'Invalid token' } })
    return route.fallback()
  })
  await page.goto(`/generations/${requestId}`)
  await expect(page.getByRole('heading', { name: /précisions/i })).toBeVisible()
  expect(reads).toBe(3)
  await expect(page.getByRole('alert')).toHaveCount(0)
})
test('bounds persistent 401 retries and allows a manual recovery on the dashboard', async ({ page }) => {
  let rejected = true
  let attempts = 0
  await page.route('**/api/generations?*', async (route) => {
    attempts++
    if (!rejected) return route.fallback()
    return route.fulfill({
      status: 401,
      headers: { 'X-Request-ID': 'auth-request', 'Access-Control-Expose-Headers': 'X-Request-ID' },
      json: { code: 'unauthenticated', message: 'Invalid token' },
    })
  })
  await page.goto('/')
  await expect(page.getByRole('alert')).toContainText('L’API n’a pas pu valider votre connexion')
  await expect(page.getByRole('alert')).toContainText('auth-request')
  expect(attempts).toBe(2)
  rejected = false
  await page.getByRole('button', { name: 'Réessayer' }).click()
  await expect(page.getByText('Aucune génération en cours.')).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(attempts).toBe(3)
})
