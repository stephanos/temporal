#!/usr/bin/env python3
"""Read frozen accountability inputs and current tracked bytes; print JSON only."""

import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys

BASE = "0988ab1041c5580b2d02763088d5048d6ee70586"
WIP = "a3b9f80efab9356c0be2080779133337e2471ac0"
ROOT = Path(__file__).resolve().parents[5]
HERE = Path(__file__).resolve().parent
A = ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/"
SOURCE = "tools/gomad3/runner/campaign_options.go"
NAMES = (
    "NotSingleBaseSeedError", "SemanticCoverageRequiredError", "NormalizeStrategy",
    "NormalizeCoverage", "ValidateCoverage", "ValidateChoiceTraceLimit",
    "ValidateChoiceCoverage", "ParseSingleBaseSeed",
)
CHECKS = []
REFERENCES = {}


def require(condition, label):
    CHECKS.append(label)
    if not condition:
        raise ValueError(label)


def git(*args):
    return subprocess.run(
        ["git", "--no-optional-locks", *args], cwd=ROOT, check=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        env={**os.environ, "GIT_OPTIONAL_LOCKS": "0"},
    ).stdout


def sha(data):
    return hashlib.sha256(data).hexdigest()


def reference(path, first=None, last=None, commit=BASE, tokens=()):
    commit = git("rev-parse", commit + "^{commit}").decode().strip()
    key = f"{commit}:{path}:{first or 1}-{last or 'EOF'}"
    blob = git("rev-parse", f"{commit}:{path}").decode().strip()
    raw = git("cat-file", "blob", blob)
    lines = raw.splitlines(keepends=True)
    begin, end = first or 1, last or len(lines)
    require(1 <= begin <= end <= len(lines), "reference interval " + key)
    selected = b"".join(lines[begin - 1:end])
    text = selected.decode()
    for token in tokens:
        require(token in text, "reference token " + key + " " + token)
    REFERENCES[key] = {
        "commit": commit, "path": path, "blob_id": blob,
        "sha256": sha(raw), "lines": [begin, end],
        "interval_sha256": sha(selected), "required_tokens": list(tokens),
    }
    return raw, selected


def task(spec, number, tokens=()):
    return reference(f".flow/tasks/{spec}.{number}.md", 4, 30, tokens=tokens)


def tree(commit):
    entries = {}
    for item in git("ls-tree", "-rz", "--full-tree", commit).split(b"\0"):
        if item:
            metadata, path = item.split(b"\t", 1)
            mode, kind, blob = metadata.decode().split()
            entries[path.decode()] = (mode, kind, blob)
    return entries


def main():
    require(git("rev-parse", "--show-toplevel").decode().strip() == str(ROOT),
            "repository root")
    admission_raw = (HERE / "root-admission.json").read_bytes()
    admission = json.loads(admission_raw)
    require(admission["base_commit"] == BASE, "admitted BASE")
    require(admission["branch"] == "gomad", "admitted branch")
    require(admission["worker_paths"] == ["accountability.md", "verify_accountability.py", "evidence.json"],
            "admitted worker writes")
    research = ROOT / admission["research_note"]
    # The scratch note is optional after checkout; its content is never authority.
    require(not research.exists() or sha(research.read_bytes()) == admission["research_note_sha256"],
            "optional research input hash")

    patch, block_patch = reference(A + "task-5/task-only.patch", 551, 633)
    require(all(line.startswith(b"+") for line in block_patch.splitlines(keepends=True)),
            "patch interval consists only of addition lines")
    stripped = b"".join(line[1:] for line in block_patch.splitlines(keepends=True))
    wip, block_wip = reference(SOURCE, 109, 191, WIP)
    current, block_current = reference(SOURCE, 109, 191)
    expected = "6a393b209153a114ba174e9a9e7212ff6df2f15039cbdcff58d858dc33800de1"
    require(stripped == block_wip == block_current, "patch/WIP/BASE exact block equality")
    require(sha(stripped) == expected, "exact block SHA256")
    require(sha(patch) == "a0f0b6c5c3b5963740b0aaab28f66d2f2d97b91ed972d16cfbe20266f670ffda",
            "task5 complete patch SHA256")
    final = json.loads(reference(A + "task-5/final-source.json")[0])
    parent = json.loads(reference(A + "task-5/parent-source-verification.json")[0])
    fix = reference(A + "task-5/review-fix.patch")[0]
    require(final["patch_sha256"] == sha(patch), "final receipt patch binding")
    require(final["review_fix_patch_sha256"] == parent["review_fix_patch_sha256"] == sha(fix)
            == "a6509889521020edf1fe5c3a09f01bb3fc672361a1052a08ee1972145f1cb7f8",
            "review-fix receipt hash chain")
    require(parent["review_fix_source_bound"] is True, "parent source-bound statement")
    require(all(parent["source_sha256"][p] == item["after_sha256"]
                for p, item in final["files"].items()), "all final/parent source receipt bindings")
    require(sha(wip) == "d44433d2d17754ba846a4c8d71964e44736b1da04ccc8d132d0b2e2851393aed",
            "whole WIP file hash")
    require(final["files"][SOURCE]["after_sha256"] ==
            "0d645d15651f0caad56a48175ce59c72bd0d37bb04431c1415ec1fe69a8020fb"
            != sha(wip), "whole WIP differs from task5 final receipt")

    old = reference(SOURCE, commit="7fe6c833c0f03b46d9b2876d9f6a1b59d5f7c2ac")[0].decode()
    inventory = reference(A + "go-interface-changes.md", 187, 210)[1].decode()
    for name in NAMES:
        declaration = ("type " if name.endswith("Error") else "func ") + name
        require(declaration in block_current.decode(), "BASE named declaration " + name)
        require(declaration not in old, "task2 does not introduce " + name)
        require(name in inventory, "inventory member " + name)
        history = git("log", "--reverse", "--format=%H", "-S" + declaration,
                      BASE, "--", SOURCE).decode().splitlines()
        require(bool(history) and history[0] == WIP, "first integrated Git introduction " + name)
    for name in NAMES[:2]:
        require(f"func (*{name}) Error() string" in block_current.decode(), "typed method " + name)
        require(f"func (*{name}) Error() string" in inventory, "inventoried method " + name)
    reference("tools/gomad3/runner/campaign_plan.go", commit="f608449c016b751a8fd4e19561518e979ebfc31e",
              tokens=("func ParseStrategy(", "func ParseCoverageMode("))
    task("fn-109-gomad-deepen-modules-and-tool-interfaces", 5,
         ("second half of R6", "options owner from task 2", "Semantic rules"))
    task("fn-109-gomad-deepen-modules-and-tool-interfaces", 2,
         ("One private serializable options", "One normalization function"))
    task("fn-109-gomad-deepen-modules-and-tool-interfaces", 26,
         ("current check point", "ParseSingleBaseSeed", "NormalizeCoverage"))
    reference(A + "task-26/independent-source-review.md", 9, 35,
              tokens=("33 behavioral tests", "all eight", "unknown external consumers"))

    cli = "tools/gomad3/cmd/gomad/internal/cli/cli.go"
    for start, end, tokens in (
        (724, 733, ("runner.ValidateChoiceCoverage(coverageMode, resolvedChoiceLimit)", "parseTarget")),
        (838, 846, ("runner.ParseSingleBaseSeed(options.Seeds)", "runner.NotSingleBaseSeedError", "errors.As")),
        (862, 870, ("runner.ParseSingleBaseSeed(options.Seeds)", "runner.NotSingleBaseSeedError", "errors.As")),
        (919, 925, ('runner.NormalizeCoverage("", true)',)),
        (945, 958, ("runner.ValidateCoverage(mode, required)", "runner.SemanticCoverageRequiredError", "errors.As")),
        (961, 974, ("if limit == 0", "runner.ValidateChoiceTraceLimit(uint64(limit))")),
    ):
        reference(cli, start, end, tokens=tokens)
        reference(cli, start, end, commit="fec3ce56e7ea2f1498617148cb2c6d47f1297bc0", tokens=tokens)
    reference(SOURCE, 271, 273, tokens=("NormalizeStrategy(options.Strategy)",))
    reference("tools/gomad3/runner/campaign_plan.go", 90, 97, tokens=("NormalizeStrategy(Strategy(value))",))
    reference("tools/gomad3/runner/runner.go", 1298, 1331,
              tokens=("ValidateCoverage(config.Coverage, config.RequiredSemanticProbes)",
                      "ValidateChoiceTraceLimit(config.ChoiceTraceLimit)",
                      "ValidateChoiceCoverage(config.Coverage, config.ChoiceTraceLimit)",
                      "NormalizeCoverage(config.Coverage, false)", "guided exploration requires a corpus and coverage"))
    reference("tools/gomad3/testdata/runnerconsumer/consumer.go", 36, 61,
              tokens=tuple("runner." + name for name in NAMES))
    reference("tools/gomad3/deterministicio/adapter_regenerate.go", 76, 86,
              commit="076cdcc344ced6e1f6e195df84540c8ca74ca2f1", tokens=("type AdapterRegeneration struct",))
    adapter_history = git("log", "--reverse", "--format=%H", "-Stype AdapterRegeneration struct",
                          BASE, "--", "tools/gomad3/deterministicio/adapter_regenerate.go").decode().splitlines()
    require(bool(adapter_history) and adapter_history[0] == "076cdcc344ced6e1f6e195df84540c8ca74ca2f1",
            "AdapterRegeneration first Git introduction predates WIP blame")
    reference("tools/gomad3/runner/guidance.go", 124, 152,
              tokens=("if !regression", "selection = excludeAnsweredSeeds(base, answered)"))
    reference(A + "go-interface-changes.md", 24, 128)
    reference(A + "go-interface-changes.md", 241, 449)
    for number in (6, 12, 11, 19):
        task("fn-109-gomad-deepen-modules-and-tool-interfaces", number)

    owners = (
        ("fn-112-gomad-determinism-assurance-and-test", 3, "diagnostic trace", "df2642da26", "tools/gomad3/choice/"),
        ("fn-112-gomad-determinism-assurance-and-test", 4, "--diagnostics", "e153d05a76", "tools/gomad3/runner/"),
        ("fn-112-gomad-determinism-assurance-and-test", 10, "soak", "bae373d147", "tools/gomad3/qualification/"),
        ("fn-113-gomad-reduce-version-pin-maintenance", 2, "adapter", "076cdcc344", "tools/gomad3/deterministicio/"),
        ("fn-113-gomad-reduce-version-pin-maintenance", 3, "refresh", "7fd67d5aae", "tools/gomad3/internal/compatibilitypack/"),
        ("fn-114-gomad-correct-search-path-defects-and", 6, "start ordinal", "80fcf2cb44", "tools/gomad3/runner/"),
        ("fn-114-gomad-correct-search-path-defects-and", 7, "answered", "2c3de4c982", "tools/gomad3/runner/"),
        ("fn-114-gomad-correct-search-path-defects-and", 7, "regression", "74a53f05b9", "tools/gomad3/"),
        ("fn-114-gomad-correct-search-path-defects-and", 8, "resume", "d00f803843", "tools/gomad3/runner/"),
        ("fn-114-gomad-correct-search-path-defects-and", 15, "workspace", "406a354f20", "tools/gomad3/runner/"),
        ("fn-114-gomad-correct-search-path-defects-and", 9, "target", "bc2e970b53", "tools/gomad3/artifact/"),
        ("fn-114-gomad-correct-search-path-defects-and", 10, "stored bytes", "6f66f744dd", "tools/gomad3/artifact/"),
        ("fn-114-gomad-correct-search-path-defects-and", 11, "readiness", "00633b2b558beb1c762decffe9d55b14631200c1", "tools/gomad3/choice/"),
        ("fn-114-gomad-correct-search-path-defects-and", 12, "ready", "1b970bc144", "tools/gomad3/runner/"),
        ("fn-114-gomad-correct-search-path-defects-and", 12, "rejected visibly", "b666b41c09ed152417e44b6c9738e16a58b8b43a", "tools/gomad3/runner/"),
    )
    owner_bindings = []
    for spec, number, token, short, prefix in owners:
        raw, selected = task(spec, number)
        require(token.lower() in selected.decode().lower(), f"owner task {spec}.{number} {token}")
        commit = git("rev-parse", short + "^{commit}").decode().strip()
        changed = git("diff-tree", "--root", "--no-commit-id", "-r", "--name-only", commit).decode().splitlines()
        source_paths = [p for p in changed if p.startswith(prefix)]
        require(bool(source_paths), f"primary owner commit {commit} touches {prefix}")
        primary_path = next((p for p in source_paths if git("ls-tree", commit, "--", p)), None)
        require(primary_path is not None, "retained owner source " + commit)
        reference(primary_path, commit=commit)
        owner_bindings.append({"task": f"{spec}.{number}", "commit": commit,
                               "changed_source_paths": source_paths, "primary_source": primary_path})

    descriptions = {"toolchain-root": 72, "terminate-grace": 126, "env": 155,
                    "io-ro-mount": 157, "world-transition-limit": 159, "observed": 264,
                    "max-bytes": 278, "min-free-bytes": 325, "prune-qualified-artifacts": 327}
    for name, line in descriptions.items():
        reference("tools/gomad3/CLI.md", line, line, tokens=("--" + name,))
    reference(A + "task-20/cli-inventory-correction/acceptance-open.md", 3, 35,
              tokens=("all 19 checks passed", "open", "nine"))
    reference("tools/gomad3/CLI.md", commit="b43aeb5b15d438eebab65f2ce48eecb19e76e55c",
              tokens=tuple("--" + name for name in descriptions))
    manifest = reference(A + "task-21/baseline-reconstruction/source.sha256")[0]
    require(sha(manifest) == "d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845",
            "original nested baseline source manifest hash")
    require(len(manifest.splitlines()) == 670, "original nested baseline input count")
    reference(A + "task-21/baseline-reconstruction/reconstruction.md", 3, 46,
              tokens=("6782b55f49a0317b230e827ea2a63a37d116d502", "dirty", "670"))
    reference(A + "task-21/baseline-reconstruction/reconstruction.md", 79, 90,
              tokens=("entire dirty repository",))
    reference(A + "task-21/preservation-audit/report.md", 17, 75,
              tokens=("18 top-level tests", "55", "did not execute a matched first-task baseline"))
    reference(A + "task-21/preservation-audit/provenance.json")
    reference(A + "task-21/preservation-disclosure-2026-10-04.md",
              tokens=("controller-v2", "v041", "v2", "incomplete"))
    qualification = reference(A + "qualification-evidence.md")[0]
    require(b"8604c07def0f97b63cbca3864b4c286d6803c4b1" in qualification,
            "historical qualification checkpoint identity")

    doc = (HERE / "accountability.md").read_bytes()
    links = re.findall(r"\[[^\]]+\]\(([^)]+)\)", doc.decode())
    require(bool(links), "supplement has primary-source links")
    for link in links:
        require("#" not in link, "fragment-free link " + link)
        destination = (HERE / link).resolve()
        require(destination.is_relative_to(ROOT) and destination.is_file(), "link exists " + link)
    for token in ("ONLY", "REQUIRED/OPEN", "whole-WIP", "nested-only", "Current-only",
                  "The default changed", "darwin/arm64", "linux/amd64", "task 5 / R6"):
        require(token in doc.decode(), "supplement bound " + token)

    allowed = set(admission["document_paths"])
    directory = str(HERE.relative_to(ROOT)) + "/"
    protected = {p: item for p, item in tree(BASE).items() if p not in allowed and not p.startswith(directory)}
    head = {p: item for p, item in tree("HEAD").items() if p not in allowed and not p.startswith(directory)}
    require(head == protected, "HEAD protected tracked paths/modes/blobs equal BASE")
    index = {}
    for item in git("ls-files", "--stage", "-z").split(b"\0"):
        if item:
            metadata, path = item.split(b"\t", 1)
            mode, blob, stage = metadata.decode().split()
            p = path.decode()
            if p not in allowed and not p.startswith(directory):
                require(stage == "0", "protected index has no conflict " + p)
                index[p] = (mode, blob)
    require(index == {p: (mode, blob) for p, (mode, kind, blob) in protected.items()},
            "index protected tracked paths/modes/blobs equal BASE")
    for p, (mode, kind, blob) in protected.items():
        full = ROOT / p
        if mode == "160000":
            require(kind == "commit" and full.is_dir(), "protected gitlink directory " + p)
            if (full / ".git").exists():
                observed = git("-C", str(full), "rev-parse", "HEAD").decode().strip()
                require(observed == blob, "initialized gitlink HEAD " + p)
                require(not git("-C", str(full), "status", "--porcelain", "--untracked-files=all"),
                        "initialized gitlink clean " + p)
            else:
                require(not any(full.iterdir()), "uninitialized gitlink directory is empty " + p)
            continue
        require(kind == "blob", "protected path is a blob " + p)
        info = full.lstat()
        if mode == "120000":
            require(stat.S_ISLNK(info.st_mode), "protected symlink mode " + p)
            content = os.fsencode(os.readlink(full))
        else:
            require(stat.S_ISREG(info.st_mode), "protected regular file " + p)
            observed_mode = "100755" if info.st_mode & stat.S_IXUSR else "100644"
            require(observed_mode == mode, "protected executable mode " + p)
            content = full.read_bytes()
        observed_blob = hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()
        require(observed_blob == blob, "protected worktree bytes " + p)

    return {
        "success": True, "task": admission["task"], "base_commit": BASE, "commits": [],
        "original_acceptance": "REQUIRED/OPEN", "required_task_state": "blocked", "required_parent_state": "open",
        "tier_line": "Tier: session (jev-unavailable(no_key))", "actual_worker_model": None,
        "review": "host-deferred; no verdict", "assertion_count": len(CHECKS),
        "assertion_labels_sha256": sha("\n".join(CHECKS).encode()),
        "admission_sha256": sha(admission_raw), "accountability_sha256": sha(doc),
        "verifier_sha256": sha(Path(__file__).read_bytes()),
        "research_note_sha256": admission["research_note_sha256"],
        "research_note_policy": "If present, hash checked; optional non-shipping research is not primary authority",
        "references": list(REFERENCES.values()), "owner_bindings": owner_bindings,
        "named_seams": list(NAMES), "flag_description_lines": descriptions,
        "protected_tracked_path_count": len(protected),
        "protected_gitlink_count": sum(mode == "160000" for mode, kind, blob in protected.values()),
        "preservation_scope": "HEAD/index/worktree tracked paths and modes outside admitted documents/directory vs BASE only; gitlinks bind recorded identity and initialized clean HEAD or empty uninitialized directory; unrelated untracked files excluded; no original first-baseline preservation verdict",
        "tests": ["python3 " + str(Path(__file__).relative_to(ROOT))], "prs": [],
        "gates": "No Go/full/native/formal gate run or pass claimed",
    }


if __name__ == "__main__":
    try:
        result = main()
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as exc:
        print(json.dumps({"success": False, "error": str(exc), "assertion_count": len(CHECKS)}))
        sys.exit(1)
    print(json.dumps(result, indent=2, sort_keys=True))
