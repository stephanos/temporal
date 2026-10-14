---
satisfies: [R5, R9, R13, R15, R16, R17]
---
# fn-126-read-each-feature-top-to-bottom-one.5 Write the remaining Models as machine objects and retire the builder forms

## Description
Convert every remaining Model to the object forms of task 4 (R15-R17), inline its single-use Scenarios (R13), retire the builder forms, and bring the docs to the final declaration shape (R9).

**Owner decisions 11-18 (spec, "Later owner decisions").** Members write `s: State` (decision 18). Add the `FailureModel` and `NegativeControl` markers with their lint rules and fixtures, and mark every Model (decision 20). Convert the remaining Models with `init`, `states`, `implements`, `refinement` and `exports` as task 4 built them, and rename the Nexus caller's `object Control` to `TrustingCaller` (pin unchanged).

**Cross-spec entry gate:**
- Task 4 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** L
**Files:**
- `model/temporal/features/{nexuscaller,nexuscaller/closepolicy,nexusoperation}/**`, `model/temporal/shared/{taskqueue,worker}/**`;
- `model/umpire` and `model/irgen` (the builder forms and their reading removed) and every lifter fixture restated in the object forms;
- `tools/umpire/internal/golden/**`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_VISION.md`, `AGENTS.md`, `.plans/DSL_OPERATORS.md`, `.plans/QUINT_MODULE_LAYOUT.md`.

**Touches:** [model/temporal/**, model/umpire/**, model/irgen/**, model/check/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, model/README.md, model/SEMANTICS.md, .plans/**, AGENTS.md]

### Approach
- **Nexus caller and Nexus operation:** as the activity.
- **Close policy:** `RejectAfterClose` holds the family's effects (`deliver(policy, redelivery, s, r)`, `reset`, …). Each of the nine designs becomes a `Derived` object whose derivation names its policy, reset and channel, with its own `queries`, so `designQueries` splits across the objects. Per-class rules where they read better (`handler.complete(…)`).
- **Task queue and worker:** machine objects (`DispatchQueue`, `MatchingQueue`, …, `Polling`), with `worker.workerStop`/`workerResume`/`serve` bound by rules.
- **Retire the builder forms.** Remove `machine[S, O, F] { … }`, `steps`, `starts`/`ends` and the `compose(…)` value form from `model/umpire` and their reading from the lifter. Restate every passing lifter fixture in the object forms, and confirm by its expected IR that nothing moved apart from recorded deltas.
- **Docs (R9):**
  - the README's "Writing a Model" shows the object forms, rules and effects, the sections and R2's order;
  - SEMANTICS states the rule lowering and disjointness;
  - `.plans/DSL_OPERATORS.md` records the owner's reversal of its rejected guard helper: `when` and `in` are rule headings, never guards inside a step;
  - QUINT_MODULE_LAYOUT and DSL_SIMPLIFICATION mark what landed.

### Investigation targets
**Required:**
- `model/temporal/features/nexuscaller/closepolicy/Model.scala:515-580` (derivations)
- task 4's done summary (the wiring and the recorded deltas)
- `model/irgen/testdata` (fixtures to restate)
**Optional:**
- `.plans/UMPIRE_MODULES.md:30,321`

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
grep -rn "machine\[\|steps(\|starts(\|ends(" model/temporal model/irgen/testdata --include=*.scala
```

### Execution constraints
- R5 holds. A table, answer or Contract change stops the task.

## Acceptance
- [ ] Every Model is written in the object forms with rules, effects and sections. The close policy's designs are `Derived` objects with their own Queries.
- [ ] The builder forms are gone from `model/umpire` and the lifter. Every lifter fixture uses the object forms, and the grep in Quick commands finds none.
- [ ] Tables, IDs, names, answers, lint findings and Contracts are unchanged apart from recorded R5 deltas.
- [ ] The R9 docs describe the final declaration shape. `.plans/DSL_OPERATORS.md` records the guard-helper reversal.
- [ ] All gates of the spec's Verification pass.


## Done summary
Every remaining Model is a machine object with rules, effects and sections. The close policy's designs are `Derived` objects with their own Queries. The failure-model and negative-control markers come with a lift-time lint, and the builder forms are retired from `model/umpire` and the lifter. Every lifter fixture is restated in the object forms.

### Per-Model conversion (`disabled` / inverted guards, before → after)
| Folder | What it became | `disabled` | Inverted |
| --- | --- | --- | --- |
| `shared/worker` | `Polling` | 3 → 0 | 3 → 0 |
| `shared/taskqueue` | `DispatchQueue`, `MatchingQueue` (Machines); `DispatchQueueUnderStorageLoss`, `LossyMatchingQueue`, `ForgetfulQueue`, `VolatileQueue` (Derived), each provider with its own `queries`; the storage-loss assumption at the file's top level | 12 → 0 | 8 → 0 |
| `nexuscaller` | `NexusProduct`, `NexusProtocol` (refinement with `toProduct` and the unobservable backoff), `HandlerWorker` (Derived), `NexusCaller` (Composition), `ForgedCompletion` (pin `Control$` and family kept) | 13 → 1 | 7 → 0 |
| `closepolicy` | `RejectAfterClose` plus 8 Derived designs | 8 → 0 | 8 → 0 |
| `nexusoperation` | `NexusOperation` | 5 → 0 | 5 → 0 |
| **All Model folders** | | **42 → 2** | **31 → 0** |

The two `disabled` left are rule-level `disabled(...)` calls on `ActivityProduct` and `NexusProduct`. Single-use Scenarios are inlined under their names (R13).

### The close policy's nine designs
- `RejectAfterClose` holds the shared effects: `deliver(policy, redelivery, s, r)`, `reset(rule, s)` and `expire`.
- The other eight are `Derived` objects. Each derivation names its policy, reset and channel, and each has its own `properties` (its progress claim) and `queries.all`.
- A bare `rebind` over rules that each fire a different class keeps their guards. Merging rules of one class with different effects is refused.

### Markers and their lint rules
`FailureModel` and `NegativeControl` mix into Machine, Derived and Composition objects, including `Composition(c.withMember(...))`. They change no byte of the IR. The lint runs at lift time and refuses, at the object:
- a negative control that nothing the run checks can refute (a `verify`, a Run expected `violated`, or a refinement check);
- a negative control that something refines, or that declares a refinement of its own;
- a failure model that binds no fault (an action of the party `fault` or of a `faults` section that some state enables; a composition counts through its members);
- a failure model whose every Query expects its Run violated;
- a machine or composition that binds a fault and is marked neither;
- an object marked both.

Each rule has a refusal fixture in `lifts/MarkerRejects.scala`. The rules are weak by construction, because the IR holds no expected check answer; the Go answer pins are the guarantee (README).

**Marked FailureModel:**
- `AdmissionResponseLoss`
- `DispatchQueueUnderStorageLoss`
- `MatchingQueue`
- `LossyMatchingQueue`
- `CurrentOverMatching`
- `CurrentOverLossyMatching`

**Marked NegativeControl:**
- `StaleAdmission`, `StaleOverQueue`, `StaleOverMatching`
- `ForgetfulQueue`, `VolatileQueue`, `CurrentOverForgetful`, `CurrentOverVolatile`
- `ForgedCompletion`
- the close policy's `RejectAfterClose`, `AckByOriginal`, `ForgetsCancelOnReset`, `TruncatesOnReset`, `RejectAfterCloseWithDeadline`, `AckByOriginalWithDeadline`

`HeldAdmission` stays unmarked because it binds no fault. `StaleRecord` stays unmarked because nothing checks it.

### Builder forms retired, fixtures restated
- **Removed:** `machine[S, O, F]{…}` in both forms, `MachineScope`, the builder statements, `refines(...)`, and `compose` with its chain.
- **Lifter:** reads only objects. It refuses `val`-declared machines and compositions, and feature-file objects that are neither machines nor compositions.
- **Fixtures:** they bind hand-written step functions with `Bindings(a ~> f, …)`, which the lint refuses in a Model. Expected IR changed only in positions, root strings, `…State` type renames, and refinement maps that became `toProduct`. Removed: the builder-only refusals; dead checks. Restated as a lifter refusal: "monitor of another state type".
- **Grep:** the Quick-commands grep finds only Scenario `.starts(state)` calls, which are the Scenario API, not the builder.

