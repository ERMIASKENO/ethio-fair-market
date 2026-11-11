-- identity: users and one-time passwords (phone login)
CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone        text NOT NULL UNIQUE,              -- E.164, e.g. +251911234567
    display_name text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_login   timestamptz
);

CREATE TABLE otp_codes (
    id          bigserial PRIMARY KEY,
    phone       text NOT NULL,
    code_hash   text NOT NULL,                      -- sha256(salt || code), never the code itself
    salt        text NOT NULL,
    attempts    int  NOT NULL DEFAULT 0,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_phone_created_idx ON otp_codes (phone, created_at DESC);
