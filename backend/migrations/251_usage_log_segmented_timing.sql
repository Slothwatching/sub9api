-- Fork-local observability only. Old rows stay NULL; legacy first_token_ms is unchanged.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS user_queue_ms INTEGER,
    ADD COLUMN IF NOT EXISTS account_queue_ms INTEGER,
    ADD COLUMN IF NOT EXISTS account_selection_ms INTEGER,
    ADD COLUMN IF NOT EXISTS upstream_connect_ms INTEGER,
    ADD COLUMN IF NOT EXISTS upstream_tls_ms INTEGER,
    ADD COLUMN IF NOT EXISTS upstream_first_byte_ms INTEGER,
    ADD COLUMN IF NOT EXISTS first_data_ms INTEGER,
    ADD COLUMN IF NOT EXISTS first_semantic_ms INTEGER,
    ADD COLUMN IF NOT EXISTS first_visible_content_ms INTEGER,
    ADD COLUMN IF NOT EXISTS client_first_content_ms INTEGER,
    ADD COLUMN IF NOT EXISTS user_duration_ms INTEGER,
    ADD COLUMN IF NOT EXISTS attempt_count INTEGER,
    ADD COLUMN IF NOT EXISTS last_attempt_ms INTEGER,
    ADD COLUMN IF NOT EXISTS last_failed_attempt_ms INTEGER;
