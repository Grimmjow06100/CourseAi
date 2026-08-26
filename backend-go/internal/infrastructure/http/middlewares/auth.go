package middlewares

import (
	"net/http"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
)

func ClerkAuthentication(authorizedParties []string, extraOptions ...clerkhttp.AuthorizationOption) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(authorizedParties))
	for _, party := range authorizedParties {
		if normalized := strings.TrimSpace(party); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	options := []clerkhttp.AuthorizationOption{
		clerkhttp.AuthorizedParty(func(party string) bool {
			_, ok := allowed[party]
			return party != "" && ok
		}),
		clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(writeClerkUnauthorized)),
	}
	options = append(options, extraOptions...)

	return func(c *gin.Context) {
		continued := false
		verified := clerkhttp.WithHeaderAuthorization(options...)(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			claims, ok := clerk.SessionClaimsFromContext(request.Context())
			if !ok || claims == nil {
				writeClerkUnauthorized(w, request)
				return
			}
			subject := strings.TrimSpace(claims.Subject)
			if !strings.HasPrefix(subject, "user_") {
				writeClerkUnauthorized(w, request)
				return
			}

			principal := contract.Principal{
				UserID:    subject,
				SessionID: strings.TrimSpace(claims.SessionID),
			}
			continued = true
			c.Request = request.WithContext(contract.ContextWithPrincipal(request.Context(), principal))
			c.Next()
		}))
		verified.ServeHTTP(c.Writer, c.Request)
		if !continued {
			c.Abort()
		}
	}
}

func RejectUnauthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		writeClerkUnauthorized(c.Writer, c.Request)
		c.Abort()
	}
}

func writeClerkUnauthorized(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusUnauthorized)
	_, _ = writer.Write([]byte(`{"code":"unauthenticated","message":"authentication is required"}`))
}
