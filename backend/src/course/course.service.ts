import { Injectable, NotFoundException } from '@nestjs/common';
import { CreateCourseDto } from './dto/create-course.dto';
import { UpdateCourseDto } from './dto/update-course.dto';
import { PrismaService } from '../prisma/prisma.service';
import { CourseGenerationStatus, Prisma } from '../generated/prisma/client';
import { UpdateModuleDto } from './dto/update-module.dto';
import { UpdateLessonDto } from './dto/update-lesson.dto';

@Injectable()
export class CourseService {
  constructor(private readonly prisma: PrismaService) {}

  async create(createCourseDto: CreateCourseDto) {
    return await this.prisma.$transaction(async (tx) => {
      const generationRequest = await tx.generationRequest.create({
        data: {
          initialUserPrompt: createCourseDto.initialUserPrompt,
          suggestedTitle: createCourseDto.title,
          shortSynopsis: createCourseDto.synopsis,
          detectedCurrentLevel: createCourseDto.currentLevel,
          detectedTargetLevel: createCourseDto.targetLevel,
          detectedLanguage: createCourseDto.language,
          detectedGoal: createCourseDto.goals?.join(', ') ?? null,
          clarificationQuestions: [],
        },
      });

      return await tx.course.create({
        data: {
          requestId: generationRequest.id,
          language: createCourseDto.language,
          status: CourseGenerationStatus.ANALYSIS_COMPLETED,
          initialUserPrompt: createCourseDto.initialUserPrompt,
          title: createCourseDto.title,
          synopsis: createCourseDto.synopsis,
          targetAudience: createCourseDto.targetAudience,
          currentLevel: createCourseDto.currentLevel,
          targetLevel: createCourseDto.targetLevel,
          goals: this.toJson(createCourseDto.goals ?? []),
        },
        include: this.courseInclude,
      });
    });
  }

  async findAll() {
    return await this.prisma.course.findMany({
      orderBy: { createdAt: 'desc' },
      include: {
        modules: {
          orderBy: { moduleOrder: 'asc' },
          include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
        },
        request: true,
      },
    });
  }

  async findOne(id: string) {
    return await this.prisma.course.findUniqueOrThrow({
      where: { id },
      include: this.courseInclude,
    });
  }

  async update(id: string, updateCourseDto: UpdateCourseDto) {
    await this.ensureCourseExists(id);

    return await this.prisma.course.update({
      where: { id },
      data: {
        title: updateCourseDto.title,
        synopsis: updateCourseDto.synopsis,
        targetAudience: updateCourseDto.targetAudience,
        currentLevel: updateCourseDto.currentLevel,
        targetLevel: updateCourseDto.targetLevel,
        language: updateCourseDto.language,
        goals: updateCourseDto.goals
          ? this.toJson(updateCourseDto.goals)
          : undefined,
      },
      include: this.courseInclude,
    });
  }

  async remove(id: string) {
    await this.ensureCourseExists(id);

    return await this.prisma.course.delete({
      where: { id },
    });
  }

  async findModules(courseId: string) {
    await this.ensureCourseExists(courseId);

    return await this.prisma.courseModule.findMany({
      where: { courseId },
      orderBy: { moduleOrder: 'asc' },
      include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
    });
  }

  async findModule(moduleId: string) {
    return await this.prisma.courseModule.findUniqueOrThrow({
      where: { id: moduleId },
      include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
    });
  }

  async updateModule(moduleId: string, updateModuleDto: UpdateModuleDto) {
    await this.ensureModuleExists(moduleId);

    return await this.prisma.courseModule.update({
      where: { id: moduleId },
      data: {
        title: updateModuleDto.title,
        description: updateModuleDto.description,
        keyLearningPoints: updateModuleDto.keyLearningPoints
          ? this.toJson(updateModuleDto.keyLearningPoints)
          : undefined,
      },
      include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
    });
  }

  async removeModule(moduleId: string) {
    await this.ensureModuleExists(moduleId);

    return await this.prisma.courseModule.delete({
      where: { id: moduleId },
    });
  }

  async findLessons(moduleId: string) {
    await this.ensureModuleExists(moduleId);

    return await this.prisma.lesson.findMany({
      where: { moduleId },
      orderBy: { lessonOrder: 'asc' },
    });
  }

  async findLesson(lessonId: string) {
    return await this.prisma.lesson.findUniqueOrThrow({
      where: { id: lessonId },
    });
  }

  async updateLesson(lessonId: string, updateLessonDto: UpdateLessonDto) {
    await this.ensureLessonExists(lessonId);

    return await this.prisma.lesson.update({
      where: { id: lessonId },
      data: {
        title: updateLessonDto.title,
        type: updateLessonDto.type,
        estimatedDurationMinutes: updateLessonDto.estimatedDurationMinutes,
        learningGoal: updateLessonDto.learningGoal,
        requiresDiagram: updateLessonDto.requiresDiagram,
        technicalKeywords: updateLessonDto.technicalKeywords
          ? this.toJson(updateLessonDto.technicalKeywords)
          : undefined,
        contentMarkdown: updateLessonDto.contentMarkdown,
      },
    });
  }

  async removeLesson(lessonId: string) {
    await this.ensureLessonExists(lessonId);

    return await this.prisma.lesson.delete({
      where: { id: lessonId },
    });
  }

  private readonly courseInclude = {
    request: true,
    modules: {
      orderBy: { moduleOrder: 'asc' },
      include: { lessons: { orderBy: { lessonOrder: 'asc' } } },
    },
  } satisfies Prisma.CourseInclude;

  private toJson(value: unknown): Prisma.InputJsonValue {
    return value as Prisma.InputJsonValue;
  }

  private async ensureCourseExists(id: string): Promise<void> {
    const course = await this.prisma.course.findUnique({ where: { id } });
    if (!course) throw new NotFoundException('Course not found');
  }

  private async ensureModuleExists(id: string): Promise<void> {
    const module = await this.prisma.courseModule.findUnique({ where: { id } });
    if (!module) throw new NotFoundException('Course module not found');
  }

  private async ensureLessonExists(id: string): Promise<void> {
    const lesson = await this.prisma.lesson.findUnique({ where: { id } });
    if (!lesson) throw new NotFoundException('Lesson not found');
  }
}
