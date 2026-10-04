# tools/umpire/export

Two other checkers read the lifted IR, and each is held to Go's reading of the same Model. Quint is
given every machine of four `model/ir/*.json` files and every composition the reader
(`tools/umpire/model`) builds of them. P is
given one monitor. An export counts only
where the tool's own run agrees with the reader; a tool that parses the export proves nothing here.

```sh
make umpire-check-backends                           # the agreement, three to ten minutes; one receipt per comparison
UMPIRE_BACKENDS_OUT=DIR make umpire-check-backends   # also keep every export, dump, report and receipts.txt under DIR
go test -tags test_dep ./tools/umpire/export/        # the Go side alone; the tool runs skip, and no tool is looked for
```

The agreement is opted into: `make umpire-check-backends` runs the package's tests with
`UMPIRE_BACKENDS=require`. They then check the pinned tools before any test starts and fail when a
tool is missing or of another version, when a tool fails, and on any disagreement; no comparison
skips. The default `go test` needs no tool and looks for none. It checks the exporters, the
comparison and the replay against dumps written from Go's own interpretation and against tampered
copies of them.

## What is compared

| Claim | Backend | What the receipt covers |
| --- | --- | --- |
| `transition-agreement` | Quint | Every state a machine's or a composition's starts reach, by every class: the results in order with outcome, state, facts, explanation and choice name, and an empty list for a disabled pair. Starts in order; reachable states, ends and classes as sets |
| `monitor-agreement` | Quint | Every step of the product of a machine and its monitors: each monitor's state after the step, whether its verdict is read there, and whether that state violates it. Quint's counterexample of each violated monitor is replayed through Go |
| `property-agreement` | Quint | Every Property a machine or a composition declares, on every step from every reachable state: whether it is about the step and whether it holds |
| `checker-coverage` | Quint | `covered`: what Quint's evaluator enumerated. `agreed`: Apalache's bounded verdict on one monitor, with its counterexample replayed through Go. `not-run`: a module Apalache did not take, with its error |
| `monitor-agreement` | P | One monitor over every event trace of a machine within five steps: P's spec machine is first violated at the step Go's monitor is, or on neither |
| `checker-coverage` | P | How many test cases P's checker ran and what each covered |
| `module-refinement` | both | Always `unsupported`. No refinement is exported or claimed. A replacement inside a composition is the reader's verdict |
| `query-agreement`, `progress-agreement` | Quint | Always `unsupported`. The Model's Queries and progress claims are not exported |

The last run compared 29 machines and 6 compositions: 2,552 reachable states, 42,506 state and class
pairs (9,748 enabled with 10,292 results, 32,758 disabled), 2,378 product steps over 946 product
states for the 11 machines that name monitors, and 52,772 Property readings of 167 Properties. It
printed 146 receipts: 78 agreed, 37 covered, 30 unsupported and 1 not run.

| Slice | Machines | Compositions | Pairs compared | Monitored machines | Machines with a violated monitor |
| --- | ---: | ---: | ---: | ---: | ---: |
| `activity` | 4 | 1 | 15,817 | 0 | 0 |
| `activity-system` | 12 | 5 of 7 | 13,839 | 2 | 1 (`staleAdmission`: both monitors) |
| `nexus-caller` | 4 | 0 | 3,716 | 0 | 0 |
| `nexus-close` | 9 | 0 | 9,134 | 9 | 6 |

The six compositions hold 886 of those states and 17,701 of those pairs, and all 18 Properties the
slices declare on compositions (6,953 readings): `standaloneActivity`, and the current and stale
admission designs over the opaque queue, the matching queue and the lossy matching queue.

The monitors are the authored ones: `terminalFinality` and `atMostOneActiveAttempt` on the two
activity admission designs, and `retainedOutcome`, `ownerAcknowledgment`, `singleOutcome` and
`cancelPrincipal` on the nine Nexus close policies.

## Quint: how the agreement is exhaustive

`Slice.Quint` translates the IR's types and functions declaration for declaration into one Quint
module. For each machine the module computes, in pure definitions, the states its starts reach and
every result of every class from each of them, then the same for the product with its monitors and
for its Properties. Its one state variable holds that dump. `quint run` takes one sample of one
step, which evaluates the initializer and writes the dump as an ITF trace. The module has one
behavior, so the run samples nothing.

Quint's simulator explores by random sampling, and its "no violation found" proves nothing (Quint
manual, `quint run`). This gate does not use it that way. The reachable set is a fixed point the
module computes itself: it takes as many rounds of successors as Go's table is deep plus one, and the
dump reports whether a step still leaves the set. The round count is the only number Go gives the
module. A count that is too small shows as `closed: false` and fails the comparison.

A step record carries the name of the alternative of a named choice it is, as `f_choice`, empty for
an unnamed one (`model/SEMANTICS.md`, Named choices). The step function keeps every alternative of
a choice in its result list, in order, each with its name; only a check module's step action picks
one, by index (`nondet n = oneOf(rs.indices())`). A composed step record carries the empty name, as
Go's composed results do.

`QuintAgreement` decodes the dump by the IR's types and compares it key by key with Go's tables: a
machine's from `umpiremodel.Build`, a composition's from `Realizer.Composition`, the reading `umpiremodel.Check`
answers it from. A pair the dump leaves out is a difference; it is never read as disabled.

Go's side of the monitor product evaluates each monitor's `next`, evaluation point and `violated`
with the reader's interpreter. `umpiremodel.Check` then confirms it through ordinary admission: one verify over
every path from each start, and one over the classes of each counterexample.

`quint verify` adds a model checker's verdict where it runs. Each of the four activity checks
(`currentAdmission` and `staleAdmission`, by two monitors) gets a module with state variables, and
Apalache 0.62.1 checks the monitor's invariant on every run up to the product's depth plus one. That
is bounded model checking: it finds a violation within the bound and says nothing of longer runs
(Quint manual, `quint verify`). It found a two-step counterexample for each monitor of
`staleAdmission`, which is Go's shortest, and none for `currentAdmission` within four steps.

Apalache does not take the Nexus close module. Its inliner stops with `Recursive substitution took
more than 100000 iterations`. The receipt for it is `not-run`, and the nine Nexus machines' monitors
are compared by the evaluator's product alone.

## Compositions

`umpiremodel.Realizer.Composition` gives the composed table `umpiremodel.Check` builds for a composition's claims,
the state record and step record each key stands for, and the composition's Properties as the
checker binds them. That is Go's side of the comparison.

Quint's side is computed from the IR's composition declaration over the member machines the module
already holds. A class is a member's own class, of an action no sync takes, or the pair of two
members' classes a sync takes together. A pair's step is the product of the members' results, the
first member's first, with the first member's outcome and both members' facts. The starts are the
product of the members' starts, and the reachable set is the same fixed point as a machine's.

A claim of a composition reads a step's outcome and facts as the strings `<field>_<key>`. Quint
builds no string, so the module spells each member's outcomes and facts out, one string per value
of the type, with the key Go's `Value.Key` gives it.

Two of the eight compositions are not exported: `currentOverForgetful` and `currentOverVolatile`.
Each puts a queue provider in place of `dispatchQueue` that does not refine it, so the reader rejects the
replacement and builds no composed table. Their receipts are `unsupported` with the reader's rejection
as the reason, and they declare no Property. For the three compositions whose replacement holds, the
module holds the composed table and a `module-refinement` receipt says the replacement is the reader's
verdict alone.

A composition past a ceiling of the scope (`OpenWithin`) has a `resource-limit` receipt and no part
of it is exported. A composition with a pair a member's hole leaves unknown is refused like a
machine with a hole row. No model checker is run on a composition.

## Counterexamples and errors

A counterexample from a backend is a path of the machine. `Slice.Replay` builds a fresh
interpretation and requires that the path starts at a start, that every step is a result of its row
under the table's Definition IDs, and that the monitor is read and violated on the last step. A path
that fails any of these makes the receipt `witness-rejected`, which stands before any difference the
comparison found. A dump or a trace that cannot be decoded is an error of the call.

## What the exporters refuse

`Slice.Quint` returns an `UnsupportedError` and writes nothing for:

- a `hole` expression in any function an exported machine reaches, and a machine whose table has a
  hole row. A hole is never written as a disabled action;
- a channel, a derived delivery or loss, and an inbox operation;
- an anonymous function anywhere but a machine's `ends`;
- a choice name Quint does not write as it is. Quint's lexer takes a string literal to the next
  double quote and keeps the text between the quotes with no escapes, so a name with a double quote
  is refused; so is one with a backslash, a control character or a character outside ASCII, which
  are not established to survive Apalache and an ITF trace unchanged. The error is at the step
  record's position.

A value no `match` case accepts and a call outside a function's precondition have no value in the
module either. They are written as an expression Quint's evaluator stops on (`QNT505`), so the run
fails instead of producing a row.

