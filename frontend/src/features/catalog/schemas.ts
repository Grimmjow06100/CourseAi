import { z } from 'zod'

export const catalogSearchSchema = z.object({
  search: z.string().trim().optional().catch(undefined),
  status: z
    .enum([
      'analysis_pending',
      'analysis_completed',
      'architecture_generating',
      'structure_generated',
      'lessons_generating',
      'lessons_generated',
      'content_generating',
      'completed',
      'failed',
    ])
    .optional()
    .catch(undefined),
  language: z.enum(['fr', 'en']).optional().catch(undefined),
  orderBy: z.enum(['created_at', 'updated_at', 'title', 'status']).catch('created_at'),
  orderDirection: z.enum(['asc', 'desc']).catch('desc'),
  page: z.coerce.number().int().positive().catch(1),
})

export type CatalogSearch = z.infer<typeof catalogSearchSchema>
