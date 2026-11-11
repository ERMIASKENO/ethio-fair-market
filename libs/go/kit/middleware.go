package kit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	claimsKey
)

// RequestID returns the request id stored in the context.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Middleware wraps the router with: request id, panic recovery, security headers,
// access log (JSON) and Prometheus metrics. Health and metrics endpoints are not logged.
func Middleware(next http.Handler, log *slog.Logger, m *Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			id = newID()
		}
		w.Header().Set("X-Request-Id", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		rec := &statusRecorder{ResponseWriter: w}
		rr := r.WithContext(ctx) // the ServeMux writes the matched pattern into this request

		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "request_id", id, "panic", p, "stack", string(debug.Stack()))
				if rec.status == 0 {
					WriteError(rec, http.StatusInternalServerError, "internal", "internal server error")
				}
			}
			route := rr.Pattern // Go 1.23+: the matched ServeMux pattern, keeps metric labels low-cardinality
			if route == "" {
				route = "unmatched"
			}
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			dur := time.Since(start).Seconds()
			if route != "GET /healthz" && route != "GET /readyz" && route != "GET /metrics" {
				m.Inc("http_requests_total", "HTTP requests handled.", Labels("route", route, "code", strconv.Itoa(rec.status)))
				m.Observe("http_request_duration_seconds", "HTTP request latency.", Labels("route", route), dur)
				log.Info("request", "request_id", id, "method", r.Method, "path", r.URL.Path, "route", route,
					"status", rec.status, "bytes", rec.bytes, "duration_ms", int(dur*1000), "remote", r.RemoteAddr)
			}
		}()
		next.ServeHTTP(rec, rr)
	})
}
