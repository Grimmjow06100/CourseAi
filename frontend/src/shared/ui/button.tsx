import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import type { ButtonHTMLAttributes } from 'react'
import { cn } from '@/shared/lib/cn'

const buttonVariants = cva(
  'inline-flex h-10 items-center justify-center gap-2 rounded-md border text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        primary: 'border-primary bg-primary px-4 text-primary-foreground hover:bg-primary-strong',
        secondary: 'border-border bg-surface px-4 text-foreground hover:bg-muted',
        ghost:
          'border-transparent bg-transparent px-3 text-muted-foreground hover:bg-muted hover:text-foreground',
        danger: 'border-danger bg-danger px-4 text-white hover:bg-danger-strong',
        icon: 'size-10 border-border bg-surface p-0 text-muted-foreground hover:bg-muted hover:text-foreground',
      },
      size: { default: '', sm: 'h-8 px-3 text-xs', lg: 'h-12 px-5 text-base' },
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
