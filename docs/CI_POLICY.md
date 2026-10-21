# Selective CI and deployment

The `CI` workflow always reports the required `release-ready` check. Do not add
workflow-level path exclusions: they can leave protected PRs waiting forever.

| Change | PR | Push to main |
| --- | --- | --- |
| Root Markdown, Markdown under `docs/` except `docs/legal/`, CODEOWNERS, PR template | Scope tests and merge gate only | Same; no image or deployment |
| Application code, UI text/translations, legal text, dependencies, Docker/deploy/CI configuration, unknown paths | Full existing checks and image verification; no deployment | Full existing checks, image publication and existing deployment |
| Manual CI run | Full validation | Full validation and existing deployment when targeting main |

This is a conservative documentation allowlist, not a file-extension-wide skip.
`docs/legal/*.md` is imported into the application. Runtime text changes need
building and publishing just like code. Future runtime imports from documentation
must also update `tools/ci_scope.py` and its regression tests.

PR classification uses the merge-base-to-head diff for the entire PR. Pushes use
the event's before/after commits. Deletions and both sides of renames count. Missing
history, an empty diff, an unknown path or manual execution select full CI.
Classification failure cannot satisfy the merge gate. Full CI requires all seven
existing jobs to succeed; documentation-only CI requires all seven to be skipped
deliberately. Unexpected skips, failures and cancellations fail the gate.

New pushes cancel outdated verification runs for the same PR. Main runs are not
cancelled by this policy. Weekly security scans and tag releases are unchanged.

Documentation-only main commits intentionally do not advance the production
revision. Operators must track the last successful production deployment separately
from Git main. Before a later release, verify that any gap contains only approved
documentation paths, using `python tools/ci_scope.py --base <production-sha>
--head <main-sha>` (expect `full=false`), and that production SHA is an ancestor of
main. Missing deployment evidence or a runtime change in the gap remains a stop
condition. Migration approval must reference the actual production revision, never
substitute a documentation-only main SHA.

Local policy validation:

```sh
python -m unittest discover -s tools -p 'test_ci_scope.py' -v
actionlint .github/workflows/backend-ci.yml
```
