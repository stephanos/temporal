#!/usr/bin/env bash
set -euo pipefail
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
scratch=/tmp/fn109-canonical-worker.f61ARZ5A
for name in amended-final-characterization amended-final-lint amended-final-errortype amended-final-gofmt amended-final-consumers amended-final-architecture amended-final-boundaries amended-final-target amended-final-validate; do
  receipt="$scratch/$name.json"
  test -f "$receipt"
  jq -e '.exit == 0 and .source_stable == true and .source_before == .source_after and (.source_before | length) == 996 and .source_before["tools/gomad3/internal/canonicaljson/canonical.go"] == "738375708f711bb44094f57a346ec9131159f5ef61da402b3a62ec82b15f9a41" and .source_before["tools/gomad3/internal/canonicaljson/canonical_characterization_test.go"] == "274b979156bd6f4ff47c6d5daa454dee51a670d65c3ba694ebed87e647bff747"' "$receipt" >/dev/null
  jq -r '.log_sha256 + "  " + .log' "$receipt" | sha256sum -c -
  jq '{argv,exit,source_stable,counts,skipped_tests}' "$receipt"
  case "$name" in
    amended-final-characterization|amended-final-consumers|amended-final-architecture|amended-final-boundaries|amended-final-target)
      jq -s -e --slurpfile report "$receipt" 'reduce (.[] | select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip"))) as $event ({}; .[$event.Package][$event.Action] += 1 | if ($event.Test | contains("/")) then . else .[$event.Package]["top_" + $event.Action] += 1 end) | . == $report[0].counts' "$scratch/$name.log" >/dev/null
      printf 'raw test event counts match metadata: %s\n' "$name"
      ;;
  esac
done
git show a683e64af560322014e14f3a1ef3953b27cad96a:tools/gomad3/internal/canonicaljson/canonical.go | diff - <(sed '/^\tcase reflect.Invalid, reflect.Bool, reflect.Int,/,+3d' tools/gomad3/internal/canonicaljson/canonical.go)
git show a683e64af560322014e14f3a1ef3953b27cad96a:tools/gomad3/internal/canonicaljson/canonical_test.go | cmp - tools/gomad3/internal/canonicaljson/canonical_test.go
printf 'source reconstruction and existing tests: unchanged\n'
