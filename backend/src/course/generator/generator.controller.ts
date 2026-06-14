import {
  Body,
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Post,
} from '@nestjs/common';
import { UserPromptDto } from './dto/user-prompt.dto';
import { GeneratorService } from './generator.service';
import { CourseContextDto } from './dto/course-context.dto';
import { LessonResponseDto } from './dto/lesson-response.dto';
import { LessonContextDto } from './dto/lesson-context.dto';
import { AnalysisGenerationResultDto } from './dto/analysis-generation-result.dto';
import { ApiOperation, ApiParam, ApiResponse, ApiTags } from '@nestjs/swagger';
import { ArchitectureGenerationResultDto } from './dto/architecture-generation-result.dto';
import { LessonContentContextDto } from './dto/lesson-content-context.dto';
import { LessonContentResponseDto } from './dto/lesson-content-response.dto';
import { FullCourseGenerationDto } from './dto/full-course-generation.dto';
import { GenerationStartResponseDto } from './dto/generation-start-response.dto';
import { GenerationStatusResponseDto } from './dto/generation-status-response.dto';
import { GenerationResultResponseDto } from './dto/generation-result-response.dto';
import { Throttle } from '@nestjs/throttler';

@ApiTags('generation')
@Controller('generator')
export class GeneratorController {
  constructor(private readonly generatorService: GeneratorService) {}

  @Post('/analysis')
  @Throttle({ default: { limit: 5, ttl: 60000 } })
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
    return await this.generatorService.promptAnalysis(userPrompt.prompt);
  }

  @Post('/architecture')
  @Throttle({ default: { limit: 5, ttl: 60000 } })
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
    return await this.generatorService.promptArchitecture(courseContext);
  }

  @Post('/lesson')
  @Throttle({ default: { limit: 5, ttl: 60000 } })
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
    return await this.generatorService.promptLessons(lessonContext);
  }

  @Post('/lesson-content')
  @Throttle({ default: { limit: 5, ttl: 60000 } })
  @ApiOperation({
    summary: 'Generate and persist the full Markdown content for one lesson',
  })
  @ApiResponse({
    status: 201,
    description: 'Lesson content generated and persisted.',
    type: LessonContentResponseDto,
  })
  public async courseLessonContent(
    @Body() lessonContentContext: LessonContentContextDto,
  ): Promise<LessonContentResponseDto> {
    return await this.generatorService.promptLessonContent(
      lessonContentContext,
    );
  }

  @Post('/full-course')
  @HttpCode(HttpStatus.ACCEPTED)
  @Throttle({ default: { limit: 2, ttl: 60000 } })
  @ApiOperation({
    summary: 'Start the full asynchronous course generation pipeline',
  })
  @ApiResponse({
    status: 202,
    description: 'Full course generation accepted.',
    type: GenerationStartResponseDto,
  })
  public async fullCourseGeneration(
    @Body() payload: FullCourseGenerationDto,
  ): Promise<GenerationStartResponseDto> {
    return await this.generatorService.startFullCourseGeneration(payload);
  }

  @Get('/requests/:requestId/status')
  @Throttle({ default: { limit: 60, ttl: 60000 } })
  @ApiOperation({ summary: 'Get generation request status' })
  @ApiParam({ name: 'requestId', format: 'uuid' })
  @ApiResponse({
    status: 200,
    description: 'Generation status returned.',
    type: GenerationStatusResponseDto,
  })
  public async generationStatus(
    @Param('requestId') requestId: string,
  ): Promise<GenerationStatusResponseDto> {
    return await this.generatorService.getGenerationStatus(requestId);
  }

  @Get('/requests/:requestId/result')
  @Throttle({ default: { limit: 60, ttl: 60000 } })
  @ApiOperation({ summary: 'Get generated course result' })
  @ApiParam({ name: 'requestId', format: 'uuid' })
  @ApiResponse({
    status: 200,
    description: 'Generation result returned.',
    type: GenerationResultResponseDto,
  })
  public async generationResult(
    @Param('requestId') requestId: string,
  ): Promise<GenerationResultResponseDto> {
    return await this.generatorService.getGenerationResult(requestId);
  }

  @Post('/requests/:requestId/retry')
  @HttpCode(HttpStatus.ACCEPTED)
  @Throttle({ default: { limit: 2, ttl: 60000 } })
  @ApiOperation({ summary: 'Retry a full generation request' })
  @ApiParam({ name: 'requestId', format: 'uuid' })
  @ApiResponse({
    status: 202,
    description: 'Generation retry accepted.',
    type: GenerationStartResponseDto,
  })
  public async retryGeneration(
    @Param('requestId') requestId: string,
  ): Promise<GenerationStartResponseDto> {
    return await this.generatorService.retryFullCourseGeneration(requestId);
  }
}
