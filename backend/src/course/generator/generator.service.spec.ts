import { Test, TestingModule } from '@nestjs/testing';
import { GeneratorService } from './generator.service';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';

describe('GeneratorService', () => {
  let service: GeneratorService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
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

    service = module.get<GeneratorService>(GeneratorService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
