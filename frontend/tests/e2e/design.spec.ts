import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { course, courseId, lessonId, mockApi, requestId } from './mock-api'

const screens = [
  ['dashboard', '/'],
  ['generate', '/generate'],
  ['library', '/courses'],
  ['history', '/generations'],
  ['clarification', `/generations/${requestId}`],
  ['course', `/courses/${courseId}`],
  ['lesson', `/courses/${courseId}/lessons/${lessonId}`],
  ['not-found', '/page-inconnue'],
] as const

for (const colorScheme of ['light', 'dark'] as const) {
  for (const [name, path] of screens) {
    test(`${name} is accessible and fits the viewport in ${colorScheme} mode`, async ({ page }, testInfo) => {
      const errors: string[] = []
      page.on('pageerror', (error) => errors.push(error.message))
      await page.emulateMedia({ colorScheme, reducedMotion: 'reduce' })
      await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
      const api = await mockApi(page)
      api.setContentGenerated(true)
      await page.route('**/api/courses?*', (route) =>
        route.fulfill({
          json: {
            items: [
              course(true),
              {
                ...course(true),
                id: 'second-course',
                title: 'Concevoir des applications React accessibles et maintenables',
                targetLevel: 'advanced',
              },
              {
                ...course(false),
                id: 'third-course',
                title: 'Développer des API performantes avec Go',
                language: 'en',
              },
            ],
            page: 1,
            pageSize: 12,
            totalItems: 15,
            totalPages: 2,
            hasNext: true,
            hasPrevious: false,
          },
        }),
      )
      await page.route('**/api/generations?*', (route) =>
        route.fulfill({
          json: {
            items: [
              {
                requestId,
                courseId: null,
                initialUserPrompt: 'Je souhaite apprendre Linux pour administrer des serveurs.',
                title: 'Mon parcours Linux',
                pipelineStatus: 'awaiting_clarification',
                courseStatus: null,
                currentStep: 'awaiting_clarification',
                progressPercent: 25,
                isOutOfScope: false,
                failureMessage: null,
                createdAt: '2026-09-04T10:00:00Z',
                updatedAt: '2026-09-04T10:00:00Z',
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
      await page.goto(path)
      await expect(page.locator('main h1')).toBeVisible()
      if (name === 'library')
        await expect(page.getByRole('link', { name: 'Linux de zéro à autonome', exact: true })).toBeVisible()
      if (name === 'dashboard' || name === 'history')
        await expect(page.getByRole('heading', { name: 'Mon parcours Linux' })).toBeVisible()
      await page.evaluate(() => document.fonts.ready)
      await page.screenshot({ path: testInfo.outputPath(`${name}-${colorScheme}.png`), fullPage: true })
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
        ),
        name,
      ).toBe(false)
      const audit = await new AxeBuilder({ page }).analyze()
      expect(
        audit.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious'),
        name,
      ).toEqual([])
      expect(errors).toEqual([])
    })
  }
}

test('theme preference persists, language switches, and suggestions focus the prompt', async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await page.goto('/')
  await page.getByRole('button', { name: 'Thème : Système. Passer à Clair' }).click()
  await page.getByRole('button', { name: 'Thème : Clair. Passer à Sombre' }).click()
  await page.reload()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.getByRole('button', { name: 'Thème : Sombre. Passer à Système' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'system')
  await page.getByRole('button', { name: 'en', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'What will you learn today?' })).toBeVisible()
  await page.getByRole('button', { name: 'Linux & systems' }).click()
  const prompt = page.getByRole('textbox', { name: 'Your learning goal' })
  await expect(prompt).toBeFocused()
  await expect(prompt).toHaveValue('Go from zero to autonomous Linux server administration')
})

test('empty library distinguishes filters and offers a useful next action', async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await page.goto('/courses?search=linux&status=failed')
  await expect(
    page.getByRole('heading', { name: 'Aucune formation ne correspond à ces filtres.' }),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Effacer les filtres' }).first().click()
  await expect(page.getByRole('textbox', { name: 'Rechercher une formation' })).toHaveValue('')
  await expect(page.getByRole('heading', { name: 'Votre bibliothèque est encore vide.' })).toBeVisible()
  await expect(page).not.toHaveURL(/search=|status=/)
  await page.getByRole('main').getByRole('link', { name: 'Nouvelle formation' }).last().click()
  await expect(page).toHaveURL(/\/generate$/)
})

test('invalid prompts expose their error as a field description', async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await page.goto('/generate')
  await page.getByRole('button', { name: 'Générer la formation' }).click()
  const prompt = page.getByRole('textbox', { name: 'Votre objectif de formation' })
  await expect(prompt).toHaveAttribute('aria-invalid', 'true')
  await expect(prompt).toHaveAccessibleDescription('Décrivez votre objectif en 10 à 4000 caractères.')
})

test('small screens fit long course titles and content', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'One dedicated 320px layout check')
  await page.setViewportSize({ width: 320, height: 800 })
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  await page.route('**/api/courses?*', (route) =>
    route.fulfill({
      json: {
        items: [
          { ...course(true), title: 'ArchitectureDesApplicationsDistribuéesAvecKubernetesEtObservabilité' },
        ],
        page: 1,
        pageSize: 12,
        totalItems: 1,
        totalPages: 1,
        hasNext: false,
        hasPrevious: false,
      },
    }),
  )
  for (const path of ['/', '/courses', '/generate', `/courses/${courseId}/lessons/${lessonId}`]) {
    await page.goto(path)
    await expect(page.locator('main h1')).toBeVisible()
    if (path === '/' || path === '/courses')
      await expect(
        page.getByRole('heading', {
          name: 'ArchitectureDesApplicationsDistribuéesAvecKubernetesEtObservabilité',
        }),
      ).toBeVisible()
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth),
      path,
    ).toBe(false)
  }
  await page.screenshot({ path: testInfo.outputPath('reader-320px.png'), fullPage: true })
})

test('generation progress, failure, and completion expose accessible next steps', async ({ page }) => {
  await mockApi(page)
  await page.addInitScript(() => localStorage.setItem('course-ai-language', 'fr'))
  let state = 'running'
  await page.route(`**/api/generations/${requestId}/status`, (route) =>
    route.fulfill({
      json: {
        requestId,
        courseId: state === 'completed' ? courseId : null,
        pipelineStatus: state,
        courseStatus: null,
        currentStep: 'content',
        progressPercent: state === 'completed' ? 100 : 80,
        failureMessage: null,
        isOutOfScope: false,
        errorMessage: null,
        warningMessage: null,
        suggestedTitle: 'Parcours Linux',
        shortSynopsis: 'Apprendre Linux.',
        detectedCurrentLevel: 'beginner',
        detectedTargetLevel: 'intermediate',
        detectedGoal: 'Administrer Linux',
        detectedLanguage: 'fr',
        clarificationQuestions: [],
        actionRequired: null,
      },
    }),
  )
  await page.goto(`/generations/${requestId}`)
  await expect(page.getByRole('progressbar', { name: 'Construction de votre formation' })).toHaveAttribute(
    'aria-valuenow',
    '80',
  )
  for (const next of ['running', 'failed', 'completed']) {
    if (next !== state) {
      state = next
      await page.reload()
    }
    if (next === 'failed') await expect(page.getByRole('button', { name: 'Réessayer' })).toBeVisible()
    if (next === 'completed')
      await expect(page.getByRole('link', { name: 'Commencer la formation' })).toBeVisible()
    const audit = await new AxeBuilder({ page }).analyze()
    expect(
      audit.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious'),
      next,
    ).toEqual([])
  }
})
