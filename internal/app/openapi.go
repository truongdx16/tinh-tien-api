package app

import (
	_ "embed"
	"net/http"
	"os"

	"tinh-tien-api/internal/pkg/httputil"
)

//go:embed swagger-ui.html
var swaggerUIHTML []byte

func OpenAPIHandler(specPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(specPath); err != nil {
			httputil.NotFound(w, "openapi spec not found")
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, specPath)
	}
}

func DocsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(swaggerUIHTML)
	}
}
