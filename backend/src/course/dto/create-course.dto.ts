import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { IsArray, IsEnum, IsOptional, IsString } from 'class-validator';
import { CourseLanguage, Level } from '../../generated/prisma/client';

export class CreateCourseDto {
  @ApiProperty()
  @IsString()
  initialUserPrompt!: string;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  synopsis!: string;

  @ApiPropertyOptional()
  @IsOptional()
  @IsString()
  targetAudience?: string;

  @ApiProperty({ enum: Level })
  @IsEnum(Level)
  currentLevel!: Level;

  @ApiProperty({ enum: Level })
  @IsEnum(Level)
  targetLevel!: Level;

  @ApiProperty({ enum: CourseLanguage })
  @IsEnum(CourseLanguage)
  language!: CourseLanguage;

  @ApiPropertyOptional({ type: [String] })
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  goals?: string[];
}
