#!/usr/bin/env python3
"""Bounded document correction checks; no production or historical writes."""
import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[5]
AREA = pathlib.Path(__file__).resolve().parent
BASE = (AREA / "base_commit").read_text().strip()
CLI = "tools/gomad3/CLI.md"
EVIDENCE = ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md"


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inventory():
    path = ROOT / ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit/inventory.json"
    return json.loads(path.read_text())["source_before"]["current"]


def protected():
    parent = AREA.parents[1]
    return {str(p.relative_to(ROOT)): sha(p) for p in parent.rglob("*")
            if p.is_file() and AREA not in p.parents and p.name != "documentation-evidence.md"}


if sys.argv[1:] == ["snapshot"]:
    result = {"base_commit": BASE, "protected": protected(), "source": inventory()}
    (AREA / "before.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({"protected_files": len(result["protected"]), "source_files": len(result["source"])}))
    sys.exit(0)

before = json.loads((AREA / "before.json").read_text())
registration_inventory = json.loads((ROOT / ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit/cli-inventory.json").read_text())["current"]["gomad"]
text = (ROOT / CLI).read_text()
original = git("show", BASE + ":" + CLI)
checks = []


def check(name, ok, detail):
    checks.append({"check": name, "passed": bool(ok), "detail": detail})


details = {
    "env": (["explore", "plan", "qualify", "NAME=VALUE", "none", "ASCII", "Duplicate", "TZ=UTC"],
            ["cmd/gomad/internal/cli/cli.go:582", "cmd/gomad/internal/cli/qualify.go:74", "runner/runner.go:1410"]),
    "io-ro-mount": (["same three commands", "HOST_DIRECTORY=TARGET_DIRECTORY", "no mounts", "working-dir", "symlink", "overlapping", "EROFS", "16 MiB", "64 MiB", "plan"],
                    ["cmd/gomad/internal/cli/cli.go:584", "cmd/gomad/internal/cli/qualify.go:76", "deterministicio/readonlymount/config.go:16", "deterministicio/readonlymount/capture.go:34", "runner/portable_plan_mounts.go:45"]),
    "max-bytes": (["minimize", "SIZE", "final", "manifest", "1 MiB", "uint64", "positive", "GiB", "status 3", "Intermediate", "unchanged parent"],
                  ["cmd/gomad/internal/cli/cli.go:1047", "runner/minimize_operation.go:208", "runner/minimize_operation.go:635"]),
    "min-free-bytes": (["qualify-set", "SIZE", "current user", "2GiB", "equality", "status 3", "positive", "GiB", "--check"],
                       ["cmd/gomad/internal/cli/qualify_set.go:38", "qualification/set/freespace.go:13", "qualification/set/set.go:596"]),
    "observed": (["replay", "DIR", "stdout", "stderr", "default empty", "replaces", "status 3", "--verify-only"],
                 ["cmd/gomad/internal/cli/cli.go:992", "runner/replay_operation.go:113", "runner/replay_operation.go:721"]),
    "prune-qualified-artifacts": (["qualify-set", "boolean", "false", "replay matches", "checkpoints", "deletes", "keeps", "artifacts_pruned", "status 3"],
                                  ["cmd/gomad/internal/cli/qualify_set.go:35", "qualification/set/set.go:536", "qualification/set/set.go:605", "qualification/set/prune.go:29"]),
    "terminate-grace": (["explore", "plan", "qualify", "DURATION", "2s", "nonnegative", "both", "zero", "SIGTERM", "SIGKILL"],
                        ["cmd/gomad/internal/cli/cli.go:548", "cmd/gomad/internal/cli/qualify.go:51", "runner/runner.go:1286", "runner/internal/execution/supervisor_unix.go:309"]),
    "toolchain-root": (["doctor", "analyze", "explore", "plan", "qualify", "replay", "minimize", "resume", "execute-shard", "default is empty", "GOMAD3_TOOLCHAIN_DIR", "gomad3-install.json", "precedence", "absolute", "clean", "non-root"],
                       ["cmd/gomad/internal/cli/cli.go:359", "cmd/gomad/internal/cli/analyze.go:87", "cmd/gomad/internal/cli/qualify.go:53", "cmd/gomad/internal/cli/resume.go:28", "cmd/gomad/internal/cli/campaign_shards.go:30", "toolchain/installation.go:38", "toolchain/installation.go:119"]),
    "world-transition-limit": (["explore", "plan", "qualify", "SIZE", "encoded", "bytes", "64MiB", "positive", "GiB", "overflow", "snapshot/model"],
                               ["cmd/gomad/internal/cli/cli.go:571", "cmd/gomad/internal/cli/qualify.go:65", "world/recording.go:184", "runner/internal/execution/worldrecord.go:142"]),
}
for flag, (required, sources) in details.items():
    paragraphs = [p for p in text.split("\n\n") if re.search(r"--" + re.escape(flag) + r"(?=[= `])", p)]
    adequate = [p for p in paragraphs if len(p.split()) >= 35 and all(term.lower() in p.lower() for term in required)]
    source_rows = []
    for source in sources:
        file, line = source.rsplit(":", 1)
        path = ROOT / "tools/gomad3" / file
        lines = path.read_text().splitlines()
        source_rows.append({"path": file, "line": int(line), "text": lines[int(line) - 1], "sha256": sha(path)})
    registered = [row for row in registration_inventory if '"' + flag + '"' in row["source"]]
    registrations_match = registered and all((ROOT / "tools/gomad3" / row["path"]).read_text().splitlines()[row["line"] - 1].strip() == row["source"].strip() for row in registered)
    check("detail:" + flag, adequate and registrations_match,
          {"paragraphs": len(paragraphs), "required": required, "doc_lines": [text[:text.index(p)].count("\n") + 1 for p in adequate], "sources": source_rows, "registered": registered})

fences = lambda value: re.findall(r"^```(?:sh|bash)\n.*?^```", value, re.M | re.S)
check("shell_fences", fences(text) == fences(original), len(fences(text)))
check("command_index", text.split("## Command index\n", 1)[1] == original.split("## Command index\n", 1)[1], "exact bytes")
spec = "tools/gomad3/SPEC.md"
check("spec_ids", (ROOT / spec).read_text() == git("show", BASE + ":" + spec), "SPEC unchanged")
check("fence_balance", sum(line.startswith("```") for line in text.splitlines()) % 2 == 0, "balanced")
errors = []
for document in [CLI, EVIDENCE]:
    value = (ROOT / document).read_text()
    for target in re.findall(r"\[[^\]]+\]\(([^)]+)\)", value):
        if "://" in target or target.startswith("mailto:"):
            continue
        path, _, anchor = target.partition("#")
        resolved = (ROOT / document).parent / path if path else ROOT / document
        if not resolved.exists():
            errors.append(document + " -> " + target)
        elif anchor and resolved.suffix == ".md":
            headings = []
            for heading in re.findall(r"^#+ (.+)$", resolved.read_text(), re.M):
                slug = re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-")
                headings.append(slug)
            if anchor not in headings:
                errors.append(document + " -> " + target)
check("local_links", not errors, errors)
changed = [p for p, digest in before["source"].items() if sha(ROOT / "tools/gomad3" / p) != digest]
check("978_source_inventory", not changed or changed == ["CLI.md"], {"files": len(before["source"]), "changed": changed})
now_protected = protected()
protected_changes = [p for p, digest in before["protected"].items() if now_protected.get(p) != digest]
check("historical_artifacts", not protected_changes, {"files": len(before["protected"]), "changed": protected_changes})
check("admission_identity", sha(AREA / "source-admission.md") == "d822e4dd0bb94ebcd8929a5ff7f0c745d7dba692c07e88c8086f8b4eadbdb3d7", "root-owned source-admission.md")
tracked = git("diff", "--name-only", BASE).splitlines()
check("allowed_tracked_diff", set(tracked) <= {CLI, EVIDENCE}, tracked)
check("root_owned_commits", git("rev-parse", "HEAD").strip() == BASE, git("rev-parse", "HEAD").strip())
passed = all(c["passed"] for c in checks)
print(json.dumps({"passed": passed, "checks": checks}, indent=2))
sys.exit(0 if passed else 1)
