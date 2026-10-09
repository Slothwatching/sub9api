# Product decisions and operations record

`Slothwatching/sub9api` contains the product source code. The companion private repository, `Slothwatching/sub9api-ops`, contains this same project's canonical decisions, release/validation records, infrastructure inventory, and operations runbooks. They must be read together; the private repository is not another product.

Before changing this repository, read the current constraints and relevant history in `sub9api-ops/decisions/sub9api-development-log.md` and follow its `runbooks/change-delivery-flow.md`. The local compatibility path is `/Users/victorwu/Developer/sub9api-development-log.md`. A repository clone without access to the private record does not establish the current decision state; report that limitation instead of reviving an older decision.

For each product decision or change batch, record the reason and authorization, affected scope, decision references, branch/PR/commit, validation, release status, and remaining work in the private log. Record material operations findings, configuration changes, incidents, backups, rollbacks, and upstream synchronization there as well, including when no product code changed. Update the record after each status change and commit/push authorized records without waiting for a separate reminder. Preserve superseded decisions as history. Never place credentials, private user data, or database dumps in either repository.

Keep the product source and public technical documentation here. Keep the canonical decision history and private operational detail in `sub9api-ops`; use links or identifiers to connect related records instead of maintaining two divergent decision logs.
