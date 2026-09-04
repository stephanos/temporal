---
satisfies: [R1, R2, R6, R7, R9]
---
# fn-64-umpire-case-runtime.2 Build typed IR admission and immutable preparation

## Description
Implement shared typed IR binding and exhaustive pre-I/O Case admission (R1, R2), including the
Profile, public Host, internal MonitorFactory, effect-handle, and opaque capability contracts used
by later tasks. Establish the root preparation facade while keeping execution machinery private.
This is the early proof point because it validates arbitrary protobuf methods and payload paths
without executing them.

**Size:** M
**Files:** `tools/umpire/internal/{ir,execution}/**`, public Profile/Host contracts, top-level static
preparation skeleton, and focused tests
**Touches:** [tools/umpire/internal/ir/**, tools/umpire/internal/execution/**, tools/umpire/*.go]

## Approach
- Reuse the existing clone-and-validate shape, replacing PortableTestPlan vocabulary rather than
  layering over it.
- Compile descriptor/path accessors, entrypoint DAG indexes, Slot/result dataflow, capability
  checks, bounds, and stable Profile/catalog identities into immutable prepared data.
- Restrict dependencies to an entrypoint; validate explicit Host-driven activation bindings for
  workflow and Nexus-handler graphs.
- Admit scalar/message/WKT fields through descriptors, defined enum values, optional/oneof
  presence, repeated fan-out, literal map keys, and whole typed `Any` Slot copies; reject unpacked
  `Any` traversal and unsupported constructs.
- Keep `PrepareCase(case, profile)` static and prepare the Contract's immutable default evaluator.
  Define root `Run(ctx, host)` preflight so it validates the live Host and creates a fresh internal
  Monitor before Run creation; Monitor construction failure is an internal invariant with no I/O.
- Define public Profile/Host/effect-handle types in the root package and a root-owned translation to
  the private execution driver. Internal execution imports neither the root nor Temporal, and the
  root imports no concrete Host implementation.
- Make completion authority an opaque Host capability type that ordinary expressions and
  projections cannot inspect; define non-blocking Host-owned effect handles for wait/cancel/drain.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/testplan/validate.go:63-100` — exhaustive pre-I/O validation pattern
- `tools/umpire/testplan/plan.go:18-79` — immutable clone/checksum precedent
- `tools/umpire/executor/portable_projection.go` — legacy schema projection and identity assumptions
- `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto` — current bounds and typed
  value inputs
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md` — complete
  typed-nil reflection cases

**Optional** (reference as needed):
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md` —
  pre-dispatch/invariant boundary regressions

## Key context
Preparation may compute stable catalog/Profile identity but must not add response/payload digests.
Profile credentials and availability remain runtime concerns behind the prepared authority identity.

## Acceptance
- [ ] Table-driven admission tests cover every schema/version/ID/DAG/bound/dataflow/context/capability
  failure and prove zero Host I/O on rejection.
- [ ] Descriptor tests bind multiple unrelated unary methods and cover nested fields,
  optional/oneof presence, repeated `[*]`, literal map keys, WKT fields, enum rejection, `Any`,
  cardinality, and type mismatch.
- [ ] Instruction-outcome references permit explicit guards over success, protocol non-success, SDK
  failure, and bounded timeout without exposing undeclared payloads.
- [ ] Prepared data is immutable and reusable; Profile typed nil fails preparation, while Host typed
  nil or identity mismatch and internal MonitorFactory typed nil/failure fail before Run creation or
  I/O.
- [ ] The public surface exposes `PrepareCase` and `PreparedCase.Run(ctx, host)` without scheduler,
  recorder, Slot, Executor, or Monitor-factory construction APIs.
- [ ] Dependency tests prove the root-owned Host-to-driver adapter introduces no root/internal/
  Temporal import cycle.
- [ ] Effect handles expose bounded wait/cancel without requiring Executor-owned goroutine wrappers;
  opaque capability Slots permit readiness/consumption but reject inspection and projection.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/ir/...
  ./tools/umpire/internal/execution/... ./tools/umpire/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
