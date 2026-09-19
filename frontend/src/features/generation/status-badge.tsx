import { useTranslation } from 'react-i18next'
import type { GenerationSummary } from '@/shared/api/types'
import { generationLabel, hasCompleteCourse } from './presentation'
import { Badge } from '@/components/ui/badge'

const tones = {
  queued: 'secondary',
  running: 'info',
  awaiting_clarification: 'warning',
  completed: 'success',
  failed: 'destructive',
  partial: 'warning',
} as const

export function PipelineStatusBadge({ generation }: { generation: GenerationSummary }) {
  const { t } = useTranslation()
  return (
    <Badge
      variant={
        hasCompleteCourse(generation) && generation.pipelineStatus !== 'completed'
          ? 'warning'
          : tones[generation.pipelineStatus]
      }
    >
      {t(generationLabel(generation))}
    </Badge>
  )
}
