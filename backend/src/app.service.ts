import { Injectable } from '@nestjs/common';
@Injectable()
export class AppService {
  getHello(): string {
    return 'Bienvenue sur la HelloAi App !!!';
  }
}
