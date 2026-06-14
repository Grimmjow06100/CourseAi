import {
  BadRequestException,
  ConflictException,
  Inject,
  Injectable,
  OnModuleInit,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import OpenAI from 'openai';
import { zodResponseFormat } from 'openai/helpers/zod';
import { z } from 'zod';
import { type CourseContextDto } from './dto/course-context.dto';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { LessonContextDto } from './dto/lesson-context.dto';
import {
  LessonPlanItemDto,
  LessonResponseDto,
} from './dto/lesson-response.dto';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';
import { AnalysisGenerationResultDto } from './dto/analysis-generation-result.dto';
import { ArchitectureGenerationResultDto } from './dto/architecture-generation-result.dto';
import { LessonContentContextDto } from './dto/lesson-content-context.dto';
import { LessonContentResponseDto } from './dto/lesson-content-response.dto';
import {
  analysisResponseSchema,
  architectureResponseSchema,
  lessonContentResponseSchema,
  lessonResponseSchema,
} from './schemas/generation-output.schemas';
import {
  ArchitectureModuleDto,
  ArchitectureResponseDto,
} from './dto/architecture-response.dto';
import { AnalysisResponseDto } from './dto/analysis-response.dto';
import { FullCourseGenerationDto } from './dto/full-course-generation.dto';
import { GenerationStartResponseDto } from './dto/generation-start-response.dto';
import { GenerationPipelineStatus } from '../../generated/prisma/client';
import { GenerationStatusResponseDto } from './dto/generation-status-response.dto';
import { GenerationResultResponseDto } from './dto/generation-result-response.dto';

interface StructuredCompletionResult<TParsed> {
  parsed: TParsed;
  rawContent: string;
}

interface GeneratedModuleLessonPlan {
  module: ArchitectureModuleDto;
  lessonPlan: LessonResponseDto;
  persistedLessons: Array<{
    id: string;
    lessonOrder: number;
  }>;
}

@Injectable()
export class GeneratorService implements OnModuleInit {
  public readonly prompts = new Map<string, string>();
  private readonly ANALYSIS: string = 'analysis';
  private readonly ARCHITECTURE: string = 'architecture';
  private readonly LESSONS: string = 'lessons';
  private readonly LESSON_CONTENT: string = 'lesson-content';
  private readonly aiModel: string;
  private readonly maxOpenAiRetries: number;

  constructor(
    @Inject('OPEN_AI_SERVICE') private openai: OpenAI,
    private parserService: GeneratorParserService,
    private readonly persistenceService: GeneratorPersistenceService,
    private readonly configService: ConfigService,
  ) {
    this.aiModel = this.configService.get<string>('AI_MODEL') ?? 'gpt-5.4';
    this.maxOpenAiRetries =
      this.configService.get<number>('OPENAI_MAX_RETRIES') ?? 2;
  }

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
        const promptNames = [
          this.ANALYSIS,
          this.ARCHITECTURE,
          this.LESSONS,
          this.LESSON_CONTENT,
        ];

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
      model: this.aiModel,
      input: 'hello ai , how are you doing.',
    });
    return response.output_text;
  }

  public async promptAnalysis(
    prompt: string,
  ): Promise<AnalysisGenerationResultDto> {
    try {
      const analysisResult = await this.generateStructuredCompletion(
        this.ANALYSIS,
        prompt,
        analysisResponseSchema,
        'analysis_response',
      );
      const analysis =
        await this.parserService.parseAndValidateAnalysisResponse(
          analysisResult.rawContent,
        );
      const requestId = await this.persistenceService.persistAnalysis(
        prompt,
        analysis,
        analysisResult.rawContent,
      );

      return { requestId, analysis };
    } catch (error) {
      this.logSafeGenerationError('Analysis generation failed', error);
      throw error;
    }
  }

  /**
   * Generates analysis and persists it into an existing request id.
   *
   * @param requestId - Existing generation request id.
   * @param prompt - Raw user prompt.
   * @returns A validated analysis response.
   */
  public async promptAnalysisForRequest(
    requestId: string,
    prompt: string,
  ): Promise<AnalysisResponseDto> {
    const analysisResult = await this.generateStructuredCompletion(
      this.ANALYSIS,
      prompt,
      analysisResponseSchema,
      'analysis_response',
    );
    const analysis = await this.parserService.parseAndValidateAnalysisResponse(
      analysisResult.rawContent,
    );

    await this.persistenceService.persistAnalysisForRequest(
      requestId,
      analysis,
      analysisResult.rawContent,
    );

    return analysis;
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
      const architectureResult = await this.generateStructuredCompletion(
        this.ARCHITECTURE,
        JSON.stringify(payload),
        architectureResponseSchema,
        'architecture_response',
      );
      const architecture =
        await this.parserService.parseAndValidateArchitectureResponse(
          architectureResult.rawContent,
        );

      const course = await this.persistenceService.persistArchitecture(
        payload,
        architecture,
        architectureResult.rawContent,
      );

      return { courseId: course.id, architecture };
    } catch (error) {
      this.logSafeGenerationError('Architecture generation failed', error);
      throw error;
    }
  }

  public async promptLessons(
    payload: LessonContextDto,
  ): Promise<LessonResponseDto> {
    try {
      const lessonResult = await this.generateStructuredCompletion(
        this.LESSONS,
        JSON.stringify(payload),
        lessonResponseSchema,
        'lesson_plan_response',
      );
      const lessonPlan =
        await this.parserService.parseAndValidateLessonResponse(
          lessonResult.rawContent,
        );

      await this.persistenceService.persistLessons(
        payload,
        lessonPlan,
        lessonResult.rawContent,
      );

      return lessonPlan;
    } catch (error) {
      this.logSafeGenerationError('Lesson plan generation failed', error);
      throw error;
    }
  }

  /**
   * Generates and persists the full Markdown content for one lesson.
   *
   * @param payload - Course, module, and lesson context for content generation.
   * @returns A validated lesson content response.
   */
  public async promptLessonContent(
    payload: LessonContentContextDto,
  ): Promise<LessonContentResponseDto> {
    try {
      const lessonContentResult = await this.generateStructuredCompletion(
        this.LESSON_CONTENT,
        JSON.stringify(payload),
        lessonContentResponseSchema,
        'lesson_content_response',
      );
      const lessonContent =
        await this.parserService.parseAndValidateLessonContentResponse(
          lessonContentResult.rawContent,
        );

      await this.persistenceService.persistLessonContent(
        payload,
        lessonContent,
        lessonContentResult.rawContent,
      );

      return lessonContent;
    } catch (error) {
      this.logSafeGenerationError('Lesson content generation failed', error);
      throw error;
    }
  }

  /**
   * Starts a background full-course generation pipeline.
   *
   * @param payload - Raw prompt used as the root generation input.
   * @returns A start response containing polling URLs.
   */
  public async startFullCourseGeneration(
    payload: FullCourseGenerationDto,
  ): Promise<GenerationStartResponseDto> {
    const requestId =
      await this.persistenceService.createQueuedGenerationRequest(
        payload.prompt,
      );

    void this.runFullCoursePipeline(requestId, payload.prompt);

    return this.buildGenerationStartResponse(requestId);
  }

  /**
   * Retries a previously created full-course generation request.
   *
   * @param requestId - Generation request id to retry.
   * @returns A start response containing polling URLs.
   */
  public async retryFullCourseGeneration(
    requestId: string,
  ): Promise<GenerationStartResponseDto> {
    const request =
      await this.persistenceService.getGenerationRequestOrThrow(requestId);

    if (request.pipelineStatus === GenerationPipelineStatus.RUNNING) {
      throw new ConflictException('Generation request is already running');
    }

    await this.persistenceService.resetRequestForRetry(requestId);
    void this.runFullCoursePipeline(requestId, request.initialUserPrompt);

    return this.buildGenerationStartResponse(requestId);
  }

  /**
   * Returns the persisted observable status for one generation request.
   *
   * @param requestId - Generation request id.
   * @returns Pollable generation status.
   */
  public async getGenerationStatus(
    requestId: string,
  ): Promise<GenerationStatusResponseDto> {
    return await this.persistenceService.getGenerationStatus(requestId);
  }

  /**
   * Returns the persisted course result for one generation request.
   *
   * @param requestId - Generation request id.
   * @returns Generated course result.
   */
  public async getGenerationResult(
    requestId: string,
  ): Promise<GenerationResultResponseDto> {
    return await this.persistenceService.getGenerationResult(requestId);
  }

  private async runFullCoursePipeline(
    requestId: string,
    prompt: string,
  ): Promise<void> {
    try {
      await this.persistenceService.updateRequestProgress(
        requestId,
        'analysis',
        5,
      );
      const analysis = await this.promptAnalysisForRequest(requestId, prompt);

      if (analysis.isOutOfScope) {
        await this.persistenceService.markRequestFailed(
          requestId,
          analysis.errorMessage ?? 'The requested course is out of scope.',
        );
        return;
      }

      await this.persistenceService.updateRequestProgress(
        requestId,
        'architecture',
        20,
      );
      const courseContext = this.buildCourseContextFromAnalysis(
        requestId,
        analysis,
      );
      const architectureResult = await this.promptArchitecture(courseContext);

      const generatedPlans = await this.generateAllLessonPlans(
        architectureResult.courseId,
        architectureResult.architecture,
        requestId,
      );

      await this.generateAllLessonContents(
        architectureResult.courseId,
        architectureResult.architecture,
        generatedPlans,
        requestId,
      );

      await this.persistenceService.markRequestCompleted(requestId);
    } catch (error) {
      await this.persistenceService.markRequestFailed(
        requestId,
        this.getSafeErrorMessage(error),
      );
      this.logSafeGenerationError('Full course pipeline failed', error);
    }
  }

  private async generateAllLessonPlans(
    courseId: string,
    architecture: ArchitectureResponseDto,
    requestId: string,
  ): Promise<GeneratedModuleLessonPlan[]> {
    const generatedPlans: GeneratedModuleLessonPlan[] = [];

    for (let index = 0; index < architecture.modules.length; index += 1) {
      const module = architecture.modules[index];
      const progress =
        30 + Math.round((index / architecture.modules.length) * 25);
      await this.persistenceService.updateRequestProgress(
        requestId,
        `lesson_plan_module_${module.order}`,
        progress,
      );

      const lessonContext = this.buildLessonContext(
        courseId,
        architecture,
        module,
      );
      const lessonPlan = await this.promptLessons(lessonContext);
      const persistedLessons =
        await this.persistenceService.findPersistedLessonsForModule(
          courseId,
          module.order,
        );

      generatedPlans.push({ module, lessonPlan, persistedLessons });
    }

    return generatedPlans;
  }

  private async generateAllLessonContents(
    courseId: string,
    architecture: ArchitectureResponseDto,
    generatedPlans: GeneratedModuleLessonPlan[],
    requestId: string,
  ): Promise<void> {
    const totalLessons = generatedPlans.reduce(
      (total, generatedPlan) => total + generatedPlan.lessonPlan.lessons.length,
      0,
    );
    let generatedLessonCount = 0;
    const previousLessonsSummary: string[] = [];

    for (const generatedPlan of generatedPlans) {
      for (const lesson of generatedPlan.lessonPlan.lessons) {
        const persistedLesson = generatedPlan.persistedLessons.find(
          (candidate) => candidate.lessonOrder === lesson.order,
        );

        if (!persistedLesson) {
          throw new BadRequestException('Persisted lesson not found');
        }

        const progress =
          55 + Math.round((generatedLessonCount / totalLessons) * 40);
        await this.persistenceService.updateRequestProgress(
          requestId,
          `lesson_content_${persistedLesson.id}`,
          progress,
        );

        const lessonContent = await this.promptLessonContent(
          this.buildLessonContentContext(
            courseId,
            architecture,
            generatedPlan.module,
            lesson,
            persistedLesson.id,
            previousLessonsSummary,
          ),
        );

        previousLessonsSummary.push(
          `${lessonContent.title}: ${lessonContent.summary}`,
        );
        generatedLessonCount += 1;
      }
    }
  }

  private buildCourseContextFromAnalysis(
    requestId: string,
    analysis: AnalysisResponseDto,
  ): CourseContextDto {
    return {
      requestId,
      title: analysis.suggestedTitle,
      synopsis: analysis.shortSynopsis,
      currentLevel: analysis.detectedCurrentLevel,
      targetLevel: analysis.detectedTargetLevel,
      goals: this.extractGoals(analysis.detectedGoal),
      language: analysis.detectedLanguage,
    };
  }

  private buildLessonContext(
    courseId: string,
    architecture: ArchitectureResponseDto,
    module: ArchitectureModuleDto,
  ): LessonContextDto {
    return {
      courseId,
      courseContext: {
        title: architecture.title,
        synopsis: architecture.synopsis,
        targetAudience: architecture.targetAudience,
        prerequisites: architecture.prerequisites,
        goals: architecture.goals,
        acquiredSkills: architecture.acquiredSkills,
        finalProject: architecture.finalProject,
      },
      moduleToExpand: module,
      globalPlanSummary: architecture.modules.map(
        (courseModule) =>
          `Module ${courseModule.order}: ${courseModule.title} - ${courseModule.description}`,
      ),
    };
  }

  private buildLessonContentContext(
    courseId: string,
    architecture: ArchitectureResponseDto,
    module: ArchitectureModuleDto,
    lesson: LessonPlanItemDto,
    lessonId: string,
    previousLessonsSummary: string[],
  ): LessonContentContextDto {
    return {
      courseId,
      courseContext: {
        title: architecture.title,
        synopsis: architecture.synopsis,
        targetAudience: architecture.targetAudience,
        prerequisites: architecture.prerequisites,
        goals: architecture.goals,
        acquiredSkills: architecture.acquiredSkills,
        finalProject: architecture.finalProject,
      },
      moduleContext: module,
      lessonToGenerate: {
        lessonId,
        order: lesson.order,
        title: lesson.title,
        type: lesson.type,
        estimatedDuration: lesson.estimatedDuration,
        learningGoal: lesson.learningGoal,
        requiresDiagram: lesson.requiresDiagram,
        technicalKeywords: lesson.technicalKeywords,
      },
      previousLessonsSummary,
    };
  }

  private extractGoals(detectedGoal: string): string[] {
    const normalizedGoal = detectedGoal.trim();

    if (!normalizedGoal || normalizedGoal.toLowerCase() === 'unknown') {
      return ['Acquérir une compétence IT exploitable en projet réel.'];
    }

    return [normalizedGoal];
  }

  private buildGenerationStartResponse(
    requestId: string,
  ): GenerationStartResponseDto {
    return {
      requestId,
      status: GenerationPipelineStatus.QUEUED,
      statusUrl: `/course/generator/requests/${requestId}/status`,
      resultUrl: `/course/generator/requests/${requestId}/result`,
    };
  }

  private async generateStructuredCompletion<TSchema extends z.ZodType>(
    promptName: string,
    userContent: string,
    schema: TSchema,
    responseName: string,
  ): Promise<StructuredCompletionResult<z.infer<TSchema>>> {
    const systemPrompt = this.prompts.get(promptName);
    if (!systemPrompt) {
      throw new Error('erreur de récuperation du prompt system');
    }

    const completion = await this.withOpenAiRetry(() =>
      this.openai.chat.completions.parse({
        model: this.aiModel,
        messages: [
          {
            role: 'system',
            content: systemPrompt,
          },
          {
            role: 'user',
            content: userContent,
          },
        ],
        response_format: zodResponseFormat(schema, responseName),
        temperature: 0.2,
      }),
    );
    const message = completion.choices[0]?.message;
    const parsed = message?.parsed;

    if (!parsed) {
      throw new Error("erreur aucune réponse structurée de l'api");
    }

    return {
      parsed,
      rawContent: message.content ?? JSON.stringify(parsed),
    };
  }

  private async withOpenAiRetry<T>(operation: () => Promise<T>): Promise<T> {
    let attempt = 0;

    while (true) {
      try {
        return await operation();
      } catch (error) {
        if (
          attempt >= this.maxOpenAiRetries ||
          !this.isRetryableOpenAiError(error)
        ) {
          throw error;
        }

        await this.delay(500 * 2 ** attempt);
        attempt += 1;
      }
    }
  }

  private isRetryableOpenAiError(error: unknown): boolean {
    const status = this.getErrorStatus(error);

    if (status === undefined) {
      return true;
    }

    return status === 408 || status === 409 || status === 429 || status >= 500;
  }

  private getErrorStatus(error: unknown): number | undefined {
    if (typeof error !== 'object' || error === null || !('status' in error)) {
      return undefined;
    }

    const status = (error as { status?: unknown }).status;
    return typeof status === 'number' ? status : undefined;
  }

  private getSafeErrorMessage(error: unknown): string {
    if (error instanceof Error) {
      return error.message;
    }

    return 'Unknown generation error';
  }

  private logSafeGenerationError(message: string, error: unknown): void {
    console.error(`${message}: ${this.getSafeErrorMessage(error)}`);
  }

  private async delay(milliseconds: number): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve, milliseconds));
  }
}
