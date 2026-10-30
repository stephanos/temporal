#!/usr/bin/env bash
set -euo pipefail
cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn25410-controls/model

defaults=$(sed -n '/^INNER_BUDGET=/,/^TIMEFORMAT=/p' scripts/time-gate.sh)
warnings=$(sed -n '/^awk -v /p' scripts/time-gate.sh)
expect_warnings() {
  local expected=$1; shift
  local actual
  actual=$(/bin/bash -euo pipefail -c "
UMPIRE_INNER_BUDGET=\$3
UMPIRE_GATE_BUDGET=\$4
$defaults
inner=\$1
total=\$2
$warnings
" -- "$@")
  if [ "$actual" != "$expected" ]; then
    printf 'timing warnings: expected <%s>, got <%s>\n' "$expected" "$actual" >&2
    exit 1
  fi
}

inner_warning='note: inner loop over soft target (fn-210 R30)'
gate_warning='note: gate over soft target (fn-210 R30)'
expect_warnings '' 10 60 '' ''
expect_warnings "$inner_warning" 10.001 60 '' ''
expect_warnings "$gate_warning" 10 60.001 '' ''
expect_warnings '' 2 4 2 4
expect_warnings "$gate_warning"$'\n'"$inner_warning" 3 5 2 4

