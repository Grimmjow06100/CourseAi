import { ApiProperty } from '@nestjs/swagger';
import { IsArray, IsString, IsUUID } from 'class-validator';

export class CourseContextDto {
  @ApiProperty({ format: 'uuid' })
  @IsUUID()
  requestId!: string;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  synopsis!: string;

  @ApiProperty()
  @IsString()
  currentLevel!: string;

  @ApiProperty()
  @IsString()
  targetLevel!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @IsString({ each: true })
  goals!: string[];

  @ApiProperty({ enum: ['fr', 'en'] })
  @IsString()
  language!: string;
}
