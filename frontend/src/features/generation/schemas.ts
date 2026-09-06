import { z } from 'zod'

export const generationPromptSchema = z.object({
  prompt: z.string().trim().min(10, 'Décrivez votre objectif en au moins 10 caractères.').max(4000),
})

export type GenerationPromptValues = z.infer<typeof generationPromptSchema>

export const generationListSearchSchema = z.object({
  status: z
    .enum(['queued', 'running', 'awaiting_clarification', 'completed', 'failed'])
    .optional()
    .catch(undefined),
  page: z.coerce.number().int().positive().catch(1),
})

export const clarificationFormSchema = z.object({
  title: z.string().trim().min(1).max(200),
  synopsis: z.string().trim().min(1).max(2000),
  language: z.enum(['fr', 'en']),
  answers: z.record(z.string(), z.array(z.string()).min(1).max(4)),
})

export type ClarificationFormValues = z.infer<typeof clarificationFormSchema>
