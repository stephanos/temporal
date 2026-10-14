---
satisfies: [R1, R2, R9]
---
# fn-92-compose-entity-machines-into-one-system.1 Worker entity module and the feature-entity-uniqueness lint

## Description
Create `Temporal.Feature.Worker.Model` and add the `feature-entity-uniqueness` lint as a declaration-level pass of `umpire-lint`, with an allowlist that records the duplicates that exist once the worker module lands (R1, R2, R9). No existing module, entity, action, fixture, or golden changes.

**Size:** M
**Files:** `model/Temporal/Feature/Worker/Model.lean` (new), `model/Temporal/Feature/Worker/Tests.lean` (new), `model/Temporal/Feature.lean` (import), `model/TemporalModelTests.lean` (import the Tests module), `model/ModelLint/Entity.lean` (new: the rule, its allowlist, its diagnostic), `model/ModelLint.lean` (run the rule in `main`), `model/ModelLint/ImportGraphTests.lean` (controlled entity violation), `Makefile` (fourth controlled-violation block under `lint-model`)
**Touches:** [model/Temporal/Feature/Worker/**, model/Temporal/Feature.lean, model/TemporalModelTests.lean, model/ModelLint.lean, model/ModelLint/**, Makefile]

### Approach
- `Worker/Model.lean`: `entity worker key: taskQueue`; actions `workerStop`, `workerResume` (party worker, no entity, as `Outage/Model.lean:55-59` spells them), and classless `serve` (party worker, `on: worker`); machine `polling` with phases polling and stopped, `starts: [polling]`, `ends: [polling, stopped]`, `serve` enabled only in polling, stop and resume moving between the phases. No set, no case, no Query, so no differential line. Tests pin the table (`Outage/Tests.lean:23-25` shape) and `#print axioms`.
- Lint: `lintModule` (`model/ModelLint.lean:92-114`) runs Batteries linters only; the new rule is its own pass in `main` (`:116-120`) over the environment `lintModules` imports with `loadExts := true`, reading `Registry.entities` (`Registry.lean:345`) and the action entries, reporting a duplicate `entity` or `action` name across `Temporal.Feature` production modules with both module names in the `[model-…/…]` diagnostic shape of `ImportGraph.lean:224-231`. Allowlist exactly `workflow` (Caller, Start, Outage), `startWorkflow` (Start, Outage), `workerStop` (Caller, Outage, Worker), `workerResume` (Outage, Worker), each with the follow-up spec named in the allowlist comment; exclude modules under `Tests`, `Success`, and `*Tests.lean`.
- Controlled violation: a fourth flag in `ModelLint.ImportGraphTests` (lakefile:149-150) and a fourth block in `Makefile:1118-1137`, same `test "$$status" -eq 1` plus exact diagnostic pattern.
- `HANDWRITTEN_INVENTORY.md` needs no row: a command-authored module imports no authoring owner (`HANDWRITTEN_INVENTORY.md:5-21`).
- Verify byte identity: `make umpire-check-case-runtime-conformance`, `make canary-check-case`, `make umpire-check-goldens`, and the Go readers of the recorded control Run pass unchanged.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:1425-1475` (entity elaborator, `Registry.localEntities` at 1450), `:1516-1660` (action elaborator)
- `model/ModelLint.lean:92-120`; `model/ModelLint/ImportGraph.lean:224-231` (diagnostic shape); `model/ModelLint/ImportGraphTests.lean`; `Makefile:1112-1140`
- `model/Temporal/Feature/Workflow/Outage/Model.lean:55-59, 87-99` (today's `workerStop` and `workerResume` and their stutter rows, for the allowlist)

**Optional:**
- `model/Temporal/Feature/System/Info/Model.lean` (the smallest complete module, as a shape)

### Key context
- The lint is declaration-level: the import rules cannot see declarations, and `lintModule`'s Batteries pass cannot host it.

### Quick commands
```bash
cd model && lake build Temporal.Feature.Worker.Tests
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-case-runtime-conformance && make canary-check-case && make umpire-check-goldens
go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/...
```
## Acceptance
- [ ] `Worker/Model.lean` elaborates with the stated entity, actions, and machine; its Tests pin the table and axioms; `model/Temporal/Feature.lean` and `model/TemporalModelTests.lean` import the new modules
- [ ] `feature-entity-uniqueness` runs as its own pass of `umpire-lint`, allowlist exactly as stated, controlled violation pinned in `ImportGraphTests` and `Makefile`
- [ ] Every fixture, the canary fixture, the goldens, and the Go readers of the recorded control Run unchanged and passing
- [ ] `LEAN_NUM_THREADS=1 make lint-model` passes
## Done summary
Added `Temporal.Feature.Worker.Model`: `entity worker` keyed by `taskQueue`, `workerStop`, `workerResume`, classless `serve`, and machine `polling` with both phases terminal. Its Tests pin the table, the `temporal.worker.*` Definition IDs, and the axioms. Also added the `feature-entity-uniqueness` pass to `umpire-lint`. The pure rule lives in `model/ModelLint/Entity.lean`. The driver imports every production feature module the source discovery found and reads `Registry.entities` and `Registry.actions`. The allowlist is exactly the four duplicates the spec names. The controlled violation is pinned in `umpire-lint-tests --controlled-entity-violation` and in a fourth `lint-model` block.

Two changes fall outside the declared Touches. `model/lakefile.lean` gains one `ModelLint.Entity` root in `ModelLintSupport`, because Lake does not build a module missing from a lib root. A binder in `ModelLint.lean` is renamed from `message` to `reason`, because `message` is a token once `Umpire.Command.Registry` is imported.

Review round 1 found two issues, both fixed in 82ef51002b. First, the lint read only the aggregate roots. Second, the worker module lacked `Temporal.Case.Conventions`, so its IDs used `temporal.feature.worker`. Round 2 returned SHIP.

Gate status: the builtin lint step of `LEAN_NUM_THREADS=1 make lint-model` is INCONCLUSIVE. Every earlier `lint-model` step passed. The last step, `lake lint --builtin-only`, failed on `.olean` files that concurrent builds from other sessions in the shared checkout removed. It must be rerun on a quiet host before `done`.

stage: impl-review - ran [round 1 NEEDS_WORK (codex fan-out, rid e30014d67b354d9aaa1047197e32b01f) .. round 2 SHIP]


The lint-model gate left inconclusive by the worker (concurrent builds) was rerun by the conductor on a quiet host after fn-88 closed: exit 0 in 4443 s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 151e282205153be568f35fd2d7a6731e18abe4fc, 82ef51002be3ebfc72e9db3e35e878e970a4c337
- Tests: baseline: not run pre-edit (the task's Quick commands were first run post-edit; no Go or fixture source changed), cd model && lake build Temporal.Feature.Worker.Tests (green; table pin confirmed red first), cd model && lake exe umpire-lint-tests (green; entity cases confirmed red with an allowance removed), cd model && lake exe umpire-lint-tests --controlled-entity-violation (exit 1, exact diagnostic), cd model && lake exe umpire-lint (green; red with the workerResume allowance removed: 'duplicate action workerResume: Temporal.Feature.Worker.Model and Temporal.Feature.Workflow.Outage.Model'), make umpire-check-case-runtime-conformance && make canary-check-case && make umpire-check-goldens (green, no fixture or golden diff; run on 151e282205), go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/... (green), LEAN_NUM_THREADS=1 make lint-model: INCONCLUSIVE - every step through umpire-lint passed (both runs, the second on the fixed code), the final lake lint --builtin-only step failed twice on missing .olean files while other sessions rebuilt Umpire.Search/Temporal.API in the shared checkout; a scoped builtin lint hit the same race, LEAN_NUM_THREADS=1 make lint-model (conductor, quiet host after fn-88 closed, 2026-09-28): exit 0 in 4443 s
- PRs: