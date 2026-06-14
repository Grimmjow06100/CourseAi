import { Test, TestingModule } from '@nestjs/testing';
import { CourseController } from './course.controller';
import { CourseService } from './course.service';
import { PrismaService } from '../prisma/prisma.service';

describe('CourseController', () => {
  let controller: CourseController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [CourseController],
      providers: [
        CourseService,
        {
          provide: PrismaService,
          useValue: {},
        },
      ],
    }).compile();

    controller = module.get<CourseController>(CourseController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
