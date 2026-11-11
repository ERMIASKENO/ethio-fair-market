-- listings: classified ads
CREATE TABLE listings (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id      uuid NOT NULL,
    title          text NOT NULL CHECK (char_length(title) BETWEEN 5 AND 120),
    description    text NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
    price_etb      numeric(14,2) NOT NULL CHECK (price_etb >= 0),   -- Ethiopian birr
    negotiable     boolean NOT NULL DEFAULT false,
    category_slug  text NOT NULL,
    region_code    text NOT NULL,
    city           text NOT NULL,
    subcity        text,
    item_condition text NOT NULL DEFAULT 'used' CHECK (item_condition IN ('new','used','not_applicable')),
    attributes     jsonb NOT NULL DEFAULT '{}'::jsonb,               -- category specific: {"year":2015,"mileage_km":90000}
    image_urls     text[] NOT NULL DEFAULT '{}',
    contact_phone  text,                                              -- shown only if the seller chose to
    status         text NOT NULL DEFAULT 'active' CHECK (status IN ('active','sold','expired','removed')),
    view_count     bigint NOT NULL DEFAULT 0,
    featured_until timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL DEFAULT now() + interval '30 days',
    -- 'simple' text search config: works for Amharic (Ge'ez script) and English without stemming
    search         tsvector GENERATED ALWAYS AS (
                     setweight(to_tsvector('simple', coalesce(title,'')), 'A') ||
                     setweight(to_tsvector('simple', coalesce(description,'')), 'B')) STORED
);

CREATE INDEX listings_search_idx        ON listings USING gin (search);
CREATE INDEX listings_active_new_idx    ON listings (created_at DESC) WHERE status = 'active';
CREATE INDEX listings_category_idx      ON listings (category_slug, created_at DESC) WHERE status = 'active';
CREATE INDEX listings_location_idx      ON listings (region_code, city) WHERE status = 'active';
CREATE INDEX listings_seller_idx        ON listings (seller_id, created_at DESC);
