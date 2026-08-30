---
satisfies: [R1, R4, R5, R6]
---

# fn-20-local-execution-semantic-conformance.4 Construct admitted Evidence and Result sets

## Description
Build the exported offline Go Run Evaluation controller around fn-18 admission, fn-19 operational status, the proven Task `.2` protocol, and Task `.3`'s private sibling adapter (R1/R4/R5/R6).

**Size:** M
**Files:** `tools/umpire/runevaluation/run_evaluation.go`, `tools/umpire/runevaluation/result.go`, `tools/umpire/runevaluation/run_evaluation_test.go`, `tools/umpire/runevaluation/result_test.go`
**Touches:** [tools/umpire/runevaluation/run_evaluation.go, tools/umpire/runevaluation/result.go, tools/umpire/runevaluation/run_evaluation_test.go, tools/umpire/runevaluation/result_test.go]

### Approach
- Export only `Check(admittedSet)`; require the exact admitted fn-19 four-member closure, validate profile/program/source restrictions, and resolve the verified sibling internally before constructing a request.
- Reuse fn-19's operational-precedence function; do not infer operational status from checker output or gate semantic checking on operational success.
- Convert the validated response directly into fn-18 Evidence and Result transports, preserving all semantic content and adding only exact artifact bindings/provenance.
- Verify `observationKnownGaps` and the exact canonical `resultKnownGaps` union against the separately bound Run/RawEvidence Known Gap collections; Go never invents, reclassifies, or drops one.
- Run fn-18 in-memory encoding/admission and complete-set validation over the four original members plus two derived members; return only the admitted set.
- Make deterministic provenance depend only on input/checker semantic sources so recomputation is byte-identical; never publish or persist an intermediate.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — exact Evidence/Result Generated View
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.8.md` — complete-set relationship admission
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.3.md` — operational precedence authority
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — admitted four-member output API
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` §Normative v2 wire contract — status, Known Gap, and identity matrix
## Acceptance
- [ ] Only the exact four-member input reaches the internally resolved checker and all original member bytes/bindings are unchanged in the six-member output.
- [ ] Every valid operational/Observation Evaluation/semantic matrix row produces exactly one admitted Evidence and Result; invalid combinations produce no set.
- [ ] Accepted-outcome identity is independently validated and has the required stability/sensitivity properties.
- [ ] Missing/duplicate/unexpected verdicts, invalid partitions/Evidence Links/dispositions/diagnostics, invalid Known Gap propagation, and crossed bindings fail as output invariants.
- [ ] Repeated checking returns byte-identical derived members and admitted manifests without publication or persisted intermediate state.
## Done summary
Implemented offline local Run Evaluation as a narrow `Check` boundary over the exact admitted four-member execution set, producing deterministic in-memory Evidence/Result admission with preserved bytes, independent status authority, exact Known Gap semantics, and plan-sensitive accepted checksums. Synchronized the bounded source-or-destination endpoint closure rule across Go and Lean, propagated the authoritative plan-aware fn20.2 fixtures, and exposed structured failure classification only through methods on the private error concrete.

Focused RED/GREEN cases, cross-language goldens, model lint, aggregate Go tests, protocol fuzz/race, non-mutating golangci, normal vet, and the physical-temp regression gate passed. Task-scoped errortype reproduced only the inherited `tools/umpire/runtime/errors.go:60:9 et:unw+`; the roadmap-wide CLI package and Make target remain deferred to task `.6` as at baseline. Codex review session `01a050b1-1a59-7fd3-b7df-a56b38aaca41` reached SHIP after one NEEDS_WORK round; both findings are fixed, and memory capture was attempted but the repository memory store is not initialized.

stage: impl-review - ran [2026-08-30T03:22:49Z..2026-08-30T03:38:35Z]
## Evidence
- Commits: 38c3cfcde5ead9b827a7c26dfd32205cf9dc5e72, d98c4a1ccffe6b99967ec3dd36c0ee38a63c9785, 87809c8e83658dea723bb4bb668e19a0e5443a5a, c558270165f71452c7bf778d6a04ba4fcd4ee794, baf73df3f106a9073613b8b85d47df2cc19d9c43, 335b110ed1ee38ada456a956e7e493ea3b88460d
- Tests: baseline: green for model checker and Go run-evaluation suites; GATE_SKIPPED:unittest:green-receipt 7fadb040; roadmap CLI package and Make target deferred to task .6, RED_EXPECTED: go test -tags test_dep -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/runtimeengine/... ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... (7 authoritative fn20.2 golden/manifest mismatches reproduced at base and current before reconciliation), RED_EXPECTED: go test -tags test_dep -count=1 ./tools/umpire/internal/artifactv2 -run TestExpectedEvaluationOutcomeChecksumChangesWithPlanOnly (plan-only mutation initially retained checksum), RED_EXPECTED: cd model && mise exec -- lake build Umpire.Artifact.Tests.Set (destination-endpoint native_decide initially false), RED_EXPECTED: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go test -tags test_dep -count=1 ./tools/umpire/runevaluation -run TestCheckWithCheckerErrorsExposeStableClassification (errors.As initially failed), cd model && mise exec -- lake build Umpire.Artifact.Tests.Set, cd model && mise exec -- lake lint, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go test -tags test_dep -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/runtimeengine/... ./tools/umpire/runtime/... ./tools/umpire/runevaluation/..., TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go test -tags test_dep -race -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/runtimeengine/... ./tools/umpire/runtime/... ./tools/umpire/runevaluation/..., TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go test -tags test_dep -run '^$' -fuzz '^FuzzDecodeCheckerResponse$' -fuzztime=5s ./tools/umpire/runevaluation, .bin/golangci-lint-v2.13.1 run --verbose --build-tags test_dep --timeout 10m --fix=false --new-from-rev=7584fbc27513b05e39607fe3a1871b54fe88fffa --config=.github/.golangci.yml ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/internal/runtimeengine/... ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... (0 issues), TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go vet -tags test_dep ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/internal/runtimeengine/... ./tools/umpire/runtime/... ./tools/umpire/runevaluation/..., INHERITED_RED: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go vet -tags test_dep -vettool=.bin/errortype -style-check=false ./tools/umpire/runevaluation/... -> tools/umpire/runtime/errors.go:60:9 et:unw+, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, GREEN_RECEIPT:unittest:335b110e for command make umpire-check-regression, impl-review codex session 01a050b1-1a59-7fd3-b7df-a56b38aaca41: SHIP
- PRs: