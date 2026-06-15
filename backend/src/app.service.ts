import { Injectable } from '@nestjs/common';
import { HealthResponseDto } from './dto/health-response.dto';

@Injectable()
export class AppService {
  getHello(): string {
    return 'Bienvenue sur la HelloAi App !!!';
  }

  getHealth(): HealthResponseDto {
    return {
      status: 'ok',
      uptimeSeconds: process.uptime(),
      timestamp: new Date().toISOString(),
    };
  }
}
