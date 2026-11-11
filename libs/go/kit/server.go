package kit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// Health tracks readiness. /healthz = process alive, /readyz = dependencies OK and not shutting down.
type Health struct {
	checks       map[string]func(context.Context) error
	shuttingDown atomic.Bool
}

// NewHealth creates a health tracker.
func NewHealth() *Health { return &Health{checks: map[string]func(context.Context) error{}} }

// AddCheck registers a readiness check (e.g. database ping).
func (h *Health) AddCheck(name string, fn func(context.Context) error) { h.checks[name] = fn }

// Register adds /healthz and /readyz to the mux.
func (h *Health) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if h.shuttingDown.Load() {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "shutting down"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		result := map[string]string{}
		ok := true
		for name, fn := range h.checks {
			if err := fn(ctx); err != nil {
				result[name] = err.Error()
				ok = false
			} else {
				result[name] = "ok"
			}
		}
		status := http.StatusOK
		if !ok {
			status = http.StatusServiceUnavailable
		}
		WriteJSON(w, status, map[string]any{"status": map[bool]string{true: "ready", false: "not ready"}[ok], "checks": result})
	})
}

// Run starts the HTTP server and shuts it down gracefully on SIGTERM/SIGINT:
// readiness turns false first, so Kubernetes stops sending traffic, then open requests may finish.
func Run(addr string, handler http.Handler, health *Health, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutdown started")
	health.shuttingDown.Store(true)
	time.Sleep(EnvDuration("SHUTDOWN_DELAY", 3*time.Second)) // let endpoints update
	sctx, cancel := context.WithTimeout(context.Background(), EnvDuration("SHUTDOWN_TIMEOUT", 20*time.Second))
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	log.Info("shutdown complete")
	return nil
}
