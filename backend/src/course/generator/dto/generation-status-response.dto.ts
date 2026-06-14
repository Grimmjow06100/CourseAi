import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import {
  CourseGenerationStatus,
  GenerationPipelineStatus,
} from '../../../generated/prisma/client';

export class GenerationStatusResponseDto {
  @ApiProperty({ format: 'uuid' })
  requestId!: string;

  @ApiPropertyOptional({ format: 'uuid', nullable: true })
  courseId!: string | null;

  @ApiProperty({ enum: GenerationPipelineStatus })
  pipelineStatus!: GenerationPipelineStatus;

  @ApiPropertyOptional({ enum: CourseGenerationStatus, nullable: true })
  courseStatus!: CourseGenerationStatus | null;

  @ApiPropertyOptional({ nullable: true })
  currentStep!: string | null;

  @ApiProperty()
  progressPercent!: number;

  @ApiPropertyOptional({ nullable: true })
  errorMessage!: string | null;

  @ApiPropertyOptional({ nullable: true })
  startedAt!: Date | null;

  @ApiPropertyOptional({ nullable: true })
  completedAt!: Date | null;
}
