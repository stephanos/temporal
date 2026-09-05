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
Implemented immutable public PrepareCase, Profile/Catalog and Host/session/effect contracts, root-owned Driver translation and private preflight. Reuses execution.Prepare and verification.Prepare/New; retains Case provenance and immutable static policy only. No public Run placeholder, concrete Host import, scheduler or Monitor construction API.

Baseline green: tagged execution/verification tests, formatting and scoped no-fix lint before edits. Final3 tagged root/execution/verification tests, race, formatting and scoped no-fix lint all exited0 before review. Logs: .flow/tmp/fn64-task13-final3-{tests,race,format,lint}.log. Initial fixture mistakes and lint issues corrected; no review dispatched before green gates.

Tests cover production evaluator composition, nil/typed-nil Profile/Host/factory forms, failed factories and identity mismatches without sessions/effects, static admission failures, provenance/catalog/source/policy mutation isolation, independent parallel prepared objects, immutable adapter inputs and external-package Host compilation/import boundary.

stage: impl-review - ran (codex:gpt-5.6-sol:high; SHIP; 2026-09-04T23:18:05.141540Z; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.13.json)
stage: plan-sync - skipped(config: planSync.enabled != true)

Review first pass returned SHIP with no findings/unaddressed requirements. Reviewer test execution was blocked by read-only sandbox; worker final3 gate exit codes are authoritative. Owned source matches reviewed immutable tree b8209ebe00f3ce49c8c2d6b91c6cabb17180af81 against task-start tree d1b5829c7b5414d30b75c08badde436e2ca25156. Actual HEAD remains 0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf. No worker-authored commits, worktrees or dependency changes. Prior staged task3 receipts/docs were preserved.

Gate classification: FULL. NO_RECEIPT: worktree dirty outside the ignore set (tools/umpire/host.go) - receipt not warrantable. No global lint or future live/cutover/model gates claimed. Public Run/lifecycle tests remain assigned to task9 after task4 scheduler.
## Evidence
- Commits:
- Tests: baseline: green (tagged execution/verification tests, make fmt-imports, scoped no-fix lint), CGO_ENABLED=0 TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -tags test_dep ./tools/umpire ./tools/umpire/internal/execution/... ./tools/umpire/verification/... (exit0; .flow/tmp/fn64-task13-final3-tests.log), CGO_ENABLED=1 CC=/usr/bin/clang TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -race -tags test_dep ./tools/umpire ./tools/umpire/internal/execution/... ./tools/umpire/verification/... (exit0; .flow/tmp/fn64-task13-final3-race.log), make fmt-imports (exit0; .flow/tmp/fn64-task13-final3-format.log), make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false (exit0; .flow/tmp/fn64-task13-final3-lint.log), gate classify: FULL, NO_RECEIPT: worktree dirty outside the ignore set (tools/umpire/host.go) - receipt not warrantable
- PRs: