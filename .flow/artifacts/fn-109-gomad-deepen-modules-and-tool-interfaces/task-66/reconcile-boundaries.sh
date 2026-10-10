#!/usr/bin/env bash
set -eu
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/fixtures
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-66
previous=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-65/sources-cbb9d9685512c60acb99596e5dfe360610c2a96c4973982c078ef2d1ed4f46a4.sha256
awk '
    $2 ~ /^tools\/gomad3\/.*\.go$/ && $2 !~ /_test\.go$/ { print; next }
    $2 ~ /^tools\/gomad3\/.*_generated_test\.go$/ { print; next }
    $2 ~ /^tools\/gomad3\/(architecture_test\.go|runner_consumer_test\.go|go\.(mod|sum)|Makefile|version_generated\.mk)$/ { print; next }
    $2 ~ /^tools\/gomad3\/(toolchain\/version|toolchain\/runtime|deterministicio\/schema|deterministicio\/boundary|choice\/schema|simulation\/schema|internal\/compatibilitypack|internal\/gomadtool\/generation|internal\/gomadtool\/manifestgen)\// { print; next }
    $2 ~ /^tools\/gomad3integration\/(go\.(mod|sum)|qualification\/tests\.(json|generator\.json))$/ { print; next }
    $2 ~ /^(Makefile|go\.(mod|sum)|\.github\/\.golangci\.yml)$/ { print }
' "$previous" | sort -u > "$packet/boundary-consumed-inputs.sha256"
test -s "$packet/boundary-consumed-inputs.sha256"
sha256sum -c "$packet/boundary-consumed-inputs.sha256" > /tmp/fn10966-boundary-input-check.log 2>&1
printf 'PASS: %s retained boundary implementation/schema/template/output input hashes match current files\n' "$(wc -l < "$packet/boundary-consumed-inputs.sha256")"
test -z "$(git diff --name-only 41ca15aefb849747a7fe73cb1821e0b7bf93bf58 b32dad53fc544ab75d56f6b9c41fba9b99a75858 -- tests go.mod go.sum tools/gomad3integration)"
printf 'PASS: qualification package source/module inputs have no tracked changes since the retained validator HEAD\n'
test "$(git diff --name-only b32dad53fc544ab75d56f6b9c41fba9b99a75858 -- tools/gomad3 tools/gomad3sim tools/gomad3integration)" = tools/gomad3/runner/runner_test.go
test -z "$(git diff --name-status b32dad53fc544ab75d56f6b9c41fba9b99a75858 --diff-filter=ACDR -- tools/gomad3 tools/gomad3sim tools/gomad3integration)"
printf 'PASS: no package/source filename additions, deletions or production modifications; only admitted existing test body changes\n'
printf 'Retained receipts: task-65/final-boundaries-corrected.json and task-65/final-validate-corrected.json. This reconciliation creates no fresh execution, native pass, external-cache qualification or closure of prior source acceptance gaps.\n'
