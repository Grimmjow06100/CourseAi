import createClient from 'openapi-fetch'
import type { paths } from './schema.gen'
import { ApiError } from './errors'

export type TokenProvider = (options?: { skipCache?: boolean }) => Promise<string | null>

function waitForToken(token: Promise<string | null>, signal: AbortSignal) {
  return new Promise<string | null>((resolve, reject) => {
    const onAbort = () => {
      const reason: unknown = signal.reason
      reject(
        reason instanceof Error || reason instanceof DOMException
          ? reason
          : new DOMException('Request aborted', 'AbortError'),
      )
    }
    signal.addEventListener('abort', onAbort, { once: true })
    token.then(resolve, reject).finally(() => signal.removeEventListener('abort', onAbort))
    if (signal.aborted) onAbort()
  })
}

export function createCourseAIClient(
  baseUrl: string,
  getToken: TokenProvider,
  getSessionSignal?: () => AbortSignal,
) {
  let tokenRefresh: Promise<string | null> | undefined
  function refreshToken() {
    // Dashboard queries and polling may receive a 401 at the same time.
    tokenRefresh ??= Promise.resolve()
      .then(() => getToken({ skipCache: true }))
      .finally(() => {
        tokenRefresh = undefined
      })
    return tokenRefresh
  }

  return createClient<paths>({
    baseUrl,
    async fetch(input) {
      const sessionSignal = getSessionSignal?.()
      const signals = [input.signal, AbortSignal.timeout(30_000)]
      if (sessionSignal) signals.push(sessionSignal)
      const signal = AbortSignal.any(signals)
      signal.throwIfAborted()
      const request = new Request(input, { signal })
      const token = await waitForToken(getToken(), signal)
      signal.throwIfAborted()
      if (!token) throw new ApiError('Authentication required', 401, 'session_expired', null)
      request.headers.set('Authorization', `Bearer ${token}`)
      request.headers.set('Accept', 'application/json')

      // Keep the original body and Idempotency-Key for one authentication-only retry.
      const response = await fetch(request.clone())
      if (response.status !== 401) return response
      signal.throwIfAborted()
      const freshToken = await waitForToken(refreshToken(), signal)
      signal.throwIfAborted()
      if (!freshToken) {
        throw new ApiError(
          'Authentication required',
          401,
          'session_expired',
          response.headers.get('X-Request-ID'),
        )
      }
      request.headers.set('Authorization', `Bearer ${freshToken}`)
      return fetch(request)
    },
  })
}

export type CourseAIClient = ReturnType<typeof createCourseAIClient>
