import { Type } from 'class-transformer';
import { ApiPropertyOptional } from '@nestjs/swagger';
import {
  IsEnum,
  IsIn,
  IsInt,
  IsOptional,
  IsString,
  Max,
  Min,
} from 'class-validator';
import {
  CourseGenerationStatus,
  CourseLanguage,
} from '../../generated/prisma/client';

export class ListCoursesQueryDto {
  @ApiPropertyOptional({ minimum: 1, default: 1 })
  @IsOptional()
  @Type(() => Number)
  @IsInt()
  @Min(1)
  page?: number = 1;

  @ApiPropertyOptional({ minimum: 1, maximum: 100, default: 20 })
  @IsOptional()
  @Type(() => Number)
  @IsInt()
  @Min(1)
  @Max(100)
  pageSize?: number = 20;

  @ApiPropertyOptional({ enum: CourseGenerationStatus })
  @IsOptional()
  @IsEnum(CourseGenerationStatus)
  status?: CourseGenerationStatus;

  @ApiPropertyOptional({ enum: CourseLanguage })
  @IsOptional()
  @IsEnum(CourseLanguage)
  language?: CourseLanguage;

  @ApiPropertyOptional({
    description: 'Search term applied to course title and synopsis.',
  })
  @IsOptional()
  @IsString()
  search?: string;

  @ApiPropertyOptional({
    enum: ['createdAt', 'updatedAt', 'title', 'status'],
    default: 'createdAt',
  })
  @IsOptional()
  @IsIn(['createdAt', 'updatedAt', 'title', 'status'])
  orderBy?: 'createdAt' | 'updatedAt' | 'title' | 'status' = 'createdAt';

  @ApiPropertyOptional({ enum: ['asc', 'desc'], default: 'desc' })
  @IsOptional()
  @IsIn(['asc', 'desc'])
  orderDirection?: 'asc' | 'desc' = 'desc';
}
