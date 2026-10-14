---
satisfies: [R4, R5, R13, R15, R16, R17]
---
# fn-126-read-each-feature-top-to-bottom-one.4 Make the machine object the machine, with effects, rules and sections, on the standalone activity

## Description
Early proof point for the declaration shape. This task reshapes the framework and the lifter once and converts the standalone activity (all three subpackages):
- the machine object is the machine (R15);
- rules say when an action fires, and effects say what it does (R16);
- every kind of member sits in a section (R17, R4 e);
- single-use Scenarios are inlined (R13, activity).

Meaning stays frozen (R5).

**Cross-spec entry gate:**
- Task 3 is done.
- Never alongside fn-124.8.
- Before fn-124.7 (the restructured functions are recorded in its harness).

**Size:** L
**Files:**
- `model/umpire/{Machine,Compose,Claims,Capabilities,Syntax}.scala`: the `Machine`, `Derived` and `Composition` object forms, `Rules`, `Syncs`, the rule sugar with `Core form:` docs, the disjointness check, and the inherited `property`, `scenario` and `capabilities(limits)` members;
- `model/irgen/{Declarations,Compositions,Claims,Syntax,Lift}.scala` and fixtures;
- `model/check/{SyntaxRule,Gate}.scala` and the R4 lint;
- `model/temporal/features/standaloneactivity/**`;
- `tools/umpire/internal/golden/{config,original}.json`.

**Touches:** [model/umpire/**, model/irgen/**, model/check/**, model/temporal/features/standaloneactivity/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**]

### Approach
- **Object forms (R15).**
  - `abstract class Machine[S, O, F]` with `start`, `end(s)`, optional `entity`, `refines`, `visible`, `unobservable` and `evidence`, and the inherited claim builders.
  - `Derived(base-derivation)`, where the derivation is today's `rebind`/`extend`/`restrict`/`refining`/`assuming`/`unmonitored`/`withMember` expression.
  - `Composition[S](members*)` with `object syncs extends Syncs`.
  - Names come from the object, with its first letter lowered (R7).
  - Pick the runtime wiring for sections without reflection (an object cannot be overridden, and nested objects initialize lazily), and prove it in munit.
  - The builder forms stay until task 5 retires them.
- **Rules (R16).**
  - Implement `Rules` and `Rules(phase)` with `when(g) { … }`, `in(p*) { … }` and `disabled(action*)`.
  - Rule-form arguments for `rebind`/`extend`, and the bare-binding `rebind(action ~> e)` that keeps guards.
  - Lower each action's rules in order to one step function `if g1 then e1 else … else Nil`, named `<object>.rules.<action>`.
  - The disjointness check runs at construction, exhaustively over the `Finite` state type and input values, and names both rules and a witness state.
  - The model gate initializes every IR-file root, so an overlap fails the gate.
  - Paired fixtures (rules / hand-written core step) lift to identical tables.
- **Lifter.** Read the object forms and their sections once. Refusal fixtures:
  - `start` or `end` missing;
  - `rules` missing;
  - a section outside a machine, composition or file top level;
  - a nested section;
  - an effect outside `effects`;
  - `disabled`/`Nil` in an effect;
  - `in` without a phase projection;
  - an unbound action class in a rule;
  - a bare binding in `extend`;
  - colliding object names;
  - an overlap (a munit or gate fixture).
- **Lint (R17, R4 e).** Section membership and order; no hand-written `action ~> step` in a Model.
- **Convert the standalone activity**, following the spec's API sketch with today's names:
  - effects named for what they do;
  - positive rules;
  - `disabled(process.workerStop)` on the product;
  - `properties`, `laws` (with `protocolCapabilities` reading the realization), `queries` with single-use Scenarios inlined (names kept);
  - `ActivityWorker`, `StandaloneActivity`, and the record designs and compositions as `Derived`/`Composition` objects;
  - `object Files` for the roots.

  The seven server-rejected pause/unpause pairs keep their reasons in the accepted `silent-rejection` findings.
- **Regenerate.** Prove by the reader's projection that tables, refinement rows, answers, lint findings and Contracts are unchanged. Record the function names and bodies (re-capturing `original.json` if the projection cannot express them).

### Investigation targets
**Required:**
- `model/umpire/Machine.scala` (whole), `model/umpire/Compose.scala:1-120`, `model/umpire/Claims.scala:1-160`
- `model/irgen/Declarations.scala:140-200` (today's machine `Block` reading), `model/irgen/Compositions.scala`
- `model/check/SyntaxRule.scala`
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.6.md` (explicit arms), `model/ir/activity.lint.json` (silent-rejection reasons)
**Optional:**
- `.plans/DSL_SIMPLIFICATION.md` section 2 (lifter costs)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

### Execution constraints
- Stop before task 5 if the activity's tables, refinement rows or Query answers move, or if the disjointness check cannot be made exhaustive in the gate.

## Acceptance
- [ ] `Machine`, `Derived` and `Composition` object forms, `Section`-based sections, `Rules` (with `when`, `in`, `disabled`) and `Syncs` exist. Each rule heading has a `Core form:` doc, and the paired fixtures lift to identical tables.
- [ ] The lifter reads the object forms and refuses each R15/R16/R17 case at its line, with a fixture per case. The disjointness check refuses an overlap naming the machine, the action class, both rules and a witness state, and the gate runs it over every root.
- [ ] The standalone activity (all three subpackages) is written in the object forms. No effect returns `disabled`/`Nil`, no inverted guard remains, single-use Scenarios are inlined with their names, and capabilities sit in `laws`.
- [ ] Tables, refinement rows, IDs, names, answers, lint findings and Contracts are unchanged. The function names and bodies are recorded as R5 deltas.
- [ ] All gates of the spec's Verification pass.


## Done summary
The standalone activity (all three subpackages) is written as machine objects with states, refinement, effects, rules, properties, implements and queries sections; the framework and lifter read the object forms; tables, refinement rows, answers, IDs, names, lint findings and Contracts are unchanged.

**Commits** (base `umpire` 61e0b81b16):
- a79b352f30 feat(umpire): machine, derived and composition objects with rules, states and refinement
- 1188d6196f feat(irgen): lift machine objects, lower rules and lint sections
- 7e887776e1 refactor(model): the standalone activity as machine objects with rules
- 41bd08735a docs(model): machine objects, rules and sections in the README
- ad6c8e1d91 merge of `umpire` (spec-only commits). The one conflict was a blank line umpire added to the old `Record.scala`; I kept this branch's version.
- Commits 2 and 3 only build together: the lifter fixtures import `ActivityProtocol`, and the lint requires `object exports`.

### Framework forms and runtime wiring (`model/umpire`)
- **`Machine[S, O, F]`** has three abstract members: `init`, `end(s)` and `rules: RuleBook`.
  - `object rules extends Rules(...)` implements the abstract `def rules`, so nothing is overridden and no reflection is used.
  - The machine reads only `rules`. Every other section stays lazy; munit shows the machine object constructs without initializing its rules.
  - The name comes from the object's class name, first letter lowered, as `Law.name` already does it.
  - `Declares[S]` exports `type State = S`, with no `Outcome` or `Fact` alias.
- **`Derived(derivation)`** makes `init`, `end` and `rules` final, so a derived object cannot declare its own.
- **`Composition[S]`** has two constructors: one taking the members, one taking `c.withMember(...)`.
  - `Syncs` holds `sync` and `replaces`.
  - `Refinement[S, P](product)` has an abstract `toProduct`, so the map is type-checked.
- **`Rules` and `Rules(_.phase)`** (sugar in `Syntax.scala`, each heading with a `Core form:` doc):
  - `when`, `in` (it needs a projection, with a clear error otherwise) and `disabled`.
  - Inline `~>` for whole actions and single classes, so error messages can name the action.
  - A top-level `when(...)` for `rebind` and `extend`.
- **Disjointness:** each new rule is checked against the earlier rules of its class as the rules object is constructed, over every state of the `Finite` type and every class. An overlap names the machine, the class, both rules (place and heading) and a witness state.
- **The gate check:** `IrFile.construct()` and `model/temporal/IrFiles.test.scala` construct every IR file's roots, so an overlap fails the gate. The test also requires the declared IR files to match `model/ir/*.json`, so a file it misses fails it.
- **munit coverage:** `model/umpire/Rules.test.scala` tests the wiring, the lowering, the derivations, the exact overlap message and composition members.

### Rules lowering
- Each action's rules lower in order to one step function, `<machine>.rules.<action>`.
- Where any rule fires one class, the lowered function matches the inputs first, with one explicit case per value, and then tries the guards on the state. A plain `if c == x && g(s)` chain would short-circuit past the state and create new `disabled-by-default` lint findings.
- `disabled(a)` lowers to a function returning `Nil`.
- Paired fixture: `Switch` (rules) and its hand-written core twin `Core.coreSwitch` must give equal tables (Go `TestRulesLowerToTheCoreTables`).

### Lifter refusal fixtures
- **`lifts/Rejects.scala`**, each refused at its line:

  | Line | Case |
  | --- | --- |
  | 1236 | effect outside `effects` |
  | 1244 | `disabled` in an effect |
  | 1253 | rule under no heading |
  | 1261 | heading under a heading |
  | 1270 | action both disabled and fired |
  | 1273 | bare binding in `extend` |
  | 1276 | `rebind` of several rules' effects to one |
  | 1280 | rules for an action the source doesn't bind |
  | 1285 | colliding object names |
  | 1293 | composition without `end` |
  | 1304 | derived composition with its own `end` |
  | 1310 | refinement member outside `refinement` |

- **Refused by the build:** `objectForms/Invalid.scala` 33 (no `init`), 39 (no `end`), 45 (no `rules`), 51 (a `Derived` with its own rules); `crossed/Rules.scala:16:43` (`in` without a projection).
- **Earlier and elsewhere:** nested and misplaced sections are task 3's lifter fixtures; the overlap is refused by munit and the gate.
- **Paired `State` fixture:** `def broken(s: State)` and `def brokenLamp(s: Lamp)` lift to identical IR. No lifter change was needed, because type lookup already dealiases.

### Lint additions (`Order.scala`)
- **Order inside a machine or composition object:** header (`init`, `end`, `entity`, `evidence`), then `states`, `refinement`, `effects`, `monitors`, `rules` (or `syncs`), `properties`, `implements`, `queries`. This is the carry-forward rework of the section ranking.
- **R17 membership:** each member must sit in its own section. Vocabulary belongs in `states`, refinement members in `refinement`, effects in `effects`, monitors in `monitors`. A hand-written `action ~> step` is refused outside `rebind`, and a section may not sit at a feature's top level.
- **IR files:** `object exports` is the only accepted name.
- **Carry-forward (a):** rule guards, `when`/`in` bodies and the `Rules(_.phase)` projection now count as initialization reads, so a guard reading a later val is refused.
- **Carry-forward P2:** a whole-index check computes every section member's Definition ID the way the lifter does and refuses twins across IR files and misplaced sections.
- **Fixtures:** new `testdata/sectionOrder/` with 14 refusals and four passing cases. In `initOrder`, `Files` became `exports` and two messages changed text (lines 72 and 102).

### The activity conversion (all three subpackages)
- **Objects:**
  - Feature file: `ActivityProduct`, `ActivityProtocol`, `ActivityWorker` (Derived), `StandaloneActivity` (Composition) and `exports.{activity, activitySystem, activityRace}`.
  - record/: `CurrentAdmission` (pin kept), `StaleAdmission` as `Derived(CurrentAdmission.rebind(when(_ => true) { ... }))`, `HeldAdmission`, `AdmissionResponseLoss`.
  - withTaskQueue/: `CurrentRecord` and `StaleRecord` (Derived), `CurrentOverQueue`, `CurrentOverMatching`, and the five `Composition(c.withMember(...))` objects.
- **Shape:**
  - Sections: `states`, `refinement`, effects named for what they do, positive rules, `properties`, `implements` (`val all`), `queries`.
  - `disabled(process.workerStop)` on the product.
  - 11 single-use Scenarios are inlined under their names; `cancelRequestedThenCanceled` is used twice and stays a val.
  - `State` is used in the members.
- **Counts:**

  | File | `disabled` before → after | Inverted guards before → after |
  | --- | --- | --- |
  | StandaloneActivity.scala | 24 → 3 | 13 → 0 |
  | Record.scala | 11 → 0 | 6 → 0 |
  | WithTaskQueue.scala | 0 → 0 | 0 → 0 |

  The 3 left are `disabled(process.workerStop)` and two comment lines. No effect returns `disabled` or `Nil`.
- The seven server-rejected pause/unpause pairs have no rule. A comment above the control rules gives their reasons, and they stay accepted `silent-rejection` findings.
- An equivalence harness in the scratchpad compared all 8 machines' lowered rules with the old step functions on every state and class: all equal.

### R5 deltas and how equality was proved
- **Reader projection** (`.flow/tmp/fn-126/fn126-4/projtool`): each machine's table plus every `model.Check` receipt (kinds, fingerprints, witnesses). Before and after are byte-identical for all 7 IR files.
- **IR projection** (`project4.py` → `ir-deltas.json`, `ok: true`): positions dropped and Function names read as tokens, the way the golden harness compares them. Every IR file, law sidecar and lint acceptance is equal, including every machine's `ends` lambda. The Nexus IR is byte-identical.
- **Recorded deltas:**
  - Positions: IR lines, two waitHint lines in Cases, one manifest line.
  - Root strings, in golden `source_root_moves` and `source_root_additions`.
  - Function symbols and bodies: 25 golden `function_name_substitutions` from the old step functions to `<machine>.rules.<action>`. `original.json` did not need re-capturing.
  - Lint acceptance prose in the three activity `.lint.json` files: "Product.effects.workerStop is disabled" now reads "ActivityProduct.rules disables workerStop". Kind, owner and subjects are unchanged.
  - The `hints` and `hintsRefused` fixture outputs name the new functions; `rules.json` is new and added to the golden `later_inventory`.
- No pinned Run needed re-recording; the testpilot and canary Cases have no diff.
- Go tests that name the activity's functions or guard text were updated: `decisions_test`, `holes_test`, `lawtable_test`, `api_test`, `quint_test`, `activity_parity_test`, and `framework_test` (framework docs reworded to a neutral orders example).

### Gates (logs in `.flow/tmp/fn-126/fn126-4/`)
- Gate update: `gen6.log`, exit 0. After the merge, `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: `model-gate.log`, `== ok`.
- `make lint-model`: `lint-model7.log`, exit 0.
- `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case`: each in its target-named log, exit 0.
- Full Go suite at `-p 2`: `go-suite2.json`. 16 of 18 packages pass; `model` and `export` were OOM-killed and pass alone at `-p 1` (`go-rerun2.json`). It ran before the merge, which added only spec files.
- `lint-code-fast`: `lint-code-fast.log`, 0 issues.

### Decisions
1. **Runtime wiring:** `rules` is an abstract member that the object implements. Optional header members are not declared in the base, because overriding a concrete member would need `override`.
2. **Incremental disjointness check:** each rule is checked as it registers, so the check runs while `rules` is constructed.
3. **Inputs matched first** in the lowered step function, to keep the lint's decisions as they were.
4. **`end`:** `def end(s) = body` lifts as `s => body`. ActivityProduct and CurrentAdmission forward to named `states.over` and `states.stopped` to keep their frozen `ends` shape.
5. **Derived compositions use `Composition(c.withMember(...))`, not `Derived(...)`.** This deviates from R15 and the API sketch: one class can't extend both Machine and Composition, and the helpers over compositions need `synced` and `own`.
6. **The rule `~>` is inline,** so the overlap message can name the action (via `codeOf`); TASTy keeps the call unexpanded.
7. **Owner decisions applied:** 11 `init`; 12 `object states` (I put `refinement` right after it, before `effects`); 13 `implements` with `val all`; 14 `object refinement extends Refinement(product)` with `toProduct`; 15 `object exports` in all four features (Nexus included, since the lint requires it), vals named after their IR files; 18 `type State`.
8. **ActivityProduct's effects write `s.copy(phase = ...)`** because `lint-model` refuses unused parameters. That changes function bodies only; `schedule` keeps `@unused s: State`.
9. **The R17 hand-binding check applies to object forms only.** The builder Models keep `steps(a ~> f)` until task 5.

### For the owner
- Decisions 16, 19, 20, 22 and 23 came in with the merge and are not in this task. Decision 23 (IDs as fully qualified names, no pins) will make the whole-index ID check and section transparency moot.
- The merge commit ad6c8e1d91 has git's default message without the Co-Authored-By trailer; I did not amend it.
- Lint tables now print `in` guards as `!(List(scheduled).contains(s.phase))`. It is display only; printing `s.phase.in(...)` would be a small Go change.
- `unobservable` can only be declared in `refinement` (decision 14), so a non-refining machine with unobservable timers has no place for them yet; task 5 should decide.

**Subagents: 2.** One converted the activity's Scala sources and reshaped them through the later decisions. One extended `Order.scala` and its fixtures. I wrote the framework, lifter, fixtures, Go and golden changes and docs, reconciled their work, ran every gate and made every commit.
### Review
claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP, no P1.
- **Proof re-run by the reviewer:**
  - `before/ir` is byte-identical to 61e0b81b16.
  - `project4.py` and `projtool` reran byte-identical on HEAD.
  - `TestProjectionIsClosed` pins the 25 substitutions.
  - Disjointness is exhaustive over `Finite` and every class, and `IrFilesTest` constructs every declared root.
- **P2s:**
  - The inline `~>` is now a documented, sanctioned exception, applied in 67d6376fac.
  - The R15 amendment for `Composition(c.withMember(...))` is recorded in the spec by the host (e2a81b1e1b).
- **P3s applied:**
  - 62765f6d25: lint tables print `x.in(a, b)` / `s.phase != scheduled` as written; display only.
  - 67d6376fac: a derivation's overlap names its actions and machine.
  - e7aeca25b1: colliding objects across packages, a `Nil` effect, a bound `rebind` source.
- **P3 carried to task 5:** `IrFile.construct()` does not follow `refinement.of`/`replaces` (`.flow/tmp/fn-126/carry-forward.md`).
- **Commit hygiene, recorded:** 1188d6196f and 7e887776e1 only build together. The merge ad6c8e1d91 lacks the attribution trailer and keeps git's `# Conflicts:` lines. History is not rewritten.
- **Reruns after the fixes, all pass:** model gate, `lint-model`, `./tools/umpire/lint` and `./tools/umpire/model` at `-p 1` (model alone after an OOM kill in the combined run), `lint-code-fast` and `umpire-check-cases`. `model/ir` and `model/cases` are unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a79b352f30, 1188d6196f, 7e887776e1, 41bd08735a, ad6c8e1d91, 67d6376fac, 62765f6d25, e7aeca25b1
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (ok; .flow/tmp/fn-126/fn126-4/model-gate.log), make lint-model (ok; lint-model7.log), make umpire-check-cases umpire-check-fixtures canary-check-case (ok), go test -count=1 -json -tags test_dep -p 2 ./tools/umpire/... (16/18; model and export OOM-killed, pass at -p 1: go-suite2.json, go-rerun2.json), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (0 issues), reader projection: tables and Check receipts byte-identical for all 7 IR files (projtool); project4.py ir-deltas.json ok, review reruns: model gate, lint-model, ./tools/umpire/lint and ./tools/umpire/model at -p 1, lint-code-fast, umpire-check-cases (all pass)
- PRs: