# Checker budgets

The generic framework gate and change-scoped gate read `UMPIRE_CHECKER_BUDGETS` as
newline-separated `key=value` entries. `default` sets the default state budget;
other keys are exact source-discovered suite names, fully qualified when the suite
has a package and simple when it is in the default package. For example:

```text
default=1048576
framework.dsl.BankTest=4096
framework.dsl.VotingTest=8192
```

With no configuration, the default is 1048576. An omitted `default` entry keeps
that default. Values must be positive Scala `Int`s. Empty configurations,
malformed entries, duplicate keys and unknown suite names fail before the gate
starts a compiler or test command. Blank lines and surrounding whitespace are
ignored. Validation includes held and unselected discovered suites.

The gate passes a normalized policy in the same environment variable to each
test JVM. `BudgetPolicy.fromEnvironment(environment).forSuite(fullyQualifiedName)`
resolves one suite's budget independently of other suites and threads. The
suite execution boundary must resolve once and pass that immutable budget into
every checker question. Explicit small budgets remain useful for engine contract
tests.

fn-195 owns the future ModelSuite registration, source-discovery support and
automatic use of this policy at the suite boundary. The direct native-test route
in `ModelSources.nativeModelTests` does not own budget policy validation; use the
generic framework gate for policy-controlled suite execution.

## Execution questions

`Checker.space(instance, steps)(select)` binds a checked instance at its caller
and supplies a finite step bound. The optional selector receives the zero-based
step position and an action class. `Checker.find(space, horizon)(target)` refuses
horizons below one or above the space bound. It returns `notFound` only after
finishing that horizon; holes and budget cuts return `unknown`. A witness reached
before a cut remains `found`. Positional searches budget their visited
state-and-position identities, so repeated states cannot bypass the budget.

`Checker.replay(instance, ReplayScript(...))` consumes ordered `ReplayStep` and
`ReplayExpectation` operations. Each expectation carries its source line, text
and deferred predicate. It reads the restored current state and the last step's
before-state, outcome and facts. Every selected finite alternative must hold.
Disabled steps refute; holes and cuts remain unknown. Replay budgets count
retained execution nodes, including positions and branches. A decisive failure
retains its concrete prefix without claiming the remaining traversal completed.

`Checker.sample(space, count, seed)` draws ordered seeded walks of at most
`space.steps` steps and checks their actual safety, monitor and progress claims.
Its `sampling` evidence retains all walks, the seed, the first failing walk index
and finite experiment completion. Successful finite walks establish no exhaustive
safety or unbounded progress result. An unfinished progress obligation remains
unknown; a sampled deadline violation or established deadlock refutes. Sampling
never turns a horizon or an excluded action into evidence of deadlock.

Every question returns the common `Result` vocabulary. A throwing effect refutes
with its action, pre-step state, message and successful prefix; independent
monitor and progress predicate errors propagate. `Checker.refuted(instance, claim)`
returns the ordinary claim answer, including verified or unknown, without an
assertion adapter. Existing Query and vacuous pinned-check semantics remain for
their separate retirement.

`Checker.forSuite(policy, exactSuiteName)` resolves an immutable `Execution`
boundary whose questions pass the same budget explicitly, including across async
test bodies. It stores no active suite context. fn-195 supplies the ModelSuite
adapter, expect syntax and registration, and extends discovery for its actual
supported declarations; this module provides execution inputs rather than a
prototype suite.
