import { Module } from '@nestjs/common';
import { GeneratorService } from './generator.service';
import { GeneratorController } from './generator.controller';
import { GeneratorParserService } from './generator-parser/generator-parser.service';
import { OpenAiService } from '../../providers/open-ai.provider';
import { GeneratorPersistenceService } from './generator-persistence/generator-persistence.service';

@Module({
  controllers: [GeneratorController],
  providers: [
    GeneratorService,
    OpenAiService,
    GeneratorParserService,
    GeneratorPersistenceService,
  ],
})
export class GeneratorModule {}
