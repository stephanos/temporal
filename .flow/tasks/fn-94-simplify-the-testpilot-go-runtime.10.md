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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