### Carry-forward decisions
- **`unobservable` without a refinement:** a header member. With a refinement it stays inside `object refinement`, and a header one is refused there (fixture).
- **`IrFile.construct`:** follows derivation sources, `refining` products, and the machine a `refinement` section refines. The gate's test fails on any machine an IR file lifts that construction never reaches.
- **Derived compositions:** the markers mix in.

### R5 deltas and the equality proof
- **Proof:** the reader projection is byte-identical for all 7 IR files; every Case is byte-identical; `project5.py` shows the IR equal with positions dropped and Function names read as tokens.
- **Deltas:** positions, IR root strings, and function symbols (`<machine>.rules.<action>`, and members moved into sections).
- **Recorded R5 change (P2-1), in `nexus-caller.lint.json`:**
  - Two acceptance reasons now name the rules instead of the old step functions.
  - The `disabled-by-default` finding for `nexusProduct` on `handlerReply-handlerError-true in scheduled` is gone, and so is its acceptance.
  - That finding (H1) can no longer fire for class-ruled actions: the lowering gives every class without a rule an explicit `Nil` case, so no class falls to a default arm.
  - The `silent-rejection` and `never-enabled` findings for that class are unchanged.

### Docs
- **README:** shows only the object forms, plus markers (with the weakness note), `Bindings`, the header `unobservable` and assumptions at a feature's top level.
- **SEMANTICS:** "Rules" covers lowering, disjointness and when a `rebind` is refused.
- **DSL_OPERATORS:** records the guard-helper reversal.
- **QUINT_MODULE_LAYOUT and DSL_SIMPLIFICATION:** mark what landed; the Quint study's builder sketch is labelled as history.
- **UMPIRE_MODULES and UMPIRE4_VISION:** updated.

### Gates
The table under "Gate results" above covers this round. The earlier round's logs are `model-gate.log`, `go-suite.json` (export passed alone at `-p 1`), `lint-model2.log`, `umpire-check-fixtures.log` and `canary-check-case.log`, all green.

### Decisions and notes
- The marking differences from decision 20's list are as above.
- The `Placebo` rename is fn-126.8's (decision 17 now names the control `TrustingCaller`).
- Faults identified only by actor name are an fn-123 question.
- Decisions 24–27 change the `val all` roots, the `Section`/`Party` words in the marker lint, and `when`; they are task 7's.
- Five subagent worktrees remain under `temporal/.claude/worktrees/agent-*`. Their work is all merged here, so they can be removed.
- **Subagents: 5** (task queue and the activity's markers; Nexus caller and operation; close policy; reject fixtures; passing fixtures). I made every commit.

### Review
- **Reviewer:** claude-opus-5-5 at high, fresh context (host-dispatched subagent); writer and reviewer are the same family (Opus).
- **Round 1: SHIP, no P1.** The reviewer reran projtool (byte-identical) and diffed the lint findings by kind, owner and subject.
- **P2-1:** the removed `disabled-by-default` finding was accepted by the host as a recorded R5 change; H1 cannot fire for class-ruled actions.
- **P2-2 and the P3s:** applied in b76409369b..4f22be8bbe.
- **Round 2: checked by the host directly** (the owner asked to return to a single lane), and accepted:
  - json4s in the Models' tests is not a new library to the project; the lifter already depends on it through scalapb-json4s.
  - The reflection in `Refinement.declaredBy` serves only the gate's construction check, not a machine's run-time wiring, and is documented. The alternative, an overridable member, would make every author write `override object refinement`.
- **Owner questions recorded in `.flow/tmp/fn-126/carry-forward.md`:**
  - faults identified by actor name (an fn-123 question);
  - the marking differences from decision 20's list.
- **Moved to fn-126.8:** the control's rename to `TrustingCaller` (decision 17).
- **Commit hygiene, recorded:**
  - d5e5aa2d14..1e0a0364d1 build only together;
  - b76409369b builds only with 4f22be8bbe.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aacd471ca0, f1ab13b06f, 888b248863, d0811706a8, 53647161d9, 68cf64c26b, d5e5aa2d14, 23e055526d, 8739e92303, 1e0a0364d1, 38aa5d5cd6, 072f04fcf2, 95c3a32845, e631abc924, 72d652caff, 787452ea6c, b76409369b, d8f01ac269, 3d1dba93f5, 2a06fbf50c, 44ec1ea2d0, 84f82d7d57, 170db92727, 4f22be8bbe
- Tests:
- PRs: