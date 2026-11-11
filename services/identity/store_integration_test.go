package main

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

// Runs only when TEST_DATABASE_URL is set (CI starts a Postgres service container).
func TestPgStoreIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	db, err := kit.OpenDB(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sub, _ := fs.Sub(migrationsFS, "migrations")
	for i := 0; i < 2; i++ { // second run must be a no-op
		if err := kit.Migrate(ctx, db, sub, 1001, kit.NewLogger("test")); err != nil {
			t.Fatal(err)
		}
	}
	s := &pgStore{db: db}
	phone := "+251911" + time.Now().Format("150405")
	if err := s.InsertOTP(ctx, phone, "h", "s", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n, err := s.CountRecentOTPs(ctx, phone, time.Now().Add(-time.Minute)); err != nil || n != 1 {
		t.Fatalf("count: %d %v", n, err)
	}
	o, err := s.LatestOTP(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.IncrementAttempts(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeOTP(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LatestOTP(ctx, phone); err != errNotFound {
		t.Fatalf("consumed otp should not be returned: %v", err)
	}
	u1, err := s.UpsertUser(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	u2, _ := s.UpsertUser(ctx, phone)
	if u1.ID != u2.ID {
		t.Fatal("upsert must return the same user")
	}
	if u, err := s.UpdateDisplayName(ctx, u1.ID, "Almaz"); err != nil || u.DisplayName != "Almaz" {
		t.Fatalf("update: %+v %v", u, err)
	}
}
