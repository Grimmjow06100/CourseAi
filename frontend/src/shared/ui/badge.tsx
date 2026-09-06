import type { HTMLAttributes } from 'react'
import { cn } from '@/shared/lib/cn'

type BadgeTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

const tones: Record<BadgeTone, string> = {
  neutral: 'border-border bg-muted text-muted-foreground',
  success: 'border-success/20 bg-success-soft text-success-strong',
  warning: 'border-warning/25 bg-warning-soft text-warning-strong',
  danger: 'border-danger/20 bg-danger-soft text-danger-strong',
  info: 'border-info/20 bg-info-soft text-info-strong',
}

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone
}

export function Badge({ className, tone = 'neutral', ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex min-h-6 items-center rounded-md border px-2 py-0.5 text-xs font-semibold',
        tones[tone],
        className,
      )}
      {...props}
    />
  )
}
