package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/apidocs"
	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersHealthAndAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{AllowedOrigins: []string{"https://app.example.com"}, WorkerEnabled: true})

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK || health.Body.String() != `{"status":"ok"}` {
		t.Fatalf("health response = %d %s", health.Code, health.Body.String())
	}

	api := httptest.NewRecorder()
	router.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/courses", nil))
	if api.Code != http.StatusUnauthorized {
		t.Fatalf("protected API route status = %d, want 401", api.Code)
	}

	for _, path := range []string{"/api/auth/signup", "/api/auth/login"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("legacy auth route %s status = %d, want 404", path, response.Code)
		}
	}

	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if docs.Code != http.StatusOK {
		t.Fatalf("documentation route status = %d, want 200", docs.Code)
	}
}

func TestOpenAPIDocumentCoversEveryPublicRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{})

	type operation struct {
		OperationID string                     `json:"operationId"`
		Summary     string                     `json:"summary"`
		Tags        []string                   `json:"tags"`
		Responses   map[string]json.RawMessage `json:"responses"`
		Security    json.RawMessage            `json:"security"`
	}
	type document struct {
		OpenAPI    string                                `json:"openapi"`
		Paths      map[string]map[string]json.RawMessage `json:"paths"`
		Security   []map[string][]string                 `json:"security"`
		Components struct {
			SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
		} `json:"components"`
	}

	var spec document
	if err := json.Unmarshal(apidocs.Specification(), &spec); err != nil {
		t.Fatalf("decode OpenAPI specification: %v", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		t.Fatalf("OpenAPI version = %q, want version 3", spec.OpenAPI)
	}
	if len(spec.Security) != 1 || spec.Security[0]["ClerkBearer"] == nil {
		t.Fatal("OpenAPI must require ClerkBearer globally")
	}
	if _, ok := spec.Components.SecuritySchemes["ClerkBearer"]; !ok {
		t.Fatal("OpenAPI is missing the ClerkBearer security scheme")
	}

	routes := make(map[string]struct{})
	operationIDs := make(map[string]string)
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/docs") {
			continue
		}
		path := openAPIPath(route.Path)
		method := strings.ToLower(route.Method)
		operationJSON, exists := spec.Paths[path][method]
		if !exists {
			t.Errorf("route %s %s is missing from OpenAPI", route.Method, route.Path)
			continue
		}
		routes[method+" "+path] = struct{}{}

		var documented operation
		if err := json.Unmarshal(operationJSON, &documented); err != nil {
			t.Errorf("decode operation %s %s: %v", method, path, err)
			continue
		}
		if documented.OperationID == "" || documented.Summary == "" || len(documented.Tags) == 0 || len(documented.Responses) == 0 {
			t.Errorf("operation %s %s is missing operationId, summary, tags, or responses", method, path)
		}
		if route.Path == "/health" && string(documented.Security) != "[]" {
			t.Errorf("public technical route %s must override global Clerk security", route.Path)
		}
		if previous, duplicate := operationIDs[documented.OperationID]; duplicate {
			t.Errorf("operationId %q is shared by %s and %s %s", documented.OperationID, previous, method, path)
		} else {
			operationIDs[documented.OperationID] = method + " " + path
		}
	}

	for path, pathItem := range spec.Paths {
		for method := range pathItem {
			if _, exists := routes[strings.ToLower(method)+" "+path]; !exists {
				t.Errorf("OpenAPI operation %s %s has no matching Gin route", strings.ToUpper(method), path)
			}
		}
	}
}

func TestProductionRouterDoesNotExposeDocumentation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{AppEnv: "production"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production docs status = %d, want 404", response.Code)
	}
}

func TestEveryBusinessRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{AllowedOrigins: []string{"http://localhost:5173"}})
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		path := strings.ReplaceAll(route.Path, ":requestID", "00000000-0000-0000-0000-000000000001")
		path = strings.ReplaceAll(path, ":courseID", "00000000-0000-0000-0000-000000000001")
		path = strings.ReplaceAll(path, ":moduleID", "00000000-0000-0000-0000-000000000001")
		path = strings.ReplaceAll(path, ":lessonID", "00000000-0000-0000-0000-000000000001")
		path = strings.ReplaceAll(path, ":jobID", "00000000-0000-0000-0000-000000000001")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(route.Method, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", route.Method, route.Path, response.Code)
		}
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/courses", nil)
	preflight.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", response.Code)
	}
}

func openAPIPath(ginPath string) string {
	segments := strings.Split(ginPath, "/")
	for index, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[index] = "{" + strings.TrimPrefix(segment, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}
