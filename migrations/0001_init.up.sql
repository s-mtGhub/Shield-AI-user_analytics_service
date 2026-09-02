CREATE TABLE IF NOT EXISTS user_activity_log (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, timestamp)
);

-- Serves COUNT(DISTINCT user_id) WHERE timestamp >= $1 AND timestamp < $2
-- (the daily/monthly active user queries) as an index-only scan.
CREATE INDEX IF NOT EXISTS idx_user_activity_log_timestamp_user_id
    ON user_activity_log (timestamp, user_id);
