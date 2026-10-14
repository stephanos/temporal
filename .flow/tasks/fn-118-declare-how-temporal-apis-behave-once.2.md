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
Behavior phase, schema half (R2, R7): adopted hints are declared once in the Temporal kit, lifted into the Umpire IR and validated by the Go reader. No lowering change; no Case byte changes.

What changed
- IR: `Realization.behavior = 16` (`ApiBehavior {visibility, causes}`) and `server_steps = 17`, with `Visibility {id, position, write: method | cause, read, eventually_within}`, `WaitBound`, `CauseBound`, `ServerStep` and `enum CauseKind`. Go bindings (`make protoc`, linux .bin) and the ScalaPB IR/API jars are regenerated.
- Framework (`model/umpire/realize`): only the open traits `Behavior` and `SystemStep`, and `Realization.behavior: Option[Behavior]`, `serverSteps: Vector[SystemStep]`. The framework guard stays green.
- Kit (`model/temporal/realize`): the vocabulary (`WaitBound`, `Visible`, `CauseKind`, `ApiBehavior`, `ServerStep`, `visibleTo`, `boundedBy`) is in `Realize.scala`. `Behavior.scala` declares the 10 visibilities and 5 cause bounds, each citing server code (re-verified at the current tree). `temporalRealization` attaches them and takes `serverSteps`. The activity realization declares `attemptStart` as delivery and `scheduleToStart`/`startToClose` as timers at `deadlineMs`; the Nexus caller declares its two timers.
- Lifter: `hintValue` writes each hint at its call line, with an id derived from the pair (`visibility.<write>.<read>`, `cause.<kind>`). It refuses, at the hint, a write or read that is no generated method constant. `constInt` now also folds Long literals.
- Go reader (`tools/umpire/model/validate_behavior.go`): refuses missing ids, duplicate ids, pairs and kinds, a missing write or read, a missing bound, and a bound or interval <= 0 or interval > bound. For server steps it refuses an unknown class, a duplicate, a performed class, an unbounded kind, a timer with no deadline and a non-timer with one. Each refusal is reported at the declaration's position.
- Lowering: only the inventory accounts for both fields, as `Unread` entries. Case bytes are unchanged.
- Coverage:
  - schema_test's closed lists gain the 2 fields, 5 messages and 1 enum (a new `schemaAddedEnums`); the stored descriptor is untouched.
  - Golden inert fields and the later inventory are updated.
  - Fixtures: lifts/Hints.scala (admitted, and refused by Go at 7 Scala lines), HintRejects.scala (lifter, 2 lines) and hintsInvalid (build, 7 lines).
- Docs: model/README.md, model/SEMANTICS.md, `.plans/API_BEHAVIOR_HINTS.md` ("As built by task 2") and the spec's API Contracts.

Decisions, and why
- Field numbers are 16/17, because fn-122.4 took 15.
- The hint vocabulary is the kit's, not the framework's: it is Temporal-specific (UMPIRE4_VISION, guard test).
- Ids are derived by the lifter, so no author names a hint and a duplicate pair is a duplicate id.
- The POST/GET binding check moves to fn-118.4, because only the lowering holds method descriptors.
- Two more pairs are declared beyond task 1's list: StartNexusOperationExecution and TerminateNexusOperationExecution -> DescribeNexusOperationExecution, both at once. They are needed by the later nexus-operation terminateSettles Case.
- `scheduleToClose` is not declared a timer: no request sets that deadline.
- The proto field is `eventually_within`, not `eventually`, because forbidigo bans the Go identifier `Eventually`.
- `temporalRealization` takes `behavior` (default: the kit's), so lifter fixtures can declare their own.
- `ApiBehavior` stays open for fn-125's sibling precondition entry.
- Source positions in model/ir and manifest.json move below edited lines. No table, ID, fingerprint, answer or Case changed, and OriginalBaseline passes.
- Gate note: the entry gate said "fn-114 closed"; the conductor started this task with fn-114 .8/.9/.10 open because those cleanup tasks don't touch the realization surface.

Review: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. The writer and the reviewer are the same model family (Opus). Round 1 was NEEDS_WORK with 2 P2 findings (two uncited bounds; a non-generated method reported at the val's line), both fixed. Round 2: SHIP. FYIs: a duplicate pair was double-reported (fixed); the fixture-only `behavior` parameter was kept on purpose.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f275cf6653, 394883c2c0, 5d1ca70612
- Tests: make protoc (exit 0), make model/gen/ir-scalapb.jar model/gen/api-scalapb.jar (exit 0), make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, incl. scala-cli lifter tests), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 46 packages, 560 s), go test -tags test_dep -count=1 -p 2 ./tools/umpire/model ./tools/umpire/lower ./tools/umpire/internal/golden ./tools/umpire/cmd/... after review fixes (exit 0), go test -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0, 0 issues), make lint-protos lint-api (only the pre-existing ir.proto:573 finding)
- PRs: