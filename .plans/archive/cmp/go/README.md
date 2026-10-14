# Umpire Models in Go

The two Models from `../SPEC.md`, written as a Go expert on the Temporal server team would write them
against a Go `umpire` package. Model 2 follows the revised spec: refinement by mapped states with any
product action class counting, `pauseRequested` mapped to `started`, and a visible retry in the product. Nothing here has been compiled; the files are close to real code, with
real imports and signatures, and the notes below say where they would push back.

## Layout

```
go.mod                                  module go.temporal.io/umpire/model (Go 1.24, tool directives for the linters)
umpire/umpire.go                        the framework surface: Step, Machine, Finite, Refines, Property, Scenario,
                                        Limits, Query, Set, Compose, and Check
common/common.go                        Timeout and Delivery, the domains both Models share
worker/worker.go                        the Worker module: entity, polling machine
nexuscaller/nexus_caller.go             Model 1, in the Lean file's section order and with its comments
nexuscaller/pins_test.go                the Nexus pins
standaloneactivity/standalone_activity.go   Model 2
standaloneactivity/pins_test.go         the activity pins
```

Go has one package per directory and no nested namespaces. The two Models share every structural name
(`Phase`, `ProtocolState`, `productOf`, `three`, `retry`), so each is its own package and the shared
domains sit in a third. The Lean file's `namespace` does the same thing in one file. Pins are white-box
tests inside the Model's package, which is where Go convention puts them and what lets them call the
unexported step functions by the spec's own lowercase names.

Spec names are kept verbatim as the `Name:` string of every declaration. Go identifiers keep the same
spelling wherever they can be unexported (`syncSucceeds`, `two`, `scheduleToStartExpires`) and are
capitalized where the Testpilot runtime needs to reach them (`NexusProtocol`, `NexusCallerTests`).
Enum constants carry a type prefix where Go's package-scoped constants would otherwise collide
(`ProductScheduled` next to `Scheduled`, `ResolutionSucceeded` next to `Succeeded`).

## The DSL mechanism

Typed struct literals of framework types, a few small generic functions, and two kinds of Go type for
the two kinds of enum. No code generation is required to author; one small generator emits `Values()`
and `String()` for integer enums.

- **Actions are package-level values.** `var schedule = &umpire.Action3[Timeout, Timeout, Timeout]{...}`
  and `var handlerReply = &umpire.Action1[Reply]{...}` carry their input types. A step function binds
  with `umpire.Bind1(handlerReply, protocolHandlerReplyStep)`, and a Scenario names a class with
  `handlerReply.With(SyncSuccess{})`. Both are checked by the compiler.
- **Machines are struct literals.** `Steps: umpire.Steps(umpire.Bind3(schedule, scheduleStep), ...)`
  is a variadic call rather than a slice literal so the type arguments are inferred once instead of
  spelled at the field. `Refines: umpire.Refines(NexusProduct, productOf)` is typed in the state of the
  machine that holds it.
- **Step functions are plain funcs** returning `[]ProtocolStep` (a package alias of
  `umpire.Step[ProtocolState, ProtocolOutcome, ProtocolFact]`). Enabled-or-not is `nil` or one row.
  Struct update is a by-value copy: `moves(state, Succeeded, ...)` mutates its own copy.
- **Plain enums** are `type Phase uint8` with `iota` constants and a generated `Values()` that
  satisfies `umpire.Finite[Phase]`. A struct state gets its domain from `umpire.Fields[ProtocolState]()`,
  which walks the fields by reflection.
- **Sum types** are sealed interfaces: `type Reply interface{ isReply() }` with one struct per
  constructor and a `//sumtype:decl` marker. `HandlerError{Retryable bool}` is one struct and two
  classes; `umpire.Sum[Reply](SyncSuccess{}, ..., HandlerError{})` lists the variants and expands the
  fields. Steps `switch reply := reply.(type)` over them.
- **Compositions** are struct literals over a struct state whose fields carry `umpire:"operation"`
  tags, the way `encoding/json` names fields. `umpire.Sync{workerStop: {"operation": workerStop,
  "worker": worker.WorkerStop}}` pairs member actions; `umpire.At("operation", scheduleToStart)`
  qualifies a member's own action on a path.
- **Protobuf schemas are typed.** `Schema: umpire.Schema(&nexuspb.HandlerError{})` reads the full
  name back through protoreflect, so a renamed message is a compile error. The Lean file carries the
  same schema as an unchecked string.

## Where each check runs

| Check | Lean | Go |
| --- | --- | --- |
| Step names an undeclared action | compile | compile (`undefined: schedul`) |
| Step signature disagrees with the action's inputs | compile | compile (type inference in `Bind1` fails) |
| Scenario passes an input of the wrong type or arity | compile | compile (`With` is typed) |
| Non-exhaustive match on an enum or sum type | compile | lint: `exhaustive`, `go-check-sumtype` (`go vet` does not) |
| A step lands outside the finite state domain | compile (`Fin`, `deriving Finite`) | test (`Check` builds the table) |
| Property has both `When` and `Transition`, or neither | compile (syntax) | test |
| Property's machine vs Query's Scenario machine mismatch | compile | test |
| Fact recorded with no `Evidence` line | compile | test |
| Refinement: every protocol row is a product row or stutter | compile | test |
| Query answered (found / verified) | compile (`#guard`) | test |
| State counts, action class counts, reachability pins | compile (`#guard`) | test (`require`) |
| Canary names a silent step | compile | test |
| Protobuf message named by an action exists | not checked | compile |

Everything on the right that says "test" runs from `go test ./...` in the same process that would
later run the Cases. Lean elaborates the file and refuses to build it; Go compiles anything that
type-checks and reports the rest as failing tests a second later.

## What an author's mistakes look like

**Naming an undeclared action** in a machine's steps or on a path is an ordinary compile error, since
actions are variables.

```
$ go vet ./...
# go.temporal.io/umpire/model/nexuscaller
nexuscaller/nexus_caller.go:410:15: undefined: schedul
```

**Binding a step to the wrong action.** `umpire.Bind1(handlerReply, protocolCompleteStep)` unifies
`A` from both arguments and fails when they disagree.

```
nexuscaller/nexus_caller.go:411:16: in call to umpire.Bind1, type func(state ProtocolState, resolution Resolution) []ProtocolStep of protocolCompleteStep does not match inferred type func(ProtocolState, Reply) []umpire.Step[ProtocolState, ProtocolOutcome, ProtocolFact] for func(S, A) []umpire.Step[S, O, F]
```

**A non-exhaustive switch** compiles. Dropping the `TimedOut` arm from `productOf` is caught by the
`exhaustive` linter; dropping `HandlerError` from a reply switch is caught by `go-check-sumtype`.
Neither runs under `go vet` or `go build`; they run under `golangci-lint` in the server repo and as
`go tool` here.

```
$ go tool exhaustive ./...
nexuscaller/nexus_caller.go:352:2: missing cases in switch of type nexuscaller.Phase: nexuscaller.TimedOut

$ go tool go-check-sumtype ./...
nexuscaller/nexus_caller.go:206:2: exhaustiveness check failed for sum type "Reply" (from nexuscaller/nexus_caller.go:63): missing cases for HandlerError
```

Note that the compiler still wants a terminating statement after a switch it cannot prove
exhaustive, so every such switch in the Models carries a `default: panic(...)` arm that the linter
knows is dead. Both linters keep reporting missing arms when a `default` is present, which is the
setting the Models rely on.

**A failed query** is a failing subtest with the search's explanation.

```
--- FAIL: TestFunctionalQueriesFind (0.04s)
    --- FAIL: TestFunctionalQueriesFind/retry (0.03s)
        pins_test.go:78: no path of retriedThenSucceeded within four reaches retrySucceeds:
            the path runs to its end and its last step lands in
            {phase: succeeded, attempts: 2, scheduleToClose: unset, scheduleToStart: unset, startToClose: unset};
            the claim fixes attempts = 1. Searched 812 of 32768 candidates.
```

**A semantic slip the type system cannot see** fails from `Check` with the declaration named.

```
--- FAIL: TestDeclarationsCheck (0.00s)
    umpire.go:322: query terminalHolds: verify names terminalIsFinal on nexusProduct, but asyncThenSucceeded runs on nexusProtocol, which does not declare Refines: nexusProduct
    umpire.go:298: machine nexusProtocol: row scheduled-2-unset-unset-unset-handlerReply-handlerError-true lands in {phase: backingOff, attempts: 3, ...}, which is outside States (Attempts.Values stops at 2)
```

## Toolchain and loop

One toolchain: the Go the server and Testpilot already build with. `go vet ./...` and
`go test ./...` on these packages take about a second each after the first build; the linters add
another second or two. Adding an enum constant means `go generate ./...` for `Values()` and `String()`,
then the linters point at every switch that now misses it.

```
go generate ./... && go vet ./... && go tool exhaustive ./... && go tool go-check-sumtype ./... && go test ./...
```

A Model author who already works in the server repo has nothing to learn beyond the `umpire` package,
and the tables, search and Case realization run in the same process as the runtime, so no bridge
format or second build exists between the Model and the tests it produces.

## Honest notes

What Go made easy:

- **No learning curve, one toolchain, instant compiles.** The Models are ordinary Go read by anyone on
  the team, edited in the same editor and CI as the server, and `go test` is the whole loop.
- **Typed protobuf references** where Lean has strings, and typed `Examples` maps keyed by the input
  class, so the realization values cannot name a class that does not exist.
- **Zero values are the first values.** `ProtocolState{Phase: Unscheduled}` says "unscheduled, no
  attempts, no deadlines" without naming the other four fields, which is exactly what Lean's `starts:
  unscheduled` defaults to.
- **Struct tags and reflection** give compositions and struct states their enumeration without a
  macro system.
- **Variadic generics helpers** (`umpire.Steps(...)`, `umpire.Sum[Reply](...)`) keep type arguments off
  most lines, and package aliases (`ProtocolStep`, `protocolProperty`) take care of the rest.

What Go made awkward:

- **No sum types.** A Lean `enum` with a payload is one line; here it is an interface, five to ten
  one-line structs, as many marker methods, a `//sumtype:decl` comment and a hand-listed
  `umpire.Sum(...)`. Exhaustiveness is a linter's word, not the compiler's, and a `default: panic`
  arm is needed anyway to satisfy the compiler.
- **No scoped enum constants.** Three enums whose members are all `scheduled` need three prefixes,
  so `ProductScheduled`, `Scheduled` and `ProductNexusOperationScheduled` coexist in one package. Both
  fact enumerations of a Model spell the same nine names twice, once as constants and once as structs.
- **Names are written twice or three times.** The identifier, the `Name:` string the tables and
  fixtures use, and for enums the generated `String()`. Lean reads the name off the declaration.
- **Everything semantic waits for `go test`.** Refinement, evidence completeness, state-domain
  membership, property shape, claim-vs-path compatibility, and every pin. Lean refuses to build the
  file; Go builds it and tells you a second later. In practice the difference is small, but a Model that
  compiles is not a Model that is right.
- **Generics have hard edges.** No type parameters on methods, so `umpire.Bind1(action, step)` rather
  than `action.Bind(step)`. No variadic type parameters, so `Action0`, `Action1[A]`, `Action3[A, B, C]`
  and `Bind0`/`Bind1`/`Bind3` are separate. No constant parameters, so `Attempts` is a `uint8` whose
  bound is a `Values()` method and a `Check`-time rejection rather than a `Fin`. Type arguments cannot
  be elided in a struct field's literal, so every Property and Scenario is written through a package
  alias. A `Query` cannot be generic over a Property of one machine and a Scenario of another, so its
  `Find` and `In` fields are erased interfaces and the pairing is a test-time check. A composition's
  step has member-typed outcomes and facts that Go cannot express as a tuple of types, so they arrive
  as `umpire.Joint{Member, Value}`. And `slices.Contains(step.Facts, NexusOperationCompleted{})` does
  not infer, which is why `Step` has a `Records` method.
- **Declarations are long.** A Property that Lean states in four lines is eight to ten here, most of
  them braces and field names. The Models are roughly twice the Lean file's length for the same content.
- **Reflection replaces derivation.** `umpire.Fields[ProtocolState]()` finds each field's `Values()`
  at run time. A field type without one is a `Check` failure, not a compile error, and the generator
  that emits `Values()` for plain enums has to be run.

## Libraries to leverage

Maintenance status checked on 2026-09-29 with `gh api repos/<owner>/<repo>`; anything with no push in
the twelve months before that, or archived, is marked no-go. Only the "use" rows are recommended.

