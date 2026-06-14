import { Module } from '@nestjs/common';
import { CourseService } from './course.service';
import { CourseController } from './course.controller';
import { GeneratorModule } from './generator/generator.module';
import { RouterModule } from '@nestjs/core';
@Module({
  controllers: [CourseController],
  providers: [CourseService],
  imports: [
    GeneratorModule,
    RouterModule.register([
      {
        path: 'course',
        module: GeneratorModule,
      },
    ]),
  ],
})
export class CourseModule {}
