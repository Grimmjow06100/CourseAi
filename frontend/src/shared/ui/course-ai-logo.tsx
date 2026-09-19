import { cn } from '@/shared/lib/cn'

export function CourseAiLogo({ compact = false, className }: { compact?: boolean; className?: string }) {
  return (
    <span className={cn('inline-flex shrink-0 items-center gap-2.5 font-semibold tracking-tight', className)}>
      <img
        src="/course-ai-mark-v2.svg"
        alt={compact ? 'Course AI' : ''}
        className="size-8 shrink-0"
        width={32}
        height={32}
      />
      {!compact && <span>Course AI</span>}
    </span>
  )
}
