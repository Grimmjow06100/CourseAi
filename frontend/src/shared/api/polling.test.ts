import { shouldPollQuery } from './polling'

it('starts bounded recovery polling after a long active generation', () => {
  const query = { state: { dataUpdateCount: 100 } }
  expect(shouldPollQuery(query, true, false)).toBe(true)
  query.state.dataUpdateCount++
  expect(shouldPollQuery(query, false, true)).toBe(true)
  query.state.dataUpdateCount += 29
  expect(shouldPollQuery(query, false, true)).toBe(true)
  query.state.dataUpdateCount++
  expect(shouldPollQuery(query, false, true)).toBe(false)
})

it('resets recovery when work resumes and never polls a stable terminal state', () => {
  const query = { state: { dataUpdateCount: 1 } }
  expect(shouldPollQuery(query, false, false)).toBe(false)
  expect(shouldPollQuery(query, false, true)).toBe(true)
  query.state.dataUpdateCount = 50
  expect(shouldPollQuery(query, false, true)).toBe(false)
  expect(shouldPollQuery(query, true, false)).toBe(true)
  expect(shouldPollQuery(query, false, true)).toBe(true)
  expect(shouldPollQuery({ state: { dataUpdateCount: 50 } }, false, true)).toBe(true)
})
