---
satisfies: [R5, R7]
---
# fn-90-resolve-the-intermittent-live-testpilot.5 Resolve or close the Nexus pair ordering failure

## Description
Resolve (2), the pair evidence-ordering failure, or close it as not reproduced (R5; R7 only if the
cause is external). Conditional on fn-90.3's receipt. Either way, confirm R5's test-side clause:
every live assertion over the pair Case correlates by key.

**Size:** S if not reproduced; M if a Model or Producer fix is needed
**Files:** always: none, or `tests/testpilot_nexus_pair_case_test.go` if the audit finds a positional assertion. On a Model fix: `model/Temporal/Feature/Nexus/Pair/Model.lean`, `model/Temporal/Feature/NexusTests.lean` (where the pair fixture is produced), the Nexus realization, the regenerated `tests/testcore/testpilot/testdata/nexusPairTests-bothComplete-case.json` (and any other fixture the generator rewrites), `tests/testpilot_nexus_control_case_test.go` (pair cross-reference control)
**Touches:** [tests/testpilot_nexus_pair_case_test.go, tests/testpilot_nexus_control_case_test.go, tests/testcore/testpilot/quarantine.go, tests/testcore/testpilot/quarantine_test.go, model/Temporal/Feature/Nexus/Pair/**, model/Temporal/Feature/NexusTests.lean, model/Temporal/Case/Realization/Nexus.lean, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/case-runtime-conformance/**]

### Approach
- Audit first (always): every live assertion that reads pair-Case evidence (`requireCorrelatedNexusPairEvidence` at `tests/testpilot_nexus_pair_case_test.go:78` and any other reader, `grep -n NexusPair tests/*.go`) indexes by rule and scheduled event id, never by global position. Fix any positional read found.
- Not reproduced in fn-90.3: close with that evidence (retired identity, successor, count) plus the audit result. No quarantine.
- Reproduced with an ordering signature: read the signature. If a rule is unresolved because the Scenario or Contract fixes one completion order, correct the pair Model (or the Producer that derives it) so every order the server may record is admitted, regenerate with `make umpire-gen-case-runtime-conformance`, and check with `make umpire-check-case-runtime-conformance`. Then add a live control that forges a completion referencing the other instance's scheduled event and expects VIOLATED, following `TestTestpilotNexusControlForgedCompletionIsViolated` at `tests/testpilot_nexus_control_case_test.go:30`.
- Before any Lean edit: `flowctl show fn-88-veil-concrete-checker-as-the-umpire`. If fn-88 is open, do not land the Model change uncoordinated; block this task (`flowctl block`) naming fn-88 and the change, so that session re-pins its Pair counts (spec Edge Cases, fn-88 overlap). The same applies to fn-89, which rewrites this fixture's Contract after fn-90.
- Cause outside the repository (server or SDK race): open an upstream issue and implement the R7 quarantine for exactly that signature in `tests/testcore/testpilot/quarantine.go` (entry table plus the retry decision, comparing signatures with fn-90.2's field-by-field compare) and `quarantine_test.go` (offline, proving the R7 error cases); the live test calls it.
- Rerun `^TestTestpilotNexusPairCase$` for 200 process-mode iterations after any change.

### Investigation targets
**Required:**
- fn-90.3 receipt (signatures for (2))
- `tests/testpilot_nexus_pair_case_test.go`
- `model/Temporal/Feature/Nexus/Pair/Model.lean`, `model/Temporal/Feature/Nexus/Pair/Tests.lean` (read only; fn-88 edits Tests.lean)
- `Makefile:617-630` (fixture generation and check)

**Optional:**
- `tests/testpilot_nexus_control_case_test.go:30-115`

### Key context
- Do not edit `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**` or `model/Temporal/API/**` (other sessions); running the generator is fine.
- QLF-05: never widen a window or drop an obligation to make the rule satisfy.
## Acceptance
- [ ] Audit result recorded; no positional pair-evidence assertion remains.
- [ ] Either "not reproduced on the successor" with fn-90.3's evidence, or a fix with regenerated fixtures, a passing pair cross-reference VIOLATED control, and 200 zero-failure iterations of the pair test.
- [ ] Any quarantine matches the spec's contract and its offline unit test passes; otherwise the receipt says "no quarantine needed".
- [ ] `make lint-code-fast` clean; `make umpire-check-case-runtime-conformance` clean if fixtures changed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
