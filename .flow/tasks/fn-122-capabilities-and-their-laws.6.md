---
satisfies: [R8, R9, R11]
---
# fn-122-capabilities-and-their-laws.6 Document capabilities and laws, count authored versus generated claims and close fn-122

## Description
Docs, the table view's `promises`/`doesNotPromise` column read from the sidecar, the authored-versus-generated counts per Model, the vision's acceptance-test evidence, and the full gates once.

**Size:** S
**Files:** `model/README.md` (capabilities, laws, `except`, `overriding`, how a new entity gets its laws, the sidecar, the core/sugar rule for this surface); the generated table view (the law's `promises`/`doesNotPromise` beside its Properties, from the sidecar); `.plans/UMPIRE_MODULES.md` (ownership of `umpire/laws`, `temporal/laws`, `temporal/features/nexusoperation`, the sidecars); `.flow/tmp/fn122-6/**`.
**Touches:** [model/README.md, .plans/UMPIRE_MODULES.md, tools/umpire/model/**, .flow/tmp/fn122-6/**]

### Approach
- README: one worked example (the activity's two declarations) and the rules that a law enters the catalog with two instantiating machines and that sugar lives in `Syntax.scala` files.
- Modality rendering (R8): on fn-120.3's per-operation table, render each law from the sidecar as the modality it pins on the cells of its capability's action (`pausedIsNotDispatched` on the `paused` cells of the dispatching action as MUST NOT, `closedIsRejectedUniformly` on the `terminal` cells), and list the cells of a capability's action no law pins; a product law the protocol has not been checked against is marked inherited. Work in `tools/umpire/model/**` beside the `promises`/`doesNotPromise` column, reusing fn-120.3's view code.
- Counts: Properties and Queries authored vs generated, before (fn-112 closing counts) and after, per Model, by one reproducible command stored with its output.
- Evidence for R11 (the vision's #PROTOCOLS acceptance test) collected from tasks 3 and 4: both entities' generated laws and Cases, the unlisted interaction law, the recorded override, the violating fixture's rejection naming law and binding.
- Run the full model gate, `make lint-model`, the Umpire Go tests with `-json` timing per MILESTONES verification instructions, and `make lint-code-fast` once; record commands, results and log paths.

### Quick commands
```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn122-6/go-test.json
```
## Acceptance
- [ ] README and module map describe capabilities, laws, waivers, the sidecar, the two-entity rule and the core/sugar file rule; the table view shows each law's `promises`/`doesNotPromise`, renders each law as the modality it pins on the per-operation table's cells, lists unpinned cells of a capability's action and marks inherited product laws (fixture on the activity IR).
- [ ] Authored vs generated Property and Query counts per Model, before and after, are in the done summary with their command.
- [ ] The done summary links the evidence for each clause of the vision's #PROTOCOLS acceptance test (R11).
- [ ] Model gate, lint-model, Umpire Go tests and lint-code-fast pass in full.
## Done summary
Documented capabilities and laws, rendered each law on the per-operation table, counted authored versus generated claims, and ran the closing gates. fn-122.7 (Pausable on the fn-119 workflow) is deferred and blocked.

**What changed**
- **Table view (R8).** `umpire-lint --tables` now prints, after each machine's per-operation table, the laws the machine is held to (`tools/umpire/lint/lawtable.go`). For each law it shows:
  - the claim, the law and the capabilities that bring it;
  - `promises` and `does not promise`, read from the sidecar;
  - the waiver, where the law is overridden;
  - the modality the law pins on the cells of its capabilities' actions, beside each cell's own modality.

  A transition law pins MUST NOT, written "MUST NOT of its results" on a MAY cell. A same-step law pins MUST, marked "on its find's path only" when only a find asks it. A law whose capabilities name no action (Closable's) pins the terminal cells of "every class" (or "every class but …").

  After the laws, the view lists the cells of the capabilities' actions that no law pins, and the `except` waivers. The product's laws, as the protocol sees them, are marked `inherited from activityProduct, unchecked`. Compositions get a laws block with no cells.

  Rows are unchanged: a test compares them with the view built without a sidecar. Example: `activityProduct.pausedIsNotDispatched` is MUST NOT on the `paused` cells of `attemptStart (Pollable.dispatch)` (where the step function is silent, `?`), `control-pause` and `control-unpause`. Unpinned cells include `attemptStart` in `scheduled`, which is a MAY.
- **Sidecar.** Each claim gains `actions`, the class each action field of its capabilities names, keyed `<Capability>.<field>` and spelled as Go keys a class (`"Pollable.dispatch": "attemptStart"`). This lets the view name no capability.
  - Changed for it: the lifter (`model/irgen/Capabilities.scala`, `actionsOf`), the Go reader (`LawClaim.Actions`), and regenerated `model/ir/*.laws.json` and `lifts/expected/capabilities.laws.json`.
  - Unchanged: no IR file and no Case. OriginalBaseline passes.
- **Docs.**
  - `model/README.md`: new section "Capabilities and their laws". It covers the capability table, the activity's two declarations as the worked example, how a law is lifted, `except`/`overriding` with the admission and Nexus examples, how a new entity gets its laws, the catalog and two-entity rule, the sidecar, the laws on the table, and core vs sugar (no sugar on this surface; any would go in a `Syntax.scala`). It also fixes a stale "allowances" sentence.
  - `.plans/UMPIRE_MODULES.md`: a new `model/temporal/capabilities` row; the DSL row names the mechanism; the sidecar is listed under the IR generator, the reader and lint. Stale fn-122.8 sentences are fixed.
  - `.plans/SEMANTIC_PROTOCOLS.md`: a "Where it stands" paragraph, and a corrected path.
  - `tools/umpire/README.md`: the `--tables` description.

**Counts (R9)**, authored/generated, from `bash .flow/tmp/fn122-6/counts.sh`, which defaults to `92fde6cf2e` (fn-112 closed), `3aad890a05^` and HEAD. Output is in `.flow/tmp/fn122-6/counts.txt`, per owner.

| IR file | Properties before → after | Queries before → after |
| --- | --- | --- |
| activity | 11/0 → 9/5 | 12/0 → 10/5 |
| activity-system | 39/0 → 24/17 | 84/0 → 70/17 |
| nexus-operation (new) | – → 0/4 | – → 0/4 |
| activity-race, nexus-caller, nexus-close, nexus-control | 122/0, unchanged | 168/0, unchanged |
| **all** | **172/0 → 155/26** | **264/0 → 248/26** |

- `3aad890a05^` equals fn-112's close.
- `nexusOperation.closedIsRejectedUniformly` is the override. It is counted as generated, though its body is the entity's own def.
- 18 authored activity-system Queries (the pinned paths and `*.product.pausedIsNotDispatched`) read generated Properties.

**R11 evidence:** `.flow/tmp/fn122-6/r11-evidence.md`, with clause-by-clause links.
- Terminable is on activityProtocol and nexusOperation.
- Pausable/Pollable is on activityProduct and the admission record (currentAdmission). This is the second entity until R10, as R11 allows.
- Generated claims are in `model/ir/{activity,activity-system,nexus-operation}.laws.json`, and the four generated Cases are under `model/cases/*{terminateSettles,cancelIsRequested}-case.json`.
- The pair law comes from `Catalog.pair(Pausable, Pollable)`, and no author lists it.
- The override is in `nexusoperation/Capabilities.scala`, with its waiver in the sidecar and in `nexus-operation.lint.json`.
- The violating fixture `rogueJob` fails with `rogueJob.pausedIsNotDispatched breaks the law pausedIsNotDispatched of Pausable and Pollable (paused = …, running = …)` (`tools/umpire/model/capabilities_test.go`).

Notes:
- Only the functional laws lower to Cases. The safety laws are verified Queries, `nothing-to-realize`.
- No single second entity declares both Terminable and Pausable.

**Gates** (`.flow/tmp/fn122-6/gates.txt`), all exit 0:
- At b44e7997e4: `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, `make lint-model`, `make umpire-check-lint`, and the full Go tooling suite with `-json` (`go-test.json`; 48 packages pass; lower 205 s, model 163 s, export 107 s).
- At 138c2728a6, after the complexity refactor and the review fix: `make lint-code-fast` and the model gate again, plus the lint and umpire-lint packages and OriginalBaseline.
- The first lint-code-fast failed on revive cognitive-complexity in three writers. Fixed by splitting them in 496e0f1f19, with output identical.

**Decisions**
- **The view lives in `tools/umpire/lint`, not `tools/umpire/model`.** fn-120.3's per-operation table code, which the task says to reuse, is there.
- **The capability's action comes from the sidecar.** A law's parameters do not carry it (Pollable's `dispatch` is no parameter of `pausedIsNotDispatched`), so the sidecar records it.
- **The modality comes from the claim's construct** (transition → MUST NOT, same-step → MUST), following MODALITIES.md §2. The view evaluates nothing new.
- **Inherited claims are `unchecked`** when no Query over the refining machine's Scenarios asks them.
- **fn-126** (capabilities as a `laws` section of machine objects) is not pre-empted. The docs describe the current `capabilities(m, limits)(…)` form.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3 (MUST NOT beside a MAY cell reads backwards). It is fixed in 138c2728a6, which writes "MUST NOT of its results", and the FYI `cellOf` helper was taken too.
- Suppressed at confidence 50, left as is: an action key matching no class is ignored, which does not happen in today's IR; a composition's `except` with no generated claim is not shown, but every composition in model/ir has claims.

**Deferred:** fn-122.7 / R10 (Pausable on fn-119's workflow) stays blocked on fn-119.4. It is recorded in the spec's coverage table.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6c8dd3b8b7, b44e7997e4, 496e0f1f19, 138c2728a6
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; only *.laws.json changed), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, at b44e7997e4 and 138c2728a6), make lint-model (exit 0), make umpire-check-lint (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, .flow/tmp/fn122-6/go-test.json), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|OriginalLaw' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 ./tools/umpire/lint ./tools/umpire/cmd/umpire-lint (exit 0, at 138c2728a6), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0 at 496e0f1f19 and 138c2728a6; first run exit 2 on cognitive-complexity, fixed), bash .flow/tmp/fn122-6/counts.sh (exit 0, counts.txt), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 1 SHIP, P3 fixed)
- PRs: