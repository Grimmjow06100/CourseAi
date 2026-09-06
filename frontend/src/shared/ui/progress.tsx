import * as ProgressPrimitive from '@radix-ui/react-progress'
import { cn } from '@/shared/lib/cn'

export function Progress({ value, className }: { value: number; className?: string }) {
  const bounded = Math.max(0, Math.min(100, value))
  return (
    <ProgressPrimitive.Root
      className={cn('h-2 w-full overflow-hidden rounded-full bg-muted', className)}
      value={bounded}
    >
      <ProgressPrimitive.Indicator
        className="h-full bg-primary transition-transform duration-500"
        style={{ transform: `translateX(-${100 - bounded}%)` }}
      />
    </ProgressPrimitive.Root>
  )
}
