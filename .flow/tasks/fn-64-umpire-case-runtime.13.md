---
satisfies: [R2, R6, R7, R9]
---
# fn-64-umpire-case-runtime.13 Compose immutable public preparation and Host adapters

## Description
Compose completed Program admission and the production Contract evaluator behind public
`PrepareCase` (R2, R6, R7, R9). Define public Profile/Host/effect contracts and private Run preflight;
task 9 supplies the public `PreparedCase.Run` implementation after the scheduler exists.

**Size:** M
**Files:** `tools/umpire/{prepare,profile,host}.go` and root facade/dependency tests
**Touches:** [tools/umpire/*.go]

### Approach
- Translate public Profile values to the immutable private policy/catalog input from task 11;
  prepare Program and Contract once, and retain no live client, credentials, Run IDs or mutable state.
- Define public Host/session/effect-handle and opaque capability types with root-owned translation
  into the private Driver contract. Alternate Hosts implement this public interface; the root does
  not import concrete Temporal adapters and execution does not import the root.
- Compose only the production evaluator prepared by task 3. Internal test fakes exercise factory
  failures without exposing arbitrary Monitor construction or replacement to callers.
- Implement private preflight used later by Run: validate every nil-capable Host/factory form,
  exact non-secret Profile/catalog identities, and fresh factory construction before creating a Run
  or Host session. Keep credential availability/rotation behind the same authorized identity.
- Do not export a placeholder Run that returns synthetic success or an unavailable error for valid
  inputs. Add the public Run method in task 9 together with working execution and lifecycle tests.

### Investigation targets
**Required**:
- `tools/umpire/temporal/local/attached.go:62` — immutable identity/live drift checks
- `tools/umpire/temporal/local/attached.go:132` — all nil-capable reflection kinds
- `tools/umpire/testplan/plan.go:49` — immutable admitted ownership pattern
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md`
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md:290` — preparation and reuse contract

## Acceptance
- [ ] Public `PrepareCase(case, profile)` composes complete typed admission and the actual prepared
  Contract evaluator with no Host/target I/O; source Case/Profile mutation cannot alter prepared data.
- [ ] Nil/typed-nil Profile values reject; private Run preflight rejects nil/typed-nil or mismatched
  Hosts and factory failures before Run/session creation. Cover pointer, map, slice, function and
  channel implementations and zero effects on every rejection.
- [ ] The public Host and effect contracts support alternate adapters, opaque capability readiness/
  consumption and bounded lifecycle operations; no public scheduler/recorder/Slot/Monitor factory
  construction or Monitor replacement API exists.
- [ ] Dependency tests prove root-owned translation has no root/internal/Temporal cycle and root
  imports no concrete Host. Independent prepared objects remain isolated under race tests; full
  sequential/concurrent Run reuse is explicitly tested by task 9.
- [ ] Tagged root/internal/execution/verification tests and applicable race/format/lint gates pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
