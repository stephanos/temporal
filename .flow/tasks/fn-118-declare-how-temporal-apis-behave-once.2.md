---
satisfies: [R2, R7]
---
# fn-118-declare-how-temporal-apis-behave-once.2 Declare adopted hints in the shared kit, lift them into the IR and validate them in Go

## Description
Behavior phase, schema half. Add the IR fields task 1 named, the Scala declaration surface in the shared Temporal kit, lifting, and Go reader validation. No lowering change yet.

**Cross-spec entry gate:** start only after fn-112.10 is done (structural Case-byte freeze verified) **and fn-114 is closed** (it freezes Case bytes through its last task; MILESTONES 'after 4'). Query.total (fn-112.11) and choice names (fn-120.1) are already in the IR; stay compatible with both.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/ir.proto` and generated `api/umpire/v1/*`; Scala IR jar via the gate's `--generate-ir`; `model/umpire/realize/**` (generic hint types); `model/temporal/realize/**` (the adopted hint declarations with server-citation comments); `model/lifter/Realizations.scala`; `tools/umpire/model/validate_realization.go`, `schema_test.go`; lifter fixtures.
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/umpire/realize/**, model/temporal/realize/**, model/lifter/**, model/gate/**, tools/umpire/model/**, model/ir/**, model/README.md, model/SEMANTICS.md]

### Approach
- Add the hint messages/fields exactly as task 1 decided. Default-empty fields must not change canonical bytes or fingerprints of existing IR (memory: default-empty extensions must preserve canonical fingerprints); compare the existing six IR files byte-for-byte.
- Regenerate Go bindings (`make protoc`) and the linked API jar; extend `TestSchemaRenameKeepsTheWireBytes` (`tools/umpire/model/schema_test.go:28,75-113`) via `schemaSupplement` to set every new field, without rewriting the stored historical descriptor.
- Scala: a hint attaches to the fn-117 typed method/message (`WorkflowServiceGrpc.METHOD_*` resolved in `model/lifter/Realizations.scala:123-170`); relationships (`visibleTo`) live once in `model/temporal/realize`, each with a comment citing the server code path (e.g. `chasm/lib/activity/handler.go:377-410`). Declare a visibility for every write->read pair task 1 listed, at once or eventually, so task 5's hints-only lowering refuses no existing Case.
- Split the refusals once: the lifter refuses an unknown method/message (it holds the descriptors); the Go reader refuses a relationship missing a required bound and a bound <= 0, reporting the Scala position carried in the IR. Wait bounds per cause kind (task 1's kind (b)) and per command kind (kind (c)) are declared beside the visibility relationships.
- Removal test for R7 lands in task 4 once lowering consumes hints.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:640-1010`
- `tools/umpire/model/schema_test.go`
- `model/lifter/Realizations.scala:110-180,500-640`
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.11.md` - the Query.total schema procedure to mirror
**Optional:**
- `.flow/memory/bug/integration/default-empty-extensions-must-preserve-2026-09-05.md`

### Quick commands
```bash
make protoc && make umpire-gen-model && git diff --stat model/ir model/cases
go test -count=1 -tags test_dep ./tools/umpire/model/...
scala-cli test model/lifter
```

### Execution constraints
- The six IR files may gain only the new hint fields (their other bytes, tables, IDs and fingerprints stay exact); every Case stays byte-identical because the lowering ignores hints until task 4.
## Acceptance
- [ ] Each adopted hint is declarable once on a typed API in `model/temporal/realize`, with a server-citation comment.
- [ ] Hints lift into the IR; the Go reader validates them; fixtures prove refusal at the Scala line for unknown method/message, missing required bound and non-positive bound.
- [ ] Go bindings and the API jar are regenerated; historical wire coverage sets every current field without replacing stored descriptors.
- [ ] Existing IR changes only by the new hint fields; fingerprints and Case bytes are unchanged; model gate and Umpire Go tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
