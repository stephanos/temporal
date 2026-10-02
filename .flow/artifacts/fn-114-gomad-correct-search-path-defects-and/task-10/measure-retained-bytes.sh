#!/bin/sh
# Usage: measure.sh ARTIFACTS_ROOT
# Prints the bytes an artifacts root retains, three ways.
set -eu
root="$1"
printf 'on_disk_kib (du -sk; a hard-linked file counts once): '
du -sk "$root" | awk '{print $1}'
printf 'distinct_file_bytes (each file once, by inode): '
find "$root" -type f -exec stat -f '%i %z' {} + | sort -u | awk '{s+=$2} END {print s+0}'
printf 'every_path_bytes (each path counts its file; the bytes without sharing): '
find "$root" -type f -exec stat -f '%z' {} + | awk '{s+=$1} END {print s+0}'
printf 'artifact_directories: '
find "$root" -name manifest.json -type f | wc -l | tr -d ' '
printf 'target_paths_in_artifacts: '
find "$root" -name target -type f -not -path '*/targets/*' | wc -l | tr -d ' '
printf 'pool_entries: '
find "$root/targets" -type f -name 'sha256-*' 2>/dev/null | wc -l | tr -d ' '
printf 'pool_bytes: '
find "$root/targets" -type f -name 'sha256-*' -exec stat -f '%z' {} + 2>/dev/null | awk '{s+=$1} END {print s+0}'
