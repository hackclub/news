-- TimescaleDB schema for email view tracking
-- Run this on your TimescaleDB instance

CREATE TABLE IF NOT EXISTS email_views (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_id TEXT NOT NULL,
    email_id TEXT NOT NULL,
    PRIMARY KEY (session_id, email_id, time_bucket('5 minutes', time))
);

SELECT create_hypertable('email_views', 'time', if_not_exists => TRUE);

-- Continuous aggregate for fast view counts
CREATE MATERIALIZED VIEW IF NOT EXISTS email_view_counts
WITH (timescaledb.continuous) AS
SELECT 
    email_id,
    COUNT(DISTINCT session_id) as view_count
FROM email_views
GROUP BY email_id
WITH NO DATA;

-- Refresh policy: update every 5 minutes
SELECT add_continuous_aggregate_policy('email_view_counts',
    start_offset => INTERVAL '1 hour',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '5 minutes',
    if_not_exists => TRUE);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_email_views_email_id ON email_views(email_id, time DESC);
