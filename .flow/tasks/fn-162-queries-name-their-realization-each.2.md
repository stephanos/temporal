---
satisfies: [R3, R8]
---
# fn-162-queries-name-their-realization-each.2 Realizations name the Queries they run: framework, lifter, docs (Scala part; Go half in fn-162.4)

## Description
Let a realization name the Queries it runs (an unnamed find Query is model-only), and detect duplicate Property declarations. This is the Scala mechanism (framework, lifter, docs, mini-scoped fixtures); the activity is migrated in fn-162.3. fn-162.1 traced today's behaviour (spec, Resolved via Codebase): a Case is already assessed against its Query's one Property plus the scenario machine's monitors, so assessment itself doesn't change.

**Scope (2026-10-12, Scala DSL batch rules):** re-scoped to its Scala part. The IR schema field (`Realization.queries`, field 21) is already committed in `realization.proto`, but the Go bindings, `lower.Realizable`, `conformance.realizationOf` and model lint cannot read it without owner-held regeneration and Go work; that half moved, unrelaxed, to the deferred fn-162.4.

**Size:** M
**Files:** `model/temporal/realize/Objects.scala` (`Realizes`/`DerivesFrom` `runs`), the lifter (`model/irgen/Realizations.scala` runs, `model/irgen/Lift.scala` duplicate Properties), `model/irgen/test/Fixtures.test.scala`, `model/README.md`, `model/SEMANTICS.md`
**Touches:** [model/framework/*.scala, model/temporal/realize/*.scala, model/irgen/*.scala, model/irgen/test/**, model/README.md, model/SEMANTICS.md]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
