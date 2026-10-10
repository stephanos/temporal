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
# CEL format and identity boundary

Task fn-146-adopt-cel-for-runtime-predicates-and.1 establishes the coordinated successor-format contract, pre-payload admission, deterministic bounded CEL identity rules, and the executable migration inventory of 65 Cases and seven recorded Run companions. Casefile remains JSON-only; runtime-schema identity belongs to Testpilot internal/ir. No CEL evaluator, descriptor adapter or public format 4.0 activation is claimed by this task.

Integrated source commits: 6934d3d87dc50d357bc55a15be27356065912fc8 and 84b1f26b30. The correction preserves ProtoJSON null-as-unset semantics: null version rejects 0.0 with FormatError and offline ReasonIncompatible before decoding retired payloads, while malformed whole documents remain malformed. Actual worker RED/green receipts cover numeric syntax, null, whitespace and malformed controls; full affected-package JSON records 474 passing test events and no failures. Existing compiler changes do not alter these Go inputs.

The original independent correctness/contracts/integration fan-out found one introduced P2 null-version issue. Single resumed Codex review returned SHIP and marked that finding fixed, with no new blocking findings. Receipt: /tmp/impl-review-receipt-401634b59261-fn-146-adopt-cel-for-runtime-predicates-and.1.json; session 01a12679-feb4-7f50-8279-f6fa10b4f529. Review snapshot 2c41ae37aea03e0d1a15e1da6133064bf48b55b6 has byte-identical Testpilot/proto/API/go.mod/go.sum inputs to the integrated target, verified by git diff --quiet. Review metadata is integrated at eb6bbf014c76341a058c95ceb257e946e0673f4b. Writer and reviewer are the same model family; the review ran in the independent existing reviewer session, not by the conductor.

Root reran the exact focused Quick on the integrated target under the actual /tmp/umpire-heavy-gates.lock: exit 0, twenty package passes and two no-test packages; seven passing packages have no selected tests. Log: .flow/tmp/fn146/boundary-integrated/quick.log. This is focused task evidence, not a full tooling or successor-format replay gate. Original correction evidence remains at the fn146 worker's .flow/tmp/fn146/task1/correction-{summary.md,evidence.json,green.jsonl,quick.log,lint.log}; original baseline/manifests remain unchanged.

Only fn-148.6 activates emission/admission of 4.0 after consumer integration; fn-148.7 regenerates all managed Cases/companions and supplies full protocol, assessment, replay, canonical gates and live evidence. Formats are not emitted as intermediate migrations. Authoring predecessors remain final integration reconciliation obligations. Those obligations and fn146 R2–R7 remain open in the actively implementing format batch.

Tier: session (jev-unavailable(no_key)); explicit project implementer/reviewer pins retained.
stage: impl-review - ran (single resumed review, SHIP; original three-axis fan-out retained)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 6934d3d87dc50d357bc55a15be27356065912fc8, 84b1f26b30, eb6bbf014c76341a058c95ceb257e946e0673f4b
- Tests: flock /tmp/umpire-heavy-gates.lock mise exec -- go test -tags test_dep -run 'Test.*(Format|Identity|Companion)' ./common/testing/testpilot/... (integrated target, exit 0; .flow/tmp/fn146/boundary-integrated/quick.log), git diff --quiet 2c41ae37aea03e0d1a15e1da6133064bf48b55b6 HEAD -- common/testing/testpilot api/testpilot proto/internal/temporal/server/api/testpilot go.mod go.sum (exit 0), worker full affected-package JSON: facade339/evaluation135 pass test events, zero fail events; actual null RED retained, Codex resumed implementation review SHIP, receipt /tmp/impl-review-receipt-401634b59261-fn-146-adopt-cel-for-runtime-predicates-and.1.json; finding fixed, git diff --check
- PRs: