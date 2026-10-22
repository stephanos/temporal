#!/usr/bin/env python3
"""fn-221.2 merged-matrix checker (independent of the harness --verify).

Usage: python3 -I merge-check.py ORIG_SWEEP_DIR RERUN_ROOT SELECTION_TSV OUT_JSON

Checks that the rerun run root's harness sidecar is exactly the original matrix with the
selected units' rows replaced by rows the rerun observed on the same candidate, recomputes
counts, and lists class changes. Exit 0 only when every check holds.
"""
import hashlib
import json
import sys
from collections import Counter, defaultdict

ORIG_KEY = "sha256:9bf9e9064761a8afcc075d6cba9aed57a57baa6a1f6de36f092233eb5c01040c"
ORIG_JOURNAL_SHA = "839654fcfc60bd65a5ea740b6ec01543336d2f0576798614220cb06be35dafb6"

orig_dir, rerun, selection_path, out_path = sys.argv[1:5]
findings = []


def check(ok, message):
    if not ok:
        findings.append(message)
    return ok


orig = json.load(open(f"{orig_dir}/matrix/sweep.json"))
try:
    merged = json.load(open(f"{rerun}/matrix/sweep.json"))
except FileNotFoundError:
    report = {"status": "incomplete", "findings": ["rerun sidecar missing: rerun is incomplete; resume the existing journal"]}
    json.dump(report, open(out_path, "w"), indent=2)
    print(report["findings"][0])
    sys.exit(1)
selection = {}
for line in open(selection_path):
    if line.startswith("#") or not line.strip():
        continue
    pkg, name, old = line.rstrip("\n").split("\t")
    check((pkg, name) not in selection, f"duplicate selection {pkg} {name}")
    selection[(pkg, name)] = old
check(len(selection) == 344, f"selection has {len(selection)} units, want 344")

# 1. Journal: original bytes, then the 344 invalidation rows, then this rerun's rows.
orig_bytes = open(f"{orig_dir}/matrix/sweep.jsonl", "rb").read()
check(hashlib.sha256(orig_bytes).hexdigest() == ORIG_JOURNAL_SHA, "original journal digest changed")
journal_bytes = open(f"{rerun}/matrix/sweep.jsonl", "rb").read()
check(journal_bytes.startswith(orig_bytes), "rerun journal does not start with the original journal bytes")
invalidation = open(f"{rerun}/invalidation-rows.jsonl", "rb").read()
rest = journal_bytes[len(orig_bytes):]
check(rest.startswith(invalidation), "invalidation rows do not follow the original journal")
new_lines = [json.loads(l) for l in rest[len(invalidation):].splitlines() if l.strip()]
inval_rows = [json.loads(l) for l in invalidation.splitlines() if l.strip()]
check({(r["test"]["package"], r["test"]["name"]) for r in inval_rows} == set(selection), "invalidation rows != selection")
check(all(r["kind"] == "test" and r["test"]["class"] == "not-run" and r["candidate"] == ORIG_KEY for r in inval_rows), "invalidation row shape")
headers = [r for r in new_lines if r["kind"] == "header"]
check(bool(headers), "rerun appended no invocation header")
check(bool(new_lines) and new_lines[0]["kind"] == "header", "rerun observations precede the first invocation header")
original_header = next(json.loads(line)["header"] for line in orig_bytes.splitlines()
                       if json.loads(line)["kind"] == "header")
check(orig["candidate"] == original_header["candidate"], "original sidecar candidate differs from immutable journal")
check(orig["budget"] == {k: original_header[k] for k in ("overall_budget", "max_tests", "retain_bytes")},
      "original sidecar budget differs from immutable journal")
check(orig["packages"] == original_header["packages"],
      "original sidecar enumeration differs from immutable journal")


# These exact relocations belong to the retained rerun-20261022T1705Z cohort.
path_relocations = {
    "/home/agent/wt/fn221": "/home/agent/wt/fn221-rerun-src",
    "/home/agent/wt/qualification-final-frozen/tools/gomad/.toolchain/stock/29cb18ad2b9f5aee/go/bin/go":
        "/home/agent/wt/evidence/fn221/rerun-tools/go1.27.1/bin/go",
    "/home/agent/wt/fn221/tools/gomad/internal/model/runtimesource":
        "/home/agent/wt/fn221-rerun-src/tools/gomad/internal/model/runtimesource",
    "/home/agent/wt/evidence/fn221/full-1d7b37f00a/matrix":
        "/home/agent/wt/evidence/fn221/rerun-20261022T1705Z/matrix",
    "/home/agent/wt/evidence/fn221/full-1d7b37f00a/work":
        "/home/agent/wt/evidence/fn221/rerun-20261022T1705Z/work",
    "/home/agent/wt/evidence/fn221/full-1d7b37f00a/retained":
        "/home/agent/wt/evidence/fn221/rerun-20261022T1705Z/retained",
}
expected_command = [path_relocations.get(arg, arg) for arg in original_header["command"]]
module_relocations = {
    "/home/agent/wt/fn221": "/home/agent/wt/fn221-rerun-src",
    "/home/agent/wt/fn221/tests/mixedbrain": "/home/agent/wt/fn221-rerun-src/tests/mixedbrain",
}
expected_packages = [{**p, "module_directory": module_relocations.get(p["module_directory"])}
                     for p in original_header["packages"]]
check(all(p["module_directory"] for p in expected_packages), "original package has an unapproved module directory")


for index, row in enumerate(headers, 1):
    header = row.get("header", {})
    prefix = f"rerun header {index}"
    check(row.get("schema") == "gomad-wasm.sweep-journal/v1", f"{prefix}: journal schema differs")
    check(header.get("candidate") == original_header["candidate"], f"{prefix}: candidate/configuration/execution budget differs")
    for field in ("statement", "overall_budget", "max_tests", "retain_bytes"):
        check(header.get(field) == original_header[field], f"{prefix}: {field} differs")
    check(header.get("packages") == expected_packages,
          f"{prefix}: package enumeration/module directories differ from authorized relocation")
    check(header.get("command") == expected_command,
          f"{prefix}: command flags/budgets/authorized paths differ")
check(len(inval_rows) == len(selection), "invalidation row count differs from selection")
check(all(r.get("kind") in ("header", "test", "warmup") and r.get("schema") == "gomad-wasm.sweep-journal/v1"
          for r in new_lines), "unknown rerun journal kind/schema")
check(all(r["candidate"] == ORIG_KEY for r in new_lines), "a rerun journal row has a different candidate key")
rerun_tests = [r["test"] for r in new_lines if r["kind"] == "test"]
warmups = [r["test"] for r in new_lines if r["kind"] == "warmup"]
latest_rerun = {}
for t in rerun_tests:
    key = (t["package"], t["name"])
    check(key in selection, f"rerun executed a non-selected unit {key}")
    check(t.get("class") != "not-run", f"rerun unit {key} is incomplete (not-run)")
    check(t.get("config_source") == orig["candidate"]["config_source"], f"rerun unit {key}: config source differs")
    latest_rerun[key] = t
check(set(latest_rerun) == set(selection), f"rerun rows cover {len(latest_rerun)} of {len(selection)} selected units")
check(len(rerun_tests) == len(selection), f"{len(rerun_tests)} rerun test rows for {len(selection)} units")

# 2. Candidate and enumeration identity.
check(merged["candidate"] == orig["candidate"], "candidate identity differs from the original")
check(merged["candidate_key"] == orig["candidate_key"] == ORIG_KEY, "candidate key differs")
check(merged["schema"] == orig["schema"] == "gomad-wasm.sweep/v2", "schema")
check(merged["statement"] == orig["statement"], "statement")
check(merged["budget"] == orig["budget"], "budget differs")
check(merged["packages"] == expected_packages, "package enumeration/module directories differ from authorized relocation")

# 3. Rows: every unit exactly once; non-selected rows identical; selected rows = rerun observation.
units = [(p["pattern"], n) for p in merged["packages"] for n in p["tests"]]
check(len(units) == len(set(units)) == 1488, f"{len(units)} enumerated units")
rows = merged["rows"]
check([(r["package"], r["name"]) for r in rows] == units, "row order/keys differ from enumeration")
orig_rows = {(r["package"], r["name"]): r for r in orig["rows"]}
check([(r["package"], r["name"]) for r in orig["rows"]] == units, "original rows differ from enumeration")
kept = replaced = 0
for r in rows:
    key = (r["package"], r["name"])
    if key in selection:
        replaced += 1
        check(r == latest_rerun.get(key), f"merged row for {key} is not the rerun row")
        check(orig_rows[key]["class"] == selection[key], f"selection class mismatch for {key}")
    else:
        kept += 1
        check(r == orig_rows[key], f"non-selected row {key} changed")
