#!/usr/bin/env python3
"""Bounded, index-free source measurement; outputs stay in the named artifact dir."""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile
import time
from datetime import datetime, timezone


def utc():
    return datetime.now(timezone.utc).isoformat()


def measure(data):
    return {"bytes": len(data), "lines": data.count(b"\n"),
            "sha256": hashlib.sha256(data).hexdigest()}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--scratch", required=True)
    args = parser.parse_args()
    repo = Path(args.repo).resolve()
    scratch = Path(args.scratch).resolve()
    assert scratch.parent == Path("/tmp") and scratch.name.startswith("fn110-source-size-")
    assert scratch.is_dir() and not list(scratch.iterdir())
    out = repo / ".flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-5"
    started = utc()
    timer = time.monotonic()
    commands = []

    def run(argv, cwd=repo, stdin=None, expected=(0,), output=None):
        begin, clock = utc(), time.monotonic()
        env = dict(os.environ, GIT_OPTIONAL_LOCKS="0", LC_ALL="C", TZ="UTC")
        result = subprocess.run(argv, cwd=cwd, input=stdin, capture_output=True,
                                env=env, timeout=120)
        source_blob = argv[:2] == ["git", "show"]
        commands.append({"argv": argv, "cwd": str(cwd), "started_utc": begin,
                         "finished_utc": utc(), "duration_seconds": time.monotonic()-clock,
                         "exit_code": result.returncode, "expected_exit_codes": list(expected),
                         "stdin": measure(stdin) if stdin is not None else None,
                         "stdout_path": str(output) if output else None,
                         "stdout_text": None if source_blob or output else result.stdout.decode("utf-8", errors="replace"),
                         "stdout": measure(result.stdout),
                         "stderr_text": result.stderr.decode("utf-8", errors="replace"),
                         "stderr": measure(result.stderr),
                         "environment_overrides": {"GIT_OPTIONAL_LOCKS": "0", "LC_ALL": "C", "TZ": "UTC"}})
        (out / "source-size-commands.json").write_text(json.dumps(commands, indent=2)+"\n")
        assert result.returncode in expected, (argv, result.returncode, result.stderr)
        if output:
            Path(output).write_bytes(result.stdout)
        return result.stdout

    def paths(patch):
        assert patch and b"\0" not in patch
        text = patch.decode("utf-8")
        assert not re.search(r"^(new file mode|deleted file mode|GIT binary patch|Binary files)", text, re.M)
        headers = re.findall(r"^diff --git a/(\S+) b/(\S+)$", text, re.M)
        assert headers and text.count("diff --git ") == len(headers)
        result = []
        for old, new in headers:
            assert old == new and old.startswith("src/")
            assert str(PurePosixPath(old)) == old and ".." not in PurePosixPath(old).parts
            assert f"--- a/{old}\n" in text and f"+++ b/{old}\n" in text
            result.append(old)
        assert len(set(result)) == len(result)
        return sorted(result)

    def patch_stats(patch):
        inventory = paths(patch)
        numstat = run(["git", "apply", "--numstat"], stdin=patch).decode()
        rows = [line.split("\t") for line in numstat.splitlines()]
        assert sorted(row[2] for row in rows) == inventory
        sections = re.split(rb"(?=^diff --git )", patch, flags=re.M)
        per_file = []
        for section in sections:
            if not section:
                continue
            name = section.splitlines()[0].decode().split()[2][2:]
            row = next(row for row in rows if row[2] == name)
            per_file.append(dict(path=name, **measure(section), hunks=section.count(b"\n@@ "),
                                 added=int(row[0]), deleted=int(row[1])))
        return dict(**measure(patch), files=len(inventory), hunks=patch.count(b"\n@@ "),
                    added=sum(int(row[0]) for row in rows), deleted=sum(int(row[1]) for row in rows),
                    inventory=per_file)

    def overlay_snapshot():
        root = repo / "tools/gomad3/toolchain/runtime/overlay"
        snapshot = {}
        for path in sorted(root.rglob("*")):
            assert not path.is_symlink(), path
            if path.is_file():
                snapshot[path.relative_to(root).as_posix()] = path.read_bytes()
        return snapshot

    def overlay_stats(snapshot):
        inventory = [dict(path=name, **measure(data)) for name, data in sorted(snapshot.items())]
        return {"files": len(inventory), "bytes": sum(row["bytes"] for row in inventory),
                "lines": sum(row["lines"] for row in inventory), "inventory": inventory}

    baseline_record = json.loads((out.parent / "task1-baseline.json").read_text())
    baseline_commit = baseline_record["baseline_commit"]
    patch_rel = "tools/gomad3/toolchain/runtime/go1.27.1.patch"
    descriptor_rel = "tools/gomad3/toolchain/version/version.json"
    patch = (repo / patch_rel).read_bytes()
    descriptor_bytes = (repo / descriptor_rel).read_bytes()
    descriptor = json.loads(descriptor_bytes)
    overlay = overlay_snapshot()
    (out / "source-size-current-U1.patch").write_bytes(patch)
    baseline_patch = run(["git", "show", f"{baseline_commit}:{patch_rel}"],
                         output=out / "source-size-original-baseline-U3.patch")
    baseline_descriptor_bytes = run(["git", "show", f"{baseline_commit}:{descriptor_rel}"])
    baseline_descriptor = json.loads(baseline_descriptor_bytes)
    assert measure(baseline_patch) == {key: baseline_record["context_variants_of_baseline_source"]["U3"][key]
                                       for key in ("bytes", "lines", "sha256")}
    assert measure(baseline_descriptor_bytes)["sha256"] == baseline_record["toolchain_inputs"]["descriptor_sha256"]
    assert paths(patch) == descriptor["patch_allowlist"]
    assert sorted(overlay) == descriptor["overlay_allowlist"]
    assert paths(baseline_patch) == baseline_descriptor["patch_allowlist"]

    baseline_overlay = {}
    overlay_prefix = "tools/gomad3/toolchain/runtime/overlay/"
    tree = run(["git", "ls-tree", "-r", "-z", baseline_commit, "--", overlay_prefix])
    for entry in tree.rstrip(b"\0").split(b"\0"):
        metadata, path_bytes = entry.split(b"\t", 1)
        mode, kind, blob = metadata.decode().split()
        name = path_bytes.decode()
        assert mode in ("100644", "100755") and kind == "blob" and name.startswith(overlay_prefix)
        baseline_overlay[name[len(overlay_prefix):]] = run(["git", "show", f"{baseline_commit}:{name}"])
    before_overlay = overlay_stats(baseline_overlay)
    assert all(before_overlay[key] == baseline_record["overlay"][key] for key in ("files", "bytes", "lines"))
    assert sorted(baseline_overlay) == baseline_descriptor["overlay_allowlist"]

    archive = repo / "tools/gomad3/.toolchain/downloads" / descriptor["archive"]["name"]
    archive_bytes = archive.read_bytes()
    archive_identity = measure(archive_bytes)
    assert archive_identity["sha256"] == descriptor["archive"]["sha256"] == baseline_descriptor["archive"]["sha256"]
    assert archive_identity["sha256"] == baseline_record["toolchain_inputs"]["archive_sha256"]
    required = sorted(set(paths(patch)+paths(baseline_patch)+["VERSION"]))
    extracted = {}
    seen = set()
    expanded = 0
    extraction_start, extraction_timer = utc(), time.monotonic()
    with tarfile.open(archive, "r:gz") as source:
        for entry in source:
            name = entry.name.rstrip("/")
            assert name and "\\" not in name and str(PurePosixPath(name)) == name
            assert not PurePosixPath(name).is_absolute() and ".." not in PurePosixPath(name).parts
            assert name == "go" or name.startswith("go/")
            assert name not in seen and (entry.isfile() or entry.isdir())
            seen.add(name)
            assert len(seen) <= 100000
            if entry.isfile():
                assert entry.size >= 0
                expanded += entry.size
                assert expanded <= 4 << 30
            relative = name[3:] if name.startswith("go/") else ""
            if relative in required:
                assert entry.isfile()
                extracted[relative] = source.extractfile(entry).read()
    assert sorted(extracted) == required
    assert extracted["VERSION"].splitlines()[0].decode() == descriptor["go_version"]
    extraction = {"implementation": "Python tarfile archive scan with entry/path/type/size checks; extract only all patched members plus VERSION",
                  "started_utc": extraction_start, "finished_utc": utc(),
                  "duration_seconds": time.monotonic()-extraction_timer, "exit_code": 0,
                  "archive_entries": len(seen), "archive_expanded_bytes": expanded,
                  "extracted_inventory": [dict(path=name, **measure(data)) for name, data in sorted(extracted.items())]}

    def make_tree(root):
        for name, data in extracted.items():
            destination = root / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)

    def apply(root, content):
        for dry in (True, False):
            argv = ["patch"] + (["--dry-run"] if dry else []) + ["--batch", "-V", "none", "-p1", "-F", "0"]
            output = run(argv, cwd=root, stdin=content)
            assert not re.search(rb"\b(fuzz|offset|FAILED|reversed|previously applied)\b", output, re.I), output
        assert not list(root.rglob("*.orig")) and not list(root.rglob("*.rej"))

    def diff_form(root, context, destination):
        return run(["git", "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--binary", "--no-prefix",
                    "--abbrev=7", "--diff-algorithm=myers", f"--unified={context}", "--", "a", "b"],
                   cwd=root, expected=(1,), output=destination)

    original = scratch / "original"
    final = scratch / "final"
    for pair in (original, final):
        make_tree(pair / "a")
        make_tree(pair / "b")
    apply(original / "b", baseline_patch)
    apply(final / "b", patch)
    reproduced_baseline = diff_form(original, 3, out / "source-size-reproduced-baseline-U3.patch")
    assert reproduced_baseline == baseline_patch
    final_u3 = diff_form(final, 3, out / "source-size-final-U3.patch")
    final_u1 = diff_form(final, 1, out / "source-size-reproduced-current-U1.patch")
    assert final_u1 == patch
    final_u3_repeat = diff_form(final, 3, scratch / "final-U3-repeat.patch")
    assert final_u3_repeat == final_u3
    equivalent = scratch / "equivalent-U3"
    make_tree(equivalent)
    apply(equivalent, final_u3)
    source_inventory = []
    for name in required:
        u1_bytes = (final / "b" / name).read_bytes()
        u3_bytes = (equivalent / name).read_bytes()
        assert u1_bytes == u3_bytes, name
        source_inventory.append({"path": name, "pristine": measure(extracted[name]),
                                 "final": measure(u1_bytes), "baseline": measure((original / "b" / name).read_bytes()),
                                 "changed_from_pristine": u1_bytes != extracted[name],
                                 "U1_U3_byte_identical": True})
    assert sorted(row["path"] for row in source_inventory if row["changed_from_pristine"]) == paths(patch)
    after_overlay = overlay_stats(overlay)
    stats = {"original_baseline_U3": patch_stats(baseline_patch),
             "current_final_U3": patch_stats(final_u3), "current_canonical_U1": patch_stats(patch)}
    overlay_changes = []
    for name in sorted(set(overlay) | set(baseline_overlay)):
        before, after = baseline_overlay.get(name), overlay.get(name)
        if before != after:
            overlay_changes.append({"path": name, "before": measure(before) if before is not None else None,
                                    "after": measure(after) if after is not None else None,
                                    "bytes_delta": len(after or b"")-len(before or b""),
                                    "lines_delta": (after or b"").count(b"\n")-(before or b"").count(b"\n")})
    stable = {"patch": (repo / patch_rel).read_bytes() == patch,
              "descriptor": (repo / descriptor_rel).read_bytes() == descriptor_bytes,
              "overlay": overlay_snapshot() == overlay,
              "archive": measure(archive.read_bytes()) == archive_identity}
    assert all(stable.values()), stable
    host = {"uname": run(["uname", "-s", "-m"]).decode().strip(),
            "git": run(["git", "--version"]).decode().strip(),
            "patch": run(["patch", "--version"]).decode().strip(),
            "head_read_only": run(["git", "rev-parse", "HEAD"]).decode().strip()}
    buildkey = repo / "tools/gomad3/.toolchain/build-key"
    evidence = {"schema": "fn110-task5-source-size-verification/v1", "started_utc": started,
                "finished_utc": utc(), "duration_seconds": time.monotonic()-timer,
                "scratch": str(scratch), "script": measure(Path(__file__).read_bytes()),
                "host": host, "requested_routing": "thinking scout gpt-6.1-sol high",
                "actual_model_metadata": "unknown; not host-observed",
                "baseline_commit": baseline_commit, "baseline_build_key": baseline_record["build_key"],
                "current_build_key_file": buildkey.read_text().strip() if buildkey.exists() else None,
                "current_build_key_qualified": False,
                "descriptor": measure(descriptor_bytes), "archive": archive_identity,
                "patches": stats, "overlay_original_baseline": before_overlay, "overlay_current": after_overlay,
                "overlay_changes": overlay_changes, "extraction": extraction,
                "materialized_source_inventory": source_inventory, "input_stability": stable,
                "comparisons": {"final_U3_smaller_than_original_baseline_U3": len(final_u3) < len(baseline_patch),
                                "final_U3_minus_original_baseline_U3_bytes": len(final_u3)-len(baseline_patch),
                                "canonical_U1_smaller_than_final_U3": len(patch) < len(final_u3),
                                "canonical_U1_minus_final_U3_bytes": len(patch)-len(final_u3),
                                "canonical_U1_minus_original_baseline_U3_bytes": len(patch)-len(baseline_patch),
                                "original_baseline_reproduced_byte_identically": True,
                                "current_U1_reproduced_byte_identically": True,
                                "final_U3_repeat_byte_identical": True,
                                "U3_U1_all_patched_members_byte_identical_zero_fuzz_zero_offset": True},
                "limits": ["Source text and patch representation only; no package loading, generation, tests or builds.",
                           "Archive scan covers every entry; extraction covers the union of every baseline/current patched member plus VERSION.",
                           "Git commands only show, ls-tree, apply --numstat, diff --no-index and rev-parse; no index/history mutation.",
                           "linux/arm64 GNU patch evidence is developmental. Native R4 darwin/arm64 and linux/amd64 equivalence remains open.",
                           "R7 behavioral/native gates and all task/spec acceptance remain open.",
                           "Current overlay includes integrated work from other specs; growth is inventoried separately and is not attributed wholly to fn110."]}
    (out / "source-size-evidence.json").write_text(json.dumps(evidence, indent=2)+"\n")
    print(json.dumps({"patches": {name: {key: value for key, value in stat.items() if key != "inventory"}
                                  for name, stat in stats.items()}, "comparisons": evidence["comparisons"],
                      "overlay_baseline": {key: before_overlay[key] for key in ("files", "bytes", "lines")},
                      "overlay_current": {key: after_overlay[key] for key in ("files", "bytes", "lines")},
                      "duration_seconds": evidence["duration_seconds"], "scratch": str(scratch)}, indent=2))


if __name__ == "__main__":
    main()
