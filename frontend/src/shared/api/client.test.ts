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

it('refreshes a rejected cached token once and retries the same generation body and idempotency key', async () => {
  const attempts: { authorization: string | null; key: string | null; body: unknown }[] = []
  server.use(
    http.post('http://test.local/api/generations', async ({ request }) => {
      attempts.push({
        authorization: request.headers.get('Authorization'),
        key: request.headers.get('Idempotency-Key'),
        body: await request.json(),
      })
      return request.headers.get('Authorization') === 'Bearer fresh-token'
        ? HttpResponse.json({ requestId: 'generation-123' }, { status: 202 })
        : HttpResponse.json({ code: 'unauthenticated', message: 'Invalid token' }, { status: 401 })
    }),
  )
  const getToken = vi.fn().mockResolvedValueOnce('cached-token').mockResolvedValueOnce('fresh-token')
  const client = createCourseAIClient('http://test.local', getToken)
  const body = { prompt: 'Learn Linux administration' }
  const result = await client.POST('/api/generations', {
    body,
    params: { header: { 'Idempotency-Key': 'same-generation' } },
  })
  expect(result.response.status).toBe(202)
  expect(getToken).toHaveBeenNthCalledWith(2, { skipCache: true })
  expect(attempts).toEqual([
    { authorization: 'Bearer cached-token', key: 'same-generation', body },
    { authorization: 'Bearer fresh-token', key: 'same-generation', body },
  ])
})

it('shares an in-flight token refresh across the two dashboard queries', async () => {
  const refresh = Promise.withResolvers<string>()
  const getToken = vi.fn((options?: { skipCache?: boolean }) =>
    options?.skipCache ? refresh.promise : Promise.resolve('cached-token'),
  )
  let rejected = 0
  const handler = http.get('http://test.local/api/*', ({ request }) => {
    if (request.headers.get('Authorization') === 'Bearer jwt-token') return HttpResponse.json({ items: [] })
    rejected++
    return HttpResponse.json({ code: 'unauthenticated', message: 'Invalid token' }, { status: 401 })
  })
  server.use(handler)
  const client = createCourseAIClient('http://test.local', getToken)
  const generations = client.GET('/api/generations', { params: { query: {} } })
  const courses = client.GET('/api/courses', { params: { query: {} } })
  await vi.waitFor(() => {
    expect(rejected).toBe(2)
    expect(getToken).toHaveBeenCalledWith({ skipCache: true })
  })
  refresh.resolve('jwt-token')
  const results = await Promise.all([generations, courses])
  expect(results.map((result) => result.response.status)).toEqual([200, 200])
  expect(getToken.mock.calls.filter(([options]) => options?.skipCache)).toHaveLength(1)
})

it('stops after a second 401 and preserves the final request ID', async () => {
  let attempts = 0
  server.use(
    http.get('http://test.local/api/generations', () => {
      attempts++
      return HttpResponse.json(
        { code: 'unauthenticated', message: 'Invalid token' },
        { status: 401, headers: { 'X-Request-ID': `request-${attempts}` } },
      )
    }),
  )
  const getToken = vi.fn().mockResolvedValue('rejected-token')
  const client = createCourseAIClient('http://test.local', getToken)
  const result = await client.GET('/api/generations', { params: { query: {} } })
  expect(attempts).toBe(2)
  expect(() => unwrapApiResult(result)).toThrow(
    expect.objectContaining({ status: 401, requestId: 'request-2' }),
  )
})

it('reports a missing session after refresh without sending an anonymous retry', async () => {
  let attempts = 0
  server.use(
    http.get('http://test.local/api/generations', () => {
      attempts++
      return HttpResponse.json(
        { code: 'unauthenticated', message: 'Invalid token' },
        { status: 401, headers: { 'X-Request-ID': 'request-123' } },
      )
    }),
  )
  const client = createCourseAIClient(
    'http://test.local',
    vi.fn().mockResolvedValueOnce('old-token').mockResolvedValueOnce(null),
  )
  await expect(client.GET('/api/generations', { params: { query: {} } })).rejects.toMatchObject({
    status: 401,
    code: 'session_expired',
    requestId: 'request-123',
  })
  expect(attempts).toBe(1)
})

it.each([403, 429, 500, 503])('does not refresh or replay a mutation after HTTP %i', async (status) => {
  let attempts = 0
  server.use(
    http.post('http://test.local/api/generations', () => {
      attempts++
      return HttpResponse.json({ code: 'error', message: 'Request failed' }, { status })
    }),
  )
  const getToken = vi.fn().mockResolvedValue('jwt-token')
  const client = createCourseAIClient('http://test.local', getToken)
  const result = await client.POST('/api/generations', { body: { prompt: 'Learn Linux administration' } })
  expect(result.response.status).toBe(status)
  expect(attempts).toBe(1)
  expect(getToken).toHaveBeenCalledTimes(1)
})

it('cancels immediately when the session ends during a token refresh', async () => {
  const refresh = Promise.withResolvers<string>()
  const getToken = vi
    .fn()
    .mockResolvedValueOnce('old-token')
    .mockImplementationOnce(() => refresh.promise)
  let attempts = 0
  server.use(
    http.get('http://test.local/api/generations', () => {
      attempts++
      return HttpResponse.json({ code: 'unauthenticated', message: 'Invalid token' }, { status: 401 })
    }),
  )
  const session = new AbortController()
  const client = createCourseAIClient('http://test.local', getToken, () => session.signal)
  const request = client.GET('/api/generations', { params: { query: {} } })
  const rejection = expect(request).rejects.toMatchObject({ name: 'AbortError' })
  await vi.waitFor(() => expect(getToken).toHaveBeenCalledTimes(2))
  session.abort()
  await rejection
  refresh.resolve('fresh-token')
  expect(attempts).toBe(1)
})

it('does not send an already-cancelled query or request its token', async () => {
  const query = new AbortController()
  query.abort()
  const getToken = vi.fn().mockResolvedValue('jwt-token')
  const client = createCourseAIClient('http://test.local', getToken)
  await expect(
    client.GET('/api/generations', { params: { query: {} }, signal: query.signal }),
  ).rejects.toMatchObject({ name: 'AbortError' })
  expect(getToken).not.toHaveBeenCalled()
})

it('does not replay a mutation after a network failure with an uncertain result', async () => {
  let attempts = 0
  server.use(
    http.post('http://test.local/api/generations', () => {
      attempts++
      return HttpResponse.error()
    }),
  )
  const getToken = vi.fn().mockResolvedValue('jwt-token')
  const client = createCourseAIClient('http://test.local', getToken)
  await expect(
    client.POST('/api/generations', { body: { prompt: 'Learn Linux administration' } }),
  ).rejects.toThrow()
  expect(attempts).toBe(1)
  expect(getToken).toHaveBeenCalledOnce()
})

it('applies the request deadline while waiting for Clerk, before sending HTTP', async () => {
  const deadline = new AbortController()
  const token = Promise.withResolvers<string>()
  const timeout = vi.spyOn(AbortSignal, 'timeout').mockReturnValue(deadline.signal)
  try {
    const client = createCourseAIClient('http://test.local', () => token.promise)
    const request = client.GET('/api/generations', { params: { query: {} } })
    const rejection = expect(request).rejects.toMatchObject({ name: 'TimeoutError' })
    deadline.abort(new DOMException('Timed out', 'TimeoutError'))
    await rejection
    expect(timeout).toHaveBeenCalledWith(30_000)
  } finally {
    timeout.mockRestore()
    token.resolve('jwt-token')
  }
})
