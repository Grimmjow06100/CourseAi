import { Type } from 'class-transformer';
import { ApiProperty } from '@nestjs/swagger';
import {
  ArrayMinSize,
  IsArray,
  IsInt,
  IsString,
  Min,
  ValidateNested,
} from 'class-validator';

export class ArchitectureModuleDto {
  @ApiProperty()
  @IsInt()
  @Min(1)
  order!: number;

  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  description!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  keyLearningPoints!: string[];
}

export class ArchitectureFinalProjectDto {
  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  description!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  constraints!: string[];
}

export class ArchitectureResponseDto {
  @ApiProperty()
  @IsString()
  title!: string;

  @ApiProperty()
  @IsString()
  synopsis!: string;

  @ApiProperty()
  @IsString()
  targetAudience!: string;

  @ApiProperty({ type: [String] })
  @IsArray()
  @IsString({ each: true })
  prerequisites!: string[];

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  goals!: string[];

  @ApiProperty({ type: [String] })
  @IsArray()
  @ArrayMinSize(1)
  @IsString({ each: true })
  acquiredSkills!: string[];

  @ApiProperty({ type: () => [ArchitectureModuleDto] })
  @IsArray()
  @ArrayMinSize(1)
  @ValidateNested({ each: true })
  @Type(() => ArchitectureModuleDto)
  modules!: ArchitectureModuleDto[];

  @ApiProperty({ type: () => ArchitectureFinalProjectDto })
  @ValidateNested()
  @Type(() => ArchitectureFinalProjectDto)
  finalProject!: ArchitectureFinalProjectDto;
}
