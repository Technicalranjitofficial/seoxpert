-- ClickHouse schemas for SeoXpert
-- Run against your existing ClickHouse instance (:8123 HTTP or :9000 native)
-- clickhouse-client --query "$(cat 001_schemas.sql)"

CREATE DATABASE IF NOT EXISTS seoxpert;

-- ── Rank History ──────────────────────────────────────────────────────────────
-- Stores daily keyword ranking positions. Designed for billions of rows.
-- Partitioned by month, ordered for fast per-project/keyword queries.

CREATE TABLE IF NOT EXISTS seoxpert.rank_history (
    project_id  UUID,
    keyword     String,
    domain      String,
    position    UInt16,          -- 0 = not ranking in top 100
    prev_position UInt16,
    engine      LowCardinality(String),  -- 'google' | 'bing'
    device      LowCardinality(String),  -- 'desktop' | 'mobile'
    location    LowCardinality(String),  -- 'us' | 'gb' | ...
    date        Date,
    checked_at  DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, keyword, date, engine, device)
TTL date + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;

-- ── Audit Metrics (time-series per project) ──────────────────────────────────

CREATE TABLE IF NOT EXISTS seoxpert.audit_metrics (
    project_id      UUID,
    audit_id        UUID,
    url             String,
    score           UInt8,
    critical_issues UInt16,
    warning_issues  UInt16,
    info_issues     UInt16,
    total_pages     UInt16,
    crawl_ms_avg    UInt32,
    date            Date DEFAULT today(),
    created_at      DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date)
TTL date + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

-- ── Traffic Analytics (future: GSC integration) ───────────────────────────────

CREATE TABLE IF NOT EXISTS seoxpert.traffic_analytics (
    project_id  UUID,
    url         String,
    keyword     String,
    clicks      UInt32,
    impressions UInt32,
    ctr         Float32,
    avg_position Float32,
    date        Date
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, url)
TTL date + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
