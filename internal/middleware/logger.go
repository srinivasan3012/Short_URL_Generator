package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterObserver struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriterObserver) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterObserver) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Logger middleware logs incoming HTTP requests and response metrics using slog.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriterObserver{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		reqID := GetRequestID(r.Context())

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"bytes", wrapped.bytesWritten,
			"ip", r.RemoteAddr,
			"request_id", reqID,
		)
	})
}
