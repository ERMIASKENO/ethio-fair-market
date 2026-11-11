package kit

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestSigner(t *testing.T) *Signer {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	s, err := NewSignerFromPEM(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestJWTRoundTripAndJWKS(t *testing.T) {
	s := newTestSigner(t)
	now := time.Now().Unix()
	tok, err := s.Sign(Claims{Subject: "u1", Phone: "+251911000000", Issuer: "efm-identity", Audience: "efm", IssuedAt: now, NotBefore: now, ExpiresAt: now + 60})
	if err != nil {
		t.Fatal(err)
	}
	// verify through a JWKS endpoint, like the listings service does
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { WriteJSON(w, 200, s.JWKS()) }))
	defer srv.Close()
	v := NewVerifier(srv.URL, "efm-identity", "efm", time.Minute)
	c, err := v.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.Subject != "u1" || c.Phone != "+251911000000" {
		t.Fatalf("claims: %+v", c)
	}
	// wrong audience
	if _, err := NewStaticVerifier(s.PublicKeys(), "efm-identity", "other").Verify(context.Background(), tok); err == nil {
		t.Fatal("expected audience error")
	}
	// tampered payload
	parts := strings.Split(tok, ".")
	bad := parts[0] + "." + parts[1] + "x." + parts[2]
	if _, err := v.Verify(context.Background(), bad); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestExpiredToken(t *testing.T) {
	s := newTestSigner(t)
	old := time.Now().Add(-time.Hour).Unix()
	tok, _ := s.Sign(Claims{Subject: "u1", Issuer: "i", Audience: "a", IssuedAt: old, NotBefore: old, ExpiresAt: old + 60})
	if _, err := NewStaticVerifier(s.PublicKeys(), "i", "a").Verify(context.Background(), tok); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestMetricsExposition(t *testing.T) {
	m := NewMetrics("test")
	m.Inc("http_requests_total", "req", Labels("route", "GET /x", "code", "200"))
	m.Observe("http_request_duration_seconds", "lat", Labels("route", "GET /x"), 0.02)
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()
	for _, want := range []string{`http_requests_total{code="200",route="GET /x"} 1`, `http_request_duration_seconds_bucket{route="GET /x",le="0.025"} 1`, `le="+Inf"} 1`, "app_build_info"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	var dst struct{ A string }
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"A":"x","B":1}`))
	if err := DecodeJSON(r, &dst); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestMiddlewareUsesRoutePattern(t *testing.T) {
	m := NewMetrics("test")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/items/{id}", func(w http.ResponseWriter, r *http.Request) { WriteJSON(w, 200, map[string]string{"id": r.PathValue("id")}) })
	h := Middleware(mux, NewLogger("test"), m)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/v1/items/42", nil))
	out := httptest.NewRecorder()
	m.Handler().ServeHTTP(out, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(out.Body.String(), `route="GET /v1/items/{id}"`) {
		t.Fatalf("route label missing:\n%s", out.Body.String())
	}
	if rr.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing request id")
	}
}
