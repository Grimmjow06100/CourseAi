import { Type } from 'class-transformer';
import { ApiProperty } from '@nestjs/swagger';
import {
  ArrayMinSize,
  IsArray,
  IsInt,
  IsString,
  IsUUID,
  Min,
  ValidateNested,
} from 'class-validator';

export class LessonCourseFinalProjectContextDto {
  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  description!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  constraints!: string[];
}

export class LessonCourseContextDto {
  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  synopsis!: string;

  @ApiProperty()
  @IsString()
  targetAudience!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @IsString({ each: true })
  prerequisites!: string[];

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  goals!: string[];

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  acquiredSkills!: string[];

  @ApiProperty({ type: () => LessonCourseFinalProjectContextDto })
  @ValidateNested()
  @Type(() => LessonCourseFinalProjectContextDto)
  finalProject!: LessonCourseFinalProjectContextDto;
}

export class LessonModuleToExpandDto {
  @ApiProperty()
  @IsInt()
  @Min(1)
  order!: number;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  description!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  keyLearningPoints!: string[];
}

export class LessonContextDto {
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
  moduleToExpand!: LessonModuleToExpandDto;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  globalPlanSummary!: string[];
}
