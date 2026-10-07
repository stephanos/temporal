---
satisfies: [R1]
---
# fn-146-adopt-cel-for-runtime-predicates-and.1 Define the breaking CEL format and identity boundary

## Description
Establish R1 before any predicate representation changes. Define current-format admission and a coordinated migration of Case files and recorded Run companions. Old formats are unsupported; verify rejection before current-schema interpretation or Driver I/O.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/case.proto`, `common/testing/testpilot/internal/execution/prepare.go`, `common/testing/testpilot/protocol_test.go`, `common/testing/testpilot/recordedrun/**`, checked-in fixtures
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/case.proto, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/protocol_test.go, common/testing/testpilot/recordedrun/**, common/testing/testpilot/**/testdata/**]

### Approach
- Inventory checked-in Cases and recorded Runs that must regenerate together.
- Require the current format and reject retired or unknown formats without translation.
- Pin identity mismatch, missing companion and pre-Driver rejection behavior.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/case.proto:8-25` - format and producer contract
- `common/testing/testpilot/internal/execution/prepare.go:60-85` - exact 1.0 admission
- `common/testing/testpilot/protocol_test.go:80-175` - protocol surface checks
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - Case and Run identity pairing
- `.plans/UMPIRE_CEL_RUNTIME_RESEARCH.md:134-150` - accepted migration decisions


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/...
```

### Format and identity boundary

Pin format 2.0 for CEL, 3.0 for Duration and 4.0 for evidence/state normalization. Migrate `casefile` parsing, Run decoding and current canonical hashing together. Define deterministic AST IDs, diagnostic metadata treatment and canonical map ordering. Do not retain legacy decoding, evaluation or identity conversion.

## Acceptance
- [ ] R1's breaking format, canonical identity and companion migration contract is executable.
- [ ] All checked-in companions are accounted for; retired formats reject explicitly.
- [ ] Unknown format and mismatched companion negative tests pass.
- [ ] Focused protocol and recordedrun suites pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
