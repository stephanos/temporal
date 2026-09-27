---
satisfies: [R1, R2, R9]
---
# fn-92-compose-entity-machines-into-one-system.1 Worker entity module and the feature-entity-uniqueness lint

## Description
Create `Temporal.Feature.Worker.Model` and add the `feature-entity-uniqueness` lint with an allowlist that records the duplicates that exist once the worker module lands. No existing module, entity, action, or fixture changes.

**Size:** M
**Files:** `model/Temporal/Feature/Worker/Model.lean` (new), `model/Temporal/Feature/Worker/Tests.lean` (new), `model/Temporal/Feature.lean` (import), `model/ModelLint.lean` (declaration-level rule in the env-importing driver), `model/ModelLint/EntityTests.lean` (new), `model/ModelLint.lean` test aggregator, `Makefile` (controlled violation under `lint-model`), `model/HANDWRITTEN_INVENTORY.md` (only if the inventory check requires a row)
**Touches:** [model/Temporal/Feature/Worker/**, model/Temporal/Feature.lean, model/ModelLint.lean, model/ModelLint/**, Makefile, model/HANDWRITTEN_INVENTORY.md]

### Approach
- `Worker/Model.lean`: `entity worker key: taskQueue`; actions `workerStop`, `workerResume` (party worker, no entity), and classless `serve` (party worker, `on: worker`); machine `polling` with phases polling and stopped, `starts: [polling]`, `ends: [polling, stopped]`, `serve` enabled only in polling, stop and resume moving between the phases. No set and no case.
- Lint: in `lintModule` (`model/ModelLint.lean:92-114`) read `Registry.entities` and the action declarations, report a duplicate `entity` or `action` name across `Temporal.Feature` production modules naming both modules; allowlist exactly `workflow` (Caller, Start, Outage), `startWorkflow` (Start, Outage), `workerStop` (Caller, Outage, Worker), `workerResume` (Outage, Worker), each with the follow-up spec named in the allowlist comment, so the new worker module's own actions are covered; exclude modules under `Tests`, `Success`, and `*Tests.lean`; pin a controlled violation in `lint-model` as `Makefile:997-1020` does.
- Verify byte identity: `make umpire-check-case-runtime-conformance`, `make canary-check-case`, `make umpire-check-goldens`, and `go test ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/...` all pass unchanged.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:1418-1470` (entity elaborator, `Registry.localEntities`), `:1509-1650` (action elaborator)
- `model/ModelLint.lean:92-114`; `model/ModelLint/ImportGraph.lean:323` (diagnostic shape)
- `model/Temporal/Feature/Workflow/Outage/Model.lean:40-90` (today's `workerStop` and `workerResume` as stutter rows, for the allowlist)

**Optional:**
- `model/Temporal/Feature/System/Info/Model.lean` (the smallest complete module, as a shape)

### Key context
- The lint is declaration-level, not import-graph: the import rules cannot see declarations.

## Acceptance
- [ ] `Worker/Model.lean` elaborates with the stated entity, actions, and machine; its Tests pin the table
- [ ] `feature-entity-uniqueness` enforced by `make lint-model`, allowlist exactly as stated, controlled violation pinned
- [ ] Every fixture, the canary fixture, the goldens, and the Go replay tests unchanged and passing
- [ ] `LEAN_NUM_THREADS=1 make lint-model` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
