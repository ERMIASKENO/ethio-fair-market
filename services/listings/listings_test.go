package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

func sp(s string) *string { return &s }
func fp(f float64) *float64 { return &f }

func TestValidate(t *testing.T) {
	ok := &ListingInput{Title: sp("Toyota Corolla 2010"), PriceETB: fp(1200000), Category: sp("vehicles/cars"), Region: sp("AA"), City: sp("Addis Ababa")}
	if err := ok.Validate(true); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	bad := &ListingInput{Title: sp("abc"), PriceETB: fp(-1), Condition: sp("broken")}
	err := bad.Validate(true)
	fe, _ := err.(FieldErrors)
	for _, f := range []string{"title", "price_etb", "category", "region", "city", "condition"} {
		if _, ok := fe[f]; !ok {
			t.Errorf("expected error for %s, got %v", f, fe)
		}
	}
	// PATCH: only sent fields are checked
	if err := (&ListingInput{PriceETB: fp(10)}).Validate(false); err != nil {
		t.Fatalf("partial update rejected: %v", err)
	}
	// Amharic title counts characters, not bytes
	if err := (&ListingInput{Title: sp("ቤት ለሽያጭ")}).Validate(false); err != nil {
		t.Fatalf("amharic title rejected: %v", err)
	}
}

func TestBuildSearch(t *testing.T) {
	where, order, args := buildSearch(SearchParams{Query: "toyota", Category: "vehicles", City: "Adama", MinPrice: fp(100), Sort: "price_asc"})
	for _, want := range []string{"status = 'active'", "plainto_tsquery('simple', $1)", "category_slug = $2 OR category_slug LIKE $2 || '/%'", "lower(city) = lower($3)", "price_etb >= $4"} {
		if !strings.Contains(where, want) {
			t.Errorf("where missing %q: %s", want, where)
		}
	}
	if !strings.Contains(order, "price_etb ASC") || len(args) != 4 {
		t.Fatalf("order=%s args=%v", order, args)
	}
}

func TestParseSearchRejectsBadInput(t *testing.T) {
	for _, q := range []string{"page=0", "page_size=500", "min_price=-5", "sort=random", "condition=bad"} {
		r := httptest.NewRequest("GET", "/v1/listings?"+q, nil)
		if _, err := parseSearch(r); err == nil {
			t.Errorf("%s should be rejected", q)
		}
	}
}

// ---- integration test with a real Postgres (TEST_DATABASE_URL) ----

func testSigner(t *testing.T) *kit.Signer {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	s, err := kit.NewSignerFromPEM(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestListingsIntegration(t *testing.T) {
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
	_, _ = db.Exec(`DROP TABLE IF EXISTS listings, schema_migrations`)
	sub, _ := fs.Sub(migrationsFS, "migrations")
	if err := kit.Migrate(ctx, db, sub, 2001, kit.NewLogger("test")); err != nil {
		t.Fatal(err)
	}
	store := &pgStore{db: db}
	if err := SeedDemo(ctx, store, kit.NewLogger("test")); err != nil {
		t.Fatal(err)
	}

	// fake catalog + locations services
	refs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/categories/vehicles/cars", strings.HasPrefix(r.URL.Path, "/v1/locations/regions/AA/cities/Addis"):
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	}))
	defer refs.Close()

	signer := testSigner(t)
	api := &API{store: store, verifier: kit.NewStaticVerifier(signer.PublicKeys(), "efm-identity", "efm"),
		refs: NewRefChecker(refs.URL, refs.URL), log: kit.NewLogger("test"), metrics: kit.NewMetrics("test")}
	mux := http.NewServeMux()
	api.routes(mux)
	now := time.Now().Unix()
	tok, _ := signer.Sign(kit.Claims{Subject: "11111111-1111-4111-8111-111111111111", Phone: "+251911223344", Issuer: "efm-identity", Audience: "efm", IssuedAt: now, NotBefore: now, ExpiresAt: now + 600})
	other, _ := signer.Sign(kit.Claims{Subject: "22222222-2222-4222-8222-222222222222", Issuer: "efm-identity", Audience: "efm", IssuedAt: now, NotBefore: now, ExpiresAt: now + 600})

	do := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	// search demo data
	w := do("GET", "/v1/listings?category=vehicles&sort=price_desc", "", "")
	var page struct {
		Items []Listing `json:"items"`
		Total int       `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &page)
	if w.Code != 200 || page.Total != 2 || page.Items[0].PriceETB < page.Items[1].PriceETB {
		t.Fatalf("vehicle search: %d total=%d %s", w.Code, page.Total, w.Body)
	}
	w = do("GET", "/v1/listings?q="+url.QueryEscape("ጤፍ"), "", "")
	_ = json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 {
		t.Fatalf("amharic full-text search: total=%d %s", page.Total, w.Body)
	}

	// create: unauthenticated, unknown category, ok
	body := `{"title":"Toyota Corolla 2008 for sale","price_etb":1450000,"category":"vehicles/cars","region":"AA","city":"Addis Ababa","show_phone":true,"attributes":{"year":2008}}`
	if w := do("POST", "/v1/listings", body, ""); w.Code != 401 {
		t.Fatalf("create without token: %d", w.Code)
	}
	if w := do("POST", "/v1/listings", strings.Replace(body, "vehicles/cars", "vehicles/boats", 1), tok); w.Code != 422 {
		t.Fatalf("unknown category: %d %s", w.Code, w.Body)
	}
	w = do("POST", "/v1/listings", body, tok)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var created Listing
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.ContactPhone != "+251911223344" {
		t.Fatalf("contact phone not stored: %+v", created)
	}

	// other users cannot edit; owner can
	if w := do("PATCH", "/v1/listings/"+created.ID, `{"price_etb":1}`, other); w.Code != 404 {
		t.Fatalf("foreign update: %d", w.Code)
	}
	w = do("PATCH", "/v1/listings/"+created.ID, `{"price_etb":1400000,"negotiable":true}`, tok)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"price_etb":1400000`) {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	if w := do("GET", "/v1/listings/mine", "", tok); !strings.Contains(w.Body.String(), created.ID) {
		t.Fatalf("mine: %s", w.Body)
	}
	if w := do("POST", "/v1/listings/"+created.ID+"/sold", "", tok); w.Code != 204 {
		t.Fatalf("sold: %d", w.Code)
	}
	if w := do("GET", "/v1/listings?q=Corolla", "", ""); strings.Contains(w.Body.String(), created.ID) {
		t.Fatal("sold listing must not appear in search")
	}
	if w := do("DELETE", "/v1/listings/"+created.ID, "", tok); w.Code != 204 {
		t.Fatalf("delete: %d", w.Code)
	}
	if w := do("GET", "/v1/listings/"+created.ID, "", ""); w.Code != 404 {
		t.Fatalf("removed listing should be 404, got %d", w.Code)
	}
	if w := do("GET", "/v1/listings/not-a-uuid", "", ""); w.Code != 404 {
		t.Fatalf("bad id should be 404, got %d", w.Code)
	}

	// expiry job
	_, _ = db.Exec(`UPDATE listings SET expires_at = now() - interval '1 minute' WHERE category_slug='sports/bicycles'`)
	if n, err := store.ExpireOld(ctx); err != nil || n != 1 {
		t.Fatalf("expire: %d %v", n, err)
	}
}
