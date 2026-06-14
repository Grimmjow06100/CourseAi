import { UsersService } from './users.service';
import { ApiTags } from '@nestjs/swagger';
// import { Get, Post, Body, Patch, Param, Delete } from '@nestjs/common';
// import { CreateUserDto } from './dto/create-user.dto';
// import { UpdateUserDto } from './dto/update-user.dto';
// import { Prisma, User } from '../generated/prisma/client';
import { Controller } from '@nestjs/common';
@ApiTags('users')
@Controller('users')
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  // @Post()
  // create(@Body() createUserDto: CreateUserDto): Promise<User> {
  //   return this.usersService.createUser({
  //     username: createUserDto.username,
  //     password: createUserDto.password,
  //   });
  // }

  // @Get()
  // findAll(): Promise<User[]> {
  //   return this.usersService.users({});
  // }

  // @Get(':id')
  // findOne(@Param('id') id: string): Promise<User | null> {
  //   return this.usersService.user({ id });
  // }

  // @Patch(':id')
  // update(
  //   @Param('id') id: string,
  //   @Body() updateUserDto: UpdateUserDto,
  // ): Promise<User> {
  //   const data: Prisma.UserUpdateInput = {};

  //   if (updateUserDto.username !== undefined) {
  //     data.username = updateUserDto.username;
  //   }

  //   if (updateUserDto.password !== undefined) {
  //     data.password = updateUserDto.password;
  //   }

  //   return this.usersService.updateUser({
  //     where: { id },
  //     data,
  //   });
  // }

  // @Delete(':id')
  // remove(@Param('id') id: string): Promise<User> {
  //   return this.usersService.deleteUser({ id });
  // }
}
