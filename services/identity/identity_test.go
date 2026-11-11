package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"0911234567":       "+251911234567",
		"911234567":        "+251911234567",
		"+251 91 123 4567": "+251911234567",
		"251911234567":     "+251911234567",
		"0711234567":       "+251711234567",
		"(091) 123-4567":   "+251911234567",
	}
	for in, want := range ok {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "0111234567", "12345", "+1 202 555 0100", "09112345678", "09a1234567"} {
		if _, err := NormalizePhone(bad); err == nil {
			t.Errorf("NormalizePhone(%q) should fail", bad)
		}
	}
}

// fakeStore is an in-memory Store for handler tests.
type fakeStore struct {
	mu    sync.Mutex
	otps  []*struct {
		phone string
		OTP
		consumed bool
		created  time.Time
	}
	users map[string]*User
}

func newFake() *fakeStore { return &fakeStore{users: map[string]*User{}} }

func (f *fakeStore) CountRecentOTPs(_ context.Context, phone string, since time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, o := range f.otps {
		if o.phone == phone && o.created.After(since) {
			n++
		}
	}
	return n, nil
}
func (f *fakeStore) InsertOTP(_ context.Context, phone, hash, salt string, exp time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.otps = append(f.otps, &struct {
		phone string
		OTP
		consumed bool
		created  time.Time
	}{phone: phone, OTP: OTP{ID: int64(len(f.otps) + 1), CodeHash: hash, Salt: salt, ExpiresAt: exp}, created: time.Now()})
	return nil
}
func (f *fakeStore) LatestOTP(_ context.Context, phone string) (*OTP, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.otps) - 1; i >= 0; i-- {
		if o := f.otps[i]; o.phone == phone && !o.consumed {
			c := o.OTP
			return &c, nil
		}
	}
	return nil, errNotFound
}
func (f *fakeStore) IncrementAttempts(_ context.Context, id int64) error {
	f.otps[id-1].Attempts++
	return nil
}
func (f *fakeStore) ConsumeOTP(_ context.Context, id int64) error { f.otps[id-1].consumed = true; return nil }
func (f *fakeStore) UpsertUser(_ context.Context, phone string) (*User, error) {
	if u, ok := f.users[phone]; ok {
		return u, nil
	}
	u := &User{ID: "00000000-0000-0000-0000-00000000000" + string(rune('1'+len(f.users))), Phone: phone, CreatedAt: time.Now()}
	f.users[phone] = u
	return u, nil
}
func (f *fakeStore) GetUser(_ context.Context, id string) (*User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errNotFound
}
func (f *fakeStore) UpdateDisplayName(ctx context.Context, id, name string) (*User, error) {
	u, err := f.GetUser(ctx, id)
	if err == nil {
		u.DisplayName = name
	}
	return u, err
}

func newTestAPI(t *testing.T) (*API, http.Handler) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	signer, err := kit.NewSignerFromPEM(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Issuer: "efm-identity", Audience: "efm", TokenTTL: time.Hour, OTPTTL: 5 * time.Minute,
		OTPMaxPerWindow: 3, OTPWindow: 10 * time.Minute, OTPMaxAttempts: 3, DevMode: true}
	a := &API{cfg: cfg, store: newFake(), signer: signer, log: kit.NewLogger("test"), metrics: kit.NewMetrics("test"), now: time.Now,
		verifier: kit.NewStaticVerifier(signer.PublicKeys(), cfg.Issuer, cfg.Audience)}
	mux := http.NewServeMux()
	a.routes(mux)
	return a, mux
}

func call(h http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestLoginFlow(t *testing.T) {
	_, h := newTestAPI(t)
	w := call(h, "POST", "/v1/auth/otp/request", `{"phone":"0911 234 567"}`, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("request: %d %s", w.Code, w.Body)
	}
	var req struct{ DevCode string `json:"dev_code"` }
	_ = json.Unmarshal(w.Body.Bytes(), &req)

	if w := call(h, "POST", "/v1/auth/otp/verify", `{"phone":"0911234567","code":"000000x"}`, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code should be 401, got %d", w.Code)
	}
	w = call(h, "POST", "/v1/auth/otp/verify", `{"phone":"+251911234567","code":"`+req.DevCode+`"}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", w.Code, w.Body)
	}
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == kit.TokenCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly {
		t.Fatal("expected HttpOnly session cookie")
	}
	// code is single use
	if w := call(h, "POST", "/v1/auth/otp/verify", `{"phone":"0911234567","code":"`+req.DevCode+`"}`, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("reused code should fail, got %d", w.Code)
	}
	if w := call(h, "GET", "/v1/auth/me", "", cookie); w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if w := call(h, "PATCH", "/v1/auth/me", `{"display_name":"Abebe Kebede"}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("update me: %d %s", w.Code, w.Body)
	}
	if w := call(h, "GET", "/v1/auth/me", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("me without token should be 401, got %d", w.Code)
	}
}

func TestRateLimit(t *testing.T) {
	_, h := newTestAPI(t)
	for i := 0; i < 3; i++ {
		if w := call(h, "POST", "/v1/auth/otp/request", `{"phone":"0922000000"}`, nil); w.Code != http.StatusAccepted {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	if w := call(h, "POST", "/v1/auth/otp/request", `{"phone":"0922000000"}`, nil); w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request should be 429, got %d", w.Code)
	}
}

func TestAttemptsLimit(t *testing.T) {
	_, h := newTestAPI(t)
	call(h, "POST", "/v1/auth/otp/request", `{"phone":"0933000000"}`, nil)
	for i := 0; i < 3; i++ {
		call(h, "POST", "/v1/auth/otp/verify", `{"phone":"0933000000","code":"999999"}`, nil)
	}
	// even a correct code is refused after too many attempts; we cannot know it here, so check the message
	w := call(h, "POST", "/v1/auth/otp/verify", `{"phone":"0933000000","code":"999998"}`, nil)
	if w.Code != http.StatusUnauthorized || !bytes.Contains(w.Body.Bytes(), []byte("expired")) {
		t.Fatalf("expected lockout, got %d %s", w.Code, w.Body)
	}
}

func TestJWKSPublished(t *testing.T) {
	_, h := newTestAPI(t)
	w := call(h, "GET", "/v1/auth/.well-known/jwks.json", "", nil)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"crv":"Ed25519"`)) {
		t.Fatalf("jwks: %d %s", w.Code, w.Body)
	}
}
