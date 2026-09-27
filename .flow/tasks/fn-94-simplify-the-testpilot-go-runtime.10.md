---
satisfies: [R4]
---
# fn-94-simplify-the-testpilot-go-runtime.10 Server Driver: shared ceiling check, InstructionPlan index, one Open, one prelude

## Description
Lane C and lane D2's server item together, because they edit the same three server files: the server's partial ceiling check calls the shared export, the Driver indexes `InstructionPlan` values, sessions read resolved defaults, `Open`/`OpenSession` merge, `InvokeRPC`/`PollRPC` share one prelude, and the claim-validity blocks become one helper called twice.

**Size:** M
**Files:** `common/testing/testpilot/temporal/server/{driver,session,handle}.go`, their tests, `temporal/server/README.md`
**Touches:** [common/testing/testpilot/temporal/server/**]

### Approach
- `validProfile` (`server/driver.go:97-107`) → shared ceiling export; drop `cloneProfile` (`:109`) in favour of `ProfileSpec.Snapshot`.
- Replace the `nodes`/`evidence` maps rebuilt from `program.Snapshot()` (`:176-199`) with an index over `InstructionPlan` values; sessions read `TimeoutMilliseconds()`/`MaxAttempts()` instead of `InstructionDefaults.Resolve` (`session.go:67,207`).
- Merge `Open`/`OpenSession` (`driver.go:146,154`).
- One `authorizeUnary` prelude for `InvokeRPC`/`PollRPC` (`session.go:73-92,108-129`); the coordinate/role/method authority checks stay and keep a rejection test each.
- `handle.go:89-112`: one claim-validity helper that runs under the lock, called at both sites (the second call re-checks after the lock is re-taken around `Accepts`).
- Update `temporal/server/README.md:4-6`.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/server/driver.go:90-200`
- `common/testing/testpilot/temporal/server/session.go:60-130,200-250`
- `common/testing/testpilot/temporal/server/handle.go:80-115`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/temporal/server/... ./common/testing/testpilot/temporal/...
make lint-code-fast
```

## Acceptance
- [ ] The server no longer rebuilds node or evidence maps from the snapshot or re-resolves defaults.
- [ ] One `Open`, one prelude; each authority check and the claim re-check after the lock re-take are pinned by tests.
- [ ] `-race` tests and lint pass; goldens unchanged.


## Done summary
The server Driver now checks Program limits at construction with the shared `ir.CheckCeilings`, against a local ceiling equal to execution's `hardLimits`. `ProfileSpec.Snapshot` replaces `cloneProfile`. Sessions index `InstructionPlan` values instead of node and evidence maps rebuilt from the snapshot, and read `TimeoutMilliseconds()`/`MaxAttempts()` instead of re-resolving `InstructionDefaults`. For both opcodes, the read method comes from `plan.Method()`. `Open` is a one-line adapter over `OpenSession`, which is the only constructor, because `Open` must keep the `testpilot.Driver` signature and the composite Driver needs the concrete `*Session`. `InvokeRPC` and `PollRPC` share `authorizeUnary`. `claimValidLocked` serves both claim checks in `InvokeHandle`.

Tests:
- The rejection table now asserts exact errors. It adds cases for an unknown instruction, an attempt beyond the instruction's `MaxAttempts`, a method the instruction does not name, and a poll on an RPC instruction.
- `TestSessionReadsPreparedInstructionBounds` pins the resolved defaults.
- `TestHandleClaimReplacedDuringContractCheckCannotInvoke` pins the re-check after the lock is re-taken.
- The new tests and the method and attempt cases were each confirmed red with their check removed.

The test fixtures now open sessions from a real `PreparedProgram` captured through `PreparedCase.Run`. The handle fixture gained a prepared workflow entrypoint with namespace and task-queue bindings.

Deviations:
- The construction check now covers all 14 `ProgramLimits` fields, not the earlier subset. Execution admission already requires every field, so such a Profile could never prepare.
- The Program ceiling literal is now duplicated in the server Driver, the worker Driver, and `execution.hardLimits`. Follow-up: export one ceiling (facade or `temporal/internal`) that is outside this task's Touches.
- An explicit opcode check in `authorizeUnary` was dropped as unpinnable. Deriving the declared role from the opcode's arm already rejects a mismatch.
- The review receipt's merged-review text holds fn-94.11's merged document. The shared scratchpad `merged.md` was overwritten by the concurrent worker before finalize, and the finalize cannot be replayed with different output. The verdict is mechanical from this task's own three SHIP draws (rid 94b6ebabf1dc4a6c9e8ac9c8a517894f), so it is unaffected.
- No gate receipt was written, because other workers' uncommitted files kept the worktree dirty.

baseline: green (go test -race temporal/... pre-edit)

stage: impl-review - ran [codex fan-out rid 26ff2437 refunded (HEAD moved by fn-94.11 commit)..codex fan-out rid 94b6ebab, 3 draws SHIP]
## Evidence
- Commits: 4380349074889cb6fec8bb348b2dcd98282a83b0
- Tests: go test -race -tags test_dep ./common/testing/testpilot/temporal/server/... ./common/testing/testpilot/temporal/... (with -overlay pinning concurrent workers' uncommitted files to HEAD), go test -tags test_dep ./common/testing/testpilot/... (with the same overlay), make lint-code GOLANGCI_LINT_FIX=false LINT_CODE_TARGETS=./common/testing/testpilot/temporal/server GOLANGCI_LINT_BASE_REV=951c5516e9 (lint-code-fast scoped to this task's package)
- PRs: