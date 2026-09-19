---
satisfies: [R4, R9]
---
# fn-64-umpire-case-runtime.14 Implement typed request and projection data plane

## Description
Implement the typed execution data plane for R4/R9, independently of Host I/O and Run recording.
Task 4 integrates these admitted-value operations with the scheduler; task 15 owns publication.

**Size:** M
**Files:** `tools/umpire/internal/ir/write.go`, focused IR path extensions/tests;
`tools/umpire/internal/execution/{values,request,projection}.go` and focused tests;
`tools/umpire/internal/execution/{dataflow,program}.go` shared overlap caller and cached runtime ceiling and `README.md` ownership docs
**Touches:** [tools/umpire/internal/ir/**, tools/umpire/internal/execution/values*, tools/umpire/internal/execution/request*, tools/umpire/internal/execution/projection*, tools/umpire/internal/execution/dataflow.go, tools/umpire/internal/execution/program.go, tools/umpire/internal/execution/README.md]

### Approach
- Reuse compiled assignment/projection paths and the `Expression.Evaluate` core; its execution entry
  charges intermediate copies without changing existing Contract accounting. Add a narrow typed request
  builder/raw projection seam in IR rather than duplicating descriptor traversal or rebinding.
- Build each request from independent bounded state. Preserve optional/oneof/map absence versus
  explicit zero, list order, and the existing no-filter wildcard/presence rules. Opaque capabilities
  are never accepted by expression, setter, or response-projection helpers.
- Own immutable per-attempt outcome snapshots and ordinary controller Slot values. Validate raw
  Host response/outcome type and bounds before staging projected writes and Observation batches.
  Missing required input, crossed types, duplicate writes and size/work/fanout overflow return
  typed failures to the caller without publishing partial Slots or events.
- Stage all projected values and stable EmitEach element indexes before publication. Task 4 commits
  the staged batch under task 15's admission/publication barrier; the data plane does not assign
  sequence/elapsed coordinates or invoke a Monitor. Bind store identity to its Run/activation owner.
- Keep static binding costs separate from executed value work. Reuse the existing declared byte,
  per-instruction emitted-event and Program fanout limits; recorder-wide event capacity belongs to
  task 15. Provide a sealing seam so task 4/task 9 can atomically prevent writes at closure.

### Investigation targets
**Required**:
- `tools/umpire/internal/execution/dataflow.go:217` — admitted projection sinks and writers
- `tools/umpire/internal/execution/dataflow.go:389` — compiled guarded request assignments
- `tools/umpire/internal/execution/program.go:72` — private compiled node and typed Slot shapes
- `tools/umpire/internal/ir/evaluate.go:15` — bounded typed evaluation
- `tools/umpire/internal/ir/evaluate_path.go:22` — existing path/protobuf semantics
- `.flow/memory/bug/integration/contract-work-bounds-must-follow-typed-2026-09-04.md`

### Early proof
Construct a typed request and project an unrelated typed response using admitted paths with no Host
or recorder; prove exact field presence, value ownership and deterministic projected element order.

## Acceptance
- [ ] Typed request tests cover nested fields, oneof/optional/map absence versus zero, numeric widths,
  malformed/crossed values and exact request-byte/work bounds through compiled paths without rebinding.
- [ ] Outcome/Slot tests prove independent snapshots, Run/activation ownership, single assignment,
  seal rejection, guarded missing values and all-or-nothing staged writes; no opaque/unprojected payload leaks.
- [ ] Projection tests preserve protobuf list order and EmitEach indexes, aligned presence booleans,
  empty/mixed/all-absent wildcard behavior, and reject wrong response types, excess bytes/fanout/events
  before publication. Exact accepted budgets bound execution work, including copies.
- [ ] The private data plane has no Host I/O, recorder/Monitor, root facade, verification or concrete
  Temporal imports. Package documentation names staged publication/sealing ownership for tasks 4/9.
- [ ] Tagged IR/execution tests and affected-package race tests, `make fmt-imports`, and authorized
  scoped no-fix `make lint-code` pass before configured implementation review.


## Done summary
Implemented the bounded typed request/outcome/Slot/projection data plane with immutable attempt snapshots, atomic staged writes, Run/activation ownership and sealing. Shared IR paths preserve protobuf presence, numeric types, ordered wildcard projections and Any handling without runtime rebinding; Host I/O, scheduling and recorder publication remain integration responsibilities of tasks 4/15/9.

Validation: all affected IR/execution/verification normal and race tests, make fmt-imports, and authorized scoped no-fix lint exited 0; exact commands, environments, timestamps and logs are recorded in `.flow/tmp/fn64-task14-fix3-results.json`. Initial final-gate lint failures were fixed before review. Baseline tests and format exited 0; baseline lint terminated with `0 issues.` but its numeric process exit-code receipt was lost when the original tool session handle was not retained. The conductor accepted that disclosed baseline limitation; final gate receipts all contain exact exit codes.

Acceptance evidence: request tests cover nested/optional/oneof/map zero versus absence, numeric widths, crossed/malformed values, conflict rejection and exact bytes/work; raw projection tests cover empty/mixed/all-absent wildcard branches and aligned presence. Execution tests cover guarded Slot reads, immutable independent attempts, ordinary and worker activation isolation, fresh-batch seal rejection, concurrent Run/seal behavior, atomic event/fanout/byte rejection and stable EmitEach indexes. Wide-expression and raw-large-response tests distinguish runtime work from binding caps, including exact and one-less budgets and checked ceiling overflow.

Approved narrow Touches extension: extracted overlap semantics into IR Path.Conflicts and updated dataflow.go; cached the admitted graph runtime ceiling in program.go during bindDataflow; updated execution/README.md ownership documentation. Task Files/Touches/Approach were updated through flowctl before review. EvaluateExecution uses the same evaluator core with intermediate-copy accounting; Contract Evaluate preserves its existing finite differently weighted admitted accounting contract. Task 4 reuses the cached ceiling rather than rescanning or inventing a schema limit.

Review round 1 found a crossed-response descriptor bug: matching message FullName alone allowed wire-compatible schema reinterpretation. The fix adds bounded structural descriptor compatibility with exact-identity fast path and recursive pair termination, while accepting independently built/generated equivalents. Regression logs `.flow/tmp/fn64-task14-review-fix-red.log` and `.flow/tmp/fn64-task14-review-fix-green.log` demonstrate the failure and fix.

No commits or pushes were made: user owns commits. Review uses an immutable tree containing task-owned staged paths over the captured task-start tree, preserving all external staged changes. Actual HEAD is recorded separately from the reviewed tree in `.flow/tmp/fn64-task14-review-snapshot.json`. No full regression/Lean gates or misleading HEAD-based green receipts were generated; fn64 task 10 owns full regression.

stage: plan-sync - skipped(config: planSync.enabled != true)

Review round 2 confirmed the original finding fixed and identified invalid default inspection for repeated bytes descriptors. The focused regression reproduced the panic; repeated scalar defaults are now skipped, with empty and populated equivalent-message coverage in TestResponseEquivalentRepeatedBytes. `.flow/tmp/fn64-task14-repeated-red.log` records the failing reproduction; the final normal/race gates cover the passing fix.

stage: impl-review - ran [2026-09-05T00:10:45.529724+00:00..2026-09-05T00:28:08.884893+00:00] | codex:gpt-5.6-sol:high | SHIP after 3 rounds in one same-receipt fix loop

Review receipt: `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.14.json`. Final reviewed tree `5e0e772934bb948e31c2aee546f116333de972d6`; actual HEAD `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`; owned source comparison exited 0. Reviewer was read-only and did not execute tests; writer gate receipts above are authoritative. Nontrivial fix memory captured at `.flow/memory/bug/integration/validate-protobuf-descriptor-structure-2026-09-05.md`.
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/... (exit 0; .flow/tmp/fn64-task14-fix3-tests.log), go test -count=1 -race -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/... (exit 0; .flow/tmp/fn64-task14-fix3-race.log), make fmt-imports (exit 0; .flow/tmp/fn64-task14-fix3-format.log), make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false (exit 0; .flow/tmp/fn64-task14-fix3-lint.log)
- PRs: