#!/usr/bin/env bash
# One validation stage as its own process: stage.sh LABEL. Started by
# scheduler.py with the driver's environment (QUALDRV_*, QUAL_ROOT) plus
# QUALDRV_LANE and QUALDRV_STAGE.
set -euo pipefail
set -o noclobber
shopt -s nullglob dotglob
export LC_ALL=C
test "$#" -eq 1
label="$1"
test "$label" = "$QUALDRV_STAGE"
# shellcheck source=lib.sh
source "$QUALDRV_DIR/lib.sh"
trap 'status=$?; if [[ "$status" -ne 0 ]]; then ledger "$(date -u +%FT%TZ) STEP-FAIL $label $status lane=${QUALDRV_LANE:-0}" > /dev/null; fi' EXIT
qual_context
qual_base_env
qual_stage_state
run_step "$label"