| Area | Library | Last push | Status | What it would replace |
| --- | --- | --- | --- | --- |
| Stateful property testing | [flyingmutant/rapid](https://github.com/flyingmutant/rapid) | 2026-09-04 | use | The exploratory Set's random walk and shrinking of a failing action sequence |
| Stateful property testing | [leanovate/gopter](https://github.com/leanovate/gopter) | 2026-04-20 | maintained, second choice | Same as rapid, with a heavier API; pick one |
| Property testing | `testing/quick` (stdlib) | frozen by the Go team | no-go | Nothing: frozen since Go 1.x, no stateful mode |
| Explicit-state model checking | [fizzbee-io/fizzbee](https://github.com/fizzbee-io/fizzbee) | 2026-08-25 | maintained, not applicable | Its checker consumes FizzBee specs, not Go step functions, so it cannot host these Models as a library |
| Explicit-state search | none found | | write it | The finite table, reachability and bounded search are a few hundred lines over `map[S]` and a queue; no maintained Go library does this over user types |
| Enum exhaustiveness | [nishanths/exhaustive](https://github.com/nishanths/exhaustive) | 2026-09-13 | use | Lean's exhaustive `match` for integer enums; also in golangci-lint |
| Sum-type exhaustiveness | [alecthomas/go-check-sumtype](https://github.com/alecthomas/go-check-sumtype) | 2026-09-20 | use | Exhaustive `match` for sealed interfaces; in golangci-lint as `gochecksumtype` |
| Sum-type exhaustiveness | [BurntSushi/go-sumtype](https://github.com/BurntSushi/go-sumtype) | 2025-03-21 | not maintained, no-go | Superseded by the alecthomas fork above |
| Enum codegen | [dmarkham/enumer](https://github.com/dmarkham/enumer) | 2026-01-22 | use | `String()` and `<Type>Values()` for every integer enum; a 40-line template on top emits the `Values()` method `umpire.Finite` wants |
| Enum codegen | [alvaroloes/enumer](https://github.com/alvaroloes/enumer) | 2024-08-03 | not maintained, no-go | The original enumer; use the dmarkham fork |
| Enum codegen | [golang/tools](https://github.com/golang/tools) `stringer` | 2026-09-29 | use if enumer is unwanted | `String()` only; the `Values()` generator would be written against `go/types` the way stringer is |
| Lint runner | [golangci/golangci-lint](https://github.com/golangci/golangci-lint) | 2026-09-29 | use | Runs both exhaustiveness linters in the server's existing `make lint-code` |
| Protobuf | [protocolbuffers/protobuf-go](https://github.com/protocolbuffers/protobuf-go) | 2026-09-23 | use | `umpire.Schema` via `proto.MessageName`; `protojson` for the Case format; `protocmp` for comparing messages in pins |
| gRPC | [grpc/grpc-go](https://github.com/grpc/grpc-go) | 2026-09-29 | use | Nothing in the model layer; the realization's transport, already a server dependency |
| Temporal client | [temporalio/sdk-go](https://github.com/temporalio/sdk-go) | 2026-09-29 | use | Nothing in the model layer; the realization drives the caller and worker parties through it |
| Deep comparison | [google/go-cmp](https://github.com/google/go-cmp) | 2026-06-18 | use | Readable diffs of rows and tables in pins and in the refinement report; `protocmp` for protobuf |
| Assertions | [stretchr/testify](https://github.com/stretchr/testify) | 2026-09-24 | use | `require` in the pins, as the server does |
| Canonical JSON | [gowebpki/jcs](https://github.com/gowebpki/jcs) | 2026-09-21 | use | RFC 8785 canonicalization of fixtures and the Behavior Fingerprint input |
| Canonical JSON | [gibson042/canonicaljson-go](https://github.com/gibson042/canonicaljson-go) | 2019-04-22 | not maintained, no-go | Same purpose as jcs |
| Struct hashing | [mitchellh/hashstructure](https://github.com/mitchellh/hashstructure) | 2023-01-03, archived | not maintained, no-go | Would have hashed the table for the Fingerprint; hash canonical JSON with `crypto/sha256` instead |
| Fast hashing | [zeebo/xxh3](https://github.com/zeebo/xxh3) | 2026-09-08 | use only if `crypto/sha256` is too slow | Row keys and search visited-set hashing |
| Fast hashing | [cespare/xxhash](https://github.com/cespare/xxhash) | 2024-07-03 | not maintained, no-go | Complete and widely used, but no push in the window |

What this buys: rapid takes the exploratory Set's generation and shrinking off the framework; the two
linters are the whole of the exhaustiveness story; enumer removes the hand-written `Values()` and
`String()`; protobuf-go and go-cmp are already server dependencies and give typed schemas, the Case
format and readable pin diffs; jcs plus `crypto/sha256` is the Fingerprint. The one thing no
maintained Go library provides is the explicit-state table build, reachability and bounded search
over user-defined types, and that is the smallest part to write.
