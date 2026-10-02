#!/bin/sh
# fn-110.1 baseline measurements. Read-only; run from the repository root:
#   sh .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-measure.sh
set -eu

patch_file="tools/gomad3/toolchain/runtime/go1.27.1.patch"
overlay_dir="tools/gomad3/toolchain/runtime/overlay"
descriptor="tools/gomad3/toolchain/version/version.json"
archive="tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz"

echo "## identity"
echo "head $(git rev-parse HEAD)"
echo "toolchain_tree_at_head $(git rev-parse HEAD:tools/gomad3/toolchain)"
echo "toolchain_status_lines $(git status --short -- tools/gomad3/toolchain | wc -l | tr -d ' ')"
echo "toolchain_diff_against_head_lines $(git diff HEAD -- tools/gomad3/toolchain | wc -l | tr -d ' ')"
echo "patch_sha256 $(shasum -a 256 "$patch_file" | cut -d' ' -f1)"
echo "archive_sha256 $(shasum -a 256 "$archive" | cut -d' ' -f1)"
echo "descriptor_archive_sha256 $(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["archive"]["sha256"])' "$descriptor")"
echo "descriptor_sha256 $(shasum -a 256 "$descriptor" | cut -d' ' -f1)"
echo "build_key $(cat tools/gomad3/.toolchain/build-key)"

echo "## patch"
echo "patch_bytes $(wc -c < "$patch_file" | tr -d ' ')"
echo "patch_lines $(wc -l < "$patch_file" | tr -d ' ')"
echo "patch_files $(grep -c '^diff --git' "$patch_file")"
echo "patch_hunks $(grep -c '^@@ ' "$patch_file")"
echo "## patch numstat (added deleted path), from git apply --numstat"
git apply --numstat "$patch_file"
git apply --numstat "$patch_file" | awk '{a+=$1; d+=$2} END {print "patch_added " a; print "patch_deleted " d}'
echo "## patch sections (bytes lines hunks path)"
awk '
	/^diff --git / { if (path != "") print bytes, lines, hunks, path; path = substr($3, 3); bytes = 0; lines = 0; hunks = 0 }
	{ bytes += length($0) + 1; lines++ }
	/^@@ / { hunks++ }
	END { print bytes, lines, hunks, path }
' "$patch_file"

echo "## overlay"
echo "overlay_files $(find "$overlay_dir" -type f | wc -l | tr -d ' ')"
echo "overlay_bytes $(find "$overlay_dir" -type f -print0 | xargs -0 cat | wc -c | tr -d ' ')"
echo "overlay_lines $(find "$overlay_dir" -type f -print0 | xargs -0 cat | wc -l | tr -d ' ')"
echo "overlay_runtime_gomad_go_bytes $(wc -c < "$overlay_dir/src/runtime/gomad.go" | tr -d ' ')"
echo "overlay_runtime_gomad_go_lines $(wc -l < "$overlay_dir/src/runtime/gomad.go" | tr -d ' ')"

echo "## source sets"
python3 - "$descriptor" "$patch_file" "$overlay_dir" <<'EOF'
import json, os, sys
descriptor, patch_file, overlay_dir = sys.argv[1:4]
d = json.load(open(descriptor))
patched = sorted(l.split()[2][2:] for l in open(patch_file) if l.startswith("diff --git "))
overlay = sorted(os.path.relpath(os.path.join(r, f), overlay_dir) for r, _, fs in os.walk(overlay_dir) for f in fs)
print("patch_allowlist_entries", len(d["patch_allowlist"]))
print("patch_allowlist_equals_patched_files", d["patch_allowlist"] == patched)
print("overlay_allowlist_entries", len(d["overlay_allowlist"]))
print("overlay_allowlist_equals_overlay_tree", d["overlay_allowlist"] == overlay)
EOF

echo "## host"
df -h . | tail -1
du -sh tools/gomad3/.toolchain/builds/* | sed 's/^/build_dir /'
