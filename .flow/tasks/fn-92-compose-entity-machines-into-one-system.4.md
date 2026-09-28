---
satisfies: [R6, R9, R10, R14]
---
# fn-92-compose-entity-machines-into-one-system.4 restrict: and extend: keys, Control over Pair with byte-identical Case

## Description
Add `from:`, `restrict:`, and `extend:` keys to `machine` with the filtering, ordering, and inheritance rules the spec states, and rewrite `Nexus/Control` as `from: pair extend: handlerReply: …` such that its Case fixture and the recorded control Run stay byte-identical (R6, R9, R10).

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (machine keys and elaborator), `model/Umpire/Command/Derived.lean` (new), `model/Umpire/Command/Tests/Derived.lean` (new), `model/UmpireTests.lean` (import), `model/Temporal/Feature/Nexus/Control/Model.lean`, `model/Temporal/Feature/Nexus/Control/Tests.lean`
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Derived.lean, model/Umpire/Command/Tests/Derived.lean, model/UmpireTests.lean, model/Temporal/Feature/Nexus/Control/**]

### Approach
- New keys in the `machineKey` category (`Syntax.lean:1663-1674`) handled in the machine elaborator (`:1873-2499`). `restrict:` keeps listed actions' rows and drops the rest from the catalog (else `CheckedTable.action_executable`, `Table.lean:118-119`, fails); filters `evidence:` for unreturned facts, `timers:`, `unobservable:`. `extend:` appends the author's results with located errors for a disabled source or a duplicate; results sorted by `stepOrderKey`; `restrict:` before `extend:`; derived machines drop `refines:` and the abstract field (`:2326-2350`) and re-own their catalogs under their own namespace so Definition IDs stay under the derived machine's family.
- Control: `machine nexusControl from: pair extend: handlerReply: controlForgedStep`, with `controlForgedStep` returning the one forged result for `handlerError false` (today's second result at `Control/Model.lean:84-85`); keep the machine name `nexusControl`, the set `nexusCallerControl` (`:141`), the state, outcome and fact spellings Pair and Control already share, the Query `forgedCompletion` (`:136`), `limits controlTwo` (`:131`), and the `case` block (`:151`). Today's result order is already sorted, so the fixture must not change.
- The differential line `Temporal.Feature.Nexus.Control.forgedCompletion` (`SearchDifferential.lean:30-104`) stays byte-identical, which pins the derived table's search behaviour on both backends.
- Verify: `make umpire-gen-case-runtime-conformance` produces no diff; the four readers of the recorded control Run pass unchanged (`go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/...` plus the live `tests/testpilot_assess_test.go:98`); `make umpire-rerecord-pinned-runs` is not run.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:1663-1674, 1873-2000, 2326-2350, 2422-2499`
- `model/Umpire/Model/Table.lean:93-119`
- `model/Temporal/Feature/Nexus/Pair/Model.lean:38-103`, `Control/Model.lean:74-155`
- `tools/umpire/replay/key_test.go:13-25`

### Key context
- fn-89 and fn-90 touched `Pair/Tests.lean` and the pair fixture only; `Pair/Model.lean`, the Control module, its fixture, and its Run are as they were when this spec was planned.

### Quick commands
```bash
cd model && lake build Umpire.Command.Tests.Derived Temporal.Feature.Nexus.Control.Tests TemporalModelTests.SearchDifferential
make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance
go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/...
```
## Acceptance
- [ ] `from:`/`restrict:`/`extend:` implemented with the stated rules; located errors pinned
- [ ] Control declares one extra-results function and no other step function; its Query outcome, witness, and differential line unchanged
- [ ] Control and Pair fixtures byte-identical; the four readers of the recorded control Run pass unchanged and the Run is not re-recorded
- [ ] `make umpire-check-case-runtime-conformance` and `make lint-model` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
