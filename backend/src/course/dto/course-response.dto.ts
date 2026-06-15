import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import {
  CourseGenerationStatus,
  CourseLanguage,
  GenerationPipelineStatus,
  LessonType,
  Level,
} from '../../generated/prisma/client';

export class GenerationRequestSummaryResponseDto {
  @ApiProperty({ format: 'uuid' })
  id!: string;

  @ApiProperty()
  initialUserPrompt!: string;

  @ApiProperty({ enum: GenerationPipelineStatus })
  pipelineStatus!: GenerationPipelineStatus;

  @ApiPropertyOptional({ nullable: true })
  currentStep!: string | null;

  @ApiProperty()
  progressPercent!: number;

  @ApiPropertyOptional({ nullable: true })
  failureMessage!: string | null;

  @ApiProperty()
  createdAt!: Date;

  @ApiProperty()
  updatedAt!: Date;
}

export class LessonEntityResponseDto {
  @ApiProperty({ format: 'uuid' })
  id!: string;

  @ApiProperty({ format: 'uuid' })
  moduleId!: string;

  @ApiProperty()
  lessonOrder!: number;

  @ApiProperty()
  title!: string;

  @ApiProperty({ enum: LessonType })
  type!: LessonType;

  @ApiProperty()
  estimatedDurationMinutes!: number;

  @ApiProperty()
  learningGoal!: string;

  @ApiProperty()
  requiresDiagram!: boolean;

  @ApiProperty({ type: [String] })
  technicalKeywords!: string[];

  @ApiPropertyOptional({ nullable: true })
  contentMarkdown!: string | null;

  @ApiProperty()
  createdAt!: Date;

  @ApiProperty()
  updatedAt!: Date;
}

export class CourseModuleResponseDto {
  @ApiProperty({ format: 'uuid' })
  id!: string;

  @ApiProperty({ format: 'uuid' })
  courseId!: string;

  @ApiProperty()
  moduleOrder!: number;

  @ApiProperty()
  title!: string;

  @ApiProperty()
  description!: string;

  @ApiProperty({ type: [String] })
  keyLearningPoints!: string[];

  @ApiProperty({ type: () => [LessonEntityResponseDto] })
  lessons!: LessonEntityResponseDto[];

  @ApiProperty()
  createdAt!: Date;

  @ApiProperty()
  updatedAt!: Date;
}

export class CourseResponseDto {
  @ApiProperty({ format: 'uuid' })
  id!: string;

  @ApiProperty({ format: 'uuid' })
  requestId!: string;

  @ApiProperty({ enum: CourseLanguage })
  language!: CourseLanguage;

  @ApiProperty({ enum: CourseGenerationStatus })
  status!: CourseGenerationStatus;

  @ApiProperty()
  initialUserPrompt!: string;

  @ApiProperty()
  title!: string;

  @ApiProperty()
  synopsis!: string;

  @ApiPropertyOptional({ nullable: true })
  targetAudience!: string | null;

  @ApiProperty({ enum: Level })
  currentLevel!: Level;

  @ApiProperty({ enum: Level })
  targetLevel!: Level;

  @ApiProperty({ type: [String] })
  prerequisites!: string[];

  @ApiProperty({ type: [String] })
  goals!: string[];

  @ApiProperty({ type: [String] })
  acquiredSkills!: string[];

  @ApiPropertyOptional({ nullable: true })
  finalProjectTitle!: string | null;

  @ApiPropertyOptional({ nullable: true })
  finalProjectDescription!: string | null;

  @ApiProperty({ type: [String] })
  finalProjectConstraints!: string[];

  @ApiPropertyOptional({ type: () => GenerationRequestSummaryResponseDto })
  request?: GenerationRequestSummaryResponseDto;

  @ApiProperty({ type: () => [CourseModuleResponseDto] })
  modules!: CourseModuleResponseDto[];

  @ApiProperty()
  createdAt!: Date;

  @ApiProperty()
  updatedAt!: Date;
}

export class PaginationMetaResponseDto {
  @ApiProperty()
  page!: number;

  @ApiProperty()
  pageSize!: number;

  @ApiProperty()
  totalItems!: number;

  @ApiProperty()
  totalPages!: number;

  @ApiProperty()
  hasNextPage!: boolean;

  @ApiProperty()
  hasPreviousPage!: boolean;
}

export class PaginatedCoursesResponseDto {
  @ApiProperty({ type: () => [CourseResponseDto] })
  data!: CourseResponseDto[];

  @ApiProperty({ type: () => PaginationMetaResponseDto })
  meta!: PaginationMetaResponseDto;
}

export class DeleteResponseDto {
  @ApiProperty()
  id!: string;

  @ApiProperty()
  deleted!: boolean;
}
