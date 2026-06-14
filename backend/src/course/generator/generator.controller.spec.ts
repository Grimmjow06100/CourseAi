import { Test, TestingModule } from '@nestjs/testing';
import { GeneratorController } from './generator.controller';
import { GeneratorService } from './generator.service';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';

describe('GeneratorController', () => {
  let controller: GeneratorController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [GeneratorController],
      providers: [
        GeneratorService,
        {
          provide: 'OPEN_AI_SERVICE',
          useValue: {},
        },
        {
          provide: GeneratorParserService,
          useValue: {},
        },
        {
          provide: GeneratorPersistenceService,
          useValue: {},
        },
      ],
    }).compile();

    controller = module.get<GeneratorController>(GeneratorController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
