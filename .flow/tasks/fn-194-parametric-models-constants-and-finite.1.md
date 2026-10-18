---
satisfies: [R1, R2, R3]
---
# fn-194-parametric-models-constants-and-finite.1 DSL completion: prototype a parametric Bank and record the verdict

## Description
**DSL completion phase (owner, 2026-10-17).** Before changing the DSL, try the class form on Bank in a prototype: `class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]` with `Addr` declared as an enum in the test file and the instance built there. Prove or refute each Architecture bullet of the spec on it: var initialisers and `oneOf` starts reading the parameters, the `VarSchema` given and the var runtime being per instance (two instances in one process, no shared state), naming an instance by its holder, and what the structure lint needs to admit the class form. Record the verdict and the chosen spelling for each bullet in `model/design/parametric-models.md`, keeping the prototype files beside a README and run script as the design-verdicts convention asks.

**Size:** M

**Files:** `model/design/parametric-models.md`, a prototype under `model/design/parametric-prototype/` (Bank copy, its test, README, run.sh).

### Verification
- The prototype's test suite passes on the native runtime with two Bank instances of different `supply` giving different reachable graphs.
- The note states a verdict per Architecture bullet and names the alternative where one does not hold.
- Quick: `cd model/framework && mise exec -- scala-cli test .` scoped to the prototype suite (the README Native checks recipe is the full gate).

**Touches:** `model/design/parametric-models.md`, `model/design/parametric-prototype/**`.

**Native checkpoint boundary (conductor, 2026-10-18).** Run against committed fn-190.16 (`d7b02a26b7`, wording `270d8cd03b`, closure `653fa13563`). This task probes the existing DSL before changing it; production framework, sample, checker and lint edits belong to .2/.3. Use the product `framework.checker` and current plain Machine form, not retired RecordMachine/Rules/expect/query/Limits or a replacement explorer. Keep prototype declarations/tests separate so the framework gate does not accidentally discover them. Use the native strict flags and a portable run script. Test two distinct instances in one process with genuinely different supply-dependent reachable graphs, parameter-dependent starts and no shared vars/schema state. Test current naming honestly: holder naming is a requirement to prove or refute, not something to simulate with overridden names and report as automatic support. If an Architecture bullet is unsupported, record observed refusals/behavior and choose a concrete implementation alternative for .2 rather than weakening the spec or changing production here. Realizations may be probed with current instance types. IR/export/Go/Temporal and historical fixtures remain held. The design verdict and prototype paths explicitly named by this task are in scope despite the general framework-only boundary. `make model-check-changed` routes to held Temporal work: record that limitation and use authorized native/prototype commands without claiming Make passed.
## Acceptance
- [ ] Verdict note in `model/design` with one verdict per Architecture bullet
- [ ] Prototype kept with README and run script; two instances checked in one process without shared state
- [ ] Instance naming by holder demonstrated or an alternative named

## Done summary
Implemented the fn-194.1 production-free Bank prototype and architecture verdict at model/design/parametric-models.md. The public checker proves independent concrete-schema instances and exact 8/27 reached-state graphs; retained compiler snapshots and expected-refusal tests distinguish that indexed runtime fallback from the unsupported intended direct generic API. The verdict records all eight Architecture bullets and concrete .2 alternatives for schema/before typing, instance claim acquisition, naming and lint, without claiming those gaps fixed.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; owner MILESTONES phase-batch review override)

baseline: green — existing focused native Bank suite, 10 tests, exit 0; baseline.log. No baseline handoff used.
Verification: final prototype suite 6 tests, exit 0; fresh OwnedVocabulary/FiniteOnly compiler refusals match schema diagnostics (each compiler exit 1, refusal runner exit 0); native framework/checker+samples 600 tests, exit 0; focused lint RuleSuite 31 tests, exit 0. Scala formatting repeated with identical SHA256 bytes; bash -n and git diff --check passed. Gate classifier returned FULL, and native unittest receipt was written at source checkpoint 9648caeb9d. No gate was skipped.

Tests cover parameter-dependent initial balances/starts, separate descriptors/state owners/runtime, exact graph budget boundaries, missing automatic holder names and unheld acceptance, empty oneOf refusal, currently absent constructor restrictions, before path typing refusal, and exact Realization instance binding. Product verifyAll refusal for class-owned claim sections is asserted as a current limitation, not a verified Model promise. Direct Addr-bearing state, instance-nested vocabulary and the requested Finite-only schema spelling remain refuted; the address-index fallback must not migrate into production .2/.3.

Failed experiments: attempt1-compiler.log (before strict equality); attempt2-compiler.log (owned vocabulary/schema/input comparison); attempt3-compiler.log (generic before projection); attempt4-compiler.log (Finite-only generic schema); attempt5/6-compiler.log (leaf evidence ambiguity/schema); attempt7-compiler.log (full-record evidence/schema); attempt8-compiler.log (eager nested actor safe-init); attempt9-test.log (class claim acquisition and an incomplete compile control). Failed snippets were corrected or retained explicitly as refusal fixtures, not left as speculative implementation. Final native/prototype/lint logs and final-commands.md are under .flow/tmp/fn1941/.

Held boundary: make model-check-changed routes to held Temporal work and was not claimed passing. No framework, samples, lint, checker, Go, IR/schema, exporter, generated corpus, Temporal or MILESTONES source was changed. No mapped user feature route changed. Spec remains open. Tracker sync: n/a (bridge inactive). Review remains deferred; no SHIP verdict is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9648caeb9d9a74de4ca3de4be31d2b82ca7fc223
- Tests: baseline: green — focused native Bank 10 tests exit 0, bash model/design/parametric-prototype/run.sh (6 tests, exit 0), bash model/design/parametric-prototype/run.sh refusals (expected compiler exits 1/1; runner exit 0), mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests (600 tests, exit 0), UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/framework-native mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite (31 tests, exit 0), Scala fmt and unchanged-byte repeat; bash -n; git diff --check (exit 0)
- PRs: