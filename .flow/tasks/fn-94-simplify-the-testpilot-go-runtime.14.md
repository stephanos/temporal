---
satisfies: [R7]
---
# fn-94-simplify-the-testpilot-go-runtime.14 Shared test support in common/testing/testpilot: closures, limits, runtime fixture, fakes

## Description
Lane F for `common/testing/testpilot`: one facade-free internal test support package for the descriptor-closure helper, a `ProgramLimits` constructor, the shared runtime fixture and the shared fakes; and the two-layer duplicate tests. Runs after the code lanes so it consolidates the final test shapes.

**Size:** M
**Files:** new `common/testing/testpilot/internal/testsupport/` (non-`_test.go`, imports only protobuf and generated code plus `contract` if needed), the 12 test files with descriptor-closure bodies, the 15 with `ProgramLimits` literals, `temporal/internal/activation/activation_test.go`, `temporal/worker/runtime_fixture_test.go`, the fake Session/Driver test files
**Touches:** [common/testing/testpilot/internal/testsupport/**, common/testing/testpilot/**/*_test.go]

### Approach
- Descriptor closure: one exported helper replaces the 12 test copies (named `descriptorClosure` in `activation_test.go:171`, `runtime_fixture_test.go:181`; `facadeDescriptorClosure` `conformance_test.go:261`; `nexusDescriptorClosure` `verification/nexus_correlation_test.go:251`; inline in `server/{poll,driver}_test.go`, `preparation_error_test.go`, `ir/catalog_test.go`, `verification/correlated_test.go`, `execution/evidence{,_lift}_test.go`). Check with `go list -deps -test` that no destination package gains an adapter dependency (memory: moved conformance tests must not import functional adapters).
- One `ProgramLimits` constructor replaces the 15 literals.
- One runtime fixture parameterized by reply kind, command types and capture step replaces the near-copies in `activation_test.go` and `runtime_fixture_test.go`.
- One scripted fake Session, one fake Driver, fake effect and reservation types. In-package facade tests (`prepare_test.go`) keep one local copy; `countingMonitorFactory`, the scheduler host and the canary's fenced session stay local. Each retained local fake gets a one-line comment saying what it tests that the shared one cannot.
- Two-layer duplicates: where facade and `execution` test the same environment rejection or concurrent preparation, keep the facade test; keep focused tests EVD-18 keeps independent of the corpus.
- Run with `-race -count=3`: consolidated fakes may share state.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/internal/activation/activation_test.go`
- `common/testing/testpilot/temporal/worker/runtime_fixture_test.go`
- `common/testing/testpilot/conformance_test.go:250-280`
- `common/testing/testpilot/prepare_test.go` — in-package fakes

### Quick commands
```sh
go test -race -count=3 -tags test_dep ./common/testing/testpilot/...
go list -deps -test ./common/testing/testpilot/internal/ir/ ./common/testing/testpilot/internal/execution/ | grep -c testpilot/temporal
make lint-code-fast
```

## Acceptance
- [ ] Each shared helper and fake has one definition; retained local fakes carry their one-line reason.
- [ ] No `ir`/`execution` test gains a facade or adapter dependency.
- [ ] `-race -count=3` tests and lint pass; the test line count drops.


## Done summary
The Testpilot tests now share one support package. `internal/testsupport` holds `DescriptorClosure`, `ProgramLimits()`, and the scripted `Session`, `Effect` and `Reservation`. It imports only `contract`, protobuf and generated code, so `ir`, `execution` and `verification` tests can use it and gain no facade or adapter dependency (`go list -deps -test ... | grep -c testpilot/temporal` = 0). `internal/testsupport/facadetest` holds the fake `Driver`, `Capture`, and the one runtime fixture `RuntimeCase`, which takes the reply kind and command types as parameters. The activation and worker tests both use it.

Every former copy's callers now use the shared definition:
- The 12 descriptor closures.
- The full-field `ProgramLimits` literals. Callers override only the field their test needs: poll and evidence set the fanout to the emission bound, and the runtime fixture keeps its 64 KiB byte ceilings.
- The four program-capture Drivers and the conformance, proof, correlated and fault-run Drivers.
- The facade, proof, correlated, recording controller and recording worker Sessions, and the scheduler host.
- The facade, recording, terminal, blocking, scheduler and runtime effects.
- The recording, scheduler and delivery reservations.

Some literals and fakes stay local, each with a one-line reason:
- Literals: the conformance and correlated `ProgramLimits` literals (existing comments explain them), the identity golden's pinned literal, and the partial callback and recorder limits.
- Fakes: the in-package `facadeDriver` and typed-nil types in `prepare_test.go`, `countingMonitorFactory`, execution's `runtimeDriver`/`runtimeSession` and `recorderHandle`, and the bridges.

Test files shrink by about 970 lines; with the 610 support lines added, the net drop is about 360. Goldens and conformance fixtures are unchanged.

The review found one problem. The first commit had deleted `TestConcurrentEnvironmentPreparationsResolveIndependently` as a two-layer duplicate, but the facade test does not check resolved requests. The test is restored in the second commit.

baseline: green (go test -race -count=3 -tags test_dep ./common/testing/testpilot/..., pre-edit)
Gate receipt: attempted. Other sessions' uncommitted model/** edits keep the worktree dirty.

stage: impl-review - ran [codex fan-out rid 77c71b5484bb458a8f63702c114ece1f: correctness SHIP, contracts NEEDS_WORK, integration SHIP -> restore test -> re-review SHIP]
## Evidence
- Commits: 5abdf52d8bc3f1b1d57f383aa6ec7b72d2270c57, ee61d21bb756240dd8613c5ef65baa8b0e32ef88
- Tests: go test -race -count=3 -tags test_dep ./common/testing/testpilot/..., go list -tags test_dep -deps -test ./common/testing/testpilot/internal/ir/ ./common/testing/testpilot/internal/execution/ | grep -c testpilot/temporal (0), make lint-code-fast, make umpire-check-retired-vocabulary
- PRs: