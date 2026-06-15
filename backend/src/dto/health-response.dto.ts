import { ApiProperty } from '@nestjs/swagger';

export class HealthResponseDto {
  @ApiProperty({ example: 'ok' })
  status!: string;

  @ApiProperty({ example: 42.15 })
  uptimeSeconds!: number;

  @ApiProperty({ example: '2026-06-15T12:00:00.000Z' })
  timestamp!: string;
}
