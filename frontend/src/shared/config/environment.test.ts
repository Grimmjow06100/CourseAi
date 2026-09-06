import { loadEnvironment } from './environment'

describe('frontend environment', () => {
  it('validates required public settings', () => {
    expect(
      loadEnvironment({
        VITE_API_BASE_URL: 'https://api.example.com',
        VITE_CLERK_PUBLISHABLE_KEY: 'pk_test_value',
      }),
    ).toMatchObject({ VITE_E2E_MODE: false })
    expect(() =>
      loadEnvironment({ VITE_API_BASE_URL: 'not-an-url', VITE_CLERK_PUBLISHABLE_KEY: '' }),
    ).toThrow('Invalid frontend environment')
  })
})

describe('deployment environment safety', () => {
  const valid = { VITE_API_BASE_URL: 'https://api.example.com', VITE_CLERK_PUBLISHABLE_KEY: 'pk_test_value' }
  it('rejects missing configuration before a bundle can be shipped', () => {
    expect(() => loadEnvironment({})).toThrow('Invalid frontend environment')
  })
  it('rejects unsafe URLs and development-only auth in builds', () => {
    expect(() => loadEnvironment({ ...valid, VITE_API_BASE_URL: 'javascript:alert(1)' })).toThrow()
    expect(() =>
      loadEnvironment({ ...valid, VITE_API_BASE_URL: 'http://api.example.com' }, { production: true }),
    ).toThrow('HTTPS')
    expect(() => loadEnvironment({ ...valid, VITE_E2E_MODE: 'true' }, { production: true })).toThrow(
      'VITE_E2E_MODE=false',
    )
  })
  it('requires live Clerk keys only for Vercel production, not previews', () => {
    expect(loadEnvironment(valid, { production: true }).VITE_E2E_MODE).toBe(false)
    expect(() => loadEnvironment(valid, { requireLiveKey: true })).toThrow('live Clerk')
    expect(
      loadEnvironment({ ...valid, VITE_CLERK_PUBLISHABLE_KEY: 'pk_live_value' }, { requireLiveKey: true }),
    ).toBeDefined()
  })
})
