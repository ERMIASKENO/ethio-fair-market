// listings: classified ads - search, detail, create/update by the seller, automatic expiry.
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
	log := kit.NewLogger("listings")
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
	jwksURL := kit.Env("JWKS_URL", "http://identity/v1/auth/.well-known/jwks.json")

	ctx := context.Background()
	db, err := kit.OpenDB(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	sub, _ := fs.Sub(migrationsFS, "migrations")
	if err := kit.Migrate(ctx, db, sub, 2001, log); err != nil {
		return err
	}
	store := &pgStore{db: db}
	if kit.EnvBool("SEED_DEMO_DATA", false) {
		if err := SeedDemo(ctx, store, log); err != nil {
			return err
		}
	}

	metrics := kit.NewMetrics("listings")
	api := &API{
		store:    store,
		verifier: kit.NewVerifier(jwksURL, kit.Env("JWT_ISSUER", "efm-identity"), kit.Env("JWT_AUDIENCE", "efm"), 10*time.Minute),
		refs:     NewRefChecker(kit.Env("CATALOG_URL", ""), kit.Env("LOCATIONS_URL", "")),
		log:      log,
		metrics:  metrics,
	}

	// expire ads older than 30 days (every replica runs it; the UPDATE is idempotent)
	go func() {
		t := time.NewTicker(kit.EnvDuration("EXPIRY_INTERVAL", 15*time.Minute))
		defer t.Stop()
		for range t.C {
			c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if n, err := store.ExpireOld(c); err != nil {
				log.Error("expiry job failed", "error", err)
			} else if n > 0 {
				log.Info("listings expired", "count", n)
				metrics.Add("listings_expired_total", "Listings expired automatically.", "", float64(n))
			}
			cancel()
		}
	}()

	mux := http.NewServeMux()
	health := kit.NewHealth()
	health.AddCheck("postgres", db.PingContext)
	health.Register(mux)
	mux.Handle("GET /metrics", metrics.Handler())
	api.routes(mux)
	return kit.Run(":"+kit.Env("PORT", "8080"), kit.Middleware(mux, log, metrics), health, log)
}
