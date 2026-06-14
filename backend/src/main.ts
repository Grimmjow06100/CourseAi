import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { ValidationPipe } from '@nestjs/common';
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  app.enableCors();
  app.enableShutdownHooks();
  app.useGlobalPipes(
    new ValidationPipe({
      transform: true,
      whitelist: true,
    }),
  );

  const swaggerConfig = new DocumentBuilder()
    .setTitle('Course AI API')
    .setDescription('API for generating and persisting IT course structures.')
    .setVersion('0.1.0')
    .build();
  const swaggerDocument = SwaggerModule.createDocument(app, swaggerConfig);
  SwaggerModule.setup('docs', app, swaggerDocument);

  await app.listen(process.env.PORT ?? 3000);
}
bootstrap()
  .then(() => {
    console.log(`Application is running on: ${process.env.PORT ?? 3000}`);
  })
  .catch((error) => {
    console.error('Error starting the application:', error);
    process.exit(1);
  });
