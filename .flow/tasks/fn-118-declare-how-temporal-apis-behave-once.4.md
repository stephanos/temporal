---
satisfies: [R3, R7]
---
# fn-118-declare-how-temporal-apis-behave-once.4 Derive Case waiting from hints in the lowering and refuse undeclared visibility

## Description
The lowering reads the hints and emits a bounded wait for a read after an eventually visible write, one read after a write visible at once, and refuses a read after a write with no declared visibility.

**Size:** M
**Files:** `tools/umpire/lower/realization.go` (commands :495-532, poll :822-853), `lower/internal/producer/build.go:89-104`, lowering tests and fixtures.
**Touches:** [tools/umpire/lower/**, tools/umpire/model/**, model/lifter/testdata/**]

### Approach
- For each path, find what each read waits on in path order across scripts, using task 1's classification: a preceding write (look up its visibility: at once -> task 3's read-once form, eventually -> bounded condition wait) or an asynchronous cause (bounded condition wait with the cause kind's declared bound). Emit task 3's instruction fields with the bound and hint position.
- Refusal names both methods (R3 errors), located at the read's source. A read that waits for an asynchronous cause whose kind has no declared bound is refused the same way, naming the cause kind and the read.
- R7: for each adopted hint, a test removes it from a fixture IR and asserts the affected Case is refused at lowering.
- Realizations still carry explicit polls until task 5; the lowering accepts them, and the final rule (also after task 5) is: an explicit poll is accepted and exempts its read from R3's refusal only when it carries a recorded reason, listed under R4's errors clause; without a reason it is refused. Existing Cases are unchanged here.

### Investigation targets
**Required:**
- `tools/umpire/lower/realization.go:480-860`
- `tools/umpire/lower/internal/producer/build.go`
**Optional:**
- `tools/umpire/lower/migration_golden_test.go`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/lower/...
make umpire-check-cases
```

### Execution constraints
- Existing checked-in Cases unchanged in this task; Program deltas arrive in task 5.
## Acceptance
- [ ] Eventually visible write -> bounded condition wait; visible-at-once write -> single read (fixtures for both).
- [ ] Read after write with no declared visibility is refused naming both methods.
- [ ] Each adopted hint has a removal test that makes lowering refuse the affected Case.
- [ ] Existing Cases unchanged; lowering tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
