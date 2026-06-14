import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import {
  CourseGenerationStatus,
  CourseLanguage,
  GenerationPipelineStatus,
  LessonType,
  Level,
  Prisma,
} from '../../../generated/prisma/client';
import { PrismaService } from '../../../prisma/prisma.service';
import { AnalysisResponseDto } from '../dto/analysis-response.dto';
import { ArchitectureResponseDto } from '../dto/architecture-response.dto';
import { CourseContextDto } from '../dto/course-context.dto';
import { LessonContextDto } from '../dto/lesson-context.dto';
import { LessonContentContextDto } from '../dto/lesson-content-context.dto';
import { LessonContentResponseDto } from '../dto/lesson-content-response.dto';
import { LessonPlanType, LessonResponseDto } from '../dto/lesson-response.dto';
import { GenerationStatusResponseDto } from '../dto/generation-status-response.dto';
import { GenerationResultResponseDto } from '../dto/generation-result-response.dto';

@Injectable()
export class GeneratorPersistenceService {
  constructor(private readonly prisma: PrismaService) {}

  /**
   * Creates the root generation request used as the asynchronous pipeline job.
   *
   * @param initialUserPrompt - Raw prompt entered by the user.
   * @returns The persisted generation request id.
   */
  public async createQueuedGenerationRequest(
    initialUserPrompt: string,
  ): Promise<string> {
    const request = await this.prisma.generationRequest.create({
      data: {
        initialUserPrompt,
        pipelineStatus: GenerationPipelineStatus.QUEUED,
        currentStep: 'queued',
        progressPercent: 0,
        clarificationQuestions: [],
      },
    });

    return request.id;
  }

  /**
   * Updates generation job progress in a single place so every pipeline phase is observable.
   *
   * @param requestId - Generation request id.
   * @param currentStep - Human-readable machine step name.
   * @param progressPercent - Integer progress from 0 to 100.
   * @returns The updated generation request.
   */
  public async updateRequestProgress(
    requestId: string,
    currentStep: string,
    progressPercent: number,
  ) {
    return await this.prisma.generationRequest.update({
      where: { id: requestId },
      data: {
        pipelineStatus: GenerationPipelineStatus.RUNNING,
        currentStep,
        progressPercent: Math.min(Math.max(progressPercent, 0), 99),
        failureMessage: null,
        startedAt: new Date(),
        completedAt: null,
      },
    });
  }

  /**
   * Marks a generation request as failed without leaking raw provider errors.
   *
   * @param requestId - Generation request id.
   * @param failureMessage - Sanitized failure message.
   * @returns The updated generation request.
   */
  public async markRequestFailed(requestId: string, failureMessage: string) {
    await this.prisma.course.updateMany({
      where: { requestId },
      data: { status: CourseGenerationStatus.FAILED },
    });

    return await this.prisma.generationRequest.update({
      where: { id: requestId },
      data: {
        pipelineStatus: GenerationPipelineStatus.FAILED,
        currentStep: 'failed',
        failureMessage,
        progressPercent: 100,
        completedAt: new Date(),
      },
    });
  }

  /**
   * Marks a generation request as completed.
   *
   * @param requestId - Generation request id.
   * @returns The updated generation request.
   */
  public async markRequestCompleted(requestId: string) {
    return await this.prisma.generationRequest.update({
      where: { id: requestId },
      data: {
        pipelineStatus: GenerationPipelineStatus.COMPLETED,
        currentStep: 'completed',
        failureMessage: null,
        progressPercent: 100,
        completedAt: new Date(),
      },
    });
  }

  /**
   * Resets status fields before retrying an existing generation request.
   *
   * @param requestId - Generation request id.
   * @returns The updated generation request.
   */
  public async resetRequestForRetry(requestId: string) {
    const request = await this.prisma.generationRequest.findUnique({
      where: { id: requestId },
    });

    if (!request) {
      throw new NotFoundException('Generation request not found');
    }

    return await this.prisma.generationRequest.update({
      where: { id: requestId },
      data: {
        pipelineStatus: GenerationPipelineStatus.QUEUED,
        currentStep: 'queued',
        progressPercent: 0,
        failureMessage: null,
        startedAt: null,
        completedAt: null,
      },
    });
  }

  /**
   * Returns the persisted prompt for retrying a generation request.
   *
   * @param requestId - Generation request id.
   * @returns The generation request.
   */
  public async getGenerationRequestOrThrow(requestId: string) {
    const request = await this.prisma.generationRequest.findUnique({
      where: { id: requestId },
    });

    if (!request) {
      throw new NotFoundException('Generation request not found');
    }

    return request;
  }

