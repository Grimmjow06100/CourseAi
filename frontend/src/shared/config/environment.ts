import { z } from 'zod'

// Check the SDK's encoded frontend-API format. This does not verify a Clerk instance exists.
function isClerkPublishableKey(value: string) {
  const encoded = /^pk_(?:test|live)_([A-Za-z0-9+/]+={0,2})$/.exec(value)?.[1]
  if (!encoded) return false
  try {
    const decoded = atob(encoded)
    return /^[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)+\$$/i.test(decoded)
  } catch {
    return false
  }
}

const environmentSchema = z.object({
  VITE_API_BASE_URL: z.url().refine((value) => {
    if (!URL.canParse(value)) return false
    const url = new URL(value)
    return (
      ['http:', 'https:'].includes(url.protocol) &&
      !url.username &&
      !url.password &&
      url.pathname === '/' &&
      !url.search &&
      !url.hash
    )
  }, 'Expected an HTTP(S) API origin without a path, credentials, query or fragment'),
  VITE_CLERK_PUBLISHABLE_KEY: z
    .string()
    .trim()
    .refine(isClerkPublishableKey, 'Expected a Clerk publishable key copied from the dashboard'),
  VITE_E2E_MODE: z
    .enum(['true', 'false'])
    .default('false')
    .transform((value) => value === 'true'),
})

export type Environment = z.infer<typeof environmentSchema>

export function loadEnvironment(
  values: Record<string, unknown>,
  options: { production?: boolean; requireLiveKey?: boolean } = {},
) {
  const result = environmentSchema.safeParse(values)
  if (!result.success) {
    const fields = result.error.issues.map((issue) => issue.path.join('.')).join(', ')
    throw new Error(`Invalid frontend environment: ${fields}`)
  }
  if (
    options.production &&
    (new URL(result.data.VITE_API_BASE_URL).protocol !== 'https:' || result.data.VITE_E2E_MODE)
  ) {
    throw new Error('Production requires HTTPS VITE_API_BASE_URL and VITE_E2E_MODE=false')
  }
  if (options.requireLiveKey && !result.data.VITE_CLERK_PUBLISHABLE_KEY.startsWith('pk_live_')) {
    throw new Error('Vercel production requires a live Clerk publishable key')
  }
  return result.data
}
