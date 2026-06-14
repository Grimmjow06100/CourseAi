import { ApiProperty } from '@nestjs/swagger';

export class GenerationResultResponseDto {
  @ApiProperty({ format: 'uuid' })
  requestId!: string;

  @ApiProperty({ format: 'uuid' })
  courseId!: string;

  @ApiProperty({ type: Object })
  course!: Record<string, unknown>;
}
