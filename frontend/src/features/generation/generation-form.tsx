import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowRight, Code2, LoaderCircle, Server, Sparkles, Terminal } from 'lucide-react'
import { useRef } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { getErrorMessage } from '@/shared/api/error-message'
import { Button } from '@/shared/ui/button'
import { Field, Textarea } from '@/shared/ui/form-controls'
import { useStartGeneration } from './api'
import { generationPromptSchema, type GenerationPromptValues } from './schemas'

const examples = [
  { prompt: 'generation.exampleLinux', label: 'design.linux', icon: Terminal },
  { prompt: 'generation.exampleReact', label: 'design.react', icon: Code2 },
  { prompt: 'generation.exampleGo', label: 'design.go', icon: Server },
] as const

export function GenerationForm({ compact = false }: { compact?: boolean }) {
  const { t } = useTranslation()
  const mutation = useStartGeneration()
  const submissionKeys = useRef(new Map<string, string>())
  const {
    register,
    handleSubmit,
    setValue,
    setFocus,
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
    <form
      onSubmit={(event) => void handleSubmit(submit)(event)}
      aria-busy={mutation.isPending}
      className="flex h-full flex-col"
    >
      <div className="mb-5 flex items-center gap-3">
        <span className="grid size-10 place-items-center rounded-xl bg-success-soft text-primary">
          <Sparkles className="size-5" aria-hidden="true" />
        </span>
        <p className="text-base font-extrabold">{t('generation.title')}</p>
      </div>
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
          className="bg-background/50"
          placeholder={t('generation.promptPlaceholder')}
          {...register('prompt')}
        />
      </Field>
      <p className="mt-5 mb-2 text-xs font-semibold text-muted-foreground">{t('design.suggestions')}</p>
      <div className="flex flex-wrap gap-2" aria-label={t('generation.examplesTitle')}>
        {examples.map(({ prompt: key, label, icon: Icon }) => (
          <button
            key={key}
            type="button"
            disabled={mutation.isPending}
            onClick={() => {
              setValue('prompt', t(key), { shouldValidate: true })
              setFocus('prompt')
            }}
            className="flex min-h-10 items-center gap-2 rounded-lg border border-border bg-surface px-3 py-2 text-left text-xs font-semibold text-muted-foreground transition hover:border-primary hover:bg-success-soft hover:text-primary disabled:opacity-50"
          >
            <Icon className="size-3.5" aria-hidden="true" />
            {t(label)}
          </button>
        ))}
      </div>
      {mutation.error ? (
        <p className="mt-4 text-sm text-danger-strong" role="alert">
          {getErrorMessage(mutation.error, t)}
        </p>
      ) : null}
      <div className="mt-auto pt-6">
        <p className="mb-4 text-xs leading-5 text-muted-foreground">{t('design.generationNote')}</p>
        <Button type="submit" size={compact ? 'default' : 'lg'} disabled={mutation.isPending}>
          {mutation.isPending ? (
            <LoaderCircle className="size-4 animate-spin" />
          ) : (
            <Sparkles className="size-4" />
          )}
          {mutation.isPending ? t('generation.submitting') : t('generation.submit')}
          {!mutation.isPending ? <ArrowRight className="ml-2 size-4" /> : null}
        </Button>
      </div>
    </form>
  )
}
