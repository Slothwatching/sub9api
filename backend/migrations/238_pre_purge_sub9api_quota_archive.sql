-- Preserve the exact pre-upgrade unlimited quota rows before upstream 238_purge.
-- This private database archive is not exposed by any API and has no foreign key:
-- deleting an operational row must not destroy its upgrade recovery evidence.
-- Restore only after reviewing current activity; never replay old counters blindly.
CREATE TABLE IF NOT EXISTS sub9api_quota_upgrade_archive (
    original_id BIGINT PRIMARY KEY,
    row_data JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO sub9api_quota_upgrade_archive (original_id, row_data)
SELECT id, to_jsonb(q)
FROM user_platform_quotas q
WHERE daily_limit_usd IS NULL
  AND weekly_limit_usd IS NULL
  AND monthly_limit_usd IS NULL
ON CONFLICT (original_id) DO NOTHING;
