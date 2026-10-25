---
satisfies: [R4]
---
# fn-250-gomad-general-purpose-output-predicates.3 Temporal softassert preset on the launcher; non-Temporal golden fixture

## Description
Move the softassert rule out of the host and into the Temporal launcher as a declared preset: json-line predicates for failed-assertion/FailedAssertion == true and msg/message prefix failed assertion: , identity field msg with the prefix stripped, plus a text-format fallback for a bare failed assertion:  line; severity fail. --soft-assertion-report-only on the launcher becomes sugar that sets the preset severity to report. Add --output-predicates FILE to the host campaign, explore and run-guest verbs. Add a go-run fixture under testdata/ that is not Temporal and declares one text/line_regex predicate and one json-line/field_equals predicate; record its golden.

## Acceptance
go-test workloads behave as before through the preset, and --soft-assertion-report-only still works. The host CLI exposes --output-predicates FILE and no package-specific flag. The non-Temporal fixture fails on a hit, records a report hit without changing outcome class, dedups identical messages and groups two seeds with the same message under one signature; its golden is retained under testdata/ and checked by cargo test.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
