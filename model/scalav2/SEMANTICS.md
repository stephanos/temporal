# Semantics of the Umpire IR

The IR's meaning is defined here, not by the lifter that writes it or the Go interpreter that reads
it. `proto/internal/temporal/server/api/modelir/v1/ir.proto` is the schema; `goir/` is one evaluator of these rules, and the parity
tests check it against the Lean Model.

## Versions

`Model.version` is the IR's major version. This document defines version 0, which every Model written
before the field existed is in. A reader rejects a Model of a version it does not know, and a Model
that uses a declaration or a field it does not implement, rather than read the Model without it.

## Values

A value is a Boolean, an integer, a string, an enum value `C(v1, …, vn)` of one case `C` of an enum
type with its fields, a record value `(v1, …, vn)` of a record type, a list `[v1, …, vn]`, or an
anonymous function. Equality is structural. The step record `umpire.Step` is a built-in record with
the fields `outcome`, `state`, `facts` (a list) and `because` (a string). The delivery
`umpire.Delivery` is a built-in record with the fields `message` and `redeliveries` (an integer): one
message a channel holds, and how many more times than once it has been delivered.

An optional value is an ordinary enum: a front end declares one type per value type, with the cases
`None` and `Some(value)` in that order. The lifter names it `scala.Option[T]`.

## Catalogs

A finite type has its members in catalog order:

- `bool` is `false, true`;
- an integer range `low..high` is its integers in increasing order;
- an enum is its cases in declaration order, each case once per assignment of its fields;
- a record, and a case's fields, are the product of the fields in declaration order, with the last
  field varying fastest;
- `channel c` is everything channel `c` can hold ([Channels](#channels)).

## Keys

`key(v)` spells a value in state, action, row and Definition keys:

- `key(true) = "true"`, `key(n)` is `n` in decimal, `key(s) = s` for a string;
- `key(C) = "C"` for a case without fields, and `key(C(v1, …, vn)) = "C-" + key(v1) + "-" + … + key(vn)`;
- `key((v1, …, vn)) = key(v1) + "-" + … + key(vn)` for a record;
- `key([v1, …, vn]) = "[" + key(v1) + "," + … + key(vn) + "]"` for a list, which is how a channel's
  contents key.

## Expressions

`E(x, σ)` is the value of expression `x` under the environment `σ`, which binds names to values.

| Expression | Value |
| --- | --- |
| `literal v` | `v` |
| `var n` | `σ(n)`; an unbound name is an error |
| `field(b, f)` | field `f` of `E(b, σ)` |
| `call(f, a1, …, an)` | `E(body_f, [p1 ↦ E(a1, σ), …, pn ↦ E(an, σ)])`, after `E(requires_f, …) = true`; a call outside the precondition is an error |
| `construct(T, C, a1, …, an)` | `C(E(a1, σ), …)`, or the record `(E(a1, σ), …)` when `C` is empty |
| `copy(b, f1 = x1, …)` | `E(b, σ)` with each named field replaced by `E(xi, σ)` |
| `not a`, `-a` | Boolean negation, integer negation |
| `a and b`, `a or b` | short-circuit: `b` is evaluated only when `a` does not decide |
| `a = b`, `a ≠ b` | structural equality |
| `a < b`, `≤`, `>`, `≥`, `+`, `-` | on integers |
| `a ++ b` | list concatenation |
| `a in b` (`OP_CONTAINS`) | whether list `E(b, σ)` has an element equal to `E(a, σ)` |
| `if c then a else b` | `E(a, σ)` when `E(c, σ) = true`, else `E(b, σ)` |
| `let n = v in b` | `E(b, σ[n ↦ E(v, σ)])` |
| `list(a1, …)` | `[E(a1, σ), …]` |
| `lambda(p1, …) b` | a function that binds its parameters over `σ` |
| `match s { p1 if g1 => b1; … }` | `E(bi, σi)` for the first case whose pattern matches `E(s, σ)`, extending `σ` to `σi`, and whose guard holds under `σi`; a value no case matches gives `⊥`, an undeclared hole ([Holes](#holes)) |
| `hole h` | `⊥h`: the declared hole `h` is reached |
| `inbox send(c, q, m)` | `E(q, σ)` with the delivery `(E(m, σ), 0)` added: last for a FIFO channel `c`, at its catalog position for an unordered one ([Channels](#channels)) |
| `inbox isEmpty(c, q)`, `isFull(c, q)` | whether `E(q, σ)` holds no delivery; whether it holds as many as `c`'s capacity |

A hole propagates: an expression that evaluates an operand to `⊥h` or `⊥` has that value itself, the
first one in evaluation order. Short-circuit operators, conditionals and `match` evaluate only what
they read, so a hole in a branch not taken is not reached.

Patterns: a wildcard matches anything; `bind n p` matches what `p` matches and binds `n`; a literal
matches an equal value; `case T.C(p1, …, pn)` matches a value of case `C` whose fields match the `pi`;
alternatives match when any does.

## Machines

A machine's table is derived from its declaration:

1. The states are the catalog of its state type; the outcomes and facts the catalogs of its outcome
   and fact types (no facts when it names none).
2. Its action classes are, for each step binding, every assignment of the action's inputs in
   catalog order, keyed `name` or `name-key(i1)-…`; all classes sorted by key. Two bindings of one
   class are an error.
3. Its rows are states-major: for each state `s` and each class `c` bound to function `f`, the list
   `E(call(f, s, i1, …))`. An empty list is a disabled pair; otherwise the row keyed `key(s)-key(c)`
   has one result per step record, and a result state outside the catalog is an error. A value that
   is a hole makes the pair a hole row of it: neither disabled nor enabled ([Holes](#holes)). The rows
   of a channel's delivery and loss are derived as [Channels](#channels) says.
4. Its starts are its start expressions' values, and its ends the states its `ends` function accepts.
5. Its evidence is, for each fact in catalog order whose case has no line yet, the case name and
   `E(call(evidence, fact))`.
6. A refining machine's refinement is checked under `Umpire.Command.deriveRefinement`'s rule, as
   model/go applies it: every outcome is a product outcome of the same name, every start maps to a
   product start, and every row result is carried by a product row from the mapped source to the
   mapped target with the same outcome whose facts are among the result's (preferring the product
   action of the row's own name), or else its source and target map to one product state and it is
   a stutter. A refinement that names the facts the product sees (`visible`, a function from a fact
   to a Boolean) narrows both cases: the carrying product result also records, by the same-named
   key, every fact the result records that the product sees, and a stutter records none of them. A
   refinement that names none reads facts as the rule above does. A refinement that names the
   outcomes the product sees (`visible_outcomes`, a function from an outcome to a Boolean) narrows the
   stutter too: its outcome is none of them. A carried result's outcome is already the carrying
   product result's, by name.
7. Its monitors ([Monitors](#monitors)) and its assumptions ([Assumptions](#assumptions)) are the
   ones it names. A machine derived by restriction keeps its source's.

Reachability, stuck states, Definition IDs and the Behavior Fingerprint are then those of model/go's
`umpire.Table` over the derived keys.

## Channels

A channel declares a message type `T`, a capacity `n` of at least 1, an order, FIFO or unordered,
whether it is lossy, and a number of duplicates `d` of at least 0. `T` is a finite type: a named type,
the Booleans or an integer range. The lifter takes an `Int` message's range from the `Finite.upTo(n)`
the channel is declared with, `0..n`, and an opaque type's from its own given; it refuses any other
`Int` catalog, a list of messages, and an order or loss that is computed rather than named.

What it holds is a list of deliveries `(m, r)`, with `m` of type `T` and `0 ≤ r ≤ d`, at most `n` of
them. Its entries, in catalog order, are each message of `T` in catalog order and, for each, the
redeliveries `0` to `d`. Its catalog is the empty list, then every list of one entry, of two, up to
`n`, as the product of the entries with the last varying fastest. An unordered channel's catalog keeps
only the lists whose entries are in catalog order, so that one multiset is one value, and a send
inserts at the position that keeps it so.

A state field of type `channel c` holds `c`; a state type holds each channel in at most one field.
For a machine whose state holds `c` in field `f`, the action that delivers `c` (`delivers = c`) has
one input, `message`, of type `T`, and the row of state `s` and message `m` is derived:

1. The delivered entry is, for a FIFO channel, the first entry of `s.f` if its message is `m`, and
   for an unordered one the first entry of `s.f` whose message is `m`. With none, the pair is
   disabled.
2. `s'` is `s` with that entry removed from `f`, and `R` the value of the bound function at `s'` and
   `m`: what receiving the message does. An empty `R` is disabled, as the receiver does not take the
   message; a hole makes a hole row.
3. The results are `R`'s, and then, when the entry's redeliveries `r` are fewer than `d`, `R`'s again
   with the entry `(m, r + 1)` put back into each result state's `f`, first for a FIFO channel and at
   its catalog position for an unordered one, explained as "the channel delivers the message again":
   the receiver's step happened and its acknowledgment was lost.

The action that loses a message of `c` (`loses = c`) is bound exactly when `c` is lossy. Its row of
`s` and `m` removes the first entry of `s.f` whose message is `m`, and its results are the bound
function's value at the resulting state and `m`: what losing the message does. With no such entry the
pair is disabled.

A send to a full channel gives contents outside the catalog, so a row that keeps them lands outside the
state domain (Machines 3).

## Holes

A declared hole names behavior the Model leaves unknown on purpose. A hole row is neither disabled,
as an empty list of steps is, nor an error. A check that explores a hole row reports its result
incomplete, naming the hole and the row, and keeps every result the rest of its exploration
establishes; a violation it found stays a violation. A hole in a row no explored path reaches affects
nothing.

`⊥`, a value no `match` case matches, is an undeclared hole: a check that reaches it reports its result
incomplete and names the row, and no declaration accounts for it. A malformed Model is never a hole:
what a reader rejects before any check is listed under [Admission](#admission).

## Monitors

A monitor declares a finite state type `M`, an initial value, a function `next` from its state, the
machine's state before a step and the step record to its state after the step, a function `violated`
from its state to a Boolean, and its evaluation point. It watches every machine that names it, whose
state type the function's second parameter is.

- Every step advances every watching monitor: taking result `r` from state `s` turns its state `μ` into
  `next(μ, s, r)`. A monitor reads steps and never disables one: a machine's rows are the same with and
  without it.
- The state a check explores is the machine's state together with every watching monitor's state, so
  two paths to one machine state whose monitors differ stay two.
- Its verdict is read only at its evaluation point: after every step (`every_step`), at the end of a
  path in a state the machine may end in (`at_ends`), or after each step whose record the `after`
  function accepts. It is violated where `violated(μ)` holds there, and a path on which it is never
  read gives it no verdict.
- Two monitors of one Model do not share a name.

## Assumptions

An assumption is a name every result of a check that relies on it carries. A machine's `assumes`
lists the ones every check of it relies on: that it stands for an opaque provider, or that it
includes a fault only the assumption allows. An assumption's `fair` actions are weakly fair under it:
a path a check of a progress claim considers does not keep one of their classes enabled at every
state of a suffix without taking it there.

## Compositions

A composition is one Model from machines of different entities, as model/go and model/scala compose
them: its state type is a record with one field per member; a member's own action class is keyed
`<field>_<class>` and steps its member alone; a sync pairs two members' actions into one step keyed by
the sync's name followed by each class's inputs, whose results are the product of the members'
results, with the first member's outcome and both members' facts, each keyed `<field>_<key>`; a
composed state is keyed by its members' state keys joined by `_`; only the rows reachable from the
members' starts belong to it; its `ends` is a predicate over the composed state; and its Definition
IDs hang off the owner `compose-<name>`.

A member that `replaces` a machine stands in for it within this composition: a detailed provider in
place of an opaque one. Its machine declares a refinement of the replaced machine (Machines 6) that
holds, and a check of the composition relies on none of the replaced machine's assumptions.

## Claims

A Property belongs to a machine or a composition. A same-step Property holds of the step record of
each step it is about: every step, the steps of one class (`when_class`), or the steps of every class
of one action (`when_action`). A transition Property (`transition`) holds of the state before a step
and the step record.

A Scenario belongs to a machine or a composition: a start state and a pinned schedule of classes, of
composed class keys for a composition, or `free`, any action at every step.

A Query asks whether its Property holds on every trace of its Scenario (`verify`) or finds one trace
on which it holds (`find`), within its Limits: `steps` bounds the depth, `actions` the length of a
pinned schedule, and `search` the product states the search may visit. It is answered as model/go's
`umpire` search answers it, whose Limit Reached proves nothing. A Query with `through` reads a
Property of the machine the Scenario's machine refines through that refinement: a state by its map,
an outcome and facts by name. Two Properties, Scenarios or Queries under one key would share a
Definition ID.

## Progress

A progress claim of a machine says that from every reachable state `from` accepts, a state `to`
accepts follows within `within` steps, on every path its assumptions admit. It is violated by a path
from such a state that reaches a state with no row before a `to` state (a deadlock); by a cycle
reachable from such a state, on which no state `to` accepts and every class a named assumption makes
fair that is enabled throughout the cycle is taken on it (a fair non-progress cycle); or by a path of
`within` steps from such a state with no `to` state after it (the deadline missed). These are reported
apart. A search that stops at a Limit before a witness is complete establishes none of them: a finite
prefix that ends open is not a counterexample.

## Admission

A reader rejects, before any check and at the position the IR gives, a Model that:

- is of a version it does not know, or uses a declaration or a field it does not implement;
- names a type, function, action, machine, composition, channel, monitor, assumption, hole, Property
  or Scenario it does not declare, or calls a function or binds a step at another arity;
- gives a state field or a monitor state a type with no finite catalog: a list, or an integer with no
  range;
- declares a channel of capacity below 1 or duplicates below 0, or with a message type that has no
  finite catalog, a state type that holds one channel in two fields, a delivery or loss of a channel the machine's state does not hold, a lossy channel whose
  holder binds no loss, or a loss of a reliable channel;
- declares Limits below 0, or a progress claim within fewer than one step;
- names the facts or the outcomes a refined machine sees on a machine that refines none, or a member replacement that
  names no member or whose member does not declare a refinement of what it replaces;
- pairs a Property with a Scenario of another machine, other than through the Scenario machine's
  declared refinement of the Property's machine;
- declares two monitors of one name, or two different declarations under one Property, Scenario,
  Query or progress key;
- contains a function that calls itself, directly or through others.

The lifter refuses the ones it can see at the Scala line that declares them; a Model written some
other way meets the same rules at its reader.

## What goir implements

The rules for the [fn-107 specimens](specimens/README.md)' constructs are above: channels and the
redeliveries derived from them, monitors, assumptions, holes, scoped replacement in a composition, the
visible projection of a refinement, Claims and progress. An internal action (`internal`) is a step of
the system that is not a timer. `goir/` implements Values through Machines 6 without the visible
projection. It refuses a Model that reaches a hole or holds a channel, as an expression or a type of
no known kind, and it still reads a Model's monitors, assumptions, compositions, Claims and progress
claims without applying them, which Versions forbids a conforming reader; task 3 of fn-107 implements
or refuses them. Of the [Admission](#admission) rules, goir's `Validate` checks only that names are
declared and bound and that arities match; the version, finite catalogs, bounds, duplicates and the
rest are checked by the lifter at the Scala source, and a Model written another way meets them only
once task 3 adds them to goir.

Two narrowings came from the specimens' evidence in `specimens/README.md`:

- **Machines 6.** A stutter records no fact the product sees only where the refinement names what the
  product sees. `activityProtocol` names nothing, so its 240 stutters that record a product fact
  (finding F3) are read as before.
- **Expressions, `match`.** A value no case matches is an undeclared hole, apart from the admission
  errors a reader rejects and from declared holes (finding F4).
