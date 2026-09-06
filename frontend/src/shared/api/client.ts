import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema.gen'
import { ApiError } from './errors'

export type TokenProvider = () => Promise<string | null>

export function createCourseAIClient(
  baseUrl: string,
  getToken: TokenProvider,
  getSessionSignal?: () => AbortSignal,
) {
  const client = createClient<paths>({ baseUrl })
  const authentication: Middleware = {
    async onRequest({ request }) {
      const sessionSignal = getSessionSignal?.()
      sessionSignal?.throwIfAborted()
      const token = await getToken()
      sessionSignal?.throwIfAborted()
      if (!token) throw new ApiError('Authentication required', 401, 'unauthorized', null)
      request.headers.set('Authorization', `Bearer ${token}`)
      request.headers.set('Accept', 'application/json')
      const signals = [request.signal, AbortSignal.timeout(30_000)]
      if (sessionSignal) signals.push(sessionSignal)
      return new Request(request, { signal: AbortSignal.any(signals) })
    },
  }
  client.use(authentication)
  return client
}

export type CourseAIClient = ReturnType<typeof createCourseAIClient>
