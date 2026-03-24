CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    short_id VARCHAR(32) NOT NULL UNIQUE,
    long_url TEXT NOT NULL,
    clicks INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    expired_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_urls_expired_at ON urls (expired_at);

