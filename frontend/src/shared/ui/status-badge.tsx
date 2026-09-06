import { useTranslation } from 'react-i18next'
import type { PipelineStatus } from '@/shared/api/types'
import { Badge } from './badge'

const tones = {
  queued: 'neutral',
  running: 'info',
  awaiting_clarification: 'warning',
  completed: 'success',
  failed: 'danger',
} as const

export function PipelineStatusBadge({ status }: { status: PipelineStatus }) {
  const { t } = useTranslation()
  return <Badge tone={tones[status]}>{t(`generation.status.${status}`)}</Badge>
}
