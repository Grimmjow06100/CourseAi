import { Test, TestingModule } from '@nestjs/testing';
import { GeneratorService } from './generator.service';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';
import { ConfigService } from '@nestjs/config';

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
        {
          provide: ConfigService,
          useValue: {
            get: jest.fn((key: string) => {
              if (key === 'AI_MODEL') return 'gpt-5.4';
              if (key === 'OPENAI_MAX_RETRIES') return 2;
              return undefined;
            }),
          },
        },
      ],
    }).compile();

    service = module.get<GeneratorService>(GeneratorService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
