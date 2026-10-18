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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: