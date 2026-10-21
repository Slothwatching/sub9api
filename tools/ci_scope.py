"""Fork-local CI policy: only explicitly non-runtime documentation may skip CI."""

import argparse
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess


FULL_JOBS = {"shell", "test", "frontend", "golangci-lint", "release-helpers", "security", "image"}


def documentation_only(paths):
    def safe(path):
        p = PurePosixPath(path)
        # docs/legal is imported into the frontend and must build/deploy.
        if path.startswith("docs/legal/"):
            return False
        return (
            (len(p.parts) == 1 and p.suffix == ".md")
            or (path.startswith("docs/") and p.suffix == ".md")
            or path in {".github/CODEOWNERS", ".github/pull_request_template.md"}
        )

    return bool(paths) and all(safe(path) for path in paths)


def changed_paths(base, head):
    # Disable rename detection: both deleted and added paths must be classified.
    for revision in (base, head):
        if not re.fullmatch(r"[0-9a-f]{40}", revision or ""):
            raise ValueError("Expected a full commit SHA")
    result = subprocess.check_output(
        ["git", "diff", "--no-renames", "--name-only", "-z", base, head, "--"]
    )
    return [p.decode("utf-8", errors="surrogateescape") for p in result.split(b"\0") if p]


def gate_errors(needs):
    changes = needs.get("changes", {})
    mode = changes.get("outputs", {}).get("full")
    if changes.get("result") != "success" or mode not in {"true", "false"}:
        return ["change classification failed or missing"]
    expected = "success" if mode == "true" else "skipped"
    return [name for name in sorted(FULL_JOBS) if needs.get(name, {}).get("result") != expected]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base")
    parser.add_argument("--head")
    parser.add_argument("--gate", action="store_true")
    args = parser.parse_args()
    if args.gate:
        errors = gate_errors(json.loads(os.environ["RESULTS"]))
        if errors:
            raise SystemExit("Required checks failed: " + ", ".join(errors))
        return

    paths = []
    full = True
    try:
        if args.base and args.head:
            paths = changed_paths(args.base, args.head)
            full = not documentation_only(paths)
        elif os.environ.get("GITHUB_EVENT_NAME") in {"pull_request", "push"}:
            event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding="utf-8"))
            if "pull_request" in event:
                pr = event["pull_request"]
                # PR scope is the entire branch diff, not just the last pushed commit.
                base = subprocess.check_output(
                    ["git", "merge-base", pr["base"]["sha"], pr["head"]["sha"]], text=True
                ).strip()
                head = pr["head"]["sha"]
            else:
                base, head = event["before"], event["after"]
            paths = changed_paths(base, head)
            full = not documentation_only(paths)
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"Cannot establish documentation-only scope; running full CI ({type(error).__name__}).")

    value = str(full).lower()
    print(f"full={value}; changed_paths={len(paths)}")
    if output := os.environ.get("GITHUB_OUTPUT"):
        with open(output, "a", encoding="utf-8") as stream:
            stream.write(f"full={value}\n")


if __name__ == "__main__":
    main()
