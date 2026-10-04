#!/usr/bin/env bash
set -eu
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
while read -r task18_digest task18_file; do
  case "$task18_file" in
    tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go|tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/process_volume.go|tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/volume.go|tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/export_test.go|tools/gomad3/toolchain/version/version.json|tools/gomad3/Makefile|tools/gomad3/simulation_gate_selection_test.go|tools/gomad3/runner/internal/execution/simulation_root_integration_test.go) continue ;;
  esac
  printf '%s  %s\n' "$task18_digest" "$task18_file" | sha256sum --check -
done < .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-18/baseline-source.sha256
for task18_file in fs.go process_volume.go volume.go export_test.go; do
  cmp "/tmp/gomad-task18.U1uS6Z/baseline-overlay/src/internal/gomadfs/$task18_file" "/tmp/gomad-task18.U1uS6Z/old-go/src/internal/gomadfs/$task18_file"
done
for task18_file in tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/*.go; do
  cmp "$task18_file" "/tmp/gomad-task18.U1uS6Z/go/src/internal/gomadfs/${task18_file##*/}"
done
sha256sum /tmp/gomad-task18.U1uS6Z/go/src/runtime/gomad_task14_development.go
