# Ops ratio alert sampling (D144)

`error_rate`, `success_rate` and `upstream_error_rate` alerts require at least
10 SLA-eligible requests (`usage_logs` successes + non-business-limited request
errors) in the rule's existing evaluation window. Below that count, the metric
is unavailable: evaluation skips it and resets the sustained-breach counter,
without firing or resolving an existing event. Dashboard percentages are unchanged.

Each rule can override the default in its existing `filters` JSON:

```json
{"platform":"openai","min_requests":25}
```

Use the existing administrator alert-rule API (`GET /api/v1/admin/ops/alert-rules`,
`PUT /api/v1/admin/ops/alert-rules/:id`). Preserve the full rule and existing scope
filters when updating it. This is an API option; the current rule editor does
not expose a minimum-request control. The backend accepts positive integers
through 2147483647 (also decimal integer strings). Missing, zero, negative,
fractional or malformed values use 10. Set 1 explicitly for the legacy behavior.
No migration or seed rewrite is necessary: old rules receive the default.

New HTTP 400 unsupported-model errors (`model_not_found` code/type or the
specific `model is not supported` message) remain logged with their existing
attribution but are classified `is_business_limited=true`, like local model
configuration rejections. Classification happens after final upstream/stream
snapshots, before enqueueing the log. A non-400 upstream status or account-auth
phase prevents this new exclusion. Other 400s, 401/403, 429, 5xx and timeouts
retain their existing classification. All existing SLA readers honor the flag;
no SQL changes, historical reclassification or alert-state backfill is performed.

## Configuring discovery for ChatGPT/Codex groups

The model catalog is configurable, with static defaults as a fallback. In
**Administration → Groups → Edit the OpenAI group → Model Display List
(模型展示列表)**, enable the display list and select only models verified for the
accounts in that group. For the D144 reported list, deselect `gpt-5.4`,
`gpt-5.4-mini`, `gpt-5.2`, and `gpt-5.3-codex-spark`; retain the other existing
entries, including the individual `gpt-image-*` IDs. Save the group. The stored
field is `models_list_config: {"enabled":true,"models":[...]}`; the array defines
discovery order. It is available through the existing group create/update API.
Do not change `model_allowlist` to achieve a display-only fix.

This affects that group's discovery, including static fallback catalogs, without
changing request permissions or platform API-key accounts in other groups. An
existing enabled request allowlist still bounds the display selection. Avoid
mixing subscription and platform accounts in a group if they need different
public discovery policies. Account `credentials.model_mapping` controls routing
and may project aliases; changing it is a different operation. Pinned upstream
catalog mode (`codex_models_manifest_config`) is also subject to group display
selection. No production configuration was read or changed for D144.
