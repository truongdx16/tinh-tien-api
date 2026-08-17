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
