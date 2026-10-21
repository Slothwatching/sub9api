"""Regression tests for CI skipping and the required merge gate."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from ci_scope import FULL_JOBS, changed_paths, documentation_only, gate_errors, main


class ScopeTests(unittest.TestCase):
    def test_docs_and_runtime_boundary(self):
        self.assertTrue(documentation_only(["README.md", "AGENTS.md", "docs/DECISIONS.md", ".github/CODEOWNERS"]))
        for path in ["docs/legal/terms.md", "frontend/src/i18n/zh.ts", "backend/prompt.md",
                     "deploy/DOCKER.md", "docs/tool.py", "Dockerfile", ".dockerignore",
                     ".github/workflows/backend-ci.yml", "tools/ci_scope.py", "unknown.txt"]:
            with self.subTest(path=path):
                self.assertFalse(documentation_only(["README.md", path]))
        self.assertFalse(documentation_only([]))

    def test_gate_rejects_failure_cancellation_and_unexpected_skip(self):
        for mode in ("true", "false"):
            expected = "success" if mode == "true" else "skipped"
            needs = {name: {"result": expected} for name in FULL_JOBS}
            needs["changes"] = {"result": "success", "outputs": {"full": mode}}
            self.assertEqual(gate_errors(needs), [])
            for result in ("failure", "cancelled", "skipped" if mode == "true" else "success"):
                needs["test"]["result"] = result
                self.assertIn("test", gate_errors(needs))
        for changes in ({}, {"result": "failure"}, {"result": "success", "outputs": {"full": ""}}):
            self.assertTrue(gate_errors({"changes": changes}))

    def test_renamed_runtime_file_is_not_documentation(self):
        with tempfile.TemporaryDirectory() as directory:
            def git(*args):
                return subprocess.check_output(["git", "-C", directory, *args], text=True).strip()
            git("init", "-q")
            git("config", "user.name", "CI test")
            git("config", "user.email", "ci@example.invalid")
            Path(directory, "runtime.py").write_text("print('example')\n")
            git("add", ".")
            git("commit", "-qm", "runtime")
            base = git("rev-parse", "HEAD")
            git("mv", "runtime.py", "README.md")
            git("commit", "-qm", "rename")
            head = git("rev-parse", "HEAD")
            previous = os.getcwd()
            try:
                os.chdir(directory)
                paths = changed_paths(base, head)
            finally:
                os.chdir(previous)
            self.assertEqual(set(paths), {"runtime.py", "README.md"})
            self.assertFalse(documentation_only(paths))

    def test_manual_and_missing_push_base_run_full(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory, "output")
            event = Path(directory, "event.json")
            event.write_text(json.dumps({"before": "0" * 40, "after": "1" * 40}))
            for name in ("workflow_dispatch", "push"):
                with patch.dict(os.environ, {"GITHUB_EVENT_NAME": name, "GITHUB_EVENT_PATH": str(event),
                                             "GITHUB_OUTPUT": str(output)}), patch("sys.argv", ["ci_scope.py"]):
                    with patch("ci_scope.changed_paths", side_effect=ValueError("missing base")):
                        main()
                self.assertIn("full=true", output.read_text())
                output.unlink()

    def test_pr_uses_whole_branch_and_push_uses_before_after(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory, "output")
            event = Path(directory, "event.json")
            base, head, ancestor = "a" * 40, "b" * 40, "c" * 40
            scenarios = [
                ("pull_request", {"pull_request": {"base": {"sha": base}, "head": {"sha": head}}}, ancestor),
                ("push", {"before": base, "after": head}, base),
            ]
            for name, payload, expected_base in scenarios:
                event.write_text(json.dumps(payload))
                for paths, expected in [(["README.md"], "false"), (["backend/app.go", "README.md"], "true")]:
                    with patch.dict(os.environ, {"GITHUB_EVENT_NAME": name, "GITHUB_EVENT_PATH": str(event),
                                                 "GITHUB_OUTPUT": str(output)}), patch("sys.argv", ["ci_scope.py"]):
                        with patch("ci_scope.subprocess.check_output", return_value=ancestor + "\n"), \
                             patch("ci_scope.changed_paths", return_value=paths) as diff:
                            main()
                            diff.assert_called_once_with(expected_base, head)
                    self.assertEqual(output.read_text().strip(), "full=" + expected)
                    output.unlink()


if __name__ == "__main__":
    unittest.main()
