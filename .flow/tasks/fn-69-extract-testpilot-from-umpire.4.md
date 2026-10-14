---
satisfies: [R1, R2, R3, R5, R6]
---
# fn-69-extract-testpilot-from-umpire.4 Publish the Testpilot facade and canonical Case ingestion

## Description
Build the small public Testpilot boundary over the extracted core, including canonical Case ingestion and a real external Driver proof (R1-R3, R5, R6). Existing Umpire callers remain on their old facade until later tasks.

**Size:** M
**Files:** `common/testing/testpilot/{prepare,prepared_case,profile,driver,case}.go`, matching tests and READMEs, migration ledger
**Touches:** [common/testing/testpilot/*.go, common/testing/testpilot/*.md, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Adapt the existing facade with `Prepare`, `PreparedCase.Run`, Profile/Catalog, Driver/Session, prepared-plan views, and DriverIdentity; keep scheduler, recorder, IR, and evaluator private.
- Fold strict `DecodeCaseProtoJSON` and deterministic `PackCaseProtoJSON` behavior into the top-level package from the existing caseartifact implementation.
- Move facade/conformance tests that own generic behavior; add an external-package non-functional Driver that prepares and executes one bounded Case through public APIs, including failure and cleanup paths.
- Preserve defensive protobuf ownership and complete nil/typed-nil checks at every interface boundary.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/prepare.go:15-45` — facade behavior to preserve
- `tools/umpire/host.go:14-68` — current public environment seam
- `tools/umpire/caseartifact/case.go:1-29` — ingestion behavior
- `tools/umpire/host_external_test.go:17-74` — external implementation and dependency proof
- `tools/umpire/conformance_test.go` — six-class public corpus
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md` — typed-nil boundary coverage

### Acceptance

## Acceptance
- [ ] Public API is exactly the planned Prepare/PreparedCase/Driver sequence plus Profile/Catalog, required plan views, and canonical Case decode/pack functions; private execution/evaluation types remain inaccessible.
- [ ] Preparation performs no Driver I/O, owns mutable inputs defensively, and rejects malformed/over-limit Case/Profile and every nil-capable interface case with frozen precedence.
- [ ] External-package proof executes a bounded Case through a non-functional Driver using only public Testpilot imports and verifies success, Driver failure, cleanup, and forbidden dependency direction.
- [ ] Generic facade/conformance tests and focused package tests pass with frozen Run/Verdict/error behavior and no Umpire, functional harness, SDK, or canary dependency.

## Done summary
Published the public Testpilot facade over the refined schema with immutable preparation, public Driver and prepared-plan views that hide private IR/evaluator types, strict canonical Case ingestion, and an external non-functional Driver proof. Focused Testpilot tests and vet pass; the user-owned checkout remains unstaged and uncommitted.

baseline: green (`go test -count=1 -tags test_dep ./common/testing/testpilot/...`)

review: SHIP; receipt `/tmp/impl-review-receipt-fn-69-extract-testpilot-from-umpire.4.json`

stage: impl-review - ran | SHIP

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/..., go vet -tags test_dep ./common/testing/testpilot/..., git diff --check -- common/testing/testpilot .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md, go doc -all go.temporal.io/server/common/testing/testpilot | rg -n 'internal/(ir|execution|verification)|tools/umpire|tests/testcore|temporal/sdk' (no matches)
- PRs:
