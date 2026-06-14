import { ApiProperty } from '@nestjs/swagger';
import { IsString, MinLength } from 'class-validator';

export class FullCourseGenerationDto {
  @ApiProperty({
    description: 'Raw user request used to generate the full course pipeline.',
    example: 'Je veux apprendre Docker pour déployer une API Node.js.',
  })
  @IsString()
  @MinLength(10)
  prompt!: string;
}
