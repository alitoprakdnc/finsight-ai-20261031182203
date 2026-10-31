package router

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed ../../pkg/openapi/openapi.yaml
var openAPISpec []byte

// RegisterDocsRoutes registers the Swagger UI and raw OpenAPI spec endpoints.
// It should be called with the API router group, e.g.:
//
//   api := router.Group("/api")
//   RegisterDocsRoutes(api)
//
func RegisterDocsRoutes(r *gin.RouterGroup) {
	docs := r.Group("/docs")
	{
		// Swagger UI landing page
		docs.GET("/", swaggerUIHandler)

		// Raw OpenAPI spec – served as YAML
		docs.GET("/openapi.yaml", openAPISpecHandler)
	}
}

// swaggerUIHandler renders a minimal HTML page that loads Swagger UI
// from a CDN and points it at the embedded OpenAPI spec.
func swaggerUIHandler(c *gin.Context) {
	// The spec URL is relative to this handler, i.e. "/api/docs/openapi.yaml"
	const html = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>FinSight AI API Documentation</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet"
	      href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.12.0/swagger-ui.min.css"
	      integrity="sha512-6cV2WJ1c5Xw2xV+ZpA6vYcGz6XU4cV6Kz6e+K6YcX5vR5lV6r9QZ3k0X5Z+K3c6Q=="
	      crossorigin="anonymous"
	      referrerpolicy="no-referrer" />
	<style>
		html, body { margin:0; height:100%; }
		#swagger-ui { height:100%; }
	</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.12.0/swagger-ui-bundle.min.js"
        integrity="sha512-6cV2WJ1c5Xw2xV+ZpA6vYcGz6XU4cV6Kz6e+K6YcX5vR5lV6r9QZ3k0X5Z+K3c6Q=="
        crossorigin="anonymous"
        referrerpolicy="no-referrer"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.12.0/swagger-ui-standalone-preset.min.js"
        integrity="sha512-6cV2WJ1c5Xw2xV+ZpA6vYcGz6XU4cV6Kz6e+K6YcX5vR5lV6r9QZ3k0X5Z+K3c6Q=="
        crossorigin="anonymous"
        referrerpolicy="no-referrer"></script>
<script>
window.onload = function() {
  SwaggerUIBundle({
    url: "./openapi.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    layout: "BaseLayout",
    requestInterceptor: function(request) {
      // If the UI has a JWT stored in a cookie named "auth_token",
      // forward it as a Bearer token. This keeps the docs usable
      // against secured endpoints without exposing credentials.
      const match = document.cookie.match(/(?:^|; )auth_token=([^;]*)/);
      if (match) {
        request.headers['Authorization'] = 'Bearer ' + decodeURIComponent(match[1]);
      }
      return request;
    }
  });
};
</script>
</body>
</html>`
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// openAPISpecHandler returns the embedded OpenAPI YAML file.
func openAPISpecHandler(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/vnd.oai.openapi+yaml", openAPISpec)
}