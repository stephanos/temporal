---
satisfies: [R3, R8]
---
# fn-162-queries-name-their-realization-each.2 Realizations name the Queries they run: framework, lifter, docs (Scala part; Go half in fn-162.4)

## Description
Let a realization name the Queries it runs (an unnamed find Query is model-only), and detect duplicate Property declarations. This is the Scala mechanism (framework, lifter, docs, mini-scoped fixtures); the activity is migrated in fn-162.3. fn-162.1 traced today's behaviour (spec, Resolved via Codebase): a Case is already assessed against its Query's one Property plus the scenario machine's monitors, so assessment itself doesn't change.

**Scope (2026-10-12, Scala DSL batch rules):** re-scoped to its Scala part. The IR schema field (`Realization.queries`, field 21) is already committed in `realization.proto`, but the Go bindings, `lower.Realizable`, `conformance.realizationOf` and model lint cannot read it without owner-held regeneration and Go work; that half moved, unrelaxed, to the deferred fn-162.4.

**Size:** M
**Files:** `model/temporal/realize/Objects.scala` (`Realizes`/`DerivesFrom` `runs`), the lifter (`model/irgen/Realizations.scala` runs, `model/irgen/Lift.scala` duplicate Properties), `model/irgen/test/Fixtures.test.scala`, `model/README.md`, `model/framework/SEMANTICS.md`
**Touches:** [model/framework/*.scala, model/temporal/realize/*.scala, model/irgen/*.scala, model/irgen/test/**, model/README.md, model/framework/SEMANTICS.md]

### Approach
- **DSL (owner, 2026-10-10):** the realization names its Queries, e.g. `Realizes(ActivityProtocol, runs = Seq(ActivityProtocol.queries.protocolCompleted, …))`. The Query syntax doesn't change, which respects the one-way layer dependency (realization → system → product). A find Query no realization names is model-only and isn't lowered.
- **Lifter:** writes each realization's find Queries into `Realization.queries`. It refuses a realization naming another machine's Query, and a Query named by two realizations, naming both.
- **Duplicates:** the lifter refuses two author Properties of one machine with the same selector and predicate (up to names and positions), naming both locations. A refusal, not a lint finding: R3 says each Property is declared once, and a refusal keeps a second declaration from ever reaching the IR or a Case; capability-generated Properties keep their own identities and are not compared.
- **Docs:** SEMANTICS (how a Query's realization is chosen; each Property declared once) and README (writing Queries and realizations without claim-container copies).
- **Fixtures (mini-scoped):** a realization naming its Queries (MiniActivityRun), an unnamed (model-only) find Query, two realizations on one machine each bound by its own Queries, a realization naming another machine's Query (refused), a Query named by two realizations (refused), and a duplicate predicate (refused, both named).

### Quick commands
```sh
mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.MiniRealizationFixtures
```
## Acceptance
- [ ] A realization names the Queries it runs (unnamed find Queries are model-only); the lifter writes the binding into the IR's `Realization.queries`, so one machine may have several realizations.
- [ ] A realization naming another machine's Query, or a Query named by two realizations, is refused; an unnamed find Query stays unbound (model-only); duplicate predicates on one machine are refused with both locations.
- [ ] README and SEMANTICS document the binding and the declared-once rule; mini-scoped fixtures cover each case. The Go consumers (`lower.Realizable`, `conformance.realizationOf`, model lint), regeneration and Go tests are fn-162.4's (deferred, held).
## Done summary
Scala half of the realization binding and the declared-once rule, re-scoped (chore commit 506f373b59) under the batch scope rules; the held Go/schema half moved unrelaxed to the new deferred fn-162.4 (precedent fn-140.7/.8/.9).

Starting state: the framework and lifter mechanism was already on `umpire` from Batch 1d (42c9865f59 `Realizes(machine, runs = Seq(...))`/`DerivesFrom(..., runs)` and `Realization.queries`, proto field 21; 48e6288e50 duplicate-Property refusal), and mini's `MiniActivityRun` already names `protocolCompleted` and `retryCompleted`. What was missing on the Scala side was mini-scoped fixture coverage for most cases and docs that still described the old rule. The IR cannot carry the binding to Go without a schema/Go change: field 21 is in `realization.proto`, but the Go bindings are not regenerated (`make protoc` is held) and `lower.Realizable`/`conformance.realizationOf` still pick the machine's single realization. That is why the Go half was split off.

Requirement evidence:
- R8 binding: `MiniRealizationFixtures` (8/8, suite-6.log). The existing tests pin MiniActivityRun's `queries == [protocolCompleted, retryCompleted]` and refuse a foreign-machine Query ("a Query of activityProduct"). New tests: two `DerivesFrom(MiniActivityRun, ActivityProtocol, runs = ...)` realizations on one machine each carry only their own Query; an unnamed find (`retryCompleted`) stays in the IR, bound by no realization (model-only); a Query two realizations name is refused, naming both (`protocolCompleted is named by the realizations againRun and completedRun`).
- R3 duplicate: a refusal rather than a lint finding, because R3 says each Property is declared once and a refusal keeps a second declaration out of the IR and the Cases. Capability-generated Properties are not compared. The new mini-scoped test confirms that two same-`when`, same-predicate Properties of one machine (different parameter names) are refused with "completesAgain states the promise completes states at …, of twice". The pre-existing `Fixtures` "strict duplicate refusals" test was not rerun (heavy suite).
- Docs: SEMANTICS §19 (a find runs through the realization naming it in `runs`, an unnamed find is model-only, several realizations per machine, refusals, assessment unchanged) and §8 (each Property declared once, duplicate refusal). README (declare Queries on the machine or facet, list them in `runs`, no claim-container copies).

Verification: `timeout 2400 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.MiniRealizationFixtures`, exit 0, 8/8; `git diff --cached --check`.

Failures retained: suite-1..5.log. These were my own fixture iterations: a root prefix bug, refusal name order, an unqualified cross-level import (R5) refused by the lifter, a Property unasked by any Query not being lifted, and the root qualification. None were product defects.

Open obligations: fn-162.4 (deferred, held): regenerate the Go API, have `lower.Realizable`, `conformance.realizationOf` and model lint read `Realization.queries`, and add the Go tests.

Deferred gates: Go/proto/regeneration, `make umpire-check-model`/`-lint`, the full `umpire.irgen.Fixtures` suite, broad validation and independent review at batch end (REVIEW_MODE=none). No source change, so no mini step-table equivalence was needed for this task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 506f373b59, 005fb76541
- Tests: timeout 2400 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.MiniRealizationFixtures (8/8, exit 0), git diff --cached --check
- PRs: