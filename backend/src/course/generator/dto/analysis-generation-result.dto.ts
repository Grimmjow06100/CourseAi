import { Type } from 'class-transformer';
import { ValidateNested, IsUUID } from 'class-validator';
import { ApiProperty } from '@nestjs/swagger';
import { AnalysisResponseDto } from './analysis-response.dto';

export class AnalysisGenerationResultDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  requestId!: string;

  @ApiProperty({ type: () => AnalysisResponseDto })
  @ValidateNested()
  @Type(() => AnalysisResponseDto)
  analysis!: AnalysisResponseDto;
}
