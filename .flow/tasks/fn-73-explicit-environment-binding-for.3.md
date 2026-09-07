---
satisfies: [R3, R4, R7]
---
# fn-73-explicit-environment-binding-for.3 Add immutable prepared-role access and Driver preflight validation

## Description
Expose the minimal copied prepared-role facade and extend the public Driver contract with binding identity and static Validate (R3, R4). Migrate every implementation and test double while pinning the pre-Open order.

**Size:** M
**Files:** `common/testing/testpilot/{driver,prepare,prepared_case}.go`, all generic Driver test doubles, and the composite/server/worker Driver implementations and tests under `common/testing/testpilot/temporal`
**Touches:** [common/testing/testpilot/driver.go, common/testing/testpilot/prepare.go, common/testing/testpilot/prepared_case.go, common/testing/testpilot/*_test.go, common/testing/testpilot/temporal/driver.go, common/testing/testpilot/temporal/driver_test.go, common/testing/testpilot/temporal/server/driver.go, common/testing/testpilot/temporal/server/driver_test.go, common/testing/testpilot/temporal/worker/driver.go, common/testing/testpilot/temporal/worker/*_test.go, tests/testcore/testpilot/artifact_test.go]

### Approach
- Add copied PreparedProgram role access containing logical IDs, kinds, symbolic binding IDs and resolved values while keeping resolved request assignments private.
- Add `DriverIdentity.Bindings` and require `Driver.Validate(context.Context, PreparedProgram) error`.
- Order Run preflight as context, Identity, exact comparison, Validate, monitor creation, then Open.
- Migrate all in-repository Drivers and test doubles explicitly; drivers without additional static rules return success.
- Preserve existing internal error behavior so fn-74 can later own public PreparationError classification.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/driver.go:68-147,199-204` — public prepared facade and Driver interface
- `common/testing/testpilot/prepare.go:23-64` — identity creation and preflight
- `common/testing/testpilot/prepare_test.go:30-121` — typed-nil, mutation and identity tests
- `common/testing/testpilot/temporal/driver.go:74-109,378-379` — composite implementation and interface assertion
- `common/testing/testpilot/temporal/server/driver.go` — server Driver implementation
- `common/testing/testpilot/temporal/worker/driver.go` — worker Driver implementation

**Optional** (reference as needed):
- `tests/testcore/testpilot/artifact_test.go:387-405` — fixture Driver

### Key context
Identity mismatch must skip Validate. Validate failure returns no Run or Verdict and must occur before any monitor/session/worker/dispatch side effect.
## Acceptance
- [ ] PreparedProgram returns copied role records with logical and resolved binding data and no mutable/private assignment state.
- [ ] Driver identity comparison includes the binding fingerprint and rejects a changed binding under the same Profile identity before Validate/Open.
- [ ] Every Driver implementation and test double implements Validate explicitly.
- [ ] Ordering tests prove identity mismatch skips Validate and Open, while Validate failure skips monitor/session creation, Open, worker registration and dispatch and returns no Run/Verdict.
- [ ] Existing typed-nil and external-facade behavior remains covered.
- [ ] Focused `go test -count=1 -tags test_dep` suites for affected packages pass.
## Done summary
Added copied prepared-role access, binding fingerprints to Driver identity, mandatory static Validate, and preflight ordering from context through identity and validation before monitor creation and Open. Migrated all in-scope Drivers and test doubles, added ordering/immutability/failure regressions, and made zero-value PreparedProgram snapshots nil-safe. Independent implementation review returned SHIP after fixing empty-facade panics in server and worker Validate. Plan sync was skipped because `planSync.enabled` is false.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./common/testing/testpilot/temporal/... ./tests/testcore/testpilot/..., gofmt -d on task-changed files, git diff --check, implementation review SHIP: /tmp/impl-review-receipt-fn-73-explicit-environment-binding-for.3.json
- PRs:
