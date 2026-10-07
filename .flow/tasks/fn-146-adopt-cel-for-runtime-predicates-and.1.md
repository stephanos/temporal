---
satisfies: [R1]
---
# fn-146-adopt-cel-for-runtime-predicates-and.1 Define the breaking CEL format and identity boundary

## Description
Establish R1 before any predicate representation changes. Define current-format admission and a coordinated migration of Case files and recorded Run companions. Old formats are unsupported; verify rejection before current-schema interpretation or Driver I/O.

**Size:** M
**Files:** actual Case decoder, casefile canonicalization and offline assessment admission, `proto/internal/temporal/server/api/testpilot/v1/case.proto`, `common/testing/testpilot/internal/execution/prepare.go`, `common/testing/testpilot/protocol_test.go`, `common/testing/testpilot/recordedrun/**`, checked-in fixtures
**Touches:** [common/testing/testpilot/case.go, common/testing/testpilot/casefile/**, common/testing/testpilot/evaluation/admission.go, common/testing/testpilot/evaluation/*_test.go, proto/internal/temporal/server/api/testpilot/v1/case.proto, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/protocol_test.go, common/testing/testpilot/recordedrun/**, common/testing/testpilot/**/testdata/**]

### Approach
- Include the actual Case decoder in `common/testing/testpilot/case.go`, JSON canonicalization in `casefile`, and independent offline assessment admission in `evaluation/admission.go`. Define current-format rejection before interpreting retired payloads.
- Unit-test the format/identity boundary first. Activate format 2.0 emission/admission and migrate checked-in companions only at the Tasks 2–6 integration point; do not require pre-CEL consumers to replay CEL payloads.
- Add successful offline assessment of a current 2.0 Case/Run pair and retired-format pre-payload rejection; run the end-to-end assessment after the verification and lowering tasks compile.
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
go test -tags test_dep -run 'Test.*(Format|Identity|Companion)' ./common/testing/testpilot/...
```

### Format and identity boundary

Pin format 2.0 for CEL, 3.0 for Duration and 4.0 for evidence/state normalization. Migrate `casefile` parsing, Run decoding and current canonical hashing together. Define deterministic AST IDs, diagnostic metadata treatment and canonical map ordering. Do not retain legacy decoding, evaluation or identity conversion.
## Acceptance
- [ ] R1's breaking format, canonical identity and companion migration contract is executable.
- [ ] All checked-in companions are accounted for; retired formats reject explicitly.
- [ ] Unknown format and mismatched companion negative tests pass.
- [ ] Focused format/identity unit proofs pass; full protocol/recordedrun and assessment gates run after consumers integrate.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
