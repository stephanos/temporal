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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
