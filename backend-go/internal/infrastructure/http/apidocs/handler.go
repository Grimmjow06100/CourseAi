package apidocs

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openAPISpec []byte

const swaggerUI = `<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Course AI API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/docs/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        displayRequestDuration: true,
        persistAuthorization: true,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>`

// Register exposes the interactive documentation and its OpenAPI specification.
func Register(router gin.IRouter) {
	router.GET("/docs", serveSwaggerUI)
	router.GET("/docs/", serveSwaggerUI)
	router.GET("/docs/openapi.json", serveOpenAPISpec)
}

// Specification returns an isolated copy of the embedded OpenAPI document.
func Specification() []byte {
	return append([]byte(nil), openAPISpec...)
}

func serveSwaggerUI(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUI))
}

func serveOpenAPISpec(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/vnd.oai.openapi+json;version=3.0", openAPISpec)
}
