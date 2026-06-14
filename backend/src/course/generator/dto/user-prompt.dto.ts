import { ApiProperty } from '@nestjs/swagger';
import { IsString } from 'class-validator';

export class UserPromptDto {
  @ApiProperty({
    example: 'Je veux apprendre Docker pour déployer mes applications backend.',
  })
  @IsString()
  prompt!: string;
}
