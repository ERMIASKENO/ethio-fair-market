package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Listing is a classified ad.
type Listing struct {
	ID            string          `json:"id"`
	SellerID      string          `json:"seller_id"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	PriceETB      float64         `json:"price_etb"`
	Negotiable    bool            `json:"negotiable"`
	CategorySlug  string          `json:"category"`
	RegionCode    string          `json:"region"`
	City          string          `json:"city"`
	Subcity       string          `json:"subcity,omitempty"`
	Condition     string          `json:"condition"`
	Attributes    json.RawMessage `json:"attributes"`
	ImageURLs     []string        `json:"image_urls"`
	ContactPhone  string          `json:"contact_phone,omitempty"`
	Status        string          `json:"status"`
	ViewCount     int64           `json:"view_count"`
	Featured      bool            `json:"featured"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
}

// ListingInput is the create/update payload. Pointers = "not sent" for PATCH.
type ListingInput struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	PriceETB    *float64         `json:"price_etb"`
	Negotiable  *bool            `json:"negotiable"`
	Category    *string          `json:"category"`
	Region      *string          `json:"region"`
	City        *string          `json:"city"`
	Subcity     *string          `json:"subcity"`
	Condition   *string          `json:"condition"`
	Attributes  *json.RawMessage `json:"attributes"`
	ImageURLs   *[]string        `json:"image_urls"`
	ShowPhone   *bool            `json:"show_phone"`
}

// FieldErrors maps field -> message.
type FieldErrors map[string]string

func (f FieldErrors) Error() string {
	parts := make([]string, 0, len(f))
	for k, v := range f {
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, "; ")
}

var conditions = map[string]bool{"new": true, "used": true, "not_applicable": true}

// Validate checks the input. create=true requires all mandatory fields.
func (in *ListingInput) Validate(create bool) error {
	errs := FieldErrors{}
	trim := func(p *string) {
		if p != nil {
			*p = strings.TrimSpace(*p)
		}
	}
	trim(in.Title)
	trim(in.Description)
	trim(in.Category)
	trim(in.Region)
	trim(in.City)
	trim(in.Subcity)

	required := func(name string, p *string) {
		if create && (p == nil || *p == "") {
			errs[name] = "is required"
		}
	}
	required("title", in.Title)
	required("category", in.Category)
	required("region", in.Region)
	required("city", in.City)
	if create && in.PriceETB == nil {
		errs["price_etb"] = "is required"
	}

	if in.Title != nil && *in.Title != "" {
		if n := utf8.RuneCountInString(*in.Title); n < 5 || n > 120 {
			errs["title"] = "must be 5-120 characters"
		}
	}
	if in.Description != nil && utf8.RuneCountInString(*in.Description) > 5000 {
		errs["description"] = "must be at most 5000 characters"
	}
	if in.PriceETB != nil && (*in.PriceETB < 0 || *in.PriceETB > 1_000_000_000) {
		errs["price_etb"] = "must be between 0 and 1,000,000,000 birr"
	}
	if in.Condition != nil && !conditions[*in.Condition] {
		errs["condition"] = "must be new, used or not_applicable"
	}
	if in.Attributes != nil {
		var m map[string]any
		if err := json.Unmarshal(*in.Attributes, &m); err != nil {
			errs["attributes"] = "must be a JSON object"
		} else if len(m) > 30 {
			errs["attributes"] = "too many attributes (max 30)"
		}
	}
	if in.ImageURLs != nil {
		if len(*in.ImageURLs) > 10 {
			errs["image_urls"] = "max 10 images"
		}
		for _, u := range *in.ImageURLs {
			if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "/media/") {
				errs["image_urls"] = fmt.Sprintf("invalid image url %q", u)
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// SearchParams are the query filters of GET /v1/listings.
type SearchParams struct {
	Query, Category, Region, City, Condition, Sort string
	MinPrice, MaxPrice                             *float64
	SellerID                                       string
	IncludeInactive                                bool
	Page, PageSize                                 int
}
