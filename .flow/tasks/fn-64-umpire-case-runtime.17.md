---
satisfies: [R2, R5, R7, R9]
---
# fn-64-umpire-case-runtime.17 Admit explicit activation carrier policy and routing topology

## Description
Compile declarative carrier authority and reservation-to-SDK-source topology for the reserved activation delivery contract.

**Size:** M
**Files:** `tools/umpire/internal/execution/{program,prepare,dataflow}.go`, new focused carrier admission code/tests, `tools/umpire/{profile,prepare,host}.go`, execution README
**Touches:** [tools/umpire/internal/execution/**, tools/umpire/profile.go, tools/umpire/prepare.go, tools/umpire/host.go, tools/umpire/prepare_test.go]

### Approach
- Extend frozen RolePolicy with bounded declarative reservation-carrier method/context/cardinality authority. Ordinary Methods authorization remains separate. Do not put Temporal method names in core.
- Validate policy subsets, duplicates and aggregate work before cloning; include carrier authority in Profile snapshot ownership and identity obligations.
- Compile explicit reservation topology once. Bind potential workflow StartNexusOperation source nodes to unique explicitly reserved matching handler entries and deterministic ordinals; include workflow ordinal for repeated activations. Reject missing/ambiguous/count-mismatched mappings.
- Preserve general bounded reservation counts for other Host carrier policies; the initial Temporal policy will admit one workflow per StartWorkflowExecution carrier. Expose immutable topology through PreparedProgram rather than rescanning source per Run.
- Migrate affected fake Profile fixtures to declare their actual carrier authority without weakening negative cases.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/execution/prepare.go` — bounded policy and activation admission
- `tools/umpire/internal/execution/program.go` — RolePolicy and prepared graphs
- `tools/umpire/internal/execution/dataflow.go` — SDK node context/dataflow checks
- `tools/umpire/internal/execution/scheduler.go:343` — explicit reservation ordering
- `tools/umpire/profile.go` — public Profile snapshot
- `.flow/tmp/fn64-task6-investigation.md` — missing routing source

### Key context
A transport carrier carries existing explicit authority; it must never create reservations by inferring effects from method names.

## Acceptance
- [ ] Preparation rejects unauthorized carrier methods, unsupported declared shapes, duplicate/oversized policy, missing or ambiguous handler mappings, crossed references, and count/overflow errors before Host I/O.
- [ ] Tests cover multiple controller workflow starts, repeated workflow/handler ordinals under a supporting fake Host policy, guarded SDK source nodes, deterministic mapping and isolated repeated preparation.
- [ ] Ordinary authorized unary methods without reservations remain unchanged; generic execution code has no Temporal RPC names or new binding-expression language.
- [ ] Focused tagged preparation/execution/root tests and race tests, formatting and scoped lint pass; document the immutable carrier topology API.

## Done summary
Added frozen declarative reservation-carrier authority to RolePolicy/Profile and compiled an immutable prepared topology with exact declared reservations plus deterministic workflow-source-to-Nexus-handler routes. Prepared and root accessors are O(1), return owned slices, preserve ordinary unreserved unary methods, and let later runtime delivery bind entrypoint ordinals without rescanning source instructions per Run.

Carrier policy validation now requires a unary method in the same endpoint's ordinary Methods set, unique supported workflow/Nexus contexts, positive per-context ceilings, and aggregate bounds before nested cloning. Reservation-bearing InvokeRPC instructions require matching authority; authored reservations remain exact. Preparation rejects missing, ambiguous, crossed, count-mismatched, cardinality, and admission-work overflow cases before Host I/O. Potential guarded StartNexusOperation sources reserve routes deterministically in workflow node order and workflow ordinal order. Generic execution/root code contains no Temporal RPC names and adds no binding-expression, delivery-ledger, header-codec, or SDK-registration behavior.

ProfileSpec.Snapshot and PrepareCase tests cover nested carrier ownership, source mutation isolation, and the existing caller-supplied Profile Identity obligation. Execution tests cover multiple controller starts, repeated workflow/handler ordinals, false guards, isolated repeated preparation, workflow-only reservations, cloned topology slices, strict policy subset/shape/cardinality validation, ordinary zero-reservation behavior, and deterministic missing/ambiguous/crossed/count rejection. The post-review fix replaced repeated handler scans with one ordered service/operation index, charged every reservation/topology record, handler insertion, potential source, emitted route, and final reconciliation, and added a one-unit-short admission-work test.

Baseline green: focused tagged execution/root tests, race tests, `make fmt-imports`, and authorized scoped no-fix lint all exited 0 before edits; logs and rc files use the `.flow/tmp/fn64-task17-baseline-*` prefix. Final post-fix formatting, normal, race, and scoped lint commands all exited 0; exact commands, environments, observations, and logs are in `.flow/tmp/fn64-task17-final-results.json`. No global lint-green claim is made because the inherited mainbase backlog remains outside this task.

stage: impl-review - ran (codex:gpt-5.6-sol:high; NEEDS_WORK then SHIP round 2; 2026-09-05T03:21:46.607421Z; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.17.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: concurrent-wave - skipped(policy: shared checkout; one writer)
Tracker sync: n/a (bridge inactive).

The sole P2 review finding identified uncharged quadratic topology matching. It was fixed with deterministic indexed matching and complete admission-work accounting; re-review reports no introduced/pre-existing findings and no unaddressed requirements. The non-trivial NEEDS_WORK-to-SHIP lesson updated the existing high-overlap memory entry `bug/integration/program-admission-must-validate-2026-09-04` rather than creating a duplicate.

The reviewed owned tree `3c64e49b4c8db63104da760070c117887db13171` matches all task17-owned source exactly against start tree `bac8d6c01c7eba4257029c0660268dfe25c4acc1`; the comparison exited 0 before lifecycle completion. Start HEAD and actual HEAD are both `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`. No commits, pushes, worktrees, destructive resets, or external messages were made: the user owns commits, evidence commits are empty, and all earlier staged changes were preserved.

Gate classification FULL. NO_RECEIPT: the task source is intentionally uncommitted in the shared user-owned staging area, so no HEAD-bound green receipt is warrantable. Delivery ledger/header codec/runtime SDK registration remain assigned to tasks 18/6.
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire, go test -count=1 -race -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire, make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, baseline: green (.flow/tmp/fn64-task17-baseline-{tests,race,format,lint}.{log,rc}), All post-fix final gate command exits, environments, timestamps and logs: .flow/tmp/fn64-task17-final-results.json, Owned source matches reviewed tree 3c64e49b4c8db63104da760070c117887db13171 against start tree bac8d6c01c7eba4257029c0660268dfe25c4acc1; comparison exit 0, GATE_CLASSIFICATION:full - executable Umpire source changed, NO_RECEIPT: user-owned shared staged source is uncommitted; no HEAD-bound receipt is warrantable
- PRs: