#!/bin/sh
# fn-113.1 pin baseline. Read-only; measures the committed tree at REV (default
# HEAD), so a working-tree development harness does not affect the counts. Run
# from the repository root:
#   sh .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/measure-pins.sh [REV]
set -eu
rev="${1:-HEAD}"
g=tools/gomad3
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git archive "$rev" "$g" go.mod | tar -x -C "$tmp"

echo "## identity"
echo "rev $(git rev-parse "$rev")"
echo "rev_date $(git log -1 --format=%cI "$rev")"

echo "## go release (toolchain/version/version.json)"
python3 - "$tmp/$g/toolchain/version/version.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
print("go_version", d["go_version"])
print("archive_sha256_pins 1")
print("supported_platforms", len(d["supported_platforms"]), " ".join(d["supported_platforms"]))
print("boundary_manifest_version", d["boundary_manifest_version"])
print("patch_allowlist_entries", len(d["patch_allowlist"]))
print("overlay_allowlist_entries", len(d["overlay_allowlist"]))
print("adapter_identities", len(d["adapters"]))
PY

echo "## runtime patch and overlay"
patch="$tmp/$g/toolchain/runtime/go1.27.1.patch"
overlay="$tmp/$g/toolchain/runtime/overlay"
echo "patch_lines $(wc -l < "$patch" | tr -d ' ')"
echo "patch_files $(grep -c '^diff --git' "$patch")"
echo "patch_hunks $(grep -c '^@@ ' "$patch")"
echo "overlay_files $(find "$overlay" -type f | wc -l | tr -d ' ')"
echo "overlay_lines $(find "$overlay" -type f -print0 | xargs -0 cat | wc -l | tr -d ' ')"

echo "## boundary manifest and interception fingerprints"
python3 - "$tmp/$g/deterministicio/boundary/manifest.json" "$tmp/$g/deterministicio/boundary/compiler-tests.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
overrides = sum(len(i.get("platform_overrides") or {}) for i in d["intercepts"])
print("manifest_platforms", " ".join(d["platforms"]))
print("intercepts", len(d["intercepts"]))
print("intercept_platform_override_fingerprints", overrides)
print("declaration_fingerprints", len(d["intercepts"]) + overrides)
print("package_fingerprints", sum(1 for i in d["intercepts"] if i.get("package_sha256")))
print("hook_policies", len(d["hook_policies"]))
print("reviewed_candidates", len(d["reviewed_candidates"]))
c = json.load(open(sys.argv[2]))
print("compiler_test_interceptions", len(c.get("intercepts", c) if isinstance(c, dict) else c))
PY
echo "expected_intercepts_lines $(wc -l < "$tmp/$g/expected-intercepts-go1.27.1.txt" | tr -d ' ')"

echo "## toolchain inventories (test-pinned)"
python3 - "$tmp/$g/toolchain/clock_inventory_test.go" "$tmp/$g/toolchain/goroutine_inventory_test.go" <<'PY'
import re, sys
clock = open(sys.argv[1]).read()
block = clock.split("var reviewedHostClockReferences = []clockReference{", 1)[1].split("\n}\n", 1)[0]
rows = re.findall(r'^\t\{"([^"]+)", "([^"]+)", "([^"]+)"', block, re.M)
print("clock_references", len(rows))
for platform in sorted({r[0] for r in rows}):
    print("clock_references_" + platform, sum(1 for r in rows if r[0] == platform))
try:
    goroutines = open(sys.argv[2]).read()
    print("goroutine_creation_sites", len(re.findall(r'^\t\{"[^"]+", "[^"]+", \d+,', goroutines, re.M)))
except FileNotFoundError:
    print("goroutine_creation_sites absent")
PY

