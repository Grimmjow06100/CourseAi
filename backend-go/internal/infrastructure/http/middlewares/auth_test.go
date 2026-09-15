package middlewares

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
	jose "github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
)

func TestClerkAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}))

	tests := []struct {
		name       string
		claims     map[string]any
		withToken  bool
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "valid", withToken: true, claims: clerkTestClaims("user_123456", "http://localhost:5173"), wantStatus: http.StatusNoContent},
		{name: "wrong authorized party", withToken: true, claims: clerkTestClaims("user_123456", "https://evil.example"), wantStatus: http.StatusUnauthorized},
		{name: "missing subject", withToken: true, claims: clerkTestClaims("", "http://localhost:5173"), wantStatus: http.StatusUnauthorized},
		{name: "non-user subject", withToken: true, claims: clerkTestClaims("machine_123456", "http://localhost:5173"), wantStatus: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ClerkAuthentication([]string{"http://localhost:5173"}, clerkhttp.JSONWebKey(publicPEM)))
			router.GET("/protected", func(c *gin.Context) {
				principal, ok := contract.PrincipalFromContext(c.Request.Context())
				if !ok || principal.UserID != "user_123456" || principal.SessionID != "sess_test" {
					c.Status(http.StatusInternalServerError)
					return
				}
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.withToken {
				request.Header.Set("Authorization", "Bearer "+signedClerkTestToken(t, privateKey, test.claims))
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func clerkTestClaims(subject string, authorizedParty string) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": "https://example.clerk.accounts.dev",
		"sub": subject,
		"sid": "sess_test",
		"azp": authorizedParty,
		"iat": now.Add(-time.Minute).Unix(),
		"nbf": now.Add(-time.Minute).Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
}

func TestClerkAuthenticationClockSkew(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}))
	now := time.Date(2026, time.September, 15, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		issuedAt   time.Duration
		expiresAt  time.Duration
		wantStatus int
	}{
		{name: "fresh token with local clock 13 seconds behind", issuedAt: 13 * time.Second, expiresAt: 73 * time.Second, wantStatus: http.StatusNoContent},
		{name: "local clock 13 seconds ahead of token expiry", issuedAt: -73 * time.Second, expiresAt: -13 * time.Second, wantStatus: http.StatusNoContent},
		{name: "future token beyond tolerance", issuedAt: 16 * time.Second, expiresAt: 76 * time.Second, wantStatus: http.StatusUnauthorized},
		{name: "expired token beyond tolerance", issuedAt: -76 * time.Second, expiresAt: -16 * time.Second, wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clock := fixedAuthenticationClock{now: now}
			router := gin.New()
			router.Use(ClerkAuthentication([]string{"http://localhost:5173"}, clerkhttp.JSONWebKey(publicPEM), clerkhttp.Clock(clock)))
			router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			claims := clerkTestClaims("user_123456", "http://localhost:5173")
			claims["iat"] = now.Add(test.issuedAt).Unix()
			claims["nbf"] = now.Add(test.issuedAt).Unix()
			claims["exp"] = now.Add(test.expiresAt).Unix()
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+signedClerkTestToken(t, privateKey, claims))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

type fixedAuthenticationClock struct {
	now time.Time
}

func (c fixedAuthenticationClock) Now() time.Time { return c.now }

func signedClerkTestToken(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := josejwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
