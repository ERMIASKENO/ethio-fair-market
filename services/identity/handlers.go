package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

// Config holds the identity settings.
type Config struct {
	Issuer, Audience string
	TokenTTL         time.Duration
	OTPTTL           time.Duration
	OTPMaxPerWindow  int
	OTPWindow        time.Duration
	OTPMaxAttempts   int
	DevMode          bool // returns the code in the API response (no SMS provider yet)
	CookieSecure     bool
}

// API implements the HTTP endpoints.
type API struct {
	cfg      Config
	store    Store
	signer   *kit.Signer
	verifier *kit.Verifier
	log      *slog.Logger
	metrics  *kit.Metrics
	now      func() time.Time
}

func (a *API) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/otp/request", a.requestOTP)
	mux.HandleFunc("POST /v1/auth/otp/verify", a.verifyOTP)
	mux.HandleFunc("POST /v1/auth/logout", a.logout)
	mux.HandleFunc("GET /v1/auth/me", kit.RequireAuth(a.verifier, a.me))
	mux.HandleFunc("PATCH /v1/auth/me", kit.RequireAuth(a.verifier, a.updateMe))
	mux.HandleFunc("GET /v1/auth/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		kit.WriteJSON(w, http.StatusOK, a.signer.JWKS())
	})
}

func hashCode(salt, code string) string {
	sum := sha256.Sum256([]byte(salt + ":" + code))
	return hex.EncodeToString(sum[:])
}

func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func newSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *API) requestOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
	}
	if err := kit.DecodeJSON(r, &in); err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	phone, err := NormalizePhone(in.Phone)
	if err != nil {
		kit.WriteError(w, http.StatusUnprocessableEntity, "invalid_phone", err.Error())
		return
	}
	ctx := r.Context()
	n, err := a.store.CountRecentOTPs(ctx, phone, a.now().Add(-a.cfg.OTPWindow))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if n >= a.cfg.OTPMaxPerWindow {
		w.Header().Set("Retry-After", fmt.Sprint(int(a.cfg.OTPWindow.Seconds())))
		kit.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many codes requested, try again later")
		return
	}
	code, err := newCode()
	if err != nil {
		a.fail(w, r, err)
		return
	}
	salt := newSalt()
	if err := a.store.InsertOTP(ctx, phone, hashCode(salt, code), salt, a.now().Add(a.cfg.OTPTTL)); err != nil {
		a.fail(w, r, err)
		return
	}
	a.metrics.Inc("identity_otp_requested_total", "One-time codes issued.", "")
	resp := map[string]any{"phone": phone, "expires_in": int(a.cfg.OTPTTL.Seconds())}
	if a.cfg.DevMode {
		// Phase 2: the notifications service sends the code by SMS instead.
		resp["dev_code"] = code
		a.log.Warn("OTP dev mode: code returned in API response", "phone", maskPhone(phone))
	}
	kit.WriteJSON(w, http.StatusAccepted, resp)
}

func (a *API) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := kit.DecodeJSON(r, &in); err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	phone, err := NormalizePhone(in.Phone)
	if err != nil {
		kit.WriteError(w, http.StatusUnprocessableEntity, "invalid_phone", err.Error())
		return
	}
	code := strings.TrimSpace(in.Code)
	ctx := r.Context()
	otp, err := a.store.LatestOTP(ctx, phone)
	if errors.Is(err, errNotFound) {
		kit.WriteError(w, http.StatusUnauthorized, "invalid_code", "request a new code")
		return
	} else if err != nil {
		a.fail(w, r, err)
		return
	}
	if a.now().After(otp.ExpiresAt) || otp.Attempts >= a.cfg.OTPMaxAttempts {
		kit.WriteError(w, http.StatusUnauthorized, "invalid_code", "code expired, request a new one")
		return
	}
	if subtle.ConstantTimeCompare([]byte(hashCode(otp.Salt, code)), []byte(otp.CodeHash)) != 1 {
		_ = a.store.IncrementAttempts(ctx, otp.ID)
		a.metrics.Inc("identity_login_total", "Login attempts.", kit.Labels("result", "wrong_code"))
		kit.WriteError(w, http.StatusUnauthorized, "invalid_code", "wrong code")
		return
	}
	if err := a.store.ConsumeOTP(ctx, otp.ID); err != nil {
		a.fail(w, r, err)
		return
	}
	user, err := a.store.UpsertUser(ctx, phone)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	now := a.now()
	exp := now.Add(a.cfg.TokenTTL)
	token, err := a.signer.Sign(kit.Claims{Subject: user.ID, Phone: user.Phone, Issuer: a.cfg.Issuer, Audience: a.cfg.Audience,
		IssuedAt: now.Unix(), NotBefore: now.Unix(), ExpiresAt: exp.Unix(), ID: newSalt()})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: kit.TokenCookie, Value: token, Path: "/", Expires: exp, MaxAge: int(a.cfg.TokenTTL.Seconds()),
		HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	a.metrics.Inc("identity_login_total", "Login attempts.", kit.Labels("result", "success"))
	kit.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "expires_at": exp.UTC(), "user": user})
}

func (a *API) logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: kit.TokenCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.store.GetUser(r.Context(), kit.ClaimsFrom(r.Context()).Subject)
	if errors.Is(err, errNotFound) {
		kit.WriteError(w, http.StatusNotFound, "not_found", "user not found")
		return
	} else if err != nil {
		a.fail(w, r, err)
		return
	}
	kit.WriteJSON(w, http.StatusOK, u)
}

func (a *API) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DisplayName string `json:"display_name"`
	}
	if err := kit.DecodeJSON(r, &in); err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	name := strings.TrimSpace(in.DisplayName)
	if n := utf8.RuneCountInString(name); n < 2 || n > 60 {
		kit.WriteError(w, http.StatusUnprocessableEntity, "invalid_name", "name must be 2-60 characters")
		return
	}
	u, err := a.store.UpdateDisplayName(r.Context(), kit.ClaimsFrom(r.Context()).Subject, name)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	kit.WriteJSON(w, http.StatusOK, u)
}

func (a *API) fail(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "request_id", kit.RequestID(r.Context()), "error", err)
	kit.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func maskPhone(p string) string {
	if len(p) < 7 {
		return "***"
	}
	return p[:6] + "****" + p[len(p)-2:]
}