echo "## dependency adapters (deterministicio/*_adapter.go)"
total=0
for file in "$tmp/$g"/deterministicio/*_adapter.go; do
	count=$(grep -o '"sha256:[0-9a-f]\{64\}"' "$file" | wc -l | tr -d ' ')
	total=$((total + count))
	echo "adapter_sha256_literals $(basename "$file") $count"
done
echo "adapter_sha256_literals_total $total"
echo "adapter_platform_pins $(cat "$tmp/$g"/deterministicio/*_adapter.go | grep -cE '^[[:space:]]+"[a-z]+/[a-z0-9]+": +"sha256:')"
echo "adapter_sha256_literals_unique $(cat "$tmp/$g"/deterministicio/*_adapter.go | grep -o '"sha256:[0-9a-f]\{64\}"' | sort -u | wc -l | tr -d ' ')"
echo "adapter_template_files $(ls "$tmp/$g"/deterministicio/adapterdata | wc -l | tr -d ' ')"
python3 - "$tmp/$g/toolchain/version/version.json" "$tmp/go.mod" <<'PY'
import json, re, sys
adapters = json.load(open(sys.argv[1]))["adapters"]
required = dict(re.findall(r'^\s*(\S+) (v\S+)', open(sys.argv[2]).read(), re.M))
absent = [a["module"] for a in adapters if a["module"] not in required]
moved = [f'{a["module"]} {a["version"]}->{required[a["module"]]}' for a in adapters if a["module"] in required and required[a["module"]] != a["version"]]
print("adapters_absent_from_root_go_mod", len(absent), " ".join(absent))
print("adapters_moved_in_root_go_mod", len(moved), " ".join(moved))
PY

echo "## compatibility packs (internal/compatibilitypack)"
python3 - "$tmp/$g/internal/compatibilitypack" <<'PY'
import glob, json, os, sys
root = sys.argv[1]
packs = [json.load(open(p)) for p in sorted(glob.glob(os.path.join(root, "packs", "*.json")))]
identities = set()
for p in packs:
    for m in p["activation"]:
        identities.add((m["path"], m["version"]))
    for r in p["rules"]:
        identities.add((r["module"]["path"], r["module"]["version"]))
print("packs", len(packs))
print("pack_rules", sum(len(p["rules"]) for p in packs))
print("pack_module_version_pins", len(identities))
print("pack_modules", len({path for path, _ in identities}))
print("pack_adapter_replacements", sum(1 for p in packs for m in p["activation"] if m["replacement"]["kind"] == "adapter"))
print("pack_source_set_digests", sum(len(p["rules"]) for p in packs))
print("pack_go_source_digests", sum(len(r["go_sources"]) for p in packs for r in p["rules"]))
print("pack_foreign_source_digests", sum(len(r["foreign_sources"]) for p in packs for r in p["rules"]))
print("requests", len(glob.glob(os.path.join(root, "requests", "*.json"))))
print("reports", len(glob.glob(os.path.join(root, "reports", "*.md"))))
print("generation_outputs", len(json.load(open(os.path.join(root, "generation.json")))["outputs"]))
for platform in sorted({pl for p in packs for pl in p["governance"]["platforms"]}):
    print("packs_for_" + platform, sum(1 for p in packs if platform in p["governance"]["platforms"]))
PY

echo "## upstream go.mod churn (2026-04-01 to 2026-10-01, the assessment window)"
# Upstream commits carry a "(#NNNNN)" pull-request suffix; Gomad branch
# commits that also touch go.mod do not and are excluded.
log() { git log --since=2026-04-01T00:00:00Z --until=2026-10-01T23:59:59Z "$@" "$rev" -- go.mod; }
echo "go_mod_commits_all $(log --format=%h | wc -l | tr -d ' ')"
echo "go_mod_commits_upstream $(log --format=%s | grep -cE '\(#[0-9]+\)$')"
echo "go_directive_bumps_upstream $(log --format='commit %s' -p | awk '/^commit /{up = ($0 ~ /\(#[0-9]+\)$/)} up && /^\+go [0-9]/{n++} END{print n+0}')"