check((kept, replaced) == (1144, 344), f"kept {kept} replaced {replaced}")

# 4. Recount.
by_class = Counter(r["class"] for r in rows)
check(dict(by_class) == merged["by_class"], "by_class does not recount")
by_pkg = defaultdict(Counter)
for r in rows:
    by_pkg[r["package"]][r["class"]] += 1
check({k: dict(v) for k, v in by_pkg.items()} == merged["by_package"], "by_package does not recount")
reasons = Counter((r["class"], r.get("reason", "")) for r in rows if r["class"] != "pass")
check({(x["class"], x["reason"]): x["count"] for x in merged["reasons"]} == dict(reasons), "reasons do not recount")
for r in rows:
    if r["class"] != "pass":
        check(bool(r.get("reason", "").strip()), f"{r['package']} {r['name']}: non-pass without reason")

# 5. Module identity: rerun rows in a package reuse the original module digest where one exists.
orig_modules = defaultdict(set)
for r in orig["rows"]:
    if r.get("module_sha256"):
        orig_modules[r["package"]].add(r["module_sha256"])
rerun_modules = defaultdict(set)
for key, r in latest_rerun.items():
    if r.get("module_sha256"):
        rerun_modules[key[0]].add(r["module_sha256"])
for pkg, mods in rerun_modules.items():
    if orig_modules.get(pkg):
        check(mods <= orig_modules[pkg], f"{pkg}: rerun module {sorted(mods)} not in original {sorted(orig_modules[pkg])}")
    check(len(mods) == 1, f"{pkg}: {len(mods)} module digests in rerun rows")

changes = []
for key, old in selection.items():
    new = latest_rerun.get(key, {})
    o = orig_rows[key]
    flag = ""
    if old == "timeout" and new.get("class") not in ("timeout", "pass"):
        if new.get("class") == "capacity-exceeded" and "network-v2-evidence-bytes" in new.get("reason", ""):
            flag = "former wall-clock timeout became WASI network-v2-evidence-bytes capacity-exceeded within the unchanged execution budget"
        else:
            flag = "former wall-clock timeout became a non-timeout non-pass class: load may have affected a non-timeout class"
    changes.append({"package": key[0], "name": key[1], "old_class": old, "old_reason": o.get("reason", ""),
                    "new_class": new.get("class"), "new_reason": new.get("reason", ""),
                    "old_wall_seconds": o.get("wall_seconds"), "new_wall_seconds": new.get("wall_seconds"),
                    "attempts": new.get("attempts"), "flag": flag})

report = {
    "checker": "fn221-rerun-tools/merge-check.py",
    "historical_checker_sha256": "aed05c442756c348db80181298b63ff83a7c14fe18967a25c4350a8e9eb317dc",
    "rerun_invocation_headers": len(headers),
    "orig_sidecar_sha256": hashlib.sha256(open(f"{orig_dir}/matrix/sweep.json", "rb").read()).hexdigest(),
    "merged_sidecar_sha256": hashlib.sha256(open(f"{rerun}/matrix/sweep.json", "rb").read()).hexdigest(),
    "merged_journal_sha256": hashlib.sha256(journal_bytes).hexdigest(),
    "units": len(units), "kept_original_rows": kept, "replaced_by_rerun": replaced,
    "by_class_before": orig["by_class"], "by_class_after": dict(sorted(by_class.items())),
    "rerun_class_transitions": dict(Counter(f"{c['old_class']} -> {c['new_class']}" for c in changes)),
    "rerun_warmups": [{"package": w["package"], "gomad_exit": w["gomad_exit"], "wall_seconds": w["wall_seconds"]} for w in warmups],
    "rerun_modules": {k: sorted(v) for k, v in rerun_modules.items()},
    "flags": [c for c in changes if c["flag"]],
    "changes": sorted(changes, key=lambda c: (c["package"], c["name"])),
    "findings": findings,
}
json.dump(report, open(out_path, "w"), indent=2)
print(json.dumps({k: report[k] for k in ("units", "kept_original_rows", "replaced_by_rerun", "by_class_before", "by_class_after", "rerun_class_transitions", "rerun_modules")}, indent=1))
print(f"flags={len(report['flags'])} findings={len(findings)}")
for f in findings[:50]:
    print("finding:", f)
sys.exit(1 if findings else 0)