  /**
   * Returns a lightweight observable status for a generation request.
   *
   * @param requestId - Generation request id.
   * @returns A status payload suitable for polling.
   */
  public async getGenerationStatus(
    requestId: string,
  ): Promise<GenerationStatusResponseDto> {
    const request = await this.prisma.generationRequest.findUnique({
      where: { id: requestId },
      include: { course: true },
    });

    if (!request) {
      throw new NotFoundException('Generation request not found');
    }

    return {
      requestId: request.id,
      courseId: request.course?.id ?? null,
      pipelineStatus: request.pipelineStatus,
      courseStatus: request.course?.status ?? null,
      currentStep: request.currentStep,
      progressPercent: request.progressPercent,
      errorMessage: request.failureMessage ?? request.errorMessage,
      startedAt: request.startedAt,
      completedAt: request.completedAt,
    };
  }

  /**
   * Returns the fully persisted generated course for a completed or partial request.
   *
   * @param requestId - Generation request id.
   * @returns Course result with modules and lessons.
   */
  public async getGenerationResult(
    requestId: string,
  ): Promise<GenerationResultResponseDto> {
    const request = await this.prisma.generationRequest.findUnique({
      where: { id: requestId },
      include: {
        course: {
          include: {
            request: true,
            modules: {
              orderBy: { moduleOrder: 'asc' },
              include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
            },
          },
        },
      },
    });

    if (!request) {
      throw new NotFoundException('Generation request not found');
    }

    if (!request.course) {
      throw new NotFoundException('Generated course not available yet');
    }

    return {
      requestId: request.id,
      courseId: request.course.id,
      course: request.course as unknown as Record<string, unknown>,
    };
  }

  /**
   * Persists the needs-analysis output before the user answers clarification questions.
   *
   * @param initialUserPrompt - Raw prompt entered by the user.
   * @param analysis - Validated model analysis.
   * @param rawAnalysisOutput - Raw JSON string returned by the model.
   * @returns The persisted generation request id.
   */
  public async persistAnalysis(
    initialUserPrompt: string,
    analysis: AnalysisResponseDto,
    rawAnalysisOutput: string,
  ): Promise<string> {
    const request = await this.prisma.generationRequest.create({
      data: {
        initialUserPrompt,
        isOutOfScope: analysis.isOutOfScope,
        errorMessage: analysis.errorMessage,
        warningMessage: analysis.warningMessage,
        suggestedTitle: analysis.suggestedTitle,
        shortSynopsis: analysis.shortSynopsis,
        detectedCurrentLevel: this.toLevelOrNull(analysis.detectedCurrentLevel),
        detectedTargetLevel: this.toLevelOrNull(analysis.detectedTargetLevel),
        detectedGoal: analysis.detectedGoal,
        detectedLanguage: this.toCourseLanguageOrNull(
          analysis.detectedLanguage,
        ),
        clarificationQuestions: this.toJson(analysis.clarificationQuestions),
        rawAnalysisOutput: this.parseJson(rawAnalysisOutput),
      },
    });

    return request.id;
  }

  /**
   * Persists analysis output into an existing request used by the full pipeline.
   *
   * @param requestId - Generation request id.
   * @param analysis - Validated model analysis.
   * @param rawAnalysisOutput - Raw JSON string returned by the model.
   * @returns The persisted generation request id.
   */
  public async persistAnalysisForRequest(
    requestId: string,
    analysis: AnalysisResponseDto,
    rawAnalysisOutput: string,
  ): Promise<string> {
    await this.getGenerationRequestOrThrow(requestId);

    await this.prisma.generationRequest.update({
      where: { id: requestId },
      data: {
        isOutOfScope: analysis.isOutOfScope,
        errorMessage: analysis.errorMessage,
        warningMessage: analysis.warningMessage,
        suggestedTitle: analysis.suggestedTitle,
        shortSynopsis: analysis.shortSynopsis,
        detectedCurrentLevel: this.toLevelOrNull(analysis.detectedCurrentLevel),
        detectedTargetLevel: this.toLevelOrNull(analysis.detectedTargetLevel),
        detectedGoal: analysis.detectedGoal,
        detectedLanguage: this.toCourseLanguageOrNull(
          analysis.detectedLanguage,
        ),
        clarificationQuestions: this.toJson(analysis.clarificationQuestions),
        rawAnalysisOutput: this.parseJson(rawAnalysisOutput),
      },
    });

    return requestId;
  }

  /**
   * Persists the course architecture and replaces generated modules atomically.
   *
   * @param payload - User-confirmed course context used for generation.
   * @param architecture - Validated architecture returned by the model.
   * @param rawArchitectureOutput - Raw JSON string returned by the model.
   * @returns The persisted course with modules.
   */
  public async persistArchitecture(
    payload: CourseContextDto,
    architecture: ArchitectureResponseDto,
    rawArchitectureOutput: string,
  ) {
    const request = await this.prisma.generationRequest.findUnique({
      where: { id: payload.requestId },
    });

    if (!request) {
      throw new BadRequestException('Generation request not found');
    }

    const currentLevel = this.toLevel(payload.currentLevel);
    const targetLevel = this.toLevel(payload.targetLevel);
    const language = this.toCourseLanguage(payload.language);

    return await this.prisma.$transaction(async (tx) => {
      const course = await tx.course.upsert({
        where: { requestId: payload.requestId },
        create: {
          requestId: payload.requestId,
          language,
          status: CourseGenerationStatus.STRUCTURE_GENERATED,
          initialUserPrompt: request.initialUserPrompt,
          title: architecture.title,
          synopsis: architecture.synopsis,
          targetAudience: architecture.targetAudience,
          currentLevel,
          targetLevel,
          prerequisites: this.toJson(architecture.prerequisites),
          goals: this.toJson(architecture.goals),
          acquiredSkills: this.toJson(architecture.acquiredSkills),
          finalProjectTitle: architecture.finalProject.title,
          finalProjectDescription: architecture.finalProject.description,
          finalProjectConstraints: this.toJson(
            architecture.finalProject.constraints,
          ),
          generationPayload: this.toJson(payload),
          rawArchitectureOutput: this.parseJson(rawArchitectureOutput),
        },
        update: {
          language,
          status: CourseGenerationStatus.STRUCTURE_GENERATED,
          title: architecture.title,
          synopsis: architecture.synopsis,
          targetAudience: architecture.targetAudience,
          currentLevel,
          targetLevel,
          prerequisites: this.toJson(architecture.prerequisites),
          goals: this.toJson(architecture.goals),
          acquiredSkills: this.toJson(architecture.acquiredSkills),
          finalProjectTitle: architecture.finalProject.title,
          finalProjectDescription: architecture.finalProject.description,
          finalProjectConstraints: this.toJson(
            architecture.finalProject.constraints,
          ),
          generationPayload: this.toJson(payload),
          rawArchitectureOutput: this.parseJson(rawArchitectureOutput),
        },
      });

      await tx.courseModule.deleteMany({ where: { courseId: course.id } });
      await tx.courseModule.createMany({
        data: architecture.modules.map((module) => ({
          courseId: course.id,
          moduleOrder: module.order,
          title: module.title,
          description: module.description,
          keyLearningPoints: this.toJson(module.keyLearningPoints),
          rawModuleOutput: this.toJson(module),
        })),
      });

      return await tx.course.findUniqueOrThrow({
        where: { id: course.id },
        include: { modules: { orderBy: { moduleOrder: 'asc' } } },
      });
    });
  }

  /**
   * Persists generated lesson plans for a module and updates the course status.
   *
   * @param payload - Context used to generate the module lesson plan.
   * @param lessonPlan - Validated lesson plan returned by the model.
   * @param rawLessonPlanOutput - Raw JSON string returned by the model.
   * @returns The persisted module with its lessons.
   */
  public async persistLessons(
    payload: LessonContextDto,
    lessonPlan: LessonResponseDto,
    rawLessonPlanOutput: string,
  ) {
    return await this.prisma.$transaction(async (tx) => {
      const module = await tx.courseModule.findUnique({
        where: {
          courseId_moduleOrder: {
            courseId: payload.courseId,
            moduleOrder: lessonPlan.moduleOrder,
          },
        },
        include: { course: true },
      });

      if (!module) {
        throw new BadRequestException('Course module not found');
      }

      await tx.lesson.deleteMany({ where: { moduleId: module.id } });
      await tx.courseModule.update({
        where: { id: module.id },
        data: {
          rawLessonsPlanOutput: this.parseJson(rawLessonPlanOutput),
        },
      });

      await tx.lesson.createMany({
        data: lessonPlan.lessons.map((lesson) => ({
          moduleId: module.id,
          lessonOrder: lesson.order,
          title: lesson.title,
          type: this.toLessonType(lesson.type),
          estimatedDurationMinutes: lesson.estimatedDuration,
          learningGoal: lesson.learningGoal,
          requiresDiagram: lesson.requiresDiagram,
          technicalKeywords: this.toJson(lesson.technicalKeywords),
        })),
      });

      const modules = await tx.courseModule.findMany({
        where: { courseId: payload.courseId },
        select: {
          id: true,
          _count: { select: { lessons: true } },
        },
      });
      const areAllModuleLessonsGenerated = modules.every(
        (courseModule) => courseModule._count.lessons > 0,
      );

      await tx.course.update({
        where: { id: payload.courseId },
        data: {
          status: areAllModuleLessonsGenerated
            ? CourseGenerationStatus.LESSONS_GENERATED
            : CourseGenerationStatus.LESSONS_GENERATING,
        },
      });

      return await tx.courseModule.findUniqueOrThrow({
        where: { id: module.id },
        include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
      });
    });
  }

  /**
   * Persists generated Markdown content for a single lesson and updates course status.
   *
   * @param payload - Context used to generate the lesson content.
   * @param lessonContent - Validated lesson content returned by the model.
   * @param rawLessonContentOutput - Raw JSON string returned by the model.
   * @returns The updated lesson.
   */
  public async persistLessonContent(
    payload: LessonContentContextDto,
    lessonContent: LessonContentResponseDto,
    rawLessonContentOutput: string,
  ) {
    if (payload.lessonToGenerate.lessonId !== lessonContent.lessonId) {
      throw new BadRequestException('Generated lesson id does not match input');
    }

    return await this.prisma.$transaction(async (tx) => {
      const lesson = await tx.lesson.findUnique({
        where: { id: lessonContent.lessonId },
        include: { module: true },
      });

      if (!lesson) {
        throw new BadRequestException('Lesson not found');
      }

      if (lesson.module.courseId !== payload.courseId) {
        throw new BadRequestException('Lesson does not belong to this course');
      }

      const updatedLesson = await tx.lesson.update({
        where: { id: lessonContent.lessonId },
        data: {
          contentMarkdown: lessonContent.contentMarkdown,
          rawContentOutput: this.parseJson(rawLessonContentOutput),
        },
      });

      const lessons = await tx.lesson.findMany({
        where: {
          module: { courseId: payload.courseId },
        },
        select: {
          contentMarkdown: true,
        },
      });
      const areAllLessonContentsGenerated = lessons.every((courseLesson) =>
        Boolean(courseLesson.contentMarkdown),
      );

      await tx.course.update({
        where: { id: payload.courseId },
        data: {
          status: areAllLessonContentsGenerated
            ? CourseGenerationStatus.COMPLETED
            : CourseGenerationStatus.CONTENT_GENERATING,
        },
      });

      return updatedLesson;
    });
  }

  /**
   * Returns persisted lessons for a generated module.
   *
   * @param courseId - Course id.
   * @param moduleOrder - Module order in the course.
   * @returns Persisted lessons ordered by lesson order.
   */
  public async findPersistedLessonsForModule(
    courseId: string,
    moduleOrder: number,
  ) {
    const module = await this.prisma.courseModule.findUnique({
      where: {
        courseId_moduleOrder: {
          courseId,
          moduleOrder,
        },
      },
      include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
    });

    if (!module) {
      throw new NotFoundException('Course module not found');
    }

    return module.lessons;
  }

  private toJson(value: unknown): Prisma.InputJsonValue {
    return value as Prisma.InputJsonValue;
  }

  private parseJson(jsonString: string): Prisma.InputJsonValue {
    return JSON.parse(jsonString) as Prisma.InputJsonValue;
  }

  private toCourseLanguage(value: string): CourseLanguage {
    const normalizedValue = value.toLowerCase();
    if (normalizedValue === 'fr') return CourseLanguage.FR;
    if (normalizedValue === 'en') return CourseLanguage.EN;
    throw new BadRequestException(`Unsupported course language: ${value}`);
  }

  private toCourseLanguageOrNull(value: string): CourseLanguage | null {
    return value ? this.toCourseLanguage(value) : null;
  }

  private toLevel(value: string): Level {
    const normalizedValue = value.toLowerCase();
    switch (normalizedValue) {
      case 'beginner':
        return Level.BEGINNER;
      case 'intermediate':
        return Level.INTERMEDIATE;
      case 'advanced':
        return Level.ADVANCED;
      case 'expert':
        return Level.EXPERT;
      case 'unknown':
      case 'unknow':
        return Level.UNKNOWN;
      default:
        throw new BadRequestException(`Unsupported level: ${value}`);
    }
  }

  private toLevelOrNull(value: string): Level | null {
    if (!value) return null;
    return this.toLevel(value);
  }

  private toLessonType(value: LessonPlanType): LessonType {
    switch (value) {
      case LessonPlanType.Theory:
        return LessonType.THEORY;
      case LessonPlanType.Practice:
        return LessonType.PRACTICE;
      case LessonPlanType.Mixed:
        return LessonType.MIXED;
      case LessonPlanType.Quiz:
        return LessonType.QUIZ;
    }
  }
}
