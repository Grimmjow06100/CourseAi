import { z } from 'zod';

export const clarificationQuestionSchema = z
  .object({
    id: z.enum(['goals', 'currentLevel', 'targetLevel']),
    question: z.string().min(1),
    options: z.array(z.string().min(1)).min(1),
  })
  .strict();

export const analysisResponseSchema = z
  .object({
    isOutOfScope: z.boolean(),
    errorMessage: z.string().nullable(),
    warningMessage: z.string().nullable(),
    suggestedTitle: z.string().min(1),
    shortSynopsis: z.string().min(1),
    detectedCurrentLevel: z.enum([
      'beginner',
      'intermediate',
      'advanced',
      'unknown',
    ]),
    detectedTargetLevel: z.enum([
      'beginner',
      'intermediate',
      'advanced',
      'expert',
      'unknown',
    ]),
    detectedGoal: z.string().min(1),
    detectedLanguage: z.enum(['fr', 'en']),
    clarificationQuestions: z.array(clarificationQuestionSchema),
  })
  .strict();

export const architectureModuleSchema = z
  .object({
    order: z.number().int().min(1),
    title: z.string().min(1),
    description: z.string().min(1),
    keyLearningPoints: z.array(z.string().min(1)).min(1),
  })
  .strict();

export const architectureFinalProjectSchema = z
  .object({
    title: z.string().min(1),
    description: z.string().min(1),
    constraints: z.array(z.string().min(1)).min(1),
  })
  .strict();

export const architectureResponseSchema = z
  .object({
    title: z.string().min(1),
    synopsis: z.string().min(1),
    targetAudience: z.string().min(1),
    prerequisites: z.array(z.string().min(1)),
    goals: z.array(z.string().min(1)).min(1),
    acquiredSkills: z.array(z.string().min(1)).min(1),
    modules: z.array(architectureModuleSchema).min(1),
    finalProject: architectureFinalProjectSchema,
  })
  .strict();

export const lessonPlanItemSchema = z
  .object({
    order: z.number().int().min(1),
    title: z.string().min(1),
    type: z.enum(['theory', 'practice', 'mixed', 'quiz']),
    estimatedDuration: z.number().int().min(1),
    learningGoal: z.string().min(1),
    requiresDiagram: z.boolean(),
    technicalKeywords: z.array(z.string().min(1)).min(1),
  })
  .strict();

export const lessonResponseSchema = z
  .object({
    moduleOrder: z.number().int().min(1),
    moduleTitle: z.string().min(1),
    lessons: z.array(lessonPlanItemSchema).min(3).max(6),
  })
  .strict();

export const lessonContentResponseSchema = z
  .object({
    lessonId: z.string().uuid(),
    title: z.string().min(1),
    contentMarkdown: z.string().min(1),
    summary: z.string().min(1),
    keyTakeaways: z.array(z.string().min(1)).min(1),
  })
  .strict();

export type AnalysisResponsePayload = z.infer<typeof analysisResponseSchema>;
export type ArchitectureResponsePayload = z.infer<
  typeof architectureResponseSchema
>;
export type LessonResponsePayload = z.infer<typeof lessonResponseSchema>;
export type LessonContentResponsePayload = z.infer<
  typeof lessonContentResponseSchema
>;
