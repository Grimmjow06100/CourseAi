package service

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

func authenticatedTestContext() context.Context {
	return contract.ContextWithPrincipal(context.Background(), contract.Principal{UserID: "user_test", SessionID: "sess_test"})
}
