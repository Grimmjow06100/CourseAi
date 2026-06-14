import { ApiPropertyOptional } from '@nestjs/swagger';
import {
  IsArray,
  IsBoolean,
  IsEnum,
  IsInt,
  IsOptional,
  IsString,
  Min,
} from 'class-validator';
import { LessonType } from '../../generated/prisma/client';

export class UpdateLessonDto {
  @ApiPropertyOptional()
  @IsOptional()
  @IsString()
  title?: string;

  @ApiPropertyOptional({ enum: LessonType })
  @IsOptional()
  @IsEnum(LessonType)
  type?: LessonType;

  @ApiPropertyOptional()
  @IsOptional()
  @IsInt()
  @Min(1)
  estimatedDurationMinutes?: number;

  @ApiPropertyOptional()
  @IsOptional()
  @IsString()
  learningGoal?: string;

  @ApiPropertyOptional()
  @IsOptional()
  @IsBoolean()
  requiresDiagram?: boolean;

  @ApiPropertyOptional({ type: [String] })
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  technicalKeywords?: string[];

  @ApiPropertyOptional()
  @IsOptional()
  @IsString()
  contentMarkdown?: string;
}
