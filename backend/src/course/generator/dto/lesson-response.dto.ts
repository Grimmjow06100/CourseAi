import { Type } from 'class-transformer';
import { ApiProperty } from '@nestjs/swagger';
import {
  ArrayMaxSize,
  ArrayMinSize,
  IsArray,
  IsBoolean,
  IsEnum,
  IsInt,
  IsString,
  Min,
  ValidateNested,
} from 'class-validator';

export enum LessonPlanType {
  Theory = 'theory',
  Practice = 'practice',
  Mixed = 'mixed',
  Quiz = 'quiz',
}

export class LessonPlanItemDto {
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

export class LessonResponseDto {
  @ApiProperty()
  @IsInt()
  @Min(1)
  moduleOrder!: number;

  @ApiProperty()
  @IsString()
  moduleTitle!: string;

  @ApiProperty({ type: () => [LessonPlanItemDto] })
  @IsArray()
  @ArrayMinSize(3)
  @ArrayMaxSize(6)
  @ValidateNested({ each: true })
  @Type(() => LessonPlanItemDto)
  lessons!: LessonPlanItemDto[];
}
