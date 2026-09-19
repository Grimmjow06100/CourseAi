import type { GenerationTracking } from '@/shared/api/types'

/** Keep the last coherent snapshot when a response from an older observation arrives. */
export function newestTracking(previous: GenerationTracking | undefined, incoming: GenerationTracking) {
  if (!previous) return incoming
  if (incoming.generationAttempt < previous.generationAttempt) return previous
  return BigInt(incoming.revision) < BigInt(previous.revision) ? previous : incoming
}

/** Availability and local incidents do not override the backend's lifecycle state. */
export function trackingSummary(snapshot: GenerationTracking) {
  if (snapshot.isOutOfScope) return 'scope'
  if (snapshot.contentComplete) return snapshot.reconciliation === 'pending' ? 'repair' : 'complete'
  if (snapshot.pipelineStatus === 'partial') return 'partial'
  if (snapshot.pipelineStatus === 'failed') return 'failed'
  if (snapshot.counts.jobsFailed > 0) return 'incidents'
  if (snapshot.reconciliation === 'attention_required') return 'attention'
  return null
}
