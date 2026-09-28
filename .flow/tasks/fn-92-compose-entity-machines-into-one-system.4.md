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
Added `from:`, `restrict:` and `extend:` to `machine`. A derived machine keeps its source's entity, state type, starts, ends, setup and evidence, generates its own Action catalog under its own name (so its Definition IDs are its own), and carries no `refines:` or abstract state field. Each step it keeps is a generated function that calls the source's dispatcher. `restrict:` keeps the listed actions, drops every timer and unobservable entry, and drops evidence lines for facts no kept row returns. `extend:` adds the author's results to an action's rows, ordered by the step order key under the derived IDs (`Umpire.Command.Derived`). The sort is a structural insertion sort, because `List.mergeSort` does not reduce in the `rfl` table law. `Nexus/Control` is now `from: pair extend: handlerReply: controlForgedStep`, and that one function is its only step code.

Pins are in `model/Umpire/Command/Tests/Derived.lean`: restricted catalogs and evidence, extended result order, restriction applied before extension, IDs owned by the derived machine, a refining source that is not inherited, and the located errors (restrict naming a timer, an absent action, or one twice; extend naming a dropped action or one twice; extension result at a disabled source; duplicate result; restrict/extend without `from:`; `from:` with authored keys; `from:` naming a non-machine). `Control/Tests.lean` pins the forged function and that the catalogs equal pair's. The Control fixture, the recorded control Run, and the `forgedCompletion` differential line are byte-identical. `make umpire-rerecord-pinned-runs` was not run. The Derived test module declares no Query, so the Umpire differential sweep, which is outside this task's touches, needs no new line (R14).

baseline: green (focused lake build, umpire-check-case-runtime-conformance, Go reader tests, pre-edit)
Gates: umpire-gen-case-runtime-conformance produced no diff; umpire-check-case-runtime-conformance rc=0; Go reader tests rc=0; lake build UmpireTests TemporalModelTests rc=0; LEAN_NUM_THREADS=1 make lint-model rc=0. The live tests/testpilot_assess_test.go:98 was not run (it needs the test cluster). The fixture and Run bytes it reads are unchanged.

stage: impl-review - ran [round 1 SHIP (codex fan-out, rid c846e9162ccb415d9a6bb5854510abd6, 3/3 draws SHIP, 0 findings)]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e3bfae2c0e99e4f4bc599e5f3213cc1ceb57fee2
- Tests: cd model && lake build Umpire.Command.Tests.Derived Temporal.Feature.Nexus.Control.Tests, cd model && lake build UmpireTests TemporalModelTests, make umpire-gen-case-runtime-conformance (no diff), make umpire-check-case-runtime-conformance, go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/..., LEAN_NUM_THREADS=1 make lint-model
- PRs: