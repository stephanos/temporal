#!/usr/bin/env bash
set -u
task18_label="$1"
task18_dir="$2"
shift 2
TIMEFORMAT='ELAPSED_SECONDS=%3R'
(
  cd "$task18_dir" || exit
  printf 'COMMAND:'
  printf ' %q' "$@"
  printf '\n'
  time timeout 600 "$@"
  task18_exit=$?
  printf 'EXIT_CODE=%s\n' "$task18_exit"
  exit "$task18_exit"
) > "/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-18/$task18_label.log" 2>&1
task18_exit=$?
printf '%s exit=%s\n' "$task18_label" "$task18_exit"
exit "$task18_exit"
