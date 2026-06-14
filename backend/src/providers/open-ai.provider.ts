import { Provider } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import OpenAI from 'openai';

export const OpenAiService: Provider = {
  provide: 'OPEN_AI_SERVICE',
  useFactory: (configService: ConfigService) => {
    const token = configService.get<string>('OPENAI_API_KEY');
    return new OpenAI({
      apiKey: token,
    });
  },
  inject: [ConfigService],
};
