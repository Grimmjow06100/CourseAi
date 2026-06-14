import { Inject, Injectable } from '@nestjs/common';
import { OnModuleInit } from '@nestjs/common';
import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import OpenAI from 'openai';
import { type CourseContextDto } from './dto/course-context.dto';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { LessonContextDto } from './dto/lesson-context.dto';
import { LessonResponseDto } from './dto/lesson-response.dto';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';
import { AnalysisGenerationResultDto } from './dto/analysis-generation-result.dto';
import { ArchitectureGenerationResultDto } from './dto/architecture-generation-result.dto';
@Injectable()
export class GeneratorService implements OnModuleInit {
  public readonly prompts = new Map<string, string>();
  private readonly ANALYSIS: string = 'analysis';
  private readonly ARCHITECTURE: string = 'architecture';
  private readonly LESSONS: string = 'lessons';

  constructor(
    @Inject('OPEN_AI_SERVICE') private openai: OpenAI,
    private parserService: GeneratorParserService,
    private readonly persistenceService: GeneratorPersistenceService,
  ) {}

  async onModuleInit(): Promise<void> {
    const promptsDirectoryPath = join(process.cwd(), 'prompts');
    const promptFileNames = await readdir(promptsDirectoryPath);
    const markdownPromptFileNames = promptFileNames.filter((fileName) =>
      fileName.endsWith('.prompt.md'),
    );

    await Promise.all(
      markdownPromptFileNames.map(async (fileName) => {
        const filePath = join(promptsDirectoryPath, fileName);
        const fileContent = await readFile(filePath, 'utf8');
        const promptNames = [this.ANALYSIS, this.ARCHITECTURE, this.LESSONS];

        const name = fileName.split('.')[0];
        if (promptNames.includes(name)) {
          this.prompts.set(name, fileContent);
          console.log(`Prompt ${name} loaded successfully.`);
        } else throw new Error('erreur de récuperation du prompt system');
      }),
    );
  }

  public async helloAi() {
    const response = await this.openai.responses.create({
      model: 'gpt-5.4',
      input: 'hello ai , how are you doing.',
    });
    return response.output_text;
  }

  public async promptAnalysis(
    prompt: string,
  ): Promise<AnalysisGenerationResultDto> {
    try {
      const content = this.prompts.get(this.ANALYSIS);
      if (!content) throw new Error('erreur de récuperation du prompt system');
      const response = await this.openai.chat.completions.create({
        model: 'gpt-5.4',
        messages: [
          {
            role: 'system',
            content: content,
          },
          {
            role: 'user',
            content: prompt,
          },
        ],
        response_format: { type: 'json_object' },
        temperature: 0.2,
      });
      const result = response.choices[0].message.content;
      if (!result) throw new Error("erreur aucune réponse de l'api");

      const analysis =
        await this.parserService.parseAndValidateAnalysisResponse(result);
      const requestId = await this.persistenceService.persistAnalysis(
        prompt,
        analysis,
        result,
      );

      return { requestId, analysis };
    } catch (error) {
      console.error('Erreur OpenAI:', error);
      throw error;
    }
  }

  /**
   * Generates the course architecture from the validated user answers payload.
   *
   * @param payload - User answers produced after the needs-analysis step.
   * @returns A validated course architecture response.
   */
  public async promptArchitecture(
    payload: CourseContextDto,
  ): Promise<ArchitectureGenerationResultDto> {
    try {
      const content = this.prompts.get(this.ARCHITECTURE);
      if (!content) throw new Error('erreur de récuperation du prompt system');

      const response = await this.openai.chat.completions.create({
        model: 'gpt-5.4',
        messages: [
          {
            role: 'system',
            content,
          },
          {
            role: 'user',
            content: JSON.stringify(payload),
          },
        ],
        response_format: { type: 'json_object' },
        temperature: 0.2,
      });

      const result = response.choices[0].message.content;
      if (!result) throw new Error("erreur aucune réponse de l'api");

      const architecture =
        await this.parserService.parseAndValidateArchitectureResponse(result);

      const course = await this.persistenceService.persistArchitecture(
        payload,
        architecture,
        result,
      );

      return { courseId: course.id, architecture };
    } catch (error) {
      console.error('Erreur OpenAI:', error);
      throw error;
    }
  }

  public async promptLessons(
    payload: LessonContextDto,
  ): Promise<LessonResponseDto> {
    try {
      const content = this.prompts.get(this.LESSONS);
      if (!content) throw new Error('erreur de récuperation du prompt system');

      const response = await this.openai.chat.completions.create({
        model: 'gpt-5.4',
        messages: [
          {
            role: 'system',
            content,
          },
          {
            role: 'user',
            content: JSON.stringify(payload),
          },
        ],
        response_format: { type: 'json_object' },
        temperature: 0.2,
      });

      const result = response.choices[0].message.content;
      if (!result) throw new Error("erreur aucune réponse de l'api");

      const lessonPlan =
        await this.parserService.parseAndValidateLessonResponse(result);

      await this.persistenceService.persistLessons(payload, lessonPlan, result);

      return lessonPlan;
    } catch (error) {
      console.error('Erreur OpenAI:', error);
      throw error;
    }
  }
}
