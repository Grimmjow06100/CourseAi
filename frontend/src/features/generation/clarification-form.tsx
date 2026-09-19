import { Checkbox } from '@/components/ui/checkbox'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { useMemo, useState, type SyntheticEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { GenerationStatus } from '@/shared/api/types'
import { getErrorMessage } from '@/shared/api/error-message'
import { Button } from '@/components/ui/button'
import { FormField as Field } from '@/components/ui/form-field'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { useSubmitClarifications } from './api'
import { clarificationFormSchema } from './schemas'

export function ClarificationForm({ status }: { status: GenerationStatus }) {
  const { t } = useTranslation()
  const mutation = useSubmitClarifications(status.requestId)
  const initialAnswers = useMemo(
    () => Object.fromEntries(status.clarificationQuestions.map((question) => [question.id, [] as string[]])),
    [status.clarificationQuestions],
  )
  const [title, setTitle] = useState(status.suggestedTitle ?? '')
  const [synopsis, setSynopsis] = useState(status.shortSynopsis ?? '')
  const [language, setLanguage] = useState<'fr' | 'en'>(status.detectedLanguage ?? 'fr')
  const [answers, setAnswers] = useState<Record<string, string[]>>(initialAnswers)
  const [validationError, setValidationError] = useState<string | null>(null)

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault()
    const parsed = clarificationFormSchema.safeParse({ title, synopsis, language, answers })
    if (
      !parsed.success ||
      status.clarificationQuestions.some((question) => (answers[question.id]?.length ?? 0) === 0)
    ) {
      setValidationError(t('generation.clarificationInvalid'))
      return
    }
    setValidationError(null)
    try {
      await mutation.mutateAsync(parsed.data)
      toast.success(t('generation.trackingTitle'))
    } catch {
      /* rendered below */
    }
  }

  return (
    <form onSubmit={(event) => void submit(event)} className="space-y-6">
      <div className="grid gap-5 sm:grid-cols-2">
        <Field label={t('generation.titleLabel')} htmlFor="course-title">
          <Input id="course-title" value={title} onChange={(event) => setTitle(event.target.value)} />
        </Field>
        <Field label={t('generation.languageLabel')} htmlFor="course-language">
          <Select
            id="course-language"
            value={language}
            onChange={(event) => setLanguage(event.target.value as 'fr' | 'en')}
          >
            <option value="fr">Français</option>
            <option value="en">English</option>
          </Select>
        </Field>
      </div>
      <Field label={t('generation.synopsisLabel')} htmlFor="course-synopsis">
        <Textarea
          id="course-synopsis"
          value={synopsis}
          onChange={(event) => setSynopsis(event.target.value)}
        />
      </Field>
      {status.clarificationQuestions.map((question) => (
        <fieldset key={question.id} className="border-t border-border pt-5">
          <legend className="mb-3 text-sm font-bold">{question.question}</legend>
          {question.allowMultiple ? (
            <div className="grid gap-2 sm:grid-cols-2">
              {question.options.map((option) => {
                const checked = answers[question.id]?.includes(option.value) ?? false
                return (
                  <label
                    key={option.value}
                    className="flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-surface p-4 text-sm transition-colors hover:border-primary has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-muted"
                  >
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(next) =>
                        setAnswers((current) => ({
                          ...current,
                          [question.id]:
                            next === true
                              ? [...(current[question.id] ?? []), option.value]
                              : (current[question.id] ?? []).filter((value) => value !== option.value),
                        }))
                      }
                      className="mt-0.5 grid size-5 shrink-0 place-items-center rounded border border-border data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground"
                    ></Checkbox>
                    {option.label}
                  </label>
                )
              })}
            </div>
          ) : (
            <RadioGroup
              aria-label={question.question}
              value={answers[question.id]?.[0] ?? ''}
              onValueChange={(value) => setAnswers((current) => ({ ...current, [question.id]: [value] }))}
              className="grid gap-2 sm:grid-cols-2"
            >
              {question.options.map((option) => (
                <label
                  key={option.value}
                  className="flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-surface p-4 text-sm transition-colors hover:border-primary has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-muted"
                >
                  <RadioGroupItem
                    value={option.value}
                    className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full border border-border data-[state=checked]:border-primary"
                  ></RadioGroupItem>
                  {option.label}
                </label>
              ))}
            </RadioGroup>
          )}
        </fieldset>
      ))}
      {validationError ? (
        <p role="alert" className="text-sm text-danger-strong">
          {validationError}
        </p>
      ) : null}
      {mutation.error ? (
        <p role="alert" className="text-sm text-danger-strong">
          {getErrorMessage(mutation.error, t)}
        </p>
      ) : null}
      <Button type="submit" disabled={mutation.isPending}>
        {t('generation.continue')}
      </Button>
    </form>
  )
}
