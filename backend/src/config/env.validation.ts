import { z } from 'zod';

const envSchema = z.object({
  DATABASE_URL: z.string().min(1),
  OPENAI_API_KEY: z.string().min(1),
  AI_MODEL: z.string().min(1).default('gpt-5.4'),
  OPENAI_MAX_RETRIES: z.coerce.number().int().min(0).max(5).default(2),
  PORT: z.coerce.number().int().positive().default(3000),
  CORS_ORIGIN: z.string().min(1).default('*'),
  THROTTLE_TTL: z.coerce.number().int().positive().default(60000),
  THROTTLE_LIMIT: z.coerce.number().int().positive().default(100),
});

export type EnvConfig = z.infer<typeof envSchema>;

export function validateEnv(config: Record<string, unknown>): EnvConfig {
  const parsedConfig = envSchema.safeParse(config);

  if (!parsedConfig.success) {
    const formattedErrors = parsedConfig.error.issues
      .map((issue) => `${issue.path.join('.')}: ${issue.message}`)
      .join('; ');

    throw new Error(`Invalid environment configuration: ${formattedErrors}`);
  }

  return parsedConfig.data;
}
