package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDContextKey contextKey = "RequestID"
const RequestIDHeader = "X-Request-ID"

// RequestID middleware injects or propagates a correlation ID for every request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Set header in response
		w.Header().Set(RequestIDHeader, reqID)

		// Attach to context
		ctx := context.WithValue(r.Context(), RequestIDContextKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context or returns an empty string.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return val
	}
	return ""
}
