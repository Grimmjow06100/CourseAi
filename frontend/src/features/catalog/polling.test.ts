import { shouldPollCourses } from './polling'

it('stops polling settled partial and completed courses', () => {
  const query = { state: { dataUpdateCount: 0 } }
  expect(shouldPollCourses([{ status: 'partial' }, { status: 'completed' }], query)).toBe(false)
  expect(shouldPollCourses([{ status: 'partial' }, { status: 'content_generating' }], query)).toBe(true)
})

it('bounds recovery reads for failed courses and restarts when work resumes', () => {
  const query = { state: { dataUpdateCount: 10 } }
  expect(shouldPollCourses([{ status: 'failed' }], query)).toBe(true)
  query.state.dataUpdateCount += 30
  expect(shouldPollCourses([{ status: 'failed' }], query)).toBe(false)
  expect(shouldPollCourses([{ status: 'lessons_generating' }], query)).toBe(true)
  expect(shouldPollCourses([{ status: 'failed' }], query)).toBe(true)
})
