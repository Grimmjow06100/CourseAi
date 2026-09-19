import { trackingFixture } from '@/test/tracking-fixture'
import { newestTracking, trackingSummary } from './tracking-state'

it('rejects a delayed older revision and an older generation attempt', () => {
  const current = trackingFixture({ revision: '9007199254740994', generationAttempt: 2 })
  expect(
    newestTracking(current, trackingFixture({ revision: '9007199254740993', generationAttempt: 2 })),
  ).toBe(current)
  expect(
    newestTracking(current, trackingFixture({ revision: '9007199254740995', generationAttempt: 1 })),
  ).toBe(current)
})

it('keeps a local job failure distinct from overall generation failure', () => {
  const active = trackingFixture({ pipelineStatus: 'running' })
  active.counts.jobsFailed = 1
  expect(trackingSummary(active)).toBe('incidents')
  expect(trackingSummary({ ...active, pipelineStatus: 'partial' })).toBe('partial')
  expect(trackingSummary({ ...active, pipelineStatus: 'failed' })).toBe('failed')
})

it('explains available content while final reconciliation is pending', () => {
  expect(trackingSummary(trackingFixture({ contentComplete: true, reconciliation: 'pending' }))).toBe(
    'repair',
  )
  expect(trackingSummary(trackingFixture({ contentComplete: true, reconciliation: 'none' }))).toBe('complete')
})
