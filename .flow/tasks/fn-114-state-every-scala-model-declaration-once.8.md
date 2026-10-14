---
satisfies: [R1, R8, R9, R11]
---
# fn-114-state-every-scala-model-declaration-once.8 Close fn-114 with literal and line counts and full gates

## Description
Count, classify and gate once at the end, and update the module map for the new root ownership.

**Size:** S
**Files:** `.flow/tmp/fn114-8/**` evidence; small literal fixes in Models; `.plans/UMPIRE_MODULES.md` ("Root lists keep their present meaning until fn-114"), `model/README.md`.
**Touches:** [model/temporal/features/nexuscaller/**, model/temporal/shared/worker/**, model/README.md, .plans/UMPIRE_MODULES.md, .flow/tmp/fn114-8/**]

### Approach
- Re-run task 1's literal and line counting command; classify each remaining literal into fn-112 R18's three kinds; list any other literal with its line and reason (R11). Remove literals that turn out to be avoidable.
- Run the full model gate, `make lint-model`, the Umpire Go tests and `make lint-code-fast` once, with `-json` timing per the MILESTONES verification instructions; record commands, results and log paths.
- Report line counts of Models and the gate program before (task 1) and after (R9), and the lift-stage times from task 1 (R8).

### Quick commands
```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -json -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... > .flow/tmp/fn114-8/go-test.json
```

### Execution constraints
- fn-120.2 (refuse unnamed branches), fn-118's behavior phase and fn-119 wait for this spec to close; say so in the done summary so the conductor can release them.
## Acceptance
- [ ] Every remaining Model string literal is classified into the three allowed kinds or listed with line and reason.
- [ ] Model and gate-program line counts before/after are in the done summary, with the lift-stage times.
- [ ] Model gate, lint-model, Umpire Go tests and lint-code-fast pass in full; R1 goldens pass.
- [ ] Module map and README describe Scala-owned roots.
## Done summary
Closed fn-114 with literal and line counts and the full gates. Commits: f9a1c9fd6b (task queue Scenario names), 1ae70ab03a (module map, review P3).

**Literals (R11)**
- The counter is fn-112 R18's, unchanged since fn-114.1: `scala-cli run model/check --main-class umpire.check.metrics`.
- Every remaining literal was read in context, in three parallel read-only Opus passes. Each is classified in `.flow/tmp/fn114-8/literals.md`.
- **Removed as avoidable**, in `shared/taskqueue/Queries.scala`:
  - five Scenario names on local vals spelled differently;
  - `any`, which repeated its own val;
  - `"lossyMatchingQueue.storageLoss"`, which now reads its machine's name by value.
  - Names, answers and Cases are unchanged. Only six Scenario positions in activity-system.json moved.
- **Listed outside the three kinds, each with lines and reason:**
  - the computed Query names (45, which the spec keeps);
  - the `DefinitionScope` owner pins;
  - evidence equal to the fact name, for facts with fields (the IR generator refuses a default);
  - `Entity` key, `refer` and `Observation.read` strings, which are never lifted;
  - realization protocol and payload values;
  - the kit's id-format fragments;
  - one composed-outcome key and one law parameter name (both fn-122's).

**Lines and literals, fn-114.1 base → now** (`metrics-before-fn114-1.txt`, `metrics-after.txt`)
- nexuscaller: 2,720/321 → 2,450/115.
- standaloneactivity: 1,567/54 → 1,748/61. The growth is fn-122's three `Capabilities.scala` (143/4) and fn-114.1's `IrFiles.scala` (47/3).
- taskqueue: 417/23 → 412/17.
- worker: 81/8 → 85/5.
- realize: 291/26 → 584/29. The growth is `Realize.scala`, which fn-114.12 moved out of model/umpire, and `Behavior.scala` from 394883c2c0.
- Together: 5,076/432 → 5,279/227.
- Added since, not in the baseline: nexusoperation 342/12, capabilities 215/36.
- Gate program: model/gate 2,654 → model/check 2,685. `Roots.scala` (-104) is gone. Additions: `Metrics.scala` (+15, folded in), `CapabilityVocabulary.test` (+31, fn-122) and Gate tests.

**Lift stage (R8)**, from fn-114.1's measurement:
- before, six JVMs: 10.9/7.0/11.1 s;
- per file: 7.4-7.8 s;
- single run: 3.4-3.7 s.
- The closing gate's single run of seven IR files took 14 s under shared load. That figure is not an isolated measurement.

**Gates** (`gates.txt`, HEAD f9a1c9fd6b, all under the shared lock). All exit 0:
- model gate;
- OriginalBaseline;
- migration goldens;
- full Go suite with `-json`: 6058 pass, 18 skip, 350 s wall. The slowest tests are in `gosuite-slowest.txt`;
- lint-model. Its first run died in scalafmt-native with SIGSEGV; the rerun passed. Its scalafix `NoSuchFieldException` noise is the same as in fn-114.9;
- lint-code-fast, umpire-check-lint, check-cases, check-fixtures, canary-check-case.

The commit after the gates touches docs only. The layout guard was rerun after it.

**Docs.** model/README.md already describes `irFile`/`IrFiles.scala` (fn-114.1, fn-114.9). The module map now states that each Model folder's `IrFiles.scala` owns its IR roots.

**Decisions**
- `Entity("worker")` and `Entity("taskQueue")` stay declared names. Capturing them would mean `shared.worker.worker`, or a clash with the kit's `taskQueue` role.
- The repeated per-machine Progress and Property names stay. They are separate declarations, and folding them moves positions.
- The two standaloneactivity Query prefixes are fn-112's and stay.
- `SourceMetrics` still names retired forms (fn-114.7's FYI). The counter must classify the fn-114.1 baseline sources the same way.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3, fixed in 1ae70ab03a: the module map listed roots instead of the rule.

**Released.** fn-120.2 (refuse unnamed branches), fn-118's behavior phase and fn-119 waited for this spec to close and can now start.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f9a1c9fd6b, 1ae70ab03a
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -run 'Migration|Golden' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0; 6058 pass, 18 skip), make lint-model (exit 0 on rerun after scalafmt-native SIGSEGV), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), make umpire-check-lint (exit 0), make umpire-check-cases (exit 0), make umpire-check-fixtures (exit 0), make canary-check-case (exit 0), scala-cli run model/check --main-class umpire.check.metrics (exit 0)
- PRs: