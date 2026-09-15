import {
  forwardRef,
  createContext,
  useContext,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react'
import { cn } from '@/shared/lib/cn'

const control =
  'w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground outline-none transition placeholder:text-muted-foreground focus:border-primary focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-60'

const FieldContext = createContext<{ describedBy?: string | undefined; invalid?: boolean }>({})

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => {
    const field = useContext(FieldContext)
    return (
      <input
        ref={ref}
        aria-describedby={field.describedBy}
        aria-invalid={field.invalid ? true : undefined}
        className={cn(control, 'h-11', className)}
        {...props}
      />
    )
  },
)
Input.displayName = 'Input'

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => {
    const field = useContext(FieldContext)
    return (
      <textarea
        ref={ref}
        aria-describedby={field.describedBy}
        aria-invalid={field.invalid ? true : undefined}
        className={cn(control, 'min-h-32 resize-y py-3 leading-6', className)}
        {...props}
      />
    )
  },
)
Textarea.displayName = 'Textarea'

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, ...props }, ref) => {
    const field = useContext(FieldContext)
    return (
      <select
        ref={ref}
        aria-describedby={field.describedBy}
        aria-invalid={field.invalid ? true : undefined}
        className={cn(control, 'h-11', className)}
        {...props}
      />
    )
  },
)
Select.displayName = 'Select'

interface FieldProps {
  label: string
  htmlFor: string
  error?: string | undefined
  hint?: string | undefined
  children: ReactNode
}

export function Field({ label, htmlFor, error, hint, children }: FieldProps) {
  return (
    <div className="space-y-2">
      <label className="block text-sm font-semibold text-foreground" htmlFor={htmlFor}>
        {label}
      </label>
      <FieldContext.Provider
        value={{ describedBy: error || hint ? `${htmlFor}-description` : undefined, invalid: Boolean(error) }}
      >
        {children}
      </FieldContext.Provider>
      {error ? (
        <p id={`${htmlFor}-description`} className="text-sm text-danger-strong" role="alert">
          {error}
        </p>
      ) : null}
      {!error && hint ? (
        <p id={`${htmlFor}-description`} className="text-xs text-muted-foreground">
          {hint}
        </p>
      ) : null}
    </div>
  )
}
