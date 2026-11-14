# Gateway segmented latency (D143, local implementation)

OpenAI HTTP Responses and Chat Completions share a request clock installed at the
first gateway route middleware, before body limits/read, authentication, request
audit, concurrency admission, reservation and scheduling. Context detachment and
outer 429/5xx account failover retain it. Instrumentation does not change billing,
rate limits, account scheduling feedback or retry policy.

`first_token_ms` and `duration_ms` retain their historical definitions and charts:
they are attempt-local, with protocol/configuration-specific TTFT. In particular,
legacy transcoded Chat may count its first upstream data payload. Compare new
traffic with `client_first_content_ms`; do not mix the two definitions in one
historical series.

All new columns are nullable integers. Old rows remain NULL. Zero means observed
within the first millisecond; NULL means no supported observation, never zero.

| Column | Meaning |
| --- | --- |
| `user_queue_ms` | Cumulative user-slot acquisition/wait duration, including immediate admission work. |
| `account_queue_ms` | Cumulative account-slot helper duration, including its admission/profit check. Immediate slot probes inside scheduling belong to selection. |
| `account_selection_ms` | Cumulative scheduler duration across selections and outer retries. |
| `upstream_connect_ms` | Cumulative actual HTTP `GetConn` → `GotConn`: pool wait, DNS/TCP, proxy CONNECT and TLS; reused connections usually approach zero. |
| `upstream_tls_ms` | Cumulative TLS handshake duration where the transport emits httptrace callbacks. Included in connection duration; do not add twice. |
| `upstream_first_byte_ms` | Ingress → first upstream response byte (including failed HTTP attempts), or first returned response header/body read if the transport lacks that callback. |
| `first_data_ms` | Ingress → first nonempty upstream SSE data event, excluding `[DONE]`; may be role/lifecycle/usage. For buffered JSON, first complete observed envelope. |
| `first_semantic_ms` | Ingress → first semantic upstream event. Responses structural output progress qualifies; Chat needs content/tool progress. Lifecycle, metadata, role-only, usage-only and keepalive events do not qualify. |
| `first_visible_content_ms` | Ingress → first observed usable upstream text/reasoning/refusal, tool arguments/input, audio or image content. Structural events with empty content do not qualify. |
| `client_first_content_ms` | Ingress → first downstream Flush after a successful write carrying usable content. Measures the gateway flush, not network delivery or rendering on the user's device. |
| `user_duration_ms` | Ingress → completion of forwarding and synchronous handler bookkeeping, captured before enqueueing usage. Includes failover; excludes asynchronous billing/storage. |
| `attempt_count` | Outer handler Forward invocations, including failed invocations; internal protocol repair/HTTP retries within Forward are part of the same attempt. |
| `last_attempt_ms` | Duration of the last outer Forward invocation. Existing duration_ms remains available too. |
| `last_failed_attempt_ms` | Duration of the last outer Forward returning an error, or NULL if none. |

The first visible event and first flush are independent observations: suppressed,
rewritten or buffered upstream content is not proof of a client flush. Completed
buffered JSON content is flushed after its successful write so its usage snapshot
can include the same client metric. SSE heartbeat/lifecycle flushes never count.
Definitions are independent of the administrator's legacy TTFT-mode setting.

HTTP observability uses composed httptrace hooks on the actual native/plugin
outbound request plus response-body observation. Unsupported transports/plugins
leave connection/TLS NULL. WebSocket frames and Anthropic/Gemini event formats
are not covered by the new content/flush metrics; existing metrics remain intact.
Images use the same observer when their payload fits its bounds.

Observers use bounded 64 KiB line/event buffers and stop after the first
usable content. Complete JSON writes/reads are inspected in place without copying, including
large buffered image responses. Oversized fragmented events/envelopes are skipped rather than copying entire
image/tool payloads; a later qualifying event can still be measured. An oversized fragmented
buffered response or a stream with only oversized content can therefore have NULL
content/flush timestamps. SSE event names and fragmented/multiline data are
supported. No payload is logged. New usage fields only attach to existing usage
records: failures that never produce a usage record do not gain one.

The usage API and admin Excel export expose the fields without changing charts.
For ad hoc operations queries, AVG naturally excludes old/missing observations:

```sql
SELECT model,
       count(client_first_content_ms) AS measured_requests,
       avg(client_first_content_ms) AS avg_client_first_content_ms,
       avg(user_queue_ms) AS avg_user_queue_ms,
       avg(account_queue_ms) AS avg_account_queue_ms,
       avg(account_selection_ms) AS avg_account_selection_ms,
       avg(attempt_count) AS avg_attempt_count
FROM usage_logs
WHERE created_at >= now() - interval '1 day'
GROUP BY model;
```

Migration 251 uses only `ADD COLUMN IF NOT EXISTS ... INTEGER`, with no defaults,
backfill or billing changes. The generated ent schema and all raw SQL insert,
batch/CTE and scan paths include these fields. Deploy compatibility fingerprints
must be computed using the existing migration_review process; a fingerprint alone
does not prove PostgreSQL baseline/upgrade/rollback/reupgrade compatibility.
