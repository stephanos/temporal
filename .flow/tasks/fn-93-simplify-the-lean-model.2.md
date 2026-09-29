---
satisfies: [R2]
---
# fn-93-simplify-the-lean-model.2 One axiom checker replaces every #print axioms pin (E2)

## Description
Lane E2. 81 `#print axioms` pins exist; 30 have no `#guard_msgs`, so they assert nothing. Add one checker command and convert every pin, including pins in modules lane B later deletes (a declined decision keeps them).

**Size:** M
**Files:** `model/Umpire/Shared/Test.lean` (or a new `model/Umpire/Shared/AxiomCheck.lean` it re-exports), `model/Umpire/Shared/Tests/AxiomCheck.lean` (new self-tests), the ~23 test files holding pins (list below), `model/UmpireTests.lean` (wire self-tests), `.plans/LEAN_GUIDELINES.md` trust-audit paragraph, `model/README.md` sentence on `#print axioms`
**Touches:** [model/Umpire/Shared/**, model/Umpire/**/Tests/**, model/Umpire/**/*Tests.lean, model/Temporal/**/Tests/**, model/Temporal/**/*Tests.lean, model/Temporal/Feature/Nexus/Success/Tests.lean, model/UmpireTests.lean, .plans/LEAN_GUIDELINES.md, model/README.md]
**Depends on other specs:** fn-88.5 and fn-92.3 add pins under `Search/Tests/*` and `Command/ComposeProofs`; convert what is there at start.

### Approach
- Command shape per spec §API Contracts (`assert_axioms [decls] allowing [axioms]`). Build on `Lean.collectAxioms` (see its use in the `machine` command's sorry guard at `model/Umpire/Command/Syntax.lean:2418,2462`); resolve names with `realizeGlobalConstNoOverloadWithInfo` so a missing name is a located error.
- Fail on `sorryAx`, on any axiom outside the allowlist, and on a missing declaration. Allowlist names are exact (`propext`, `Classical.choice`, `Quot.sound`, and `Lean.ofReduceBool`/`Lean.trustCompiler` only where `native_decide` is already used).
- Self-tests under `#guard_msgs` (use `drop warning` or pin the "declaration uses 'sorry'" line) for: seeded sorry, extra axiom, missing name.
- Convert each pin to one entry with the exact axiom set it prints today. The 8 duplicated targets (`nexusProduct`, `Execution.closed_property`, `Projection.Correlated.Monitor.admitMany_append`, `evaluateProperty`, `evaluateProperty_agrees`, `evaluatePropertyPredicate_agrees`, `Lowered.window_property`, `Lowered.evidence_validation`) keep one entry each; the per-machine "no sorryAx" guarantee stays one entry per machine.
- The checker module must stay inside `testSupportNamespaces` (`model/ModelLint/ImportGraph.lean:170`); importing `Lean.Elab` there is allowed for test support only.

### Investigation targets
**Required:**
- `model/Umpire/Shared/Test.lean` — current test-support home (23 lines)
- `model/Umpire/Command/Syntax.lean:2405-2470` — existing `collectAxioms` guard
- Pin sites with the most bare pins: `model/Umpire/Evidence/Tests/Compilation.lean:568-571`, `model/Umpire/Model/Tests/FiniteMachine.lean`, `model/Temporal/Feature/Nexus/Success/Tests.lean`, `model/Umpire/Property/Tests/Endpoints.lean`, `model/Umpire/Scenario/Tests/Authoring.lean`
**Optional:**
- `.plans/LEAN_GUIDELINES.md:164-173` — trust-audit prose to update

### Quick commands
```sh
grep -rn '#print axioms' model --include='*.lean' | grep -v .lake | wc -l   # 0 after, except comments
cd model && lake build UmpireTests TemporalModelTests
```

## Acceptance
- [ ] No `#print axioms` pin remains (comments aside); every former pin is one checker entry with its exact prior axiom set
- [ ] Self-tests pin the three failure modes; seeding `sorry` into a checked declaration makes `lake build` fail (receipt shows it)
- [ ] No declaration's axiom inventory widened; LEAN_GUIDELINES trust-audit text and README sentence updated
- [ ] `lake build` of all roots and `LEAN_NUM_THREADS=1 make lint-model` green


## Done summary
Replaced every `#print axioms` pin (92 real pins across 30 test files, 30 of them formerly
unguarded and asserting nothing) with `assert_axioms [decls] allowing [axioms]`
(`Umpire.Shared.Test.AxiomCheck`), a command built on `Lean.collectAxioms` that fails elaboration
itself -- so `lake build` fails -- when a checked declaration depends on `sorryAx` (rejected
unconditionally, never admissible via `allowing`), on any axiom outside the allowed set, or names a
declaration that does not exist. Self-tests (`Umpire.Shared.Tests.AxiomCheck`, wired into
`UmpireTests.lean`) pin all of that under `#guard_msgs`, including that `allowing [sorryAx, ...]`
still fails and that a genuinely axiom-free declaration passes `allowing []`.

The Codex fan-out review (round 1) found that the initial conversion widened several
declarations' checked axiom ceilings by grouping them with same-file siblings that had a wider
prior inventory, that the `allowing` grammar couldn't express an empty (axiom-free) inventory, and
that `lampedRefused` (the composition-refusal fixture, which deliberately stays declared carrying
`sorryAx`) was exempted from the `sorryAx` rejection via an `allowing` entry rather than proven
rejected. All three were fixed: mixed groups were split so every entry's `allowing` list is exactly
the union of its members' identical prior inventory, `allowing` now accepts `ident,*` (so `[]`
parses), and `lampedRefused` is now a `#guard_msgs`-pinned expectation that the checker rejects it.
Round 2 (fresh fan-out against the fixed commit) shipped with zero findings across all three axes.

