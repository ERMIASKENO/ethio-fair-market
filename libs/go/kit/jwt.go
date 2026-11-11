package kit

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Claims are the JWT claims used across the platform.
type Claims struct {
	Subject   string `json:"sub"`           // user id (uuid)
	Phone     string `json:"phone"`         // E.164, e.g. +251911234567
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	ExpiresAt int64  `json:"exp"`
	ID        string `json:"jti"`
}

var b64 = base64.RawURLEncoding

// Signer signs EdDSA (Ed25519) JWTs. Ed25519 keys are small, fast and have no weak parameters.
type Signer struct {
	key ed25519.PrivateKey
	kid string
}

// NewSignerFromPEM loads a PKCS#8 PEM Ed25519 private key (openssl genpkey -algorithm ed25519).
func NewSignerFromPEM(pemData []byte) (*Signer, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("signing key: no PEM block found")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("signing key: %w", err)
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("signing key: not an Ed25519 key")
	}
	pub := priv.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(pub)
	return &Signer{key: priv, kid: b64.EncodeToString(sum[:8])}, nil
}

// Sign returns a compact JWT.
func (s *Signer) Sign(c Claims) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": s.kid})
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	unsigned := b64.EncodeToString(header) + "." + b64.EncodeToString(payload)
	sig := ed25519.Sign(s.key, []byte(unsigned))
	return unsigned + "." + b64.EncodeToString(sig), nil
}

// JWK is a public key in JSON Web Key format (RFC 8037 for Ed25519).
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

// JWKS returns the public key set to publish at /.well-known/jwks.json.
func (s *Signer) JWKS() map[string][]JWK {
	pub := s.key.Public().(ed25519.PublicKey)
	return map[string][]JWK{"keys": {{Kty: "OKP", Crv: "Ed25519", X: b64.EncodeToString(pub), Kid: s.kid, Use: "sig", Alg: "EdDSA"}}}
}

// PublicKeys returns kid -> key for local verification.
func (s *Signer) PublicKeys() map[string]ed25519.PublicKey {
	return map[string]ed25519.PublicKey{s.kid: s.key.Public().(ed25519.PublicKey)}
}

// Verifier validates JWTs against keys from a JWKS URL (cached) or a static key set.
type Verifier struct {
	issuer, audience string
	jwksURL          string
	client           *http.Client
	ttl              time.Duration

	mu      sync.RWMutex
	keys    map[string]ed25519.PublicKey
	fetched time.Time
	now     func() time.Time
}

// NewVerifier builds a verifier that loads keys from jwksURL and refreshes them every ttl.
func NewVerifier(jwksURL, issuer, audience string, ttl time.Duration) *Verifier {
	return &Verifier{jwksURL: jwksURL, issuer: issuer, audience: audience, ttl: ttl,
		client: &http.Client{Timeout: 3 * time.Second}, keys: map[string]ed25519.PublicKey{}, now: time.Now}
}

// NewStaticVerifier verifies with fixed keys (used by the identity service itself and tests).
func NewStaticVerifier(keys map[string]ed25519.PublicKey, issuer, audience string) *Verifier {
	return &Verifier{issuer: issuer, audience: audience, keys: keys, fetched: time.Now().Add(100 * 365 * 24 * time.Hour), now: time.Now}
}

func (v *Verifier) refresh(ctx context.Context) error {
	if v.jwksURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch: status %d", resp.StatusCode)
	}
	var set struct {
		Keys []JWK `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("jwks decode: %w", err)
	}
	keys := map[string]ed25519.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" {
			continue
		}
		raw, err := b64.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	if len(keys) == 0 {
		return errors.New("jwks: no usable Ed25519 keys")
	}
	v.mu.Lock()
	v.keys, v.fetched = keys, v.now()
	v.mu.Unlock()
	return nil
}

func (v *Verifier) key(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	v.mu.RLock()
	k, ok := v.keys[kid]
	stale := v.now().Sub(v.fetched) > v.ttl
	v.mu.RUnlock()
	if ok && !stale {
		return k, nil
	}
	if err := v.refresh(ctx); err != nil && !ok {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, errors.New("unknown signing key")
}

// Verify checks signature, algorithm, issuer, audience and time claims.
func (v *Verifier) Verify(ctx context.Context, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	hb, err := b64.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("malformed header")
	}
	var h struct{ Alg, Kid string }
	if err := json.Unmarshal(hb, &h); err != nil || h.Alg != "EdDSA" {
		return nil, errors.New("unsupported algorithm")
	}
	key, err := v.key(ctx, h.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), sig) {
		return nil, errors.New("invalid signature")
	}
	pb, err := b64.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("malformed payload")
	}
	var c Claims
	if err := json.Unmarshal(pb, &c); err != nil {
		return nil, errors.New("malformed claims")
	}
	now := v.now().Unix()
	const leeway = 30
	switch {
	case c.Issuer != v.issuer:
		return nil, errors.New("wrong issuer")
	case c.Audience != v.audience:
		return nil, errors.New("wrong audience")
	case c.ExpiresAt < now-leeway:
		return nil, errors.New("token expired")
	case c.NotBefore > now+leeway:
		return nil, errors.New("token not valid yet")
	case c.Subject == "":
		return nil, errors.New("missing subject")
	}
	return &c, nil
}

// TokenCookie is the name of the HttpOnly session cookie set by the identity service.
const TokenCookie = "efm_token"

// TokenFromRequest reads "Authorization: Bearer <jwt>" or the session cookie.
func TokenFromRequest(r *http.Request) string {
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
		return strings.TrimPrefix(a, "Bearer ")
	}
	if c, err := r.Cookie(TokenCookie); err == nil {
		return c.Value
	}
	return ""
}

// RequireAuth rejects requests without a valid token and stores the claims in the context.
func RequireAuth(v *Verifier, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := TokenFromRequest(r)
		if tok == "" {
			WriteError(w, http.StatusUnauthorized, "unauthenticated", "login required")
			return
		}
		c, err := v.Verify(r.Context(), tok)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
	}
}

// ClaimsFrom returns the claims set by RequireAuth.
func ClaimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
