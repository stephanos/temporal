---
satisfies: [R1]
---
# fn-221-gomad-functional-suite-triage-sweep-on.1 Build the sweep harness and prove it on two packages

## Description
Add a sweep driver under internal/backends/wasm (developer command or test-only producer; decide and record, following the wasmmeasure precedent of explicit flags and refused inherited GOMAD_* variables) that enumerates test functions under tests, reconciles the count against go test -list, builds each package once with backend.TemporalGuestBuildTags(), and runs each function through the existing explore go-test path with one seed, --parallel=1, a per-test timeout and an overall budget. Use the one-command setup installation if fn-220 has landed, otherwise a throwaway configuration derived from the integration fixtures, and record which. Classify outcomes into pass, guest failure, refused import, capacity exceeded, timeout, build failure, infrastructure (retried once). Early proof point: run on tests/gomadfunctional and one ordinary tests package capped at ten functions.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/** (harness location decided in task), tools/gomad/internal/backends/wasm/README.md

## Acceptance
- [ ] Harness refuses inherited GOMAD_* variables and unbounded runs
- [ ] Enumerated count reconciles against go test -list for the same packages
- [ ] Every function in the two proof packages receives exactly one class
- [ ] Configuration source (fn-220 or throwaway) recorded

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
