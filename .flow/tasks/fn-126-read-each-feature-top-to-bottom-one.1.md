---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R12]
---
# fn-126-read-each-feature-top-to-bottom-one.1 Write the standalone activity as one feature file per folder, rename record/ and withTaskQueue/, and lint declaration order

## Description
Early proof point for the layout. Convert `features/standaloneactivity` and its two subpackages into one feature file each (R1, R2, R3), using today's declaration forms: module objects hold vocabulary, step functions, the machine `val`s, Properties, capabilities, Scenarios and Queries. Rename the subpackages `admission/` → `record/` and `compositions/` → `withTaskQueue/` (R12). Land the declaration-order lint (R4 a-d). Meaning and identity stay frozen (R5), with only recorded deltas.

**Cross-spec entry gate:**
- fn-114, fn-118 and fn-122 are closed, and fn-127 (Simplify the DSL's words) is closed.
- Never alongside fn-124.8.
- fn-126 lands before fn-124.7 (this task records into the golden harness).
- fn-125 stays paused until fn-126 closes.

**Size:** L
**Files:**
- `model/temporal/features/standaloneactivity/**`: `StandaloneActivity.scala`, `record/Record.scala` and `withTaskQueue/WithTaskQueue.scala` replace the per-kind files; `Realization.scala` gets imports and qualified references only;
- `model/check/**` (the lint) and its refusal fixtures;
- `model/project.scala` (if Scala's safe-init checkers are chosen);
- `tools/umpire/internal/golden/config.json`;
- `model/ir/activity*.json`, `model/cases/**`, the generated fixtures.

**Touches:** [model/temporal/features/standaloneactivity/**, model/check/**, model/project.scala, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**]

### Approach
- Record the R11 baseline first: files, lines, hops from state to Query, cross-file references, the `disabled` count and the inverted-guard count, all under `.flow/tmp/fn-126/`.
- Layout per R2:
  - header comment;
  - top-level types;
  - signature (actions stay top-level here; task 3 groups them);
  - module objects in dependency order (`Product`, `Protocol`; the record designs; the compositions) built from today's vocabulary objects, so status-set function symbols do not move;
  - irFile roots last.
- Machine `val`s keep their names inside their module objects (R7).
- The feature section holds only what has no module object yet: the composition with the worker, `protocolCapabilities` (it reads the realization) and the irFile roots. Task 4 moves each to its final home.
- Pins (R6): the file-level pin stays. A module object that holds a monitor, assumption, hole or channel pins its former owner (`…System$package$` in `record/`). Two owners may pin one former owner; add the lifter fixture that proves it.
- Lint (R4 a-d):
  - decide between `-Wsafe-init`/`-Ysafe-init-global` with `-Werror` and an irgen pass for (a) and (b), and record the decision;
  - (c) and (d) read the R2 order and R3 placement;
  - write one refusal fixture per kind, and confirm no passing lifter fixture is refused.
- Regenerate. Record function-symbol moves, `source_root_moves`, path moves and the package renames in the golden configuration. Confirm by the reader's table projection that tables, IDs, type names, answers, lint findings and Contracts are unchanged.

### Investigation targets
**Required:**
- `.plans/QUINT_MODULE_LAYOUT.md` sections 2-6
- `model/temporal/features/standaloneactivity/*.scala`, `admission/*.scala`, `compositions/*.scala`
- `model/irgen/Context.scala:225-335` (Definition IDs, pins, type names)
- `tools/umpire/internal/golden/config.json` (projection, substitutions)
**Optional:**
- `model/README.md:500-560` (today's layout section)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

### Execution constraints
- Stop if R5 needs a delta beyond positions, root strings and function symbols. Report it before task 2.

## Acceptance
- [ ] `standaloneactivity`, `record/` and `withTaskQueue/` each hold one feature file in R2's order, plus `Realization.scala` and tests. No `Model.scala`, `Properties.scala`, `Queries.scala`, `Capabilities.scala` or `IrFiles.scala` remains there.
- [ ] Definition IDs, IR type names, machine/Property/Scenario/Query/Limits/law names, tables, answers, lint findings and Contracts are unchanged. The IR and Case diffs hold only R5's deltas, recorded in the golden configuration.
- [ ] The declaration-order lint runs in `make umpire-check-model` and refuses each R4 (a)-(d) kind at its line, with one fixture each. It passes on all Models and on every passing lifter fixture. The choice of checker for (a)/(b) is recorded.
- [ ] A fixture proves that two owners may pin one former owner.
- [ ] The R11 baseline is recorded under `.flow/tmp/fn-126/`.
- [ ] The model gate, `make lint-model`, the original-baseline check, the Umpire Go tests, the Case/fixture/canary checks and `make lint-code-fast` pass.


## Done summary
# fn-126.1 done summary

Branch `umpire-fn126-1` in wt/lane-a. Commits: 67bb8f5778 (layout), 9fbb7f1a38 (lint),
09222600af (pin fixture), 846c5423a8 (docs), c4836d7131 (merge of `umpire` a766d10cba),
bcadfc979c (Go test positions); review P3s: 7216cafd5c (exemption limited to
model/irgen/testdata), 48824b20f9 (companions and nested objects placed), ac150ce729 (remaining
placement fixtures), 5dac7a5cf1 (docs).

### What landed
- **One feature file per folder (R1, R2, R3, R12).**
  - `standaloneactivity/StandaloneActivity.scala`: header; types; signature (actions, timers, bounds); `Product`, `Protocol`, `ActivityWorker`, `StandaloneActivity`; `object Files`.
  - `record/Record.scala` (was `admission/`): `Admission`, `StaleAdmission`, `HeldAdmission`, `ResponseLoss`.
  - `withTaskQueue/WithTaskQueue.scala` (was `compositions/`): `CurrentRecord`, `StaleRecord`, `CurrentOverQueue`, `StaleOverQueue`, `CurrentOverMatching`, `StaleOverMatching`, `CurrentOverForgetful`, `CurrentOverVolatile`, `CurrentOverLossyMatching`.
  - `Realization.scala` changed imports only. No `Model/Properties/Queries/Capabilities/IrFiles.scala` remains in the three folders.
- **Machine `val`s keep their names (R7):** `Product.activityProduct`, `Admission.currentAdmission`, and so on.
- **Real bugs fixed:** `notFoundCode` is now declared before `productCapabilities`; `jobsCode` in `lifts/Capabilities.scala` moved up the same way.
- **Pins (R6):** the file-level pins stay. `record/`'s `Admission` holds the two monitors and pins `temporal.standaloneactivity.System$package$` beside its file's own pin of the same owner. No other module object holds an ID-bearing declaration.
- **Pin fixture:** `lifts/Captured.scala` `object Watched` pins `fixture.spelled.Spelled$package$` as its file does. `Fixtures.test.scala` asserts both monitors' IDs (`…storedOnce`, `…storedTwice`) under that owner.

### R11 baseline (recorded first)
`.flow/tmp/fn-126/baseline.md`, `baseline-metrics.txt` (`umpire.check.metrics`), `baseline-counts.txt` and `baseline.py` (re-runnable for task 6).
- 33 files and 5016 own lines across the Model folders; 301 cross-file same-package references.
- 74 `disabled` and 45 inverted guards, reproducing the study's numbers.
- State-to-Query hops: `activityProduct` 4 files and 649 lines; `nexusProduct` 4 files and 814 lines.

After task 1 (same script): the standalone activity tree is 3 feature files plus `Realization.scala`, 741 + 448 + 283 lines.

### R5 deltas, and where they are recorded
All are in `.flow/tmp/fn-126/fn126-1/ir-deltas.json`, produced by `.flow/tmp/fn-126/project.py` run against `.flow/tmp/fn-126/before/{ir,cases}`. That projection reports every IR, law sidecar, lint acceptance and Case either "identical" or "equal under the recorded deltas" (OK).
- **Positions:** file paths and lines in the IR and in Cases.
- **Source root strings:** for example `Functional$.all` → `Protocol$.queries$.all`, and `Capabilities$package$.productCapabilities` → `Product$.laws$.productCapabilities`.
- **Function symbols (49):**
  - step functions move into `effects`: `Product$.attemptStart` → `Product$.effects$.attemptStart`, and likewise for `Protocol$`, `Admission$` and `ResponseLoss$`;
  - the end predicate is renamed `Product$.ends` → `Product$.end` and `Admission$.ends` → `Admission$.end`, because inside the object `ends` would shadow the builder word;
  - the package renames `admission.` → `record.` and `compositions.` → `withTaskQueue.`, including the `through` function names.
- **Law sidecar binding text:** two binding strings in `activity.laws.json` print the action's owner, which changes from `…Model$package.control` to `…StandaloneActivity$package.control`. This is the same file-rename symbol change; it is display-only text that messages read.
- **Golden configuration (`tools/umpire/internal/golden/config.json`):**
  - 15 `source_root_moves` retargeted and 2 added (`activityProduct`, `standaloneActivity`);
  - `source_root_additions` renamed;
  - `source_package_moves` and `source_path_moves` gain `admission`→`record` and `compositions`→`withTaskQueue`.
  - Function symbols cannot go in `function_name_substitutions`, because no frozen input declares them (`FunctionsRenamed`). They are recorded in the projection instead, as fn-127.2 did.
- **Unchanged:** Definition IDs, IR type names, machine/Property/Scenario/Query/Limits/law names, tables, answers, `*.lint.json` and the nexus IR. The original-baseline check passes.

### Declaration-order lint (R4 a–d)
`model/irgen/Order.scala` is a TASTy pass that the lifter runs in `Lifter.inspect` over every inspected source, in both lift modes, before lifting. Each finding is reported as `lift: <file>:<line>: …`.
- **(a)** A val read during its owner's initialization before it is declared. Reads in a def, a non-context lambda, a by-name argument or a lazy val are skipped. Context functions such as `machine { … }` are entered. Constant `final val`s, `Inlined` and `Typed` trees are handled, and parent constructor arguments are read first.
- **(b)** Initialization cycles between owners, found as strongly connected components. Each is reported once, at its first read in source order.
- **(c)** R2 order in a feature file:
  - top level: header, types, signature, machine objects, `Files`;
  - inside a machine object: vocabulary, `effects`, monitors, machine, `properties`, `laws`, `queries`;
  - within `queries`: Scenarios before Queries.
- **(d)** R3 placement:
  - each kind (classified by type) sits in its section;
  - nothing of a Model's kinds sits at the top level;
  - `Files` holds only IR files;
  - a Property, capabilities or Scenario sits with its machine's object, and a Query with its Scenario;
  - a sibling file of a feature file declares no Model kinds.
- **Nested objects (review P3):**
  - a Model declaration in a type's companion, in an object of the signature, or in an object nested in a section is refused;
  - so is a nested object that holds one where no section may: a machine object inside a plain object, or a non-section object of a machine object.
- **Feature file:** a source named after its folder (ignoring case) in a package under `features` or `shared`. This excludes the kit's `Capabilities.scala`/`Realize.scala` and the `samestate` fixture.
- **Exemption:** lifter refusal-specimen files `*Rejects.scala` under `model/irgen/testdata/` are exempt. Their deliberate forward reads (`loopFirst`, `aliasFirst`, the `AskedA`/`AskedB` cycle) keep their own refusals.
- **Refusal fixture `model/irgen/testdata/initOrder/`:** one test asserts the exact 15 refusals, and the exempt neighbours (def, lambda, by-name, lazy, unread object, constant) are not refused.
  - (a) `Forward.scala:7`, plus a context-function read at `:17`;
  - (b) `InitOrder.scala:54`, `Switch.laws -> SwitchRealization -> Switch.laws`;
  - (c) `:69` (a section out of order), `:72` (a type after an object), `:97` (a Scenario after a Query);
  - (d) `:83` (a Property outside `properties`), `:86` (a Property over another object's machine), `:24` (a machine in a type's companion), `:30` (a Property at the top level), `:34` (a machine object in a signature object), `:75` (a non-section object holding a Property), `:98` (a Query over another object's Scenario), `:102` (a non-IR-file val in `Files`), and `Forward.scala:23` (a Scenario beside the feature file).
- **Coverage:** it passes on every Model and on every passing lifter fixture, since every fixture lift in the gate runs it. Run over the base revision's Models, it refuses exactly `standaloneactivity/Capabilities.scala:21 notFoundCode` (`lint-on-base.txt`).
- **Checker decision (also in README "The reading order and its lint"):** an irgen pass, not Scala's checkers. `-Wsafe-init` checks classes only (3.9). `-Ysafe-init-global` crashes on ScalaPB gRPC `METHOD_*` reads through 3.10.0-RC3. `model/project.scala` is unchanged.

### Gates (logs under `.flow/tmp/fn-126/fn126-1/`)
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`: `gen2.log`, then `gen3.log` after the formatting change.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` after the merge: `model-gate.log`, ok. Rerun after the review P3s: `model-gate2.log`, ok. This includes the lifter fixture tests and the munit tests.
- `make lint-model`: `lint-model.log`, 0; after the P3s, `lint-model2.log`, 0. The scalafix JDK reflection traces in it are pre-existing noise.
- Original baseline, `-run 'OriginalBaseline|Migration'`: `original-baseline.log`. `tools/umpire/model` was OOM-killed at `-p 2` and passed alone at `-p 1` (`original-baseline-model.log`).
- Full Go suite, `-json -p 2`: `go-suite.json`, 339 s wall. The only failures were 2 positions-only expectations (`lint` api_test `Realization.scala:61→63`; `model` hints fixture lines −4). Both are fixed in bcadfc979c, and `lint` and `model` were rerun in full at `-p 1` and pass (`go-rerun.json`). Slowest tests: `TestOriginalBaselineCases` 50 s, `TestMigrationProjectionPreservesSemantics` 48 s, `TestOriginalBaselineModel` 46 s.
- `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case`: 0 each (logs by target name). Regenerating fixtures and the canary Case changed nothing.
- `make lint-code-fast`: 0 issues (`lint-code-fast.log`).

- After the P3s, the lifted IR and the lifter's expected fixture outputs are unchanged (Models relifted byte-identical). So the original-baseline check and the `lint`/`model` Go packages were not rerun.

### Decisions
1. **Module object names.** Module objects keep today's vocabulary names (`Product`, `Protocol`, `Admission`, `ResponseLoss`), so status sets keep their symbols. Machines with no vocabulary object take R1's pre-R18 names (`StaleAdmission`, `HeldAdmission`, `ActivityWorker`, `StandaloneActivity`, `CurrentOverQueue`, …). Renaming to `ActivityProduct` and the rest is left to task 4, where the symbols move anyway.
2. **Section objects now.** Step functions, Properties, capabilities and Scenarios/Queries sit in plain nested `effects`/`properties`/`laws`/`queries` objects, without the `Section` marker. Inside an object, a step function or Property named like an action or phase would shadow it, because Scala 3 lets a member silently win over a wildcard import (verified). Monitors and the machine `val` stay direct members, so `Admission`'s pin covers the monitors.
3. **No transitional feature section.** The composition got its object `StandaloneActivity` now. `protocolCapabilities` sits in `Protocol.laws`, which is lazy, so the realization can read `Protocol` without a cycle. This is R3's final placement rather than the task text's interim "feature section". `Files` holds only the IR files.
4. **`ends` → `end`.** The machine objects' end predicate is renamed (`Product.end`, `Admission.end`) so it does not shadow the `ends` builder. This is a function-symbol delta.
5. **The lint lives in irgen, with (c)/(d) beside (a)/(b).** It runs in both lift modes. The lifter's `*Rejects.scala` files are exempt rather than moved, because those specimens must still reach their own refusals.
6. **Nested objects are refused, not ranked.** A Model declaration in a companion or a plain object is refused rather than ranked, since R2 gives such objects no place.

### For the owner
- The law sidecar binding text (`…Model$package.control.apply(…)` → `…StandaloneActivity$package…`) is a symbol string outside the IR proper. I counted it under R5's symbol deltas rather than as a stop condition.
- `model/ir/activity.lint.json` acceptance prose still says "(Model.scala, workerStop is disabled)". It is unchanged because lint findings are frozen; task 2's R10 sweep may want it.
- Left for later tasks, per review: feature-file naming enforcement (task 2 / R10); `rules`/`monitors`/`syncs` ranking and the guard-lambda note (task 4).
- The README layout section and the `UMPIRE_MODULES.md` Models and Capabilities rows now describe the activity's feature file and say the Nexus and `shared/` Models are still per-kind until task 2.
- Merging lane-b is safe, since it does not touch Model files. The merge of `umpire` here brought in docs and flow only.

Subagents used: 0.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP, no P1/P2.
- The reviewer checked `before/` byte for byte against the base and reran `project.py` (OK, 49 renames).
- It compared every declaration-name set raw against base, and structurally diffed all Cases: only `source.path`, `waitHints.line` and one `Realization.scala` line change.
- It judged the law-sidecar binding strings display-only (not in any key or fingerprint) and `ends` → `end` within R5.
- P3s applied in 7216cafd5c..5dac7a5cf1. P3s carried to later tasks: `.flow/tmp/fn-126/carry-forward.md` (feature-file naming and `activity.lint.json` prose → task 2; section ranking for `rules`/`monitors`/`syncs` and guard lambdas → task 4).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 67bb8f5778, 9fbb7f1a38, 09222600af, 846c5423a8, c4836d7131, bcadfc979c, 7216cafd5c, 48824b20f9, ac150ce729, 5dac7a5cf1
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, make lint-model, go test -count=1 -tags test_dep -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (model rerun -p 1), go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... (lint, model rerun -p 1 after fixes), make umpire-check-cases, make umpire-check-fixtures, make canary-check-case, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast, python3 .flow/tmp/fn-126/project.py .flow/tmp/fn-126/before/ir model/ir .flow/tmp/fn-126/before/cases model/cases
- PRs: