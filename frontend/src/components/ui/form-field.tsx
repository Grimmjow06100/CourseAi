import { cloneElement, type ReactElement } from 'react'
import { Field, FieldDescription, FieldError, FieldLabel } from '@/components/ui/field'

// Associates a controlled or registered input with its visible label and error.
export function FormField({
  label,
  htmlFor,
  error,
  hint,
  children,
}: {
  label: string
  htmlFor: string
  error?: string | undefined
  hint?: string | undefined
  children: ReactElement<{ 'aria-describedby'?: string | undefined; 'aria-invalid'?: boolean | undefined }>
}) {
  const descriptionId = error || hint ? `${htmlFor}-description` : undefined
  return (
    <Field data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={htmlFor}>{label}</FieldLabel>
      {cloneElement(children, { 'aria-describedby': descriptionId, 'aria-invalid': Boolean(error) })}
      {error ? (
        <FieldError id={descriptionId}>{error}</FieldError>
      ) : hint ? (
        <FieldDescription id={descriptionId}>{hint}</FieldDescription>
      ) : null}
    </Field>
  )
}
