export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId: string | null

  constructor(message: string, status: number, code: string, requestId: string | null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.requestId = requestId
  }
}

interface ApiResult<T> {
  data?: T
  error?: unknown
  response: Response
}

function publicError(error: unknown) {
  if (typeof error !== 'object' || error === null) return null
  const candidate = error as Record<string, unknown>
  if (typeof candidate.code !== 'string' || typeof candidate.message !== 'string') return null
  return { code: candidate.code, message: candidate.message }
}

export function unwrapApiResult<T>(result: ApiResult<T>): T {
  if (result.response.ok && result.data !== undefined) return result.data
  const error = publicError(result.error)
  throw new ApiError(
    error?.message ?? `HTTP ${result.response.status}`,
    result.response.status,
    error?.code ?? 'unknown_error',
    result.response.headers.get('X-Request-ID'),
  )
}

export function ensureApiSuccess(result: ApiResult<unknown>) {
  if (result.response.ok) return
  const error = publicError(result.error)
  throw new ApiError(
    error?.message ?? `HTTP ${result.response.status}`,
    result.response.status,
    error?.code ?? 'unknown_error',
    result.response.headers.get('X-Request-ID'),
  )
}
