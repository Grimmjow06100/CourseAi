import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common';
import { CourseService } from './course.service';
import { CreateCourseDto } from './dto/create-course.dto';
import { UpdateCourseDto } from './dto/update-course.dto';
import {
  ApiOperation,
  ApiParam,
  ApiResponse,
  ApiTags,
} from '@nestjs/swagger';
import { UpdateModuleDto } from './dto/update-module.dto';
import { UpdateLessonDto } from './dto/update-lesson.dto';

@ApiTags('courses')
@Controller('course')
export class CourseController {
  constructor(private readonly courseService: CourseService) {}

  @Post()
  @ApiOperation({ summary: 'Create a course manually' })
  @ApiResponse({ status: 201, description: 'Course created.' })
  create(@Body() createCourseDto: CreateCourseDto) {
    return this.courseService.create(createCourseDto);
  }

  @Get()
  @ApiOperation({ summary: 'List courses with modules and lessons' })
  @ApiResponse({ status: 200, description: 'Courses returned.' })
  findAll() {
    return this.courseService.findAll();
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get one course by id' })
  @ApiParam({ name: 'id', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Course returned.' })
  @ApiResponse({ status: 404, description: 'Course not found.' })
  findOne(@Param('id') id: string) {
    return this.courseService.findOne(id);
  }

  @Patch(':id')
  @ApiOperation({ summary: 'Update one course by id' })
  @ApiParam({ name: 'id', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Course updated.' })
  @ApiResponse({ status: 404, description: 'Course not found.' })
  update(@Param('id') id: string, @Body() updateCourseDto: UpdateCourseDto) {
    return this.courseService.update(id, updateCourseDto);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete one course by id' })
  @ApiParam({ name: 'id', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Course deleted.' })
  @ApiResponse({ status: 404, description: 'Course not found.' })
  remove(@Param('id') id: string) {
    return this.courseService.remove(id);
  }

  @Get(':id/modules')
  @ApiOperation({ summary: 'List modules for a course' })
  @ApiParam({ name: 'id', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Modules returned.' })
  findModules(@Param('id') id: string) {
    return this.courseService.findModules(id);
  }

  @Get('modules/:moduleId')
  @ApiOperation({ summary: 'Get one module with lessons' })
  @ApiParam({ name: 'moduleId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Module returned.' })
  findModule(@Param('moduleId') moduleId: string) {
    return this.courseService.findModule(moduleId);
  }

  @Patch('modules/:moduleId')
  @ApiOperation({ summary: 'Update one module' })
  @ApiParam({ name: 'moduleId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Module updated.' })
  updateModule(
    @Param('moduleId') moduleId: string,
    @Body() updateModuleDto: UpdateModuleDto,
  ) {
    return this.courseService.updateModule(moduleId, updateModuleDto);
  }

  @Delete('modules/:moduleId')
  @ApiOperation({ summary: 'Delete one module' })
  @ApiParam({ name: 'moduleId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Module deleted.' })
  removeModule(@Param('moduleId') moduleId: string) {
    return this.courseService.removeModule(moduleId);
  }

  @Get('modules/:moduleId/lessons')
  @ApiOperation({ summary: 'List lessons for a module' })
  @ApiParam({ name: 'moduleId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Lessons returned.' })
  findLessons(@Param('moduleId') moduleId: string) {
    return this.courseService.findLessons(moduleId);
  }

  @Get('lessons/:lessonId')
  @ApiOperation({ summary: 'Get one lesson' })
  @ApiParam({ name: 'lessonId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Lesson returned.' })
  findLesson(@Param('lessonId') lessonId: string) {
    return this.courseService.findLesson(lessonId);
  }

  @Patch('lessons/:lessonId')
  @ApiOperation({ summary: 'Update one lesson' })
  @ApiParam({ name: 'lessonId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Lesson updated.' })
  updateLesson(
    @Param('lessonId') lessonId: string,
    @Body() updateLessonDto: UpdateLessonDto,
  ) {
    return this.courseService.updateLesson(lessonId, updateLessonDto);
  }

  @Delete('lessons/:lessonId')
  @ApiOperation({ summary: 'Delete one lesson' })
  @ApiParam({ name: 'lessonId', format: 'uuid' })
  @ApiResponse({ status: 200, description: 'Lesson deleted.' })
  removeLesson(@Param('lessonId') lessonId: string) {
    return this.courseService.removeLesson(lessonId);
  }
}
