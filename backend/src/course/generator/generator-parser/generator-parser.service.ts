import { BadRequestException, Injectable } from '@nestjs/common';
import { ArchitectureResponseDto } from '../dto/architecture-response.dto';
import { plainToInstance } from 'class-transformer';
import { validate } from 'class-validator';
import { AnalysisResponseDto } from '../dto/analysis-response.dto';
import { LessonResponseDto } from '../dto/lesson-response.dto';
import { LessonContentResponseDto } from '../dto/lesson-content-response.dto';

@Injectable()
export class GeneratorParserService {
  /**
   * Parses and validates the raw needs-analysis JSON returned by the model.
   *
   * @param jsonString - Raw JSON string returned by the model.
   * @returns A validated needs-analysis response.
   */
  public async parseAndValidateAnalysisResponse(
    jsonString: string,
  ): Promise<AnalysisResponseDto> {
    try {
      const rawObject: unknown = JSON.parse(jsonString);
      const instance = plainToInstance(AnalysisResponseDto, rawObject);
      const errors = await validate(instance);

      if (errors.length > 0) {
        const errorMessages = errors
          .map((err) => Object.values(err.constraints || {}))
          .flat();
        throw new BadRequestException({
          message: 'Validation échouée',
          errors: errorMessages,
        });
      }

      return instance;
    } catch (error) {
      console.error(error);
      if (error instanceof BadRequestException) throw error;
      throw new BadRequestException(
        `Le format de la chaîne n'est pas un JSON valide ou structure incorrecte `,
      );
    }
  }

  /**
   * Parses and validates the raw architecture JSON returned by the model.
   *
   * @param jsonString - Raw JSON string returned by the model.
   * @returns A validated course architecture response.
   */
  public async parseAndValidateArchitectureResponse(
    jsonString: string,
  ): Promise<ArchitectureResponseDto> {
    try {
      const rawObject: unknown = JSON.parse(jsonString);
      const instance = plainToInstance(ArchitectureResponseDto, rawObject);

      const errors = await validate(instance);

      if (errors.length > 0) {
        const errorMessages = errors
          .map((err) => Object.values(err.constraints || {}))
          .flat();
        throw new BadRequestException({
          message: 'Validation échouée',
          errors: errorMessages,
        });
      }

      return instance;
    } catch (error) {
      console.error(error);
      if (error instanceof BadRequestException) throw error;
      throw new BadRequestException(
        `Le format de la chaîne n'est pas un JSON valide ou structure incorrecte `,
      );
    }
  }

  /**
   * Parses and validates the raw lessons plan JSON returned by the model.
   *
   * @param jsonString - Raw JSON string returned by the model.
   * @returns A validated lessons plan response.
   */
  public async parseAndValidateLessonResponse(
    jsonString: string,
  ): Promise<LessonResponseDto> {
    try {
      const rawObject: unknown = JSON.parse(jsonString);
      const instance = plainToInstance(LessonResponseDto, rawObject);

      const errors = await validate(instance);

      if (errors.length > 0) {
        const errorMessages = errors
          .map((err) => Object.values(err.constraints || {}))
          .flat();
        throw new BadRequestException({
          message: 'Validation échouée',
          errors: errorMessages,
        });
      }

      return instance;
    } catch (error) {
      console.error(error);
      if (error instanceof BadRequestException) throw error;
      throw new BadRequestException(
        `Le format de la chaîne n'est pas un JSON valide ou structure incorrecte `,
      );
    }
  }

  /**
   * Parses and validates the raw lesson content JSON returned by the model.
   *
   * @param jsonString - Raw JSON string returned by the model.
   * @returns A validated lesson content response.
   */
  public async parseAndValidateLessonContentResponse(
    jsonString: string,
  ): Promise<LessonContentResponseDto> {
    try {
      const rawObject: unknown = JSON.parse(jsonString);
      const instance = plainToInstance(LessonContentResponseDto, rawObject);

      const errors = await validate(instance);

      if (errors.length > 0) {
        const errorMessages = errors
          .map((err) => Object.values(err.constraints || {}))
          .flat();
        throw new BadRequestException({
          message: 'Validation échouée',
          errors: errorMessages,
        });
      }

      return instance;
    } catch (error) {
      console.error(error);
      if (error instanceof BadRequestException) throw error;
      throw new BadRequestException(
        `Le format de la chaîne n'est pas un JSON valide ou structure incorrecte `,
      );
    }
  }
}
