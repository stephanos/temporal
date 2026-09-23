---
satisfies: [R6]
---
# fn-22-deterministic-replay-semantic.2 Declare every result's evidence on witnessed rows in the Producer

## Description
Change the Producer so that each witnessed row declares the evidence of every result of its (state, action) pair, each kind projected to the result whose fact records it, while the witness's own step stays the confirmed one: `derivedEvidence` and `resolveEvidence` enumerate the transition table's results for the witnessed rows rather than the witness's steps alone, `evidenceDeclarations` declares every kind so found, and the `evidence.action-repeated` rejection applies per (action, result) rather than per action. A kind is declared once: one the witness's step already declares, or that another result of the same row records, is projected to the row that records it; a kind two *different* rows would record is rejected by name (`evidence.kind-ambiguous`, naming both rows) rather than admitted as a generic `projection.admission` failure. An alternative result's evidence is projected to its own row and confirms the same silent prefix as the witness's step, since the steps before the row that record nothing are confirmed by whichever result's evidence arrives. Regenerate the goldens and the functional fixtures; the live suite's satisfied Runs stay satisfied, since an undeclared kind that does not occur was never read. Add the Lean check the proof needs: on a produced Case whose Model carries an alternative result, the alternative's event kind is declared and projected to its own row, pinned on a lamp-sized Model in `Umpire.Command.Tests` before the control exists.

### Approach
- The transition table already carries every result of a witnessed pair (`Projection.check`); this task changes the evidence, not the table.
- `make umpire-gen-goldens umpire-gen-case-runtime-conformance` regenerate; the diff is reviewed as the change's own evidence.

### Quick commands
`cd model && lake build UmpireTests && cd .. && make umpire-gen-goldens umpire-gen-case-runtime-conformance umpire-check-goldens umpire-check-case-runtime-conformance && go test -count=1 -tags test_dep ./tests/testcore/testpilot/`

**Size:** M
**Files:** `model/Umpire/Case/Producer.lean`, `model/Umpire/Case/Projection/**`, `model/Umpire/Case/CompilerTests.lean`, `model/Umpire/Command/Tests/**`, `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `tests/testcore/testpilot/testdata/*-case.json`, `model/**/Fixtures/**`, `common/testing/testpilot/testdata/case-runtime-conformance/**`
**Touches:** `model/Umpire/Case/**`, `model/Umpire/Command/Tests/**`, `model/Temporal/Feature/**/Tests.lean`, `tests/testcore/testpilot/testdata/**`, `common/testing/testpilot/testdata/case-runtime-conformance/**`, `model/**/Fixtures/**`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK and revised through the six plan review rounds the spec's **Plan review** section records (SHIP on round six, 2026-09-22).
## Acceptance
- [x] Every result of a witnessed row has its evidence kind declared and projected to its own row; the witness's step is the confirmed one; a Case with no alternative results is byte-identical to before.
- [x] Goldens and functional fixtures are regenerated, the conformance check passes, and every fixture still prepares.
- [x] A Lean test pins, on a Model with an alternative result, that the alternative's kind is declared and projected to the row that records it, that a kind two rows would record is rejected by name, and that the alternative confirms the witness's silent prefix.
## Done summary
`Umpire.Case.Producer.alternativeRules` walks the witness with its prior states and, for every result of a witnessed (state, action) pair other than the one taken, declares the evidence its facts imply under the machine's `evidence:` lines and projects it to that result's own row, inheriting the silent steps before the row; a kind the witness's step or another result of the same row records is projected once to every row that records it, a kind two different rows would record rejects as `evidence.kind-ambiguous` naming it, and a kind the realization does not admit as `evidence.kind-unknown`. The alternatives' rules feed the projection declaration, the Program's evidence declarations and its lifts; the witness's rules alone feed the field relations. Every existing Model's rows are deterministic, so every existing fixture and golden is byte-identical; pinned in `Umpire.Case.Tests.Producer` on a lamp whose toggle ends bright or stuck.

## Evidence
- Commits:
- Tests:
- PRs:
