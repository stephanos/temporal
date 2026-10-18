# Native ModelSuite prototype

This throwaway prototype tests the four-section authoring shape against the shipped Scala checker.
The [verdict](../runs-in-tests.md) records each architecture decision and its remaining proofs.
No public DSL or checker implementation changes here.

From the repository root, run the positive suites with:

```sh
env -u BASH_ENV bash model/design/modelsuite-prototype/run.sh
```

The expected inventory is 18 passing tests. BankTest has eight, QuintBankTest one, KettleTest two
and LifecycleTest seven. The script selects the suite names explicitly and requires tests.
It keeps compiler workspaces under `.flow/tmp/fn1951`; set `MODELSUITE_WORKSPACE` to use a fresh
directory. It supplies MUnit 1.2.0 to this invocation's ordinary compile classpath, leaving the
library's test-scoped dependency unchanged.

Run these experiments separately:

```sh
env -u BASH_ENV bash model/design/modelsuite-prototype/run.sh framework.prototype.WrongExpectationProof
env -u BASH_ENV bash model/design/modelsuite-prototype/run.sh --early-refusal
env -u BASH_ENV bash model/design/modelsuite-prototype/run.sh --inventory
```

WrongExpectationProof must exit nonzero with the first Boolean's line, expression and concrete
balances, even though its second Boolean is true. Early.refusal.scala must fail strict safe-init
compilation because an eager helper touches its section before the suite has finished construction.
Inventory must exit zero after demonstrating that the actual source Registry recognizes only
LifecycleTest here. Registry recognition of indirect ModelSuite subclasses remains fn-195.2 work.

Bank uses named Send record fields, a finite oneOf family, a reusable any-action space, an overflow
find, reproducible six-step sampled walks and a Leaky supply counterexample. QuintBankTest replays
the literal retained sendTest with supply 10,000 and sends of 2,000 bananas and 1,000 burgers.
Kettle starts at 20, executes `repeat(80)(heater.heat)` and checks 100 degrees then failover.

LifecycleTest covers initialized eager helpers, independent receivers, fixed section order,
repeated/concurrent enumeration, deferred bodies, test/suite transforms, runner description and
filtering, setup ordering, ignore/only, unit-only/empty/absent sections, top-level/invalid-member
refusal, cross-section initialization, sticky partial failure and recursive registration.

## Deliberate prototype limitations

ModelSuite.scala adapts private-to-framework native declaration/context carriers in its own
`framework.prototype` package. Its action wrappers append real ReplayStep operations; its expect
macro appends real positioned ReplayExpectation operations. The shipped checker performs all
replay, find, sampling and claim refutation. Execution comes from the immutable suite BudgetPolicy
boundary, with the caller's existing budget semantics.

Conformance has prototype string metadata for expectation and through. Its paths replay natively.
This declares no generated Case and proves no realization resolution, Contract lifting, system
execution or recorded-Run conformance. The final typed RunExpectation/realization API is open.

The inherited String test overload preserves `test("..."):`. Named run parameters return a small
builder whose apply registers the same body. Plain unit bodies remain ordinary MUnit bodies.
The prototype does not lift arbitrary Scala test bodies or validate the complete public subset.

Scala 3.9 emits E176 unused-Boolean warnings before the ordinary expect macro expands. The suite
annotations suppress only the intended Boolean unused-value diagnostic. The raw strict failure is
preserved in `.flow/tmp/fn1951/prototype-fourth.log`. This is a prototype-only limitation.
Fn-195.2 still requires unannotated authored expect blocks under all existing strict flags.
The before/outcome adapters likewise do not change the public claim macros.

The existing Kettle singleton remains unchanged; its class-only migration belongs to fn-195.6.
No Temporal, Go, exporter/lifter, IR/schema, backend or generated route runs in this prototype.
