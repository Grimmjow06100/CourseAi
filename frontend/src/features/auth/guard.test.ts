import { requireAuthentication } from './guard'

describe('authentication guard', () => {
  it('allows an authenticated session', () => {
    expect(() => requireAuthentication(true)).not.toThrow()
  })

  it('redirects an anonymous visitor', () => {
    expect(() => requireAuthentication(false)).toThrow()
  })
})
