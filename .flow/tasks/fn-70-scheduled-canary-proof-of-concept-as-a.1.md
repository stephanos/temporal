---
satisfies: [R1, R3, R4]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.1 Expose no-I/O prepared Driver validation and reuse it in Run

## Description
Expose no-I/O prepared Driver validation and reuse it in Run.

**Size:** M
**Files:** common/testing/testpilot/prepare.go; common/testing/testpilot/prepared_case.go; common/testing/testpilot/*external_test.go; common/testing/testpilot/prepare_test.go; common/testing/testpilot/README.md
**Touches:** [common/testing/testpilot/prepare.go, common/testing/testpilot/prepared_case.go, common/testing/testpilot/*external_test.go, common/testing/testpilot/prepare_test.go, common/testing/testpilot/README.md]

### Approach
- Add PreparedCase.ValidateDriver(ctx, Driver) error using the existing private identity and Driver.Validate sequence; separate this from Monitor creation and reuse it in Run preflight. Do not expose private PreparedProgram construction or duplicate temporal validation.
- Preserve existing errors and cancellation checks, immutable binding identity and Prepare behavior. Test nil/typed-nil inputs, canceled context, identity/binding mismatch, Driver validation failure and successful repeat validation with counters proving no Monitor, Open, Run or target effect.
- Add external-package TestPreparedCaseValidateDriverPublicAdmission and existing Run preflight compatibility coverage; document the no-I/O public boundary.

### Investigation targets
**Required:**
- common/testing/testpilot/prepare.go:25,53 — admission and existing preflight.
- common/testing/testpilot/driver.go:76,245 — private prepared view and Driver hook.
- common/testing/testpilot/prepared_case.go:11 — fresh Run dispatch.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot`
`mise exec -- go test -race -count=1 -tags test_dep ./common/testing/testpilot`

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] Public validation reuses identity and static Driver checks without Monitor/Open/effects or a Run.
- [ ] External negative/success tests execute and unchanged Run rejects at the same authority boundary.
- [ ] Cancellation, typed nils, source snapshots and full binding fingerprint remain enforced.
- [ ] Focused unit/race suites pass; public documentation accurately distinguishes Prepare, ValidateDriver and Run.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
