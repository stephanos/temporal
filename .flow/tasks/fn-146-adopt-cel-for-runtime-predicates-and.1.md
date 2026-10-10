---
satisfies: [R1]
---
# fn-146-adopt-cel-for-runtime-predicates-and.1 Define the breaking CEL format and identity boundary

## Description
Establish R1 before any predicate representation changes. Define current-format admission and a coordinated migration of Case files and recorded Run companions. Old formats are unsupported; verify rejection before current-schema interpretation or Driver I/O.

Execution-order amendment, explicitly authorized by the owner on 2026-10-10: implement this Testpilot lane in parallel with fn-156 while fn-157 is deferred. Start from the integrated local umpire source. Existing authoring predecessors remain final integration holds; do not fabricate their closure or guess their future APIs. Task .6 must re-anchor to those integrated APIs and fn-156's actual compiler checks. CEL adoption is already decided: execute the reviewed implementation tasks, not another feasibility prototype. Preserve the shared format contract: only fn-148.6 activates 4.0, and fn-148.7 owns complete companion regeneration and replay gates.

**Size:** M
**Files:** actual Case decoder, casefile canonicalization and offline assessment admission, `proto/internal/temporal/server/api/testpilot/v1/case.proto`, `common/testing/testpilot/internal/execution/prepare.go`, `common/testing/testpilot/protocol_test.go`, `common/testing/testpilot/recordedrun/**`, checked-in fixtures
**Touches:** [common/testing/testpilot/case.go, common/testing/testpilot/case_test.go, common/testing/testpilot/casefile/**, common/testing/testpilot/evaluation/admission.go, common/testing/testpilot/evaluation/*_test.go, proto/internal/temporal/server/api/testpilot/v1/case.proto, api/testpilot/v1/case.pb.go, common/testing/testpilot/internal/ir/cel_identity.go, common/testing/testpilot/internal/ir/cel_identity_test.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/protocol_test.go, common/testing/testpilot/recordedrun/**, common/testing/testpilot/**/testdata/**]

Ownership re-anchor: `casefile` remains JSON-only. Place CEL AST identity canonicalization in Testpilot's existing `internal/ir` runtime-schema boundary, without introducing an evaluator or an Umpire dependency. Regenerate the exact generated Case Go file when proto comments change. Reuse the existing `cel.dev/expr` dependency pin; dependency upgrades remain outside scope. Source metadata remains diagnostic for evaluation while its exact stored bytes participate in artifact identity. Canonical map ordering must not silently reorder observable evaluation or errors: bound the admitted literal domain and test/refuse unsafe constructions; do not indiscriminately reorder message constructor entries.

### Approach
- Include the actual Case decoder in `common/testing/testpilot/case.go`, JSON canonicalization in `casefile`, and independent offline assessment admission in `evaluation/admission.go`. Define current-format rejection before interpreting retired payloads.
- Unit-test the shared format/identity boundary first. Only fn-148.6 activates format 4.0 emission/admission after all three specs' consumers integrate; fn-148.7 migrates checked-in companions. Do not require pre-CEL consumers to replay CEL payloads.
- Define successful offline assessment of a current 4.0 Case/Run pair and retired-format pre-payload rejection. Run focused proofs when their consumers compile; prove public 4.0 acceptance at fn-148.6 and the complete managed companion/replay surface at fn-148.7.
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

Pin one format 4.0 contract for CEL, Duration and evidence/state normalization. Formats 1.0, 2.0 and 3.0 are rejected, not emitted as migration stages. Migrate `casefile` parsing, Run decoding and current canonical hashing together. Define deterministic AST IDs, diagnostic metadata treatment and canonical map ordering. Do not retain legacy decoding, evaluation or identity conversion.
## Acceptance
- [ ] R1's breaking format, canonical identity and companion migration contract is executable.
- [ ] All checked-in companions are accounted for; retired formats reject explicitly.
- [ ] Unknown format and mismatched companion negative tests pass.
- [ ] Focused format/identity unit proofs pass; full protocol/recordedrun and assessment gates run at the shared fn-148.7 close after fn-148.6 activates 4.0.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
