import * as Checkbox from '@radix-ui/react-checkbox'
import * as RadioGroup from '@radix-ui/react-radio-group'
import { Check, ChevronDown, Lightbulb, LockKeyhole } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Exercise, LessonSolutions, Quiz } from '@/shared/api/types'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/form-controls'
import { Markdown } from '@/shared/ui/markdown'

function ExerciseBlock({
  exercise,
  solutions,
  reveal,
}: {
  exercise: Exercise
  solutions: LessonSolutions | undefined
  reveal: () => void
}) {
  const { t } = useTranslation()
  const [hintsOpen, setHintsOpen] = useState(false)
  const [revealed, setRevealed] = useState(false)
  const correction = revealed
    ? solutions?.exercises.find((item) => item.exerciseId === exercise.id)
    : undefined
  return (
    <article className="rounded-lg border border-border bg-surface p-5">
      <p className="font-mono text-xs uppercase text-primary">
        {exercise.type.replaceAll('_', ' ')} · {exercise.difficulty}
      </p>
      <h3 className="mt-2 text-lg font-bold">{exercise.title}</h3>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">{exercise.objective}</p>
      <div className="mt-5">
        <Markdown>{exercise.instructionsMarkdown}</Markdown>
      </div>
      {exercise.contentMarkdown ? (
        <div className="mt-5">
          <Markdown>{exercise.contentMarkdown}</Markdown>
        </div>
      ) : null}
      {exercise.payload.tasks.length ? (
        <div className="mt-5">
          <h4 className="text-sm font-bold">{t('lesson.tasks')}</h4>
          <ol className="mt-2 list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
            {exercise.payload.tasks.map((task) => (
              <li key={task}>{task}</li>
            ))}
          </ol>
        </div>
      ) : null}
      {exercise.payload.starterCode ? (
        <div className="mt-5">
          <Markdown>{`\`\`\`\n${exercise.payload.starterCode}\n\`\`\``}</Markdown>
        </div>
      ) : null}
      {exercise.payload.expectedOutput ? (
        <div className="mt-5 rounded-md border border-border bg-muted p-3">
          <h4 className="mb-2 text-sm font-bold">{t('lesson.expectedOutput')}</h4>
          <pre className="overflow-x-auto font-mono text-xs">{exercise.payload.expectedOutput}</pre>
        </div>
      ) : null}
      {exercise.payload.resources.length ? (
        <div className="mt-5">
          <h4 className="text-sm font-bold">{t('lesson.resources')}</h4>
          <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            {exercise.payload.resources.map((resource) => (
              <li key={resource}>{resource}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {exercise.payload.hints.length ? (
        <div className="mt-5">
          <Button variant="ghost" size="sm" onClick={() => setHintsOpen((value) => !value)}>
            <Lightbulb className="size-4" />
            {t('lesson.showHints')}
            <ChevronDown className={`size-4 transition ${hintsOpen ? 'rotate-180' : ''}`} />
          </Button>
          {hintsOpen ? (
            <ul className="mt-3 space-y-2 border-l-2 border-accent pl-4 text-sm text-muted-foreground">
              {exercise.payload.hints.map((hint) => (
                <li key={hint}>{hint}</li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
      <div className="mt-6 border-t border-border pt-4">
        {correction ? (
          <div>
            <h4 className="mb-3 text-sm font-bold">{t('lesson.solution')}</h4>
            <Markdown>{correction.correctionMarkdown}</Markdown>
          </div>
        ) : (
          <Button
            variant="secondary"
            onClick={() => {
              setRevealed(true)
              reveal()
            }}
          >
            <LockKeyhole className="size-4" />
            {t('lesson.revealSolution')}
          </Button>
        )}
      </div>
    </article>
  )
}

function QuizBlock({
  quiz,
  solutions,
  reveal,
}: {
  quiz: Quiz
  solutions: LessonSolutions | undefined
  reveal: () => void
}) {
  const { t } = useTranslation()
  const [answers, setAnswers] = useState<Record<number, string[]>>({})
  const [revealed, setRevealed] = useState(false)
  const solution = revealed ? solutions?.quizzes.find((item) => item.quizId === quiz.id) : undefined
  return (
    <article className="rounded-lg border border-border bg-surface p-5">
      <p className="font-mono text-xs uppercase text-primary">
        {quiz.type.replaceAll('_', ' ')} · {quiz.difficulty}
      </p>
      <h3 className="mt-2 text-lg font-bold">{quiz.title}</h3>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">{quiz.objective}</p>
      <div className="mt-6 space-y-7">
        {quiz.questions.map((question) => {
          const questionSolution = solution?.questions.find((item) => item.order === question.order)
          const multiple = question.type === 'multiple_choice'
          return (
            <fieldset key={question.order}>
              <legend className="text-sm font-bold">
                {question.order}. {question.question}
              </legend>
              {question.type === 'short_answer' ? (
                <Input
                  className="mt-3"
                  aria-label={t('lesson.shortAnswer')}
                  value={answers[question.order]?.[0] ?? ''}
                  onChange={(event) =>
                    setAnswers((current) => ({ ...current, [question.order]: [event.target.value] }))
                  }
                />
              ) : multiple ? (
                <div className="mt-3 space-y-2">
                  {question.options.map((option) => {
                    const checked = answers[question.order]?.includes(option.text) ?? false
                    return (
                      <label key={option.order} className="flex cursor-pointer items-center gap-3 text-sm">
                        <Checkbox.Root
                          checked={checked}
                          onCheckedChange={(next) =>
                            setAnswers((current) => ({
                              ...current,
                              [question.order]:
                                next === true
                                  ? [...(current[question.order] ?? []), option.text]
                                  : (current[question.order] ?? []).filter((value) => value !== option.text),
                            }))
                          }
                          className="grid size-5 place-items-center rounded border border-border data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground"
                        >
                          <Checkbox.Indicator>
                            <Check className="size-3.5" />
                          </Checkbox.Indicator>
                        </Checkbox.Root>
                        {option.text}
                      </label>
                    )
                  })}
                </div>
              ) : (
                <RadioGroup.Root
                  aria-label={question.question}
                  className="mt-3 space-y-2"
                  value={answers[question.order]?.[0] ?? null}
                  onValueChange={(value) =>
                    setAnswers((current) => ({ ...current, [question.order]: [value] }))
                  }
                >
                  {question.options.map((option) => (
                    <label key={option.order} className="flex cursor-pointer items-center gap-3 text-sm">
                      <RadioGroup.Item
                        value={option.text}
                        className="grid size-5 place-items-center rounded-full border border-border data-[state=checked]:border-primary"
                      >
                        <RadioGroup.Indicator className="size-2.5 rounded-full bg-primary" />
                      </RadioGroup.Item>
                      {option.text}
                    </label>
                  ))}
                </RadioGroup.Root>
              )}
              {questionSolution ? (
                <div className="mt-3 border-l-2 border-primary bg-success-soft px-3 py-2 text-sm">
                  <p className="font-semibold">
                    {[questionSolution.answer.answer, ...questionSolution.answer.answers]
                      .filter(Boolean)
                      .join(', ')}
                  </p>
                  <p className="mt-1 text-muted-foreground">{questionSolution.correction}</p>
                </div>
              ) : null}
            </fieldset>
          )
        })}
      </div>
      {!solution ? (
        <Button
          className="mt-6"
          variant="secondary"
          onClick={() => {
            setRevealed(true)
            reveal()
          }}
        >
          <LockKeyhole className="size-4" />
          {t('lesson.checkAnswers')}
        </Button>
      ) : null}
    </article>
  )
}

export function LessonActivities({
  exercises,
  quizzes,
  solutions,
  onReveal,
}: {
  exercises: Exercise[]
  quizzes: Quiz[]
  solutions: LessonSolutions | undefined
  onReveal: () => void
}) {
  const { t } = useTranslation()
  if (exercises.length === 0 && quizzes.length === 0) return null
  return (
    <div className="mt-12 space-y-10">
      {exercises.length ? (
        <section>
          <h2 className="mb-4 text-xl font-bold">{t('lesson.exercises')}</h2>
          <div className="space-y-5">
            {exercises.map((exercise) => (
              <ExerciseBlock key={exercise.id} exercise={exercise} solutions={solutions} reveal={onReveal} />
            ))}
          </div>
        </section>
      ) : null}
      {quizzes.length ? (
        <section>
          <h2 className="mb-4 text-xl font-bold">{t('lesson.quizzes')}</h2>
          <div className="space-y-5">
            {quizzes.map((quiz) => (
              <QuizBlock key={quiz.id} quiz={quiz} solutions={solutions} reveal={onReveal} />
            ))}
          </div>
        </section>
      ) : null}
    </div>
  )
}
