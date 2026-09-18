import type { GenerationJob } from '@/shared/api/types'

export function isJobActive(job: GenerationJob) {
  return job.status === 'queued' || job.status === 'running' || job.status === 'retry_scheduled'
}

// A retry supersedes an older terminal job for the same operation and target.
export function latestTargetJobs(jobs: GenerationJob[]) {
  const attempt = Math.max(1, ...jobs.map((job) => job.generationAttempt ?? 1))
  const latest = new Map<string, GenerationJob>()
  for (const job of jobs) {
    if ((job.generationAttempt ?? 1) !== attempt) continue
    const key = `${job.kind}:${job.targetId ?? ''}`
    const previous = latest.get(key)
    if (
      !previous ||
      Date.parse(job.createdAt) > Date.parse(previous.createdAt) ||
      (Date.parse(job.createdAt) === Date.parse(previous.createdAt) &&
        Date.parse(job.updatedAt) > Date.parse(previous.updatedAt))
    ) {
      latest.set(key, job)
    }
  }
  return [...latest.values()]
}

export function contentJobState(jobs: GenerationJob[], targetIds: string[]) {
  const targets = new Set(targetIds)
  const relevant = latestTargetJobs(jobs).filter(
    (job) =>
      job.targetId !== null &&
      targets.has(job.targetId) &&
      (job.kind === 'lesson_content' || job.kind === 'module_content'),
  )
  return {
    active: relevant.some(isJobActive),
    failed: relevant.some((job) => job.status === 'failed' || job.status === 'cancelled'),
  }
}
