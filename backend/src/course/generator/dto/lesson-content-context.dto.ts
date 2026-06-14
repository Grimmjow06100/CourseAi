import { Type } from 'class-transformer';
import { ApiProperty } from '@nestjs/swagger';
import {
  ArrayMinSize,
  IsArray,
  IsBoolean,
  IsEnum,
  IsInt,
  IsOptional,
  IsString,
  IsUUID,
  Min,
  ValidateNested,
} from 'class-validator';
import { LessonPlanType } from './lesson-response.dto';
import {
  LessonCourseContextDto,
  LessonModuleToExpandDto,
} from './lesson-context.dto';

export class LessonContentTargetDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  lessonId!: string;

  @ApiProperty()
  @IsInt()
  @Min(1)
  order!: number;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty({ enum: LessonPlanType })
  @IsEnum(LessonPlanType)
  type!: LessonPlanType;

  @ApiProperty()
  @IsInt()
  @Min(1)
  estimatedDuration!: number;

  @ApiProperty()
  @IsString()
  learningGoal!: string;

  @ApiProperty()
  @IsBoolean()
  requiresDiagram!: boolean;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  technicalKeywords!: string[];
}

export class LessonContentContextDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  courseId!: string;

  @ApiProperty({ type: () => LessonCourseContextDto })
  @ValidateNested()
  @Type(() => LessonCourseContextDto)
  courseContext!: LessonCourseContextDto;

  @ApiProperty({ type: () => LessonModuleToExpandDto })
  @ValidateNested()
  @Type(() => LessonModuleToExpandDto)
  moduleContext!: LessonModuleToExpandDto;

  @ApiProperty({ type: () => LessonContentTargetDto })
  @ValidateNested()
  @Type(() => LessonContentTargetDto)
  lessonToGenerate!: LessonContentTargetDto;

  @ApiProperty({ type: [String], required: false })
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  previousLessonsSummary?: string[];
}