Listed as `unsupported` and left out: the two compositions the reader builds no table of, refinements
(9 of machines, 3 replacements), progress claims (10) and the Queries (259, one receipt per slice).
A Query's Scenario and Limits stay the reader's: Quint is given the Properties the Queries ask and the
monitors that watch them. A Property of a composition about one composed class is refused; none of
the slices declares one. Evidence lines, Definition IDs, fingerprints and realizations are outside
both backends.

## P: one monitor

`Slice.PMonitor` writes `terminalFinality` as a P spec machine. It translates the monitor's IR
functions into P functions: `finality`, `terminal`, and the monitor's `next` and `violated`.
A driver machine announces each event trace to it. The event carries the state before the step and
the step's outcome and state.

The traces are every path of the machine from its starts within five steps, each ended by its first
violation, by a state with no step, or by the bound. `staleAdmission` has 1,171 (996 accepted, 175
rejected) and `currentAdmission` has 483, all accepted. The specimen's four-step path, on which a
stale delivery starts a completed activity, is one of the rejected ones.

P's checker explores schedules by random sampling (P manual, `p check`). A driver that only
announces events has one schedule, so each test case runs once with `-s 1` and that run is the
whole behavior. The test cases are:

- `tcAgreement`: every trace against a copy of the monitor that records its first violation and
  asserts, at the trace's end, that it is the step Go gave;
- `tcAccepted`: every accepted trace against the monitor's own assertion, which must not fail;
- `tcRejected<n>`: the first 16 rejected traces, each against the monitor's own assertion, which
  must fail at Go's step;
- `tcControlWrongExpectation`: an accepted trace announced as violated, which must fail.

`p check -tc` selects every test case a name is a prefix of and runs none for a name that matches
none, without an error. `RunP` therefore requires that the checker's output names exactly the test
case asked for.

The P export covers one monitor on two machines, traces of at most five steps, and steps without
their facts. It refuses a monitor that reads a step's facts, a type whose enum cases carry values
(P's enums carry none, which excludes `atMostOneActiveAttempt` and every Nexus monitor), a monitor
read at the machine's ends, and a conditional inside an expression. The machine's transitions are
not exported to P and no P module refinement is checked.

## Tools

| Tool | Version | How it is installed |
| --- | --- | --- |
| Quint | 0.33.0 | `tools/umpire/export/quint.sh`: `npm exec --yes --package=@informalsystems/quint@0.33.0 -- quint`. The TypeScript evaluator runs the dump |
| Apalache | 0.62.1 | `quint verify` downloads it into `~/.quint` on first use. It runs on the repository's JDK (`mise.toml`) |
| .NET SDK | 8.0.425 | `curl -sSL https://dot.net/v1/dotnet-install.sh | bash -s -- --channel 8.0 --install-dir .build/umpire-backend-tools/dotnet` |
| P | 3.1.0 | `dotnet tool install P --version 3.1.0 --tool-path .build/umpire-backend-tools/p` |

.NET and P are installed once, by the two commands of the table, run from the repository's root;
`make umpire-install-backends` runs both and skips a tool that is already there.
`UMPIRE_BACKEND_TOOLS` moves the tool directory. Nothing is added to `mise.toml` and no binary is
checked in. `quint verify` starts an Apalache server and leaves it running; the checks use port
38822 and stop the server on that port when they end.

The pins are in `tools_test.go`. Under `UMPIRE_BACKENDS=require` the tests take Quint from
`quint.sh` and P from the tool directory, whatever else is on the path, and set `UMPIRE_QUINT` and
`UMPIRE_P` to them for the tool runs.

## Layout

| File | What it holds |
| --- | --- |
| `slice.go` | `Slice`, receipts, Go's reading of a machine, its monitor product and its Properties, and `Replay` |
| `quint.go` | The IR to Quint translation, the dump module and the check module |
| `composed.go` | The composition's translation and the reading of its part of a dump |
| `itf.go` | Reading ITF values back by the IR's types |
| `agreement.go` | The comparison of a dump with Go |
| `checked.go` | Confirmation of monitor verdicts and counterexamples through `umpiremodel.Check` |
| `verify.go` | The Apalache run and its verdict's comparison |
| `p.go` | The IR to P translation, the traces, and the comparison of P's reports |
| `tool.go` | Running the tools |
| `tools_test.go`, `quint.sh` | The opt-in, the pinned tools and the Quint launcher |
