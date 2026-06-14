import { SignInUserDto } from './../users/dto/signIn-user.dto';
import { Injectable } from '@nestjs/common';
import { UsersService } from '../users/users.service';
import { AuthError } from './auth.erros';
import { ConfigService } from '@nestjs/config';
import * as bcrypt from 'bcrypt';
import { JwtService } from '@nestjs/jwt';
import { CreateUserDto } from '../users/dto/create-user.dto';
import { AuthTokenResponse } from '../interfaces/auth-token-response.interface';

@Injectable()
export class AuthService {
  constructor(
    private readonly usersService: UsersService,
    private readonly config: ConfigService,
    private readonly jwtService: JwtService,
  ) {}

  async signIn(signInUserDto: SignInUserDto): Promise<AuthTokenResponse> {
    try {
      const user = await this.usersService.user({
        username: signInUserDto.username,
      });

      if (!user) {
        throw new AuthError('User not found');
      }

      const isPasswordValid = await bcrypt.compare(
        signInUserDto.password,
        user.password,
      );
      if (!isPasswordValid) {
        throw new AuthError('Invalid password');
      }

      return {
        accessToken: await this.generateJwtToken(user.id, user.username),
      };
    } catch (error) {
      console.error('Error during sign-in:', error);
      if (error instanceof AuthError) {
        throw error;
      } else {
        throw new AuthError('An error occurred during sign-in');
      }
    }
  }
  async signUp(createUser: CreateUserDto): Promise<AuthTokenResponse> {
    try {
      const existingUser = await this.usersService.user({
        username: createUser.username,
      });
      if (existingUser) {
        throw new AuthError('user already exists');
      }
      const salt = this.config.get<string>('PASSWORD_HASH_SALT');
      if (!salt) {
        throw new AuthError('Password hash salt is not configured');
      }
      const hashedPassword = await bcrypt.hash(createUser.password, salt);

      const user = await this.usersService.createUser({
        username: createUser.username,
        password: hashedPassword,
      });
      return {
        accessToken: await this.generateJwtToken(user.id, user.username),
      };
    } catch (error) {
      console.error('Error during sign-up:', error);
      if (error instanceof AuthError) {
        throw error;
      }
      throw new AuthError('An error occurred during sign-up');
    }
  }

  /**
   * Generates a signed access token for an authenticated user.
   *
   * @param userId - UUID of the authenticated user.
   * @param username - Username embedded in the token payload.
   * @returns A signed JWT access token.
   */
  private async generateJwtToken(
    userId: string,
    username: string,
  ): Promise<string> {
    const payload = {
      sub: userId,
      username: username,
    };
    return await this.jwtService.signAsync(payload);
  }
}
