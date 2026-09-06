import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { createCourseAIClient } from './client'
import { ApiError, unwrapApiResult } from './errors'

const server = setupServer(
  http.get('http://test.local/api/generations', ({ request }) => {
    if (request.headers.get('Authorization') !== 'Bearer jwt-token')
      return HttpResponse.json({ code: 'unauthorized', message: 'Missing token' }, { status: 401 })
    return HttpResponse.json({
      items: [],
      page: 1,
      pageSize: 20,
      totalItems: 0,
      totalPages: 0,
      hasNext: false,
      hasPrevious: false,
    })
  }),
  http.get('http://test.local/api/courses', () =>
    HttpResponse.json(
      { code: 'rate_limited', message: 'Quota reached' },
      { status: 429, headers: { 'X-Request-ID': 'request-123' } },
    ),
  ),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('Course AI API client', () => {
  it('adds the current bearer token to every request', async () => {
    const client = createCourseAIClient('http://test.local', () => Promise.resolve('jwt-token'))
    const result = await client.GET('/api/generations', { params: { query: {} } })
    expect(result.response.status).toBe(200)
  })

  it('normalizes public API errors', async () => {
    const client = createCourseAIClient('http://test.local', () => Promise.resolve('jwt-token'))
    const result = await client.GET('/api/courses', { params: { query: {} } })
    expect(() => unwrapApiResult(result)).toThrow(ApiError)
    try {
      unwrapApiResult(result)
    } catch (error) {
      expect(error).toMatchObject({ status: 429, code: 'rate_limited', requestId: 'request-123' })
    }
  })
})

it('rejects unauthenticated calls without sending an anonymous request', async () => {
  const client = createCourseAIClient('http://test.local', () => Promise.resolve(null))
  await expect(client.GET('/api/generations', { params: { query: {} } })).rejects.toMatchObject({
    status: 401,
  })
})

it('does not send a request when its session ends while Clerk resolves the token', async () => {
  let resolveToken: (token: string) => void = () => undefined
  const token = new Promise<string>((resolve) => {
    resolveToken = resolve
  })
  const session = new AbortController()
  const client = createCourseAIClient(
    'http://test.local',
    () => token,
    () => session.signal,
  )
  const request = client.GET('/api/generations', { params: { query: {} } })
  session.abort()
  resolveToken('jwt-token')
  await expect(request).rejects.toMatchObject({ name: 'AbortError' })
})
