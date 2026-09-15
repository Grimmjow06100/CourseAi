import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import type { ButtonHTMLAttributes } from 'react'
import { cn } from '@/shared/lib/cn'

const buttonVariants = cva(
  'inline-flex min-h-11 shrink-0 items-center justify-center gap-2 rounded-xl border text-sm font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        primary: 'border-primary bg-primary px-4 text-primary-foreground hover:bg-primary-strong',
        secondary: 'border-border bg-surface px-4 text-foreground hover:bg-muted',
        ghost:
          'border-transparent bg-transparent px-3 text-muted-foreground hover:bg-muted hover:text-foreground',
        danger: 'border-danger-strong bg-danger-strong px-4 text-surface hover:opacity-90',
        icon: 'size-11 border-border bg-surface p-0 text-muted-foreground hover:bg-muted hover:text-foreground',
      },
      size: { default: 'py-2', sm: 'min-h-9 px-3 py-1.5 text-xs', lg: 'min-h-12 px-5 py-3 text-sm' },
    },
    defaultVariants: { variant: 'primary', size: 'default' },
  },
)

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export function Button({ className, variant, size, asChild, ...props }: ButtonProps) {
  const Component = asChild ? Slot : 'button'
  return <Component className={cn(buttonVariants({ variant, size }), className)} {...props} />
}
