package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ermiaskeno/ethio-fair-market/libs/go/kit"
)

// API implements the listings endpoints.
type API struct {
	store    Store
	verifier *kit.Verifier
	refs     *RefChecker
	log      *slog.Logger
	metrics  *kit.Metrics
}

func (a *API) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/listings", a.search)
	mux.HandleFunc("GET /v1/listings/mine", kit.RequireAuth(a.verifier, a.mine))
	mux.HandleFunc("GET /v1/listings/{id}", a.get)
	mux.HandleFunc("POST /v1/listings", kit.RequireAuth(a.verifier, a.create))
	mux.HandleFunc("PATCH /v1/listings/{id}", kit.RequireAuth(a.verifier, a.update))
	mux.HandleFunc("POST /v1/listings/{id}/sold", kit.RequireAuth(a.verifier, a.statusHandler("sold")))
	mux.HandleFunc("DELETE /v1/listings/{id}", kit.RequireAuth(a.verifier, a.statusHandler("removed")))
}

func parseSearch(r *http.Request) (SearchParams, error) {
	q := r.URL.Query()
	p := SearchParams{
		Query:     strings.TrimSpace(q.Get("q")),
		Category:  q.Get("category"),
		Region:    q.Get("region"),
		City:      q.Get("city"),
		Condition: q.Get("condition"),
		Sort:      q.Get("sort"),
		Page:      1,
		PageSize:  20,
	}
	if len(p.Query) > 100 {
		return p, errors.New("q is too long")
	}
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			return p, errors.New("page must be 1-500")
		}
		p.Page = n
	}
	if v := q.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 50 {
			return p, errors.New("page_size must be 1-50")
		}
		p.PageSize = n
	}
	for name, dst := range map[string]**float64{"min_price": &p.MinPrice, "max_price": &p.MaxPrice} {
		if v := q.Get(name); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f < 0 {
				return p, errors.New(name + " must be a positive number")
			}
			*dst = &f
		}
	}
	switch p.Sort {
	case "", "newest", "price_asc", "price_desc", "relevance":
	default:
		return p, errors.New("sort must be newest, price_asc, price_desc or relevance")
	}
	if p.Condition != "" && !conditions[p.Condition] {
		return p, errors.New("invalid condition")
	}
	return p, nil
}

func (a *API) search(w http.ResponseWriter, r *http.Request) {
	p, err := parseSearch(r)
	if err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	items, total, err := a.store.Search(r.Context(), p)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	for i := range items {
		items[i].ContactPhone = "" // contact details only on the detail page
	}
	w.Header().Set("Cache-Control", "public, max-age=15")
	kit.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": p.Page, "page_size": p.PageSize})
}

func (a *API) mine(w http.ResponseWriter, r *http.Request) {
	p := SearchParams{SellerID: kit.ClaimsFrom(r.Context()).Subject, IncludeInactive: true, Page: 1, PageSize: 50}
	items, total, err := a.store.Search(r.Context(), p)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	kit.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	l, err := a.store.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, errNotFound) {
		kit.WriteError(w, http.StatusNotFound, "not_found", "listing not found")
		return
	} else if err != nil {
		a.fail(w, r, err)
		return
	}
	go func(id string) { // view counting must never slow down or break the page
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.store.IncrementViews(ctx, id)
	}(l.ID)
	kit.WriteJSON(w, http.StatusOK, l)
}

func (a *API) checkRefs(w http.ResponseWriter, r *http.Request, in *ListingInput, current *Listing) bool {
	cat, region, city := str(in.Category, ""), str(in.Region, ""), str(in.City, "")
	if current != nil && (in.Region != nil || in.City != nil) { // PATCH of one location field: check the pair
		if region == "" {
			region = current.RegionCode
		}
		if city == "" {
			city = current.City
		}
	}
	errs, err := a.refs.Check(r.Context(), cat, region, city)
	if err != nil {
		a.log.Warn("reference check failed", "error", err)
		kit.WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "please try again in a moment")
		return false
	}
	if len(errs) > 0 {
		kit.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation", "message": errs.Error(), "fields": errs}})
		return false
	}
	return true
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	var in ListingInput
	if err := kit.DecodeJSON(r, &in); err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := in.Validate(true); err != nil {
		var fe FieldErrors
		errors.As(err, &fe)
		kit.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation", "message": err.Error(), "fields": fe}})
		return
	}
	if !a.checkRefs(w, r, &in, nil) {
		return
	}
	c := kit.ClaimsFrom(r.Context())
	l, err := a.store.Create(r.Context(), c.Subject, c.Phone, &in)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.metrics.Inc("listings_created_total", "Listings created.", kit.Labels("category", topCategory(l.CategorySlug)))
	w.Header().Set("Location", "/v1/listings/"+l.ID)
	kit.WriteJSON(w, http.StatusCreated, l)
}

func (a *API) update(w http.ResponseWriter, r *http.Request) {
	var in ListingInput
	if err := kit.DecodeJSON(r, &in); err != nil {
		kit.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := in.Validate(false); err != nil {
		var fe FieldErrors
		errors.As(err, &fe)
		kit.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation", "message": err.Error(), "fields": fe}})
		return
	}
	c := kit.ClaimsFrom(r.Context())
	current, err := a.store.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, errNotFound) || (err == nil && current.SellerID != c.Subject) {
		kit.WriteError(w, http.StatusNotFound, "not_found", "listing not found")
		return
	} else if err != nil {
		a.fail(w, r, err)
		return
	}
	if !a.checkRefs(w, r, &in, current) {
		return
	}
	l, err := a.store.Update(r.Context(), current.ID, c.Subject, c.Phone, &in)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	kit.WriteJSON(w, http.StatusOK, l)
}

func (a *API) statusHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := a.store.SetStatus(r.Context(), r.PathValue("id"), kit.ClaimsFrom(r.Context()).Subject, status)
		if errors.Is(err, errNotFound) {
			kit.WriteError(w, http.StatusNotFound, "not_found", "listing not found")
			return
		} else if err != nil {
			a.fail(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *API) fail(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "request_id", kit.RequestID(r.Context()), "error", err)
	kit.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func topCategory(slug string) string {
	if i := strings.IndexByte(slug, '/'); i > 0 {
		return slug[:i]
	}
	return slug
}
