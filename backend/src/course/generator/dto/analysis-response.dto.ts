import { Type } from 'class-transformer';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import {
  IsArray,
  IsBoolean,
  IsEnum,
  IsIn,
  IsOptional,
  IsString,
  ValidateNested,
} from 'class-validator';

export enum AnalysisLevel {
  Beginner = 'beginner',
  Intermediate = 'intermediate',
  Advanced = 'advanced',
  Expert = 'expert',
  Unknown = 'unknown',
}

export enum AnalysisLanguage {
  French = 'fr',
  English = 'en',
}

export enum ClarificationQuestionId {
  Goals = 'goals',
  CurrentLevel = 'currentLevel',
  TargetLevel = 'targetLevel',
}

export class ClarificationQuestionDto {
  @ApiProperty({ enum: ClarificationQuestionId })
  @IsEnum(ClarificationQuestionId)
  id!: ClarificationQuestionId;

  @ApiProperty()
  @IsString()
  question!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @IsString({ each: true })
  options!: string[];
}

export class AnalysisResponseDto {
  @ApiProperty()
  @IsBoolean()
  isOutOfScope!: boolean;

  @ApiPropertyOptional({ nullable: true })
  @IsOptional()
  @IsString()
  errorMessage!: string | null;

  @ApiPropertyOptional({ nullable: true })
  @IsOptional()
  @IsString()
  warningMessage!: string | null;

  @ApiProperty()
  @IsString()
  suggestedTitle!: string;

  @ApiProperty()
  @IsString()
  shortSynopsis!: string;

  @ApiProperty({
    enum: [
      AnalysisLevel.Beginner,
      AnalysisLevel.Intermediate,
      AnalysisLevel.Advanced,
      AnalysisLevel.Unknown,
    ],
  })
  @IsIn([
    AnalysisLevel.Beginner,
    AnalysisLevel.Intermediate,
    AnalysisLevel.Advanced,
    AnalysisLevel.Unknown,
  ])
  detectedCurrentLevel!: Exclude<AnalysisLevel, AnalysisLevel.Expert>;

  @ApiProperty({ enum: AnalysisLevel })
  @IsEnum(AnalysisLevel)
  detectedTargetLevel!: AnalysisLevel;

  @ApiProperty()
  @IsString()
  detectedGoal!: string;

  @ApiProperty({ enum: AnalysisLanguage })
  @IsEnum(AnalysisLanguage)
  detectedLanguage!: AnalysisLanguage;

  @ApiProperty({ type: () => [ClarificationQuestionDto] })
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => ClarificationQuestionDto)
  clarificationQuestions!: ClarificationQuestionDto[];
}
