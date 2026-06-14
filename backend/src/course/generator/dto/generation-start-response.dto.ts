import { ApiProperty } from '@nestjs/swagger';
import { GenerationPipelineStatus } from '../../../generated/prisma/client';

export class GenerationStartResponseDto {
  @ApiProperty({ format: 'uuid' })
  requestId!: string;

  @ApiProperty({ enum: GenerationPipelineStatus })
  status!: GenerationPipelineStatus;

  @ApiProperty()
  statusUrl!: string;

  @ApiProperty()
  resultUrl!: string;
}
