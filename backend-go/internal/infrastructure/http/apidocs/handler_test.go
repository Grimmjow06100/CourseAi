package apidocs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterServesOpenAPISpecificationAndSwaggerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register(router)

	specResponse := httptest.NewRecorder()
	router.ServeHTTP(specResponse, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if specResponse.Code != http.StatusOK {
		t.Fatalf("specification status = %d, want %d", specResponse.Code, http.StatusOK)
	}
	if !json.Valid(specResponse.Body.Bytes()) {
		t.Fatal("embedded OpenAPI specification is not valid JSON")
	}
	if contentType := specResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "application/vnd.oai.openapi+json") {
		t.Fatalf("specification Content-Type = %q", contentType)
	}

	uiResponse := httptest.NewRecorder()
	router.ServeHTTP(uiResponse, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if uiResponse.Code != http.StatusOK {
		t.Fatalf("Swagger UI status = %d, want %d", uiResponse.Code, http.StatusOK)
	}
	if !strings.Contains(uiResponse.Body.String(), "/docs/openapi.json") {
		t.Fatal("Swagger UI does not reference the embedded specification")
	}
}

func TestSpecificationReturnsCopy(t *testing.T) {
	first := Specification()
	first[0] = 'x'
	second := Specification()
	if second[0] == 'x' {
		t.Fatal("Specification exposes mutable embedded storage")
	}
}

func TestSpecificationLocalReferencesResolve(t *testing.T) {
	var document any
	if err := json.Unmarshal(Specification(), &document); err != nil {
		t.Fatalf("decode OpenAPI specification: %v", err)
	}

	var walk func(any)
	walk = func(value any) {
		switch current := value.(type) {
		case map[string]any:
			if reference, ok := current["$ref"].(string); ok {
				if _, found := resolveLocalReference(document, reference); !found {
					t.Errorf("unresolved OpenAPI reference %q", reference)
				}
			}
			for _, nested := range current {
				walk(nested)
			}
		case []any:
			for _, nested := range current {
				walk(nested)
			}
		}
	}
	walk(document)
}

func TestGenerationCommandsDocumentAdmissionLimits(t *testing.T) {
	var document struct {
		Paths map[string]map[string]struct {
			RequestBody json.RawMessage            `json:"requestBody"`
			Responses   map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(Specification(), &document); err != nil {
		t.Fatalf("decode OpenAPI specification: %v", err)
	}

	for path, operations := range document.Paths {
		operation, isGenerationCommand := operations["post"]
		if !isGenerationCommand || !strings.HasPrefix(path, "/api/generations") {
			continue
		}
		if _, documented := operation.Responses["429"]; !documented {
			t.Errorf("POST %s does not document rate limiting", path)
		}
		if len(operation.RequestBody) > 0 {
			if _, documented := operation.Responses["413"]; !documented {
				t.Errorf("POST %s does not document the body size limit", path)
			}
		}
	}
}

func resolveLocalReference(document any, reference string) (any, bool) {
	if !strings.HasPrefix(reference, "#/") {
		return nil, false
	}
	current := document
	for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
