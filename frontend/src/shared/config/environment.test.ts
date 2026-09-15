import { loadEnvironment } from './environment'

const testKey = `pk_test_${btoa('clerk.course-ai.test$')}`
const liveKey = `pk_live_${btoa('clerk.example.com$')}`
const valid = { VITE_API_BASE_URL: 'https://api.example.com', VITE_CLERK_PUBLISHABLE_KEY: testKey }

describe('frontend environment', () => {
  it('validates required public settings', () => {
    expect(
      loadEnvironment({
        VITE_API_BASE_URL: 'https://api.example.com',
        VITE_CLERK_PUBLISHABLE_KEY: testKey,
      }),
    ).toMatchObject({ VITE_E2E_MODE: false })
    expect(() =>
      loadEnvironment({ VITE_API_BASE_URL: 'not-an-url', VITE_CLERK_PUBLISHABLE_KEY: '' }),
    ).toThrow('Invalid frontend environment')
  })
})

describe('deployment environment safety', () => {
  it('keeps localhost available for development', () => {
    expect(loadEnvironment({ ...valid, VITE_API_BASE_URL: 'http://localhost:8080' })).toBeDefined()
  })
  it.each([
    'https://api.example.com/api',
    'https://user:password@api.example.com',
    'https://api.example.com?token=secret',
    'https://api.example.com/#fragment',
  ])('rejects an API URL that is not a bare HTTP(S) origin: %s', (url) => {
    expect(() => loadEnvironment({ ...valid, VITE_API_BASE_URL: url })).toThrow('VITE_API_BASE_URL')
  })
  it.each([
    'pk_test_replace_me',
    'pk_live_xxx',
    'sk_live_do_not_expose',
    `pk_live_${btoa('invalid-host$')}`,
    `pk_live_${btoa('clerk.example.com')}`,
    `pk_live_${btoa('clerk.example.com$$')}`,
  ])('rejects placeholder, secret or malformed Clerk keys without printing their value', (key) => {
    expect(() => loadEnvironment({ ...valid, VITE_CLERK_PUBLISHABLE_KEY: key })).toThrow(
      /^Invalid frontend environment: VITE_CLERK_PUBLISHABLE_KEY$/,
    )
  })
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
      loadEnvironment(
        { ...valid, VITE_CLERK_PUBLISHABLE_KEY: liveKey },
        { production: true, requireLiveKey: true },
      ),
    ).toBeDefined()
  })
})
