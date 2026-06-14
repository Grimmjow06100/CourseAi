export class AuthError extends Error {
  private readonly error?: unknown;
  constructor(message: string, error?: unknown) {
    super(message);
    this.name = 'AuthError';
    this.error = error;
  }
}
