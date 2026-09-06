import { z } from 'zod'

const environmentSchema = z.object({
  VITE_API_BASE_URL: z.url().refine((value) => /^https?:\/\//.test(value)),
  VITE_CLERK_PUBLISHABLE_KEY: z
    .string()
    .trim()
    .regex(/^pk_(test|live)_[A-Za-z0-9_-]+$/),
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
