import { BadRequestException } from '@nestjs/common';
import { GeneratorParserService } from './generator-parser.service';

describe('GeneratorParserService', () => {
  let service: GeneratorParserService;

  beforeEach(() => {
    service = new GeneratorParserService();
  });

  it('validates a lesson content response', async () => {
    const response = await service.parseAndValidateLessonContentResponse(
      JSON.stringify({
        lessonId: '7f75ecbb-1c4c-4d74-8203-2edfd35b4569',
        title: 'Comprendre les volumes Docker',
        contentMarkdown:
          '# Comprendre les volumes Docker\n\n## Objectif\nComprendre la persistance.',
        summary: 'La leçon explique la persistance avec les volumes.',
        keyTakeaways: ['Un volume persiste les données.'],
      }),
    );

    expect(response.lessonId).toBe('7f75ecbb-1c4c-4d74-8203-2edfd35b4569');
    expect(response.keyTakeaways).toHaveLength(1);
  });

  it('rejects invalid JSON', async () => {
    await expect(
      service.parseAndValidateLessonContentResponse('{not-json'),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('rejects an invalid lesson content response shape', async () => {
    await expect(
      service.parseAndValidateLessonContentResponse(
        JSON.stringify({
          lessonId: 'not-a-uuid',
          title: 'Leçon invalide',
          contentMarkdown: '',
          summary: 'Résumé',
          keyTakeaways: [],
        }),
      ),
    ).rejects.toBeInstanceOf(BadRequestException);
  });
});
