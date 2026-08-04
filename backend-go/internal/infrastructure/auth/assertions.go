package auth

import "github.com/Grimmjow06100/course-ai/backend-go/internal/contract"

var _ contract.PasswordManager = (*PasswordManager)(nil)
var _ contract.TokenManager = (*TokenManager)(nil)