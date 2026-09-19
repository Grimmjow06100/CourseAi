import { expect, test } from '@playwright/test'
import { courseId, mockApi, requestId, tracking } from './mock-api'

test('keeps completed content accessible and refreshes status without retrying generation', async ({
  page,
}) => {
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await mockApi(page)
  let complete = false
  let mutations = 0
  page.on('request', (request) => {
    if (request.method() === 'POST') mutations++
  })
  await page.route(`**/api/generations/${requestId}/tracking`, (route) =>
    route.fulfill({
      json: tracking('completed', {
        pipelineStatus: complete ? 'completed' : 'failed',
        reconciliation: complete ? 'none' : 'pending',
        revision: complete ? '21' : '20',
      }),
    }),
  )
  await page.goto(`/generations/${requestId}`)
  await expect(
    page.getByText('Le contenu est disponible. La finalisation du suivi est en cours.'),
  ).toBeVisible()
  await expect(page.getByRole('link', { name: 'Ouvrir la formation' })).toHaveAttribute(
    'href',
    `/courses/${courseId}`,
  )
  await expect(page.getByRole('button', { name: 'Reprendre la génération', exact: true })).toHaveCount(0)
  complete = true
  await page.getByRole('button', { name: 'Vérifier maintenant' }).click()
  await expect(page.getByText('Toutes les leçons sont disponibles.')).toBeVisible()
  expect(mutations).toBe(0)
})
