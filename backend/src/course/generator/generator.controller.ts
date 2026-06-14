import { Body, Controller, Post } from '@nestjs/common';
import { UserPromptDto } from './dto/user-prompt.dto';
import { GeneratorService } from './generator.service';
import { CourseContextDto } from './dto/course-context.dto';
import { LessonResponseDto } from './dto/lesson-response.dto';
import { LessonContextDto } from './dto/lesson-context.dto';
import { AnalysisGenerationResultDto } from './dto/analysis-generation-result.dto';
import { ApiOperation, ApiResponse, ApiTags } from '@nestjs/swagger';
import { ArchitectureGenerationResultDto } from './dto/architecture-generation-result.dto';

@ApiTags('generation')
@Controller('generator')
export class GeneratorController {
  constructor(private readonly generatorService: GeneratorService) {}

  @Post('/analysis')
  @ApiOperation({
    summary: 'Analyze a raw user prompt and persist the generation request',
  })
  @ApiResponse({
    status: 201,
    description: 'Needs analysis generated and persisted.',
    type: AnalysisGenerationResultDto,
  })
  public async userInputAnalysis(
    @Body() userPrompt: UserPromptDto,
  ): Promise<AnalysisGenerationResultDto> {
    try {
      const analysis = await this.generatorService.promptAnalysis(
        userPrompt.prompt,
      );
      return analysis;
    } catch (error) {
      console.error(error);
      throw error;
    }
  }

  @Post('/architecture')
  @ApiOperation({
    summary: 'Generate and persist the course architecture',
  })
  @ApiResponse({
    status: 201,
    description: 'Course architecture generated and persisted.',
    type: ArchitectureGenerationResultDto,
  })
  public async courseArchitecture(
    @Body() courseContext: CourseContextDto,
  ): Promise<ArchitectureGenerationResultDto> {
    try {
      const architecture =
        await this.generatorService.promptArchitecture(courseContext);
      return architecture;
    } catch (error) {
      console.error(error);
      throw error;
    }
  }

  @Post('/lesson')
  @ApiOperation({
    summary: 'Generate and persist the lesson plan for one module',
  })
  @ApiResponse({
    status: 201,
    description: 'Module lesson plan generated and persisted.',
    type: LessonResponseDto,
  })
  public async courseLessons(
    @Body() lessonContext: LessonContextDto,
  ): Promise<LessonResponseDto> {
    try {
      const lessons = await this.generatorService.promptLessons(lessonContext);
      return lessons;
    } catch (error) {
      console.error(error);
      throw error;
    }
  }
}
