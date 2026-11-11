package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var errNotFound = errors.New("not found")

// Store persists listings.
type Store interface {
	Create(ctx context.Context, sellerID, phone string, in *ListingInput) (*Listing, error)
	Get(ctx context.Context, id string) (*Listing, error)
	Update(ctx context.Context, id, sellerID, phone string, in *ListingInput) (*Listing, error)
	SetStatus(ctx context.Context, id, sellerID, status string) error
	Search(ctx context.Context, p SearchParams) ([]Listing, int, error)
	IncrementViews(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
	ExpireOld(ctx context.Context) (int64, error)
}

type pgStore struct{ db *sql.DB }

const cols = `id, seller_id, title, description, price_etb, negotiable, category_slug, region_code, city,
	coalesce(subcity,''), item_condition, attributes, image_urls, coalesce(contact_phone,''), status, view_count,
	(featured_until IS NOT NULL AND featured_until > now()), created_at, updated_at, expires_at`

func scanListing(row interface{ Scan(...any) error }) (*Listing, error) {
	var l Listing
	var attrs []byte
	err := row.Scan(&l.ID, &l.SellerID, &l.Title, &l.Description, &l.PriceETB, &l.Negotiable, &l.CategorySlug, &l.RegionCode,
		&l.City, &l.Subcity, &l.Condition, &attrs, pq.Array(&l.ImageURLs), &l.ContactPhone, &l.Status, &l.ViewCount,
		&l.Featured, &l.CreatedAt, &l.UpdatedAt, &l.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	l.Attributes = json.RawMessage(attrs)
	if l.ImageURLs == nil {
		l.ImageURLs = []string{}
	}
	return &l, nil
}

func str(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func (s *pgStore) Create(ctx context.Context, sellerID, phone string, in *ListingInput) (*Listing, error) {
	attrs := json.RawMessage(`{}`)
	if in.Attributes != nil {
		attrs = *in.Attributes
	}
	images := []string{}
	if in.ImageURLs != nil {
		images = *in.ImageURLs
	}
	var contact any
	if in.ShowPhone != nil && *in.ShowPhone {
		contact = phone
	}
	var subcity any
	if in.Subcity != nil && *in.Subcity != "" {
		subcity = *in.Subcity
	}
	negotiable := in.Negotiable != nil && *in.Negotiable
	row := s.db.QueryRowContext(ctx, `INSERT INTO listings
		(seller_id, title, description, price_etb, negotiable, category_slug, region_code, city, subcity, item_condition,
		 attributes, image_urls, contact_phone)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+cols,
		sellerID, *in.Title, str(in.Description, ""), *in.PriceETB, negotiable, *in.Category, *in.Region, *in.City, subcity,
		str(in.Condition, "used"), []byte(attrs), pq.Array(images), contact)
	return scanListing(row)
}

func (s *pgStore) Get(ctx context.Context, id string) (*Listing, error) {
	if !isUUID(id) {
		return nil, errNotFound
	}
	return scanListing(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM listings WHERE id=$1 AND status <> 'removed'`, id))
}

func (s *pgStore) Update(ctx context.Context, id, sellerID, phone string, in *ListingInput) (*Listing, error) {
	set := []string{}
	args := []any{}
	add := func(col string, v any) {
		args = append(args, v)
		set = append(set, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.Title != nil {
		add("title", *in.Title)
	}
	if in.Description != nil {
		add("description", *in.Description)
	}
	if in.PriceETB != nil {
		add("price_etb", *in.PriceETB)
	}
	if in.Negotiable != nil {
		add("negotiable", *in.Negotiable)
	}
	if in.Category != nil {
		add("category_slug", *in.Category)
	}
	if in.Region != nil {
		add("region_code", *in.Region)
	}
	if in.City != nil {
		add("city", *in.City)
	}
	if in.Subcity != nil {
		add("subcity", *in.Subcity)
	}
	if in.Condition != nil {
		add("item_condition", *in.Condition)
	}
	if in.Attributes != nil {
		add("attributes", []byte(*in.Attributes))
	}
	if in.ImageURLs != nil {
		add("image_urls", pq.Array(*in.ImageURLs))
	}
	if in.ShowPhone != nil {
		if *in.ShowPhone {
			add("contact_phone", phone)
		} else {
			add("contact_phone", nil)
		}
	}
	if len(set) == 0 {
		return s.Get(ctx, id)
	}
	set = append(set, "updated_at=now()")
	args = append(args, id, sellerID)
	q := fmt.Sprintf(`UPDATE listings SET %s WHERE id=$%d AND seller_id=$%d AND status <> 'removed' RETURNING %s`,
		strings.Join(set, ", "), len(args)-1, len(args), cols)
	return scanListing(s.db.QueryRowContext(ctx, q, args...))
}

func (s *pgStore) SetStatus(ctx context.Context, id, sellerID, status string) error {
	if !isUUID(id) {
		return errNotFound
	}
	res, err := s.db.ExecContext(ctx, `UPDATE listings SET status=$3, updated_at=now() WHERE id=$1 AND seller_id=$2 AND status <> 'removed'`, id, sellerID, status)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

// buildSearch returns the WHERE clause, ORDER BY and args. Kept separate so it can be unit tested.
func buildSearch(p SearchParams) (where, order string, args []any) {
	conds := []string{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if !p.IncludeInactive {
		conds = append(conds, "status = 'active'", "expires_at > now()")
	} else {
		conds = append(conds, "status <> 'removed'")
	}
	if p.SellerID != "" {
		conds = append(conds, "seller_id = "+arg(p.SellerID))
	}
	if p.Query != "" {
		conds = append(conds, "search @@ plainto_tsquery('simple', "+arg(p.Query)+")")
	}
	if p.Category != "" {
		// category and its sub-categories: "vehicles" matches "vehicles" and "vehicles/cars"
		a := arg(p.Category)
		conds = append(conds, "(category_slug = "+a+" OR category_slug LIKE "+a+" || '/%')")
	}
	if p.Region != "" {
		conds = append(conds, "region_code = "+arg(p.Region))
	}
	if p.City != "" {
		conds = append(conds, "lower(city) = lower("+arg(p.City)+")")
	}
	if p.Condition != "" {
		conds = append(conds, "item_condition = "+arg(p.Condition))
	}
	if p.MinPrice != nil {
		conds = append(conds, "price_etb >= "+arg(*p.MinPrice))
	}
	if p.MaxPrice != nil {
		conds = append(conds, "price_etb <= "+arg(*p.MaxPrice))
	}
	featured := "(featured_until IS NOT NULL AND featured_until > now()) DESC, "
	switch p.Sort {
	case "price_asc":
		order = featured + "price_etb ASC, created_at DESC"
	case "price_desc":
		order = featured + "price_etb DESC, created_at DESC"
	case "relevance":
		if p.Query != "" {
			order = featured + "ts_rank(search, plainto_tsquery('simple', $" + fmt.Sprint(indexOf(args, p.Query)) + ")) DESC, created_at DESC"
		} else {
			order = featured + "created_at DESC"
		}
	default:
		order = featured + "created_at DESC"
	}
	return strings.Join(conds, " AND "), order, args
}

func indexOf(args []any, v any) int {
	for i, a := range args {
		if a == v {
			return i + 1
		}
	}
	return 1
}

func (s *pgStore) Search(ctx context.Context, p SearchParams) ([]Listing, int, error) {
	where, order, args := buildSearch(p)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM listings WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	q := fmt.Sprintf(`SELECT %s FROM listings WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, cols, where, order, len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Listing{}
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *l)
	}
	return items, total, rows.Err()
}

func (s *pgStore) IncrementViews(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE listings SET view_count = view_count + 1 WHERE id=$1`, id)
	return err
}

func (s *pgStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM listings`).Scan(&n)
	return n, err
}

func (s *pgStore) ExpireOld(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE listings SET status='expired', updated_at=now() WHERE status='active' AND expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return false
			}
		}
	}
	return true
}
