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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
