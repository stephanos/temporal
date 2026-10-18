# Parametric Models: Bank prototype verdict

The intended class form needs framework work before sample migration. Ordinary constructor
parameters, instance-local vars and parameter-dependent starts already run on the public checker;
direct generic state schemas, instance-owned vocabulary, `before` typing, class claim sections,
holder naming and class lint admission do not yet satisfy the spec.

This is the fn-194.1 probe against the committed fn-190.16 native checkpoint, before any DSL
change. [The prototype](parametric-prototype/README.md) has a portable pinned runner, a Bank copy,
one six-test suite and retained rejected source forms. Only design files changed. The product
`framework.checker.Checker` supplies all exploration; the probe implements no explorer.

## Architecture verdicts

| Spec bullet | Observed verdict and evidence | Chosen spelling or next implementation |
| --- | --- | --- |
| A Model is a class | **Partial.** `class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]` runs with concrete native state fields. Direct `Balances[Addr]` fails schema derivation; full intended form is **refuted today**. The existing object Bank's 10 native tests pass unchanged. | Keep the requested class constructor and parameterless object forms. In .2 fix class lexical type/evidence acquisition; do not ship the prototype's index representation as the sample API. |
| Two kinds of parameter | **Partial/refuted.** The enum `Addr` is declared in the test and its selected `Finite` reaches each constructor. `holder` and `supply` affect initialization and domains. `Finite` alone does not let the current schema derive generic map keys/input records. Instance-nested `Denom` and records additionally cause projected-type macro errors. Ordinary constructors currently admit an unbounded type parameter and a `Path` value parameter (compile controls in the suite). | Keep `[Addr: Finite]` and ordinary immutable value parameters. Keep Bank-owned `Denom`, `Balances[Addr]` and `Send[Addr]` in `Bank`'s companion, matching the spec's API example; the concrete world stays in the test. Add located constructor validation in .2. Preserve existing collection admissibility when acquiring native schema evidence. |
| Instantiation | **Proved for the explicit representation fallback; full desired API remains partial.** Two instances coexist with independent var descriptors, state owners, balances, `oneOf` starts and reachable graphs. The public checker completes at exactly 8/27 states for supplies 1/2 and cuts at 7/26. A temporary write in one instance leaves the other unchanged and is restored afterwards. **Refuted for claim answers:** `Checker.verifyAll(graph)` refuses `Bank$Model$invariants$` as a claim section of an instance. | Continue using `Bank.Model(holder = Addr.king, supply = 2)`. Extend native claim-section acquisition to the actual Machine instance, using its enclosing-instance reference rather than requiring a singleton module. Keep state/schema/runtime acquisition on each instance. |
| Naming | **Refuted.** Both `val smallBank` and `val largeBank` return `name == "model"`; public exploration reports `subject == "model"`. Passing an unheld constructor expression also succeeds as `model`. No names are overridden in the probe. | Capture a stable val at the public check call with an inline macro, carry that holder name into the report, and refuse constructor expressions with no stable holder, naming `Model` in the error. The later ModelSuite can provide its own suite-owned binding. Preserve object naming. A handwritten `name` argument or override would not prove automatic holder naming. |
| Vars and starts | **Proved for concrete field schemas.** The holder's initial balances equal its supply, `opening = oneOf(0 to supply)` yields ordered starts `0,1` / `0,1,2`, and supply `-1` retains `opening: oneOf takes a non-empty domain`. **Refuted for generic/path-dependent fields:** direct generic schema and `before.balances` typing fail even before execution. | Retain ordinary var initializers and `oneOf`. Fix the shared declared-field type computation to use the lexical instance prefix rather than a class projection; test both schema generation and `before`, including generic records. No `.equals` substitution belongs in the final sample. |
| Lint | **Unsupported today, by source inspection.** `framework/lint/Structure.scala` selects only `Flags.Module` class definitions in `Source.objects` and `form`; `Order.scala` likewise recognizes object forms and its refusal text names the retired three-argument Machine. Its section inventory still names `rules/properties`, not native `transitions/invariants/postconditions`. No successful prototype compile establishes structure-lint admission. | Extend these existing passes to Machine class definitions and current sections, constructor/type parameter checks and located refusals. Keep object checks and their controls. Test a valid class, invalid parameters and misplaced sections; do not exempt classes or invent a second linter. |
| Realizations | **Proved for native construction.** `Realization(machine = smallBank, ...)` retains that exact instance and not `largeBank`. Its public parameter is already `MachineDeclaration[?, ?, ?]`. This checks construction only. | Keep `machine = smallBank`; a class type is not a Scala value. No new realization abstraction is needed. Fixture migration and class-as-instance compile controls remain .3 work. |
| Later phase (bounded) | **Unproved and held.** The probe performs no lifting, IR identity, export, Case lowering or Go/Temporal execution. | Keep instance parameter transport and Case identity for the IR phase; native success is not export evidence. |

## What the failed experiments establish

`refused/OwnedVocabulary.scala.txt` retains the attempted instance-nested `Denom`, `Balances`
and `Send`. Strict `balances == before.balances` first failed because the types were
`Model.this.Balances` and `Bank.Model#Balances`. The retained snapshot uses ordinary `.equals`
only to expose the next errors: `cannot establish a reference to ...Model#Denom`, a missing
balances schema and `input records require structural comparison`. The suite independently pins
the strict-equality refusal on a minimal instance-owned record.

`refused/FiniteOnly.scala.txt` moves the vocabulary to `Bank` and retains the intended generic
map. It still refuses `balances: Balances[Addr] has no IR schema` and input structural comparison.
Adding `Data.Structural[Addr]` and `StateData.CollectionMember[Addr]` allowed input comparison
but introduced ambiguous `Data[Addr]` selection alongside `Finite[Addr]`; explicit `CanEqual`
resolved only that comparison. Passing evidence for the complete `Balances[Addr]` and `Send[Addr]`
types still failed the superclass schema expansion. That last source is retained as
`refused/RecordEvidence.scala.txt`. These are observed failed experiments, not supported APIs.

The shared seam is `VarSchemaMacro.declaredType` in `dsl/Vars.scala`:
`cls.typeRef.memberType(v).widenByName` computes the class projection. `ClaimMacros.before`
uses it too. A concrete .2 candidate is to preserve `This(cls)`'s lexical type while reading
members and resolving selected evidence. That candidate is not implemented or proved here.
`Finite` already retains its selected structural comparator privately; reuse selected evidence
without catalog enumeration. Do not add a blanket `Finite => CollectionMember` conversion:
the existing collection rules deliberately exclude unsupported leaves such as Inbox keys.

The runnable fallback maps each external address to its finite catalog index and uses concrete
Bank-owned record types. This isolates R2/runtime behavior from the refuted schema path.
Its `client.send` is a `lazy val`: the eager nested-actor declaration triggered a genuine
`-Wsafe-init` warning promoted by `-Werror`; lazy construction compiles under the same flags.
No warning suppression or relaxed compiler option is used. This distinction needs a class-actor
control in .2, not a claim that the original eager spelling passed.

## Verification boundary

The fresh baseline is the existing focused native Bank suite: 10 tests, exit 0. The probe runner
requires executed tests and checks both runtime results and explicit current refusals. Its refusal
mode compiles fresh snapshots and requires their actual schema diagnostics, so an unrelated tool
failure cannot count as a demonstrated refusal. See `.flow/tmp/fn1941/` for captured output and
the task's done evidence for terminal commands.

The prototype is outside normal framework source discovery. Native formatter and framework
checks remain the authorized boundary. `make model-check-changed` is held because it routes to
Temporal work; it is not recorded as passing. Schema/generated IR, exporter fixtures, Go and
disabled Temporal sources remain untouched. This design checkpoint proves/refutes the requested
architecture before .2 changes the DSL; it does not complete R1/R3/R4 or native claim evaluation
for class Models.
