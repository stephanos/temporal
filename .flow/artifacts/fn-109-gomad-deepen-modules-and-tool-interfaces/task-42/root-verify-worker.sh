#!/usr/bin/env bash
set -euo pipefail
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
out=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42
for scope in characterization lint errortype gofmt consumers architecture boundaries target validate; do
  name=final-$scope-nil-selection
  receipt=$out/raw/$name.json
  expected_exit=0
  if test "$scope" = lint; then expected_exit=1; fi
  jq -e --argjson expected "$expected_exit" '.exit == $expected and .source_stable == true and .source_count == 997 and .source_before_sha256 == .source_after_sha256' "$receipt" >/dev/null
  before=$(jq -r '.source_before' "$receipt")
  after=$(jq -r '.source_after' "$receipt")
  cmp "$out/$before" "$out/$after"
  jq -r '.source_before_sha256 + "  " + .source_before, .source_after_sha256 + "  " + .source_after, .log_sha256 + "  " + .log' "$receipt" | (cd "$out" && sha256sum -c -)
  test "$(sed -n '\|  tools/gomad3/internal/compatibilitypack/policy.go$|s/ .*//p' "$out/$before")" = c7d137b5d1df036f25cbdc11ebeb31da6c1dc0e1a26adc5e00dfe276df7d9642
  test "$(sed -n '\|  tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go$|s/ .*//p' "$out/$before")" = e3e498aa5750ec118480f68416170fbf002e2d5fa4cb879d59599f9e5a0266d5
  sha256sum -c "$out/$before" >/dev/null
  if test "$scope" = characterization || test "$scope" = consumers || test "$scope" = architecture || test "$scope" = boundaries || test "$scope" = target; then
    jq -s -e --slurpfile report "$receipt" 'reduce (.[] | select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip"))) as $event ({}; .[$event.Package][$event.Action] += 1 | if ($event.Test | contains("/")) then . else .[$event.Package]["top_" + $event.Action] += 1 end) | . == $report[0].counts' "$out/raw/$name.log" >/dev/null
  fi
  jq '{name,argv,exit,source_count,source_stable,counts,skipped_tests}' "$receipt"
done
jq -e '.exit==0 and .source_stable and .source_count==997' "$out/raw/base-characterization-nil-selection.json" >/dev/null
base_manifest=$(jq -r '.source_before' "$out/raw/base-characterization-nil-selection.json")
test "$(sed -n '\|  tools/gomad3/internal/compatibilitypack/policy.go$|s/ .*//p' "$out/$base_manifest")" = c7bde3286e60001d4235897c1b36dd7a5cdb4a0390114097cfc09971fddaf84f
test "$(sed -n '\|  tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go$|s/ .*//p' "$out/$base_manifest")" = e3e498aa5750ec118480f68416170fbf002e2d5fa4cb879d59599f9e5a0266d5
jq -r '.log_sha256 + "  " + .log' "$out/raw/base-characterization-nil-selection.json" | (cd "$out" && sha256sum -c -)
cmp <(jq -s '[.[] | select(.Action == "output" and (.Output | contains("decision=") or contains("loader_error="))) | {Test,Output}]' "$out/raw/base-characterization-nil-selection.log") <(jq -s '[.[] | select(.Action == "output" and (.Output | contains("decision=") or contains("loader_error="))) | {Test,Output}]' "$out/raw/final-characterization-nil-selection.log")
git show 533aa074337bc1e21e52c306ddb5a19e93fdfdbe:tools/gomad3/internal/compatibilitypack/policy.go | cmp - <(sed '/^\t\t\tcase FactMalformedLinkname, FactNoReviewedGoSource:$/,+1d' tools/gomad3/internal/compatibilitypack/policy.go)
sha256sum -c "$out/manifests/protected.sha256" >/dev/null
test "$(wc -l < "$out/manifests/protected.sha256")" = 995
cmp "$out/admission-base/lint.log" "$out/raw/base-additive-lint-nil-selection.log"
cmp <(sed '/policy.go:196:4: missing cases/,+2d; /^8 issues:/,$d' "$out/raw/base-additive-lint-nil-selection.log") <(sed '/^7 issues:/,$d' "$out/raw/final-lint-nil-selection.log")
for kind in production test; do
  expected=$(jq -r '.sha256' "$out/$kind-diff.json")
  actual=$(jq -jr '.raw_diff' "$out/$kind-diff.json" | sha256sum | cut -d ' ' -f1)
  test "$actual" = "$expected"
done
printf 'Root verified all nine latest captures, raw counts/hashes, BASE/final literals, exact production reconstruction, 995 protected inputs, unchanged lint residuals and encoded diffs.\n'
