import { clarificationFormSchema, generationListSearchSchema, generationPromptSchema } from './schemas'

describe('generation schemas', () => {
  it('rejects empty and oversized prompts', () => {
    expect(generationPromptSchema.safeParse({ prompt: 'short' }).success).toBe(false)
    expect(generationPromptSchema.safeParse({ prompt: 'a'.repeat(4001) }).success).toBe(false)
    expect(
      generationPromptSchema.safeParse({ prompt: 'Learn Go by building a production API' }).success,
    ).toBe(true)
  })

  it('normalizes generation URL filters', () => {
    expect(generationListSearchSchema.parse({ page: '-2', status: 'invalid' })).toEqual({ page: 1 })
  })

  it('requires at least one selected value per clarification answer', () => {
    const result = clarificationFormSchema.safeParse({
      title: 'Linux',
      synopsis: 'A Linux course',
      language: 'fr',
      answers: { goals: [] },
    })
    expect(result.success).toBe(false)
  })
})
