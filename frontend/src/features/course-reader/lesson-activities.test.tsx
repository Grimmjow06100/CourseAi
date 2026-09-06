import { fireEvent, render, screen } from '@testing-library/react'
import '@/shared/i18n'
import type { Exercise } from '@/shared/api/types'
import { LessonActivities } from './lesson-activities'

it('does not reveal other activities when their solutions are already in the shared response', () => {
  const exercise = (id: string): Exercise => ({
    id,
    lessonId: 'lesson',
    type: 'command_line',
    difficulty: 'beginner',
    title: id,
    objective: 'Learn',
    instructionsMarkdown: 'Do the task',
    contentMarkdown: '',
    payload: { tasks: [], resources: [], starterCode: null, expectedOutput: null, hints: [] },
    createdAt: '2026-09-06',
    updatedAt: '2026-09-06',
  })
  const reveal = vi.fn()
  render(
    <LessonActivities
      exercises={[exercise('first'), exercise('second')]}
      quizzes={[]}
      solutions={{
        lessonId: 'lesson',
        quizzes: [],
        exercises: [
          { exerciseId: 'first', correctionMarkdown: 'First secret' },
          { exerciseId: 'second', correctionMarkdown: 'Second secret' },
        ],
      }}
      onReveal={reveal}
    />,
  )
  expect(screen.queryByText('First secret')).not.toBeInTheDocument()
  const firstButton = screen.getAllByRole('button')[0]
  if (!firstButton) throw new Error('Missing reveal button')
  fireEvent.click(firstButton)
  expect(screen.getByText('First secret')).toBeVisible()
  expect(screen.queryByText('Second secret')).not.toBeInTheDocument()
  expect(reveal).toHaveBeenCalledOnce()
})
