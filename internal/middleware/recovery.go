package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer catches unhandled panics, logs stack traces, and returns an RFC 7807 500 error.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				reqID := GetRequestID(r.Context())
				stack := string(debug.Stack())

				slog.Error("panic recovered in HTTP handler",
					"error", fmt.Sprintf("%v", rvr),
					"stack", stack,
					"request_id", reqID,
				)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)

				resp := map[string]interface{}{
					"error": map[string]interface{}{
						"code":       "INTERNAL_ERROR",
						"message":    "An unexpected internal server error occurred",
						"request_id": reqID,
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
