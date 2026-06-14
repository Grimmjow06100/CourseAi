import { ApiProperty } from '@nestjs/swagger';
import { ArrayMinSize, IsArray, IsString, IsUUID } from 'class-validator';

export class LessonContentResponseDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  lessonId!: string;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  contentMarkdown!: string;

  @ApiProperty()
  @IsString()
  summary!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  keyTakeaways!: string[];
}
