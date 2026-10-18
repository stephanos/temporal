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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: