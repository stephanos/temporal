# Nim sample

The two Models, authored in Nim against a small framework surface (`umpire.nim`). Nothing here has
been run through a toolchain; the sample shows what the authoring would look like and where each
check lands.

The activity Model follows the revised SPEC.md Model 2: the refinement is by mapped states with
any product action class allowed, `pauseRequested` reads as `started`, and the product sees a
retryable failure as a reschedule.

Files:

- `umpire.nim`: `Step`, `Machine`, finite enumeration, refinement, `Property`, `Scenario`,
  `Limits`, `Query`, `Set`, composition, and the command macros.
- `worker.nim`: the worker entity the compositions synchronize with.
- `nexus_caller.nim`, `standalone_activity.nim`: the two Models, in the Lean file's section order.
- `pins.nim`: the pins, as `static: doAssert` at compile time and `std/unittest` at run time.

## The DSL mechanism

Nim macros receive Nim's own parse tree, and Nim's parser already has a colon-block form: a call
followed by an indented block of `key: value` lines. So

```nim
machine nexusProduct:
  `for`: operation
  state: ProductState
  starts: [scheduled]
  steps:
    handlerReply: handlerReplyStep
```

arrives at `macro machine(name, body: untyped)` as `Command(machine, nexusProduct, StmtList(...))`
where each line is `Call(key, StmtList(value))`. The macro needs no parser: it reads the keys,
rejects unknown ones at their own line, and emits one `const`. The whole `machine` macro is about
a hundred lines, and half of that is building the loop nest that fills the table.

The consts are the point. A `const` in Nim is evaluated by the compile-time VM, and the VM runs
ordinary procs. `enumerate`, the row loops, `refinementOf` and `search` are plain procs with no
compile-time special casing, so the state table is built, the refinement walked and every Query
searched while the Model module compiles. `pins.nim` then pins the results twice with the same
procs: once in a `static:` block, once in `unittest`.

The vocabulary commands (`entity`, `action`, `observation`) emit no symbol at all. They register
in `macrocache` tables, which is the one compile-time store that survives across modules, and the
commands that reference them read the tables at expansion time. That is how `machine`'s `steps:`
knows `handlerReply` takes one `Reply` and enumerates six classes from one line, and how
`nexus_caller.nim` can step on the `workerStop` that `worker.nim` declared.

Two small pieces of macro work carry the rest of the surface:

- `finite ProtocolState` derives `enumerate` for an object as the cartesian product of its fields'
  enumerations. Enums and `range[0..n]` are finite for free through `low..high`. A `case kind`
  object contributes one member per assignment of each branch's fields, so `handlerError(
  retryable: bool)` is two classes, as in Lean. `key` needs no macro at all: `fieldPairs` visits
  only the active branch of a variant, which yields `"handlerError-true"` from one generic proc.
- `holds: step => ...` is the `std/sugar` arrow, rewritten by the `property` macro into a
  `{.nimcall.}` lambda typed by the machine's step type, so a predicate is written with no type
  annotation and still type-checks against the right `Step[S, O, F]`.

## Where each check runs

| Check | When | How |
| --- | --- | --- |
| Unknown field in a command block | compile time, macro | `error("unknown field ...", node)` at the line |
| Step names an undeclared action | compile time, macro | `machine` consults the action registry and the machine's `timers:` |
| Reference to an undeclared machine, property, scenario, limits, query | compile time, macro | registry lookup, pinned to the reference |
| Non-exhaustive `case` over a phase or a variant kind | compile time, compiler | Nim's ordinary exhaustiveness check on enums |
| Wrong class in `when:` or a scenario action | compile time, compiler | the class is emitted as a call to the constructor, so it type-checks |
| State table, catalog, `starts`/`ends` expansion | compile time, VM | inside the `const` the macro emits |
| Refinement (a protocol row is a stutter if its mapped states agree, else the product must have some row between them, of any class) | compile time, VM then macro | `refinementOf` in the const; `admit(refinement, node)` takes a `static` parameter and pins a rejection to the `refines:` line |
| Query search (`find` and `verify`) | compile time, VM then macro | `search` in the const; `admit(query, node)` pins to the `find:` line |
| Pins | compile time and test | `static: doAssert` in `pins.nim`; `unittest` for the same values with a printed diff |
| Canary admissibility (silent steps on a path) | not in this sample | would be another `static` check over the witness and the machine's `unobservable` |
| Composition table | sketched | `product` is a signature and a comment |

The `admit` trick is the one worth knowing. A macro parameter declared `static Refinement` is
evaluated in the VM before the macro body runs, so the macro sees the *value* the const computed
and still holds the author's `untyped` node to pin the error to. It is how a check that needs the
whole table can still report at a single token.

## What an author's error looks like

A step that names an action nothing declared:

```
nexus_caller.nim(233, 5) Error: step names an undeclared action `handlerRepy`; declare it with `action`, or list it under `timers:`
```

A `case` over `Phase` missing an arm, which is Nim's own check and needs no framework code:

```
standalone_activity.nim(314, 3) Error: not all cases are covered; missing: {pauseRequested}
```

A `when:` naming a class the domain does not have (`handlerReply(handlerErorr(true))`):

```
nexus_caller.nim(407, 22) Error: undeclared identifier: 'handlerErorr'
```

A query whose path never reaches its claim (say `retrySucceeds` searched in `syncReplied`):

```
nexus_caller.nim(520, 3) Error: query `retry`: retrySucceeds never holds on syncReplied
```

A refinement the product cannot account for:

```
nexus_caller.nim(378, 3) Error: refinement rejected: protocol row scheduled-0-unset-unset-unset-handlerReply-handlerError-true maps scheduled -> backingOff, which is no product step and no stutter
```

## Where the spec's names fight Nim's scope

Two things in the spec cost real friction in Nim, and the sample keeps the spec's spelling rather
than hide either.

**Five field names are Nim keywords.** `for`, `in`, `from`, `when` and `bind` cannot start a
`key: value` line, because the parser sees a `for` loop or a `when` statement. Nim lets any keyword
be an identifier when stropped in backticks, so the sample writes `` `for`: operation ``,
`` `in`: syncReplied ``, `` `when`: handlerReply(async) ``, `` `from`: Worker.polling `` and
`` `bind`: ``. It reads worse than the Lean and it is what an author would type on every machine,
query and property. A port that owned the vocabulary would rename these five (`entity:`, `path:`,
`on:`, `base:`, `parties:`) and the backticks would disappear.

**One flat namespace per module.** Lean's enum constructors live in their type's namespace, so
`query handlerError` sits beside `Reply.handlerError` without contact. Nim puts enum fields and
top-level consts in the same module scope, and a const may not share a name with an existing
symbol. Four declarations in the spec collide as written, each marked with a `NOTE` in the source:

| Declaration | Collides with | Error |
| --- | --- | --- |
| `query handlerError` (Nexus) | the `Reply` constructor `handlerError(retryable)` | `redefinition of 'handlerError'` |
| `scenario completed` (Activity) | `Phase.completed`, `ProductPhase.completed` | `redefinition of 'completed'` |
| `property terminated` (Activity) | `Phase.terminated`, `ProductPhase.terminated` | `redefinition of 'terminated'` |
| `query terminate` (Activity) | `Control.terminate` | `redefinition of 'terminate'` |

The payload domains already dodge this: their kind enums are `{.pure.}`, so `syncSuccess` the
`Reply` const and `ReplyKind.syncSuccess` coexist, at the price of qualifying every `of` arm in a
step function. Making every phase enum pure too would clear the table above, and would cost the
same qualification in every `case` over a phase, which is most of a step function. Splitting each
Model over a few modules (domains, machines, claims) is the more Nim-shaped fix, and it is what a
real port would do; this spec asks for one file per Model, so the sample shows the collision instead.

Nim 2's overloadable enum fields cover the cases that are not consts: `scheduled` is a member of
both `ProductPhase` and `Phase` in one module, and resolves by the expected type in `case` arms
and object constructors. Where no expected type is available, the sample qualifies (`Phase.started`
inside a set literal).

## Libraries to leverage

Maintenance status checked on 2026-09-29 with `gh api repos/<owner>/<repo>`. "Maintained" means a
push within the last twelve months and not archived; anything older or archived is marked no-go
and should not be built on, whatever its star count.

| Package | Area | Last push | Status | What it would replace |
| --- | --- | --- | --- | --- |
| `nim-lang/Nim` (`std/macros`, `std/macrocache`, `std/unittest`) | macro tooling, tests | 2026-09-29 | maintained | Everything the command macros use is stdlib; no macro helper library is needed. `std/macrocache` is the cross-module registry the vocabulary commands rely on. |
| `nim-lang/fusion` (`fusion/matching`) | macro tooling | 2025-11-24 | maintained, low activity | Pattern matching over `NimNode` trees would shorten `fieldsOf` and the `sync:` and `cover:` parsers; optional, and it is a thin dependency that moves slowly. |
| `status-im/nim-protobuf-serialization` | protobuf | 2026-09-11 | maintained | Reading and writing the protobuf Case format from Nim objects; `importProto3` generates the types at compile time, which fits a `const`-built table. Proto3 only. |
| `nitely/nim-grpc` | gRPC | 2026-06-13 | maintained | A pure-Nim gRPC client and server over HTTP/2, tested against go-grpc interop; it is the one gRPC option and it depends on the protobuf package above. Young (25 stars), OpenSSL required. |
| `treeform/jsony` | JSON | 2026-05-24 | maintained | Fast object-to-JSON for fixtures. It writes fields in declaration order and has no canonical mode, so a Behavior Fingerprint would still sort keys itself through its hooks. |
| `status-im/nim-json-serialization` (+ `nim-serialization`) | JSON | 2026-09-22 | maintained | The alternative to jsony when the same object types must also round-trip through protobuf: one serialization framework, several formats. Heavier. |
| `cheatfate/nimcrypto` | hashing | 2026-07-06 | maintained | SHA-2/3, BLAKE2 and Keccak for the fingerprint of the tables. Pure Nim, works in the VM for small inputs. |
| `nim-lang/checksums` | hashing | 2026-07-21 | maintained | SHA-1/2/3 and MD5 as the official successor to the removed `std/sha1`; enough if only a SHA-256 is wanted and a smaller dependency than nimcrypto. |
| `status-im/nim-unittest2` | tests | 2026-08-24 | maintained | A drop-in for `std/unittest` with parallel runs and better output; `pins.nim` would switch by changing one import. |
| `disruptek/balls` | tests | 2026-02-16 | maintained | A test runner with a compile-matrix (backends, GC, defines) if the pins should run under several Nim configurations. |
| `status-im/nim-stew` | utilities | 2026-09-24 | maintained | `stew/results`, `stew/bitseqs`, `stew/endians2` and friends; useful glue, not essential. |
| `PMunch/protobuf-nim` | protobuf | 2023-10-17 | not maintained, no-go | Was the best-known protobuf macro library; three years without a push. |
| `oskaritimperi/nimpb` | protobuf | 2021-08-29 | not maintained, no-go | |
| `PMunch/macroutils` | macro tooling | 2021-12-03 | not maintained, no-go | |
| `status-im/nim-drchaos` | fuzzing | 2023-05-10 | not maintained, no-go | A libFuzzer front end, not a property-based tester; a `planetis-m/drchaos` repo has 2026 pushes but no description, release or stars, so there is nothing dependable to build on. |
| `nim-works/npbt` | property-based testing | 2024-03-28 | not maintained, no-go | Self-described WIP. |
| `alehander92/nim-quicktest` | property-based testing | 2020-03-10 | not maintained, no-go | |
| `schneiderfelipe/quickcheck` | property-based testing | 2021-07-26 | archived, no-go | |

Gaps, plainly:

- **No model checker or explicit-state search library.** A GitHub search for Nim model checkers
  returns nothing. `enumerate`, the table loops, `reachable`, `refinementOf` and `search` in
  `umpire.nim` are hand-written, and would stay so. They are small, and the finite domains here are
  small, so that is acceptable; it is still code the Lean version gets from its library.
- **No maintained property-based testing library.** Every QuickCheck-style package is stale or
  archived. For this framework it matters less than it sounds: the domains are finite and
  enumerated exhaustively, so the exploratory set is a full walk within a budget rather than a
  random one. A generator-based shrinker for larger models would have to be written.
- **One protobuf library and one gRPC library**, both maintained, both from small teams, and the
  gRPC one depends on the protobuf one. That is a single point of failure for anything that talks
  to a server; for a model layer that only emits tables and fixtures it is a manageable risk, and
  the Go runtime is where the wire work happens anyway.

## Toolchain and loop

`nim c -r pins.nim` compiles the framework, both Models and the pins, running every table build,
refinement walk and Query search in the VM, then the unittest suites. Nim compiles this amount of
code in a second or two from cold; the VM work is the part that grows with the tables (the
activity protocol has 288 states and its `six` limits cover a 262144-candidate search), and the VM
is roughly an order of magnitude slower than native, so a Model that gets heavy would move `search`
out of the const and into the test, keeping `admit` for the refinement. Nothing in the design
depends on which side of that line a check sits: the same proc runs either way.

Edit-to-feedback is one compile. Errors from the macros are ordinary compiler errors with file,
line and column, and land in any editor that reads them. `nimsuggest` gives hover and go-to on the
procs; it does not see through the macros, so a `const` a macro emitted is not discoverable from
the editor by name.

## What Nim made easy

- The DSL is nearly free. The block syntax, the arrow lambdas, the `a.x || b.y` sync pairs and the
  `a | b | c` cover lists all parse as Nim, so the macros only walk trees.
- Compile-time evaluation of ordinary code. No second language for the static half.
- `case` exhaustiveness over enums is on by default and is the check that catches a phase added to
  `Phase` but not to a step function.
- `range[0..attemptBound]` is a first-class finite type and a proc over it (`saturatingSucc`)
  reads as the design reads.
- `key` from `fieldPairs`, and structural `==` on variant objects, fall out of the language.

## What was awkward, and the costs

- The keyword and scope collisions above.
- Macro debugging. When a `quote do` block produces a tree that does not type-check, the error
  points at the use site of the macro, and `expandMacros` or `treeRepr` is how you find out what
  was emitted. The `machine` macro's loop nest is exactly the kind of code that takes a few rounds.
- `WorkerFact` is uninhabited in Lean; Nim has no empty enum, so the sample uses the unit type and
  a comment. `Option` and `seq` in a `const` are fine, but a `const` cannot hold a closure, which is
  why the `holds` lambdas are `{.nimcall.}` and cannot capture.
- The `set` macro shadows the system `set` type constructor in call position. It works because
  `set` is not a reserved word, and a production framework would still rename it.
- Ecosystem and hiring. Nim is a small community; there is one maintained protobuf library and
  one maintained gRPC library, each from a small team and the second depending on the first (see
  "Libraries to leverage"), so anything that talks to a server rests on two thin dependencies. That
  is fine for a model layer that only emits tables and fixtures. Few engineers arrive knowing Nim,
  and the macro system in particular is a skill that is learned on the job.
- Nim 2's overloadable enums are recent enough that the rules for resolving a bare field against a
  same-named const or proc are still surprising in places; the table above is where the sample
  expects to be corrected by a compiler.
