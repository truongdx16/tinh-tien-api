package app

import (
	"net/http"
	"runtime/debug"

	"tinh-tien-api/internal/pkg/httputil"
)

func RecoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				_ = debug.Stack()
				if httputil.IsJSONRequest(r) {
					httputil.InternalError(w, "internal server error", rec)
					return
				}
				panic(rec)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func NotFoundJSON(w http.ResponseWriter, r *http.Request) {
	if httputil.IsJSONRequest(r) {
		httputil.NotFound(w, "endpoint not found")
		return
	}
	http.NotFound(w, r)
}

func MethodNotAllowedJSON(w http.ResponseWriter, r *http.Request) {
	if httputil.IsJSONRequest(r) {
		httputil.MethodNotAllowed(w, r.Method)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// CORS lets the web build of the app (served from another origin) call the
// API. Requests are authenticated with a bearer token, never cookies, so any
// origin is allowed and credentials are not.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Add("Vary", "Origin")
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			h.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
