import { Type } from 'class-transformer';
import { IsUUID, ValidateNested } from 'class-validator';
import { ApiProperty } from '@nestjs/swagger';
import { ArchitectureResponseDto } from './architecture-response.dto';

export class ArchitectureGenerationResultDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  courseId!: string;

  @ApiProperty({ type: () => ArchitectureResponseDto })
  @ValidateNested()
  @Type(() => ArchitectureResponseDto)
  architecture!: ArchitectureResponseDto;
}
