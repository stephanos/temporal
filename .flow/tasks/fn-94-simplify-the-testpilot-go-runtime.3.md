---
satisfies: [R1]
---
# fn-94-simplify-the-testpilot-go-runtime.3 Correlated monitor carries states as atom plus fields, as Lean does

## Description
The monitor half of E1 (decision: read, recorded in fn-94.1): Go's correlated monitor tracks the current state as atom plus fields and matches transitions on both, exactly as Lean's `StateValue` equality does. No wire, Lean or fixture change. This is the spec's early proof point.

**Size:** M
**Files:** `common/testing/testpilot/internal/verification/correlated.go`, its test file(s) (`correlated_test.go` or a new `correlated_state_test.go`)
**Touches:** [common/testing/testpilot/internal/verification/correlated.go, common/testing/testpilot/internal/verification/correlated_test.go, common/testing/testpilot/internal/verification/correlated_state_test.go]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.5 (`verification`)

## Approach
- `correlatedOperation.state` (`correlated.go:37`) becomes the state atom plus its fields. It starts from `InitialState` with `InitialStateFields` (`:439`), and moves to `out.State` with `out.StateFields` (`:528`).
- Transition authorization (`:453-457`) and candidate counting (`:478-483`) match `PriorState` and `PriorFields` against the current state's atom and fields. Candidate counting feeds obligation work accounting, so Lean-produced Cases (whose fields follow from the atom) keep identical work totals; the corpus pins that.
- Compare fields as ordered lists of model values (`proto.Equal` element-wise), matching Lean's derived `BEq` on `List Atom`.
- Focused tests, each on a hand-built correlated Case: (a) `prior_fields` that disagree with the fields the run reached, where the transition must not be taken (the same outcome Lean's monitor gives for an unmatched prior); (b) `initial_state_fields` that disagree with the first transition's `prior_fields`; (c) a consistent Case whose outcome and work totals match today's. If an existing Lean/Go differential harness covers correlated Cases (fn-89.3's Verdict-identity test), add (a) to it; otherwise pin Lean's outcome in the test comment from the Lean definitions.
- Behavior pin: `make umpire-check-case-runtime-conformance` with no diff, including `expected.json` work fields.

## Investigation targets
**Required:**
- `common/testing/testpilot/internal/verification/correlated.go:30-45,430-535`
- `model/Testpilot/Correlated.lean:280-290,370-400` — plan initial state and transition decode
- `model/Shared/SemanticData.lean:10-30` — `StateValue` equality
**Optional:**
- the Lean correlated monitor's transition step (`CorrelatedObligation`), to confirm the unmatched-prior outcome
- `common/testing/testpilot/testdata/case-runtime-conformance/correlated.json`

## Key context
- If any committed fixture diverges after the change, stop: that is a Lean/Go disagreement the corpus never showed. Report it and re-evaluate E1 per the spec's early proof point.

## Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/internal/verification/...
make umpire-check-case-runtime-conformance
go test -tags test_dep ./common/testing/testpilot/...
make lint-code-fast
```

## Acceptance
- [ ] The monitor's state is atom plus fields, initialized from `initial_state_fields` and advanced with `state_fields`.
- [ ] Transition authorization and candidate counting match `prior_state` with `prior_fields`.
- [ ] Tests (a)-(c) pass; the corpus, including work totals, shows no diff.
- [ ] `-race` tests and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
