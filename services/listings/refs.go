package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RefChecker validates category and location references against the catalog and locations services.
// Positive answers are cached for 10 minutes so creating ads does not hammer the other services.
type RefChecker struct {
	catalogURL, locationsURL string
	client                   *http.Client
	mu                       sync.Mutex
	cache                    map[string]time.Time
}

// NewRefChecker builds a checker. Empty URLs disable the check (local development).
func NewRefChecker(catalogURL, locationsURL string) *RefChecker {
	return &RefChecker{catalogURL: catalogURL, locationsURL: locationsURL,
		client: &http.Client{Timeout: 2 * time.Second}, cache: map[string]time.Time{}}
}

func (c *RefChecker) exists(ctx context.Context, u string) (bool, error) {
	c.mu.Lock()
	if t, ok := c.cache[u]; ok && time.Since(t) < 10*time.Minute {
		c.mu.Unlock()
		return true, nil
	}
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("reference check unavailable: %w", err)
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		c.mu.Lock()
		c.cache[u] = time.Now()
		c.mu.Unlock()
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("reference check: status %d", resp.StatusCode)
	}
}

// Check returns field errors for unknown references, or an error if a service is down.
func (c *RefChecker) Check(ctx context.Context, category, region, city string) (FieldErrors, error) {
	errs := FieldErrors{}
	if c.catalogURL != "" && category != "" {
		ok, err := c.exists(ctx, c.catalogURL+"/v1/categories/"+escapeSegments(category))
		if err != nil {
			return nil, err
		}
		if !ok {
			errs["category"] = "unknown category"
		}
	}
	if c.locationsURL != "" && region != "" {
		u := c.locationsURL + "/v1/locations/regions/" + url.PathEscape(region)
		if city != "" {
			u += "/cities/" + url.PathEscape(city)
		}
		ok, err := c.exists(ctx, u)
		if err != nil {
			return nil, err
		}
		if !ok {
			errs["city"] = "unknown region or city"
		}
	}
	return errs, nil
}

// escapeSegments escapes each part of a slug like "vehicles/cars" but keeps the slashes.
func escapeSegments(slug string) string {
	parts := strings.Split(slug, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
