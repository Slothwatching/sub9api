# Product decisions and operations record

`Slothwatching/sub9api` contains the product source code. The companion private repository, `Slothwatching/sub9api-ops`, contains this same project's canonical decisions, release/validation records, infrastructure inventory, and operations runbooks. They must be read together; the private repository is not another product.

Before changing this repository, read the current constraints and relevant history in `sub9api-ops/decisions/sub9api-development-log.md` and follow its `runbooks/change-delivery-flow.md`. The local compatibility path is `/Users/victorwu/Developer/sub9api-development-log.md`. A repository clone without access to the private record does not establish the current decision state; report that limitation instead of reviving an older decision.

For each product decision or change batch, record the reason and authorization, affected scope, decision references, branch/PR/commit, validation, release status, and remaining work in the private log. Record material operations findings, configuration changes, incidents, backups, rollbacks, and upstream synchronization there as well, including when no product code changed. Update the record after each status change and commit/push authorized records without waiting for a separate reminder. Preserve superseded decisions as history. Never place credentials, private user data, or database dumps in either repository.

Keep the product source and public technical documentation here. Keep the canonical decision history and private operational detail in `sub9api-ops`; use links or identifiers to connect related records instead of maintaining two divergent decision logs.

## Intent-first collaboration (D74, 2026-10-21)

The canonical record is `../sub9api-ops/decisions/sub9api-development-log.md` when both repositories are sibling checkouts, including on Windows. Resolve the actual ops checkout when using a worktree. The old macOS absolute paths below are historical compatibility paths, not a requirement to create a duplicate log.

Before implementation, record the user's request, why it matters, expected outcome, constraints, and a stable decision ID in that private log. Unknown motivation must be marked pending, not invented. After every meaningful commit, push, PR, or review attempt, append its actual result and relevant branch/commit/PR reference. Record rejected, failed, cancelled, superseded, and unmerged work too. Preserve earlier reasoning and link replacement decisions. Push authorized ops records even when product changes are rejected; an ops commit is not product approval.

D76 (2026-10-21) supersedes D74's mandatory peer/two-maintainer approval requirement: this small team may submit and merge its own PRs once required CI passes. Keep PRs, strict release-ready checks and administrator protection; no peer or code-owner approval is required. Optional review remains available. The daily upstream routine follows D66's existing release authorization and guardrails without an additional human approval gate. Every intent and attempt outcome must still be recorded and pushed to the canonical private log. The previous review requirement and its rationale remain historical records in ops; this policy grants no new production release scope.

Public PRs reference only a decision ID and a safe technical summary, never private log contents or credentials. External contributors without ops access explain their intent in the PR; an internal maintainer records it privately before accepting the change. If an internal maintainer lacks ops access, report the missing record rather than inventing a public replacement. This is a documented collaboration workflow, not automatic cross-repository synchronization.
