package kit

import (
	"log/slog"
	"os"
	"strings"
)

// Version is set at build time: -ldflags "-X github.com/ermiaskeno/ethio-fair-market/libs/go/kit.Version=..."
var Version = "dev"

// NewLogger returns a JSON logger (one line per event, easy for Loki/Elasticsearch) with service metadata.
func NewLogger(service string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h).With("service", service, "version", Version)
}
