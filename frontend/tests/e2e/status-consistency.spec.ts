import { expect, test } from '@playwright/test'
import { courseId, mockApi, requestId } from './mock-api'

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
  await page.route(`**/api/generations/${requestId}/status`, (route) =>
    route.fulfill({
      json: {
        requestId,
        courseId,
        pipelineStatus: complete ? 'completed' : 'failed',
        courseStatus: 'completed',
        contentComplete: true,
        generationAttempt: 2,
        progressPercent: complete ? 100 : 95,
        isOutOfScope: false,
        failureCode: complete ? null : 'finalization_failed',
        failureMessage: 'generation failed; retry the operation or contact support with the request id',
        clarificationQuestions: [],
        suggestedTitle: 'Linux',
      },
    }),
  )
  await page.goto(`/generations/${requestId}`)
  await expect(page.getByRole('heading', { name: 'Contenu disponible' })).toBeVisible()
  await expect(page.getByRole('link', { name: /commencer la formation/i })).toHaveAttribute(
    'href',
    `/courses/${courseId}`,
  )
  await expect(page.getByText(/generation failed;/)).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Réessayer', exact: true })).toHaveCount(0)
  complete = true
  await page.getByRole('button', { name: 'Actualiser le statut' }).click()
  await expect(page.getByRole('heading', { name: /formation est prête/i })).toBeVisible()
  expect(mutations).toBe(0)
})
