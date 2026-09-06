import { zodResolver } from '@hookform/resolvers/zod'
import { Sparkles } from 'lucide-react'
import { useRef } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { getErrorMessage } from '@/shared/api/error-message'
import { Button } from '@/shared/ui/button'
import { Field, Textarea } from '@/shared/ui/form-controls'
import { useStartGeneration } from './api'
import { generationPromptSchema, type GenerationPromptValues } from './schemas'

const examples = ['generation.exampleLinux', 'generation.exampleReact', 'generation.exampleGo'] as const

export function GenerationForm({ compact = false }: { compact?: boolean }) {
  const { t } = useTranslation()
  const mutation = useStartGeneration()
  const submissionKeys = useRef(new Map<string, string>())
  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors },
  } = useForm<GenerationPromptValues>({
    resolver: zodResolver(generationPromptSchema),
    defaultValues: { prompt: '' },
  })
  const prompt = useWatch({ control, name: 'prompt' })
  async function submit(values: GenerationPromptValues) {
    const idempotencyKey = submissionKeys.current.get(values.prompt) ?? crypto.randomUUID()
    submissionKeys.current.set(values.prompt, idempotencyKey)
    try {
      await mutation.mutateAsync({ prompt: values.prompt, idempotencyKey })
    } catch {
      // The mutation error remains visible; an unchanged prompt reuses its key.
    }
  }

  return (
    <form onSubmit={(event) => void handleSubmit(submit)(event)} className={compact ? '' : 'max-w-4xl'}>
      <Field
        label={t('generation.promptLabel')}
        htmlFor="generation-prompt"
        error={errors.prompt ? t('generation.promptInvalid') : undefined}
        hint={t('generation.promptHint', { count: prompt.length })}
      >
        <Textarea
          id="generation-prompt"
          disabled={mutation.isPending}
          aria-invalid={Boolean(errors.prompt)}
          maxLength={4000}
          rows={compact ? 4 : 7}
          placeholder={t('generation.promptPlaceholder')}
          {...register('prompt')}
        />
      </Field>
      <div className="mt-4 flex flex-wrap gap-2" aria-label={t('generation.examplesTitle')}>
        {examples.map((key) => (
          <button
            key={key}
            type="button"
            disabled={mutation.isPending}
            onClick={() => setValue('prompt', t(key), { shouldValidate: true })}
            className="rounded-md border border-border bg-surface px-3 py-2 text-left text-xs font-semibold text-muted-foreground transition hover:border-primary hover:text-primary"
          >
            {t(key)}
          </button>
        ))}
      </div>
      {mutation.error ? (
        <p className="mt-4 text-sm text-danger-strong" role="alert">
          {getErrorMessage(mutation.error, t)}
        </p>
      ) : null}
      <div className="mt-5 flex justify-end">
        <Button type="submit" size={compact ? 'default' : 'lg'} disabled={mutation.isPending}>
          <Sparkles className="size-4" />
          {mutation.isPending ? t('generation.submitting') : t('generation.submit')}
        </Button>
      </div>
    </form>
  )
}
