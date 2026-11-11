// identity: phone-number login with one-time codes, issues EdDSA JWTs and publishes the JWKS.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	log := kit.NewLogger("identity")
	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	dsn, err := kit.MustEnv("DATABASE_URL")
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(kit.Env("JWT_SIGNING_KEY_FILE", "/etc/efm/jwt/private.pem"))
	if err != nil {
		return err
	}
	signer, err := kit.NewSignerFromPEM(keyPEM)
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := kit.OpenDB(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	sub, _ := fs.Sub(migrationsFS, "migrations")
	if err := kit.Migrate(ctx, db, sub, 1001, log); err != nil {
		return err
	}

	cfg := Config{
		Issuer:          kit.Env("JWT_ISSUER", "efm-identity"),
		Audience:        kit.Env("JWT_AUDIENCE", "efm"),
		TokenTTL:        kit.EnvDuration("TOKEN_TTL", 24*time.Hour),
		OTPTTL:          kit.EnvDuration("OTP_TTL", 5*time.Minute),
		OTPMaxPerWindow: kit.EnvInt("OTP_MAX_PER_WINDOW", 3),
		OTPWindow:       kit.EnvDuration("OTP_WINDOW", 10*time.Minute),
		OTPMaxAttempts:  kit.EnvInt("OTP_MAX_ATTEMPTS", 5),
		DevMode:         kit.EnvBool("OTP_DEV_MODE", false),
		CookieSecure:    kit.EnvBool("COOKIE_SECURE", true),
	}
	metrics := kit.NewMetrics("identity")
	api := &API{cfg: cfg, store: &pgStore{db: db}, signer: signer, log: log, metrics: metrics, now: time.Now,
		verifier: kit.NewStaticVerifier(signer.PublicKeys(), cfg.Issuer, cfg.Audience)}

	mux := http.NewServeMux()
	health := kit.NewHealth()
	health.AddCheck("postgres", db.PingContext)
	health.Register(mux)
	mux.Handle("GET /metrics", metrics.Handler())
	api.routes(mux)

	if cfg.DevMode {
		log.Warn("OTP_DEV_MODE is on - codes are returned by the API (never enable this in production)")
	}
	return kit.Run(":"+kit.Env("PORT", "8080"), kit.Middleware(mux, log, metrics), health, log)
}
