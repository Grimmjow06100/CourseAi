import { createQueryClient } from './query-client'
import { SessionLifetime } from './session-lifetime'

describe('authenticated runtime lifetime', () => {
  it('does not share cached private data across sessions', () => {
    const first = createQueryClient()
    const second = createQueryClient()
    first.setQueryData(['courses'], ['private course'])
    expect(second.getQueryData(['courses'])).toBeUndefined()
    first.clear()
    expect(first.getQueryData(['courses'])).toBeUndefined()
    second.clear()
  })

  it('aborts pending operations on sign-out and supports StrictMode effect replay', () => {
    const lifetime = new SessionLifetime()
    const previous = lifetime.getSignal()
    lifetime.dispose()
    expect(previous.aborted).toBe(true)
    lifetime.activate()
    expect(lifetime.getSignal().aborted).toBe(false)
    expect(previous.aborted).toBe(true)
    lifetime.dispose()
  })
})