9 declarations were pinned from two files each (the spec's 8 plus `evaluatePropertyEndpoint_closed`,
found during conversion); each keeps one `assert_axioms` entry, with a comment at the other site
naming where. Two turned out not to be true duplicates despite sharing a bare-pin spelling
(`Temporal.Feature.Nexus.{Caller,Tests.Machines}` each declare their own `nexusProduct`), so both
kept independent entries.

Updated `.plans/LEAN_GUIDELINES.md`'s trust-audit paragraph and `model/README.md`'s `#print axioms`
sentence to describe the checker.

stage: impl-review - ran fan-out (correctness/contracts/integration; round 1 NEEDS_WORK on all 3
axes -- fixed; round 2 SHIP on all 3 axes, 0 findings) -> finalize SHIP

stage: plan-sync - skipped(config: planSync.enabled != true)

Verification: `grep -rn '#print axioms' model --include='*.lean' | grep -v .lake` finds only the
two prose mentions (no executable pins). `lake build UmpireTests TemporalModelTests` is green
(786 jobs), run both before and after the review-fix commit. The checker's own self-tests are
green. `lake exe umpire-lint-tests` and `lake exe umpire-lint` (the custom import-graph and
entity-uniqueness policy checks `lint-model` runs) are green against the fixed commit.
`umpire-check-goldens`, `canary-check-case` and `umpire-check-case-runtime-conformance` are green
(byte identity, per the coordinator's narrowed gate set for this task's surface).

One gate is INCONCLUSIVE, not green: `LEAN_NUM_THREADS=1 make lint-model-builtin
LINT_MODEL_MODULES=<32 touched modules>` (the isolated-copy Lean *builtin* linter, a sub-step of
`make lint-model` distinct from the custom checks above) was attempted 5 times. Every attempt's own
743/849-job isolated rebuild completed green, but the sandbox's disk quota was exhausted
immediately afterward (SIGKILL, disk at 98-99% full each time) before the lint analysis pass itself
could run -- never a code-related failure, and reproducible at the exact same point regardless of
which commit was checked out. `.build/lint-model` (the isolated copy) is cleaned up after each
attempt and is not left behind. This is an environment disk-capacity limit on this host, not
something fixable within this task's Touches; fn-93.43 runs the spec's full lint and regression
once, and would hit the same wall if the host's disk quota is unchanged by then.
## Evidence
- Commits: ee423e999e784d490c4eaa50eb0493706b2b9130, 1764991c3dd09e74b2b961b406825453f31b59ee
- Tests: grep -rn '#print axioms' model --include='*.lean' | grep -v .lake | wc -l -> 0 real pins (2 comment mentions only), cd model && lake build UmpireTests TemporalModelTests (786 jobs, green, run twice: before and after the review-fix commit), cd model && lake build Umpire.Shared.Tests.AxiomCheck (self-tests: seeded sorry, sorryAx-in-allowlist, extra axiom, missing declaration, empty-allowlist positive -- all green), make umpire-check-goldens (green, no diff), make canary-check-case (green), make umpire-check-case-runtime-conformance (green), cd model && lake build umpire-lint-tests umpire-lint && lake exe umpire-lint-tests (green: module index + import-graph synthetic tests passed), cd model && lake exe umpire-lint (green: import-graph, Batteries per-module, feature-entity-uniqueness linting passed), INCONCLUSIVE: LEAN_NUM_THREADS=1 make lint-model-builtin LINT_MODEL_MODULES=<32 touched modules> -- attempted 5 times; the isolated rsync-mirrored full rebuild (743/849 jobs) always completed green, but the sandbox's disk quota was exhausted immediately after (SIGKILL, disk at 98-99% each time) before the builtin-lint analysis pass itself could run; this is the only Quick-command/AC gate not independently confirmed, and it is an environment disk-capacity limit, not a code defect -- the equivalent non-isolated umpire-lint / umpire-lint-tests custom checks (which are the load-bearing import-graph and entity-uniqueness policy, distinct from lake's builtin linters) passed cleanly against the same fixed commit
- PRs: