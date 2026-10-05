# Semantics of the Umpire IR

The IR's meaning is defined here, not by the IR generator that writes it or the Go interpreter that reads
it. `proto/internal/temporal/server/api/umpire/v1/ir.proto` is the schema;
`tools/umpire/model` is the Go evaluator of these rules, and its goldens hold what it derives.

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
`None` and `Some(value)` in that order. The IR generator names it `scala.Option[T]`.

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

## Levels

An expression is at one of five levels, by what it is computed from:

- a **value** is computed from constants and its arguments, none of them a state or a step record;
- a **reading of a state** is a value computed from one state of a machine or a composition;
- a **step** is the list of step records one action class gives from one state: what may happen
  next ([Machines](#machines) 3);
- a **reading of a step** is a value computed from one step record, and from the state before it;
- a **claim over a path** says what holds of the paths a check explores.

A higher level reads the lower ones: a step reads its state, a reading of a step reads the step's
`state`, and each calls functions of values. Each place of a declaration takes one level:

| Level | Places |
| --- | --- |
| value | a machine's starts and a Scenario's start; a machine's evidence, of a fact; what a refinement names visible, of a fact or an outcome; a monitor's initial state, and its `violated`, of the monitor's state |
| reading of a state | a machine's and a composition's `ends`; a refinement's map; a progress claim's `from` and `to` |
| step | the function a step binding names, and each function it calls that gives steps |
| reading of a step | a same-step Property's `holds` and a transition Property's; a monitor's `next` and `after` |
| claim over a path | a Query, which reads a Property on the paths of a Scenario within Limits ([Claims](#claims)); a monitor's verdict ([Monitors](#monitors)); a progress claim ([Progress](#progress)) |

Only a step makes a step record. A `construct` of `umpire.Step`, or a call of a function that gives
step records, belongs to the function a step binding names and the functions it calls; anywhere
else it is at the wrong level, as an `ends` that asks whether a step function gives a step, or a
Property that calls the step function on the state after its step, would be. The IR does not mark
levels; the IR generator keeps them, refusing at its line an expression that makes a step in a function
that gives anything else, in a function's precondition or in a declared value. Its fixtures hold one
refusal for each place whose Scala type admits one: a machine's start, `ends`, evidence and
refinement, a monitor, a precondition, a same-step Property, a transition Property, a claim
pattern, a progress claim, a composition's `ends` and a Scenario's start (`level*` in
`model/irgen/testdata/lifts/Rejects.scala`). The Scala types rule out every other wrong level, so
it has no fixture:

- a value reads no state and no step record: these enter an expression only as parameters of the
  places that read them, and a start, evidence, what a refinement names visible and a monitor's
  initial state and `violated` have none;
- a reading of a state reads no step record: it has none as a parameter, and may not make one;
- no expression reads a claim over a path: no Scala value of a Property, a Scenario, a Query, a
  progress claim or a monitor holds whether it holds, and a Query's total and Limits are literals
  ([Query totals](#query-totals)).

Umpire's claims over a path are safety claims, a Query that asks whether a Property holds on every
path of its Scenario or finds one on which it does and a monitor's verdict, and bounded progress:
TLA's leads-to `from ~> to`, cut to `within` steps, on the paths the weak fairness (`WF`) of the
classes its assumptions make fair admits. Any temporal operator Umpire adds takes its meaning from
TLA, read over the paths of the machine's table, and an unbounded one takes the TLA operator's name:
`eventually` is TLA's `<>`, never another spelling of a progress claim.

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
6. A refining machine's refinement is checked under this rule:
   every outcome is a product outcome of the same name, every start maps to a
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

Reachability, stuck states, Definition IDs and the Behavior Fingerprint are then those of the
reader's table (`tools/umpire/model`, `Table`) over the derived keys.

### Rules

The IR binds each action to one step function; a Model writes when it fires and what it does
apart, and the IR generator lowers the two to that function (`model/umpire/Syntax.scala`, `Rules`).
A machine object's `rules` lists rules under headings: `when(g)` fires while the guard `g` of the
state holds, and `in(p1, …)` while the rules' phase projection, `Rules(_.phase)`, is one of the
phases listed. A rule names a whole action, `a ~> e`, or one class of it, `a(v) ~> e`, and an effect
`e` of the machine's `effects`, which says what happens and never whether: it gives at least one
step. `disabled(a)` binds an action no state enables.

The rules of one action lower, in the order written, to the step function
`<machine>.rules.<action>`: `if g1(s) then e1(s, i) else if g2(s) then e2(s, i) else Nil`, where
`gk` is the k-th rule's guard (for `in`, `List(p1, …).contains(projection(s))`). Where a rule fires
one class, the function first matches the inputs, one case per class in catalog order, and tries
there the rules that fire that class, so the state alone chooses among them. `disabled(a)` lowers to
a function that gives `Nil` for every state. The table derived from the lowered function is the one
Machines 3 gives; nothing else of the IR records the rules.

Rules of one action class are disjoint: no state and class are fired by two of them. The framework
checks it when the machine's rules are constructed, over every state of the state type's catalog
and every class, and refuses the second rule of an overlap, naming the machine, the class, both
rules by their place and heading, and a state where both hold; the model gate constructs every IR
file's roots, so an overlap fails it. A step that can go more than one way is one effect that names
each result with `choose` ([Named choices](#named-choices)), never two rules. Because the rules are
disjoint, their order changes no row: it fixes only the order of the arms of the lowered function.

A derivation rebinds rules in its source's place: `rebind(a ~> e)` keeps the guards (and classes) of
`a`'s rules and gives each the effect `e`; `rebind(when(g) { a ~> e })` replaces them; `extend` adds
rules for actions its source does not bind. `rebind(a ~> e)` is refused where it would merge rules
the source tells apart by their effects: two rules of the whole action, or two rules of one class,
with different effects. Rules that each fire another class may differ, since `e` reads the class's
inputs. A machine of the IR generator's core fixtures may bind
its step functions by hand, `object rules extends Bindings(a ~> f, …)`, which is what rules lower to.

### Modalities

A machine's table says what may happen, and the claims about it what must. A row is permission with
fixed results: its class may happen in its state, with exactly the results the row lists, each named
choice among them. A disabled pair is prohibition for an action the system takes, a timer or an
internal action: the system does not take it there. For an action of a party, such as a caller's
request, it is silence: the party can always try it, and the Model does not say what the system
answers, as a row with a rejecting outcome and its `because` would. A hole row is neither
([Holes](#holes)). Obligations are the claims: a same-step Property is a postcondition on the results
of its class's rows, read only where a row exists; a progress claim says that a state must follow;
and an assumption's fairness obliges a class that stays enabled to be taken. A refinement narrows
permission, since every row of the refining machine is carried by a row of the machine it refines or
is a stutter (Machines 6), and does not by itself preserve obligation: a Property of the refined
machine holds of the refining one only where a verify Query reads it `through` the refinement, and a
progress claim only where the refining machine declares its own. Lint (`tools/umpire/lint`) prints
each machine's table in these terms and reports where they leave a gap as specification holes: a
pair disabled by a wildcard arm, a party's request the Model is silent on, a result no claim
constrains and a Property only a `find` asks.

### Named choices

A step that can go more than one way may name each of its results. The name rides on the step
record's construct: `construct(umpire.Step, "", outcome, state, facts, because)` with `choice` set
to the alternative's name, as the IR generator writes each alternative of a Scala `choose`. An alternative
that calls a function is lifted as a call of a copy of it, `<function>$<choice>`, whose every step
record carries the name; the choose is then the concatenation of its alternatives' lists in the
order written. The IR generator writes every branching of a Model this way: it refuses several results
written without names. Evaluation ignores the name. The construct's value is the four-field record
above, and the name is metadata the record carries: no key, equality, Property, monitor or
refinement reads it, and it travels with the record through lists, `let`s, calls and branches. A
row's result reports the name of the step record it was read from, empty for an unnamed one; the
redelivery a channel derives from a result ([Channels](#channels)) is unnamed, as its explanation is
its own. The table's rows, their results and their order, the Definition IDs, the Behavior
Fingerprint, Query answers and lowered Cases are the same with and without names, and so are
exploration and Model identities for names written on steps. A named call adds the copies it calls,
which enter those content-hashed identities as any change of a Model's functions does. An
alternative is a result of its action class and never another class, so the names do not enter a
Query's total ([Query totals](#query-totals)). A `choice` on a construct of any other type is an
error of the Model, and so are two results of one row with the same name. The Quint export writes
the name on the step record of the ordered result list (`f_choice`), and its checker action, not the
step function, picks one result of that list.

## Channels

A channel declares a message type `T`, a capacity `n` of at least 1, an order, FIFO or unordered,
whether it is lossy, and a number of duplicates `d` of at least 0. `T` is a finite type: a named type,
the Booleans or an integer range. The IR generator takes an `Int` message's range from the `Finite.upTo(n)`
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

A hole reached inside a function a claim reads, a Property's `holds`, a monitor's `next`, `violated` or
`after`, or a progress claim's `from` or `to`, is unknown evidence of the step or the state it was read
on: the claim neither holds nor fails there. A search does not continue through such a step, a
progress check explores no step from such a state and counts it in no path, and the result is
incomplete unless it found a witness elsewhere. A hole reached outside any step leaves unknown what
it was reached in: a start, an `ends` or the evidence of a machine leaves that machine without a
table, and a refinement's map or what it names visible leaves that refinement neither held nor
rejected. The results of what depends on it, the machine's own claims, the compositions it is a
member of and the Queries that read through its refinement, are incomplete. Every other machine is
checked as if the hole were not there.

What a refinement names visible only narrows which steps carry a row and which rows are stutters. A
refinement rejected with a fact or an outcome of unknown visibility read as unseen is therefore
rejected whatever the hole hides, and the rejection stands with the hole beside it. One that holds
only that way is incomplete, and a Query reads through it as through one a hole row leaves unknown.

A declaration read at one value after another, an `ends` at each state or what a refinement names
visible at each fact, is read on past a hole. A result at a later value that makes the Model malformed
is the declaration's error whatever hole came before it, and a result names every hole reached.
A machine's starts, its `ends` and its evidence are one such reading, and so are a refinement's map
and what it names visible, as far as the check reads them. A composition reads every member: a
malformed member is its error, then a member's rejected replacement, then a ceiling, and only then
the holes of its members, all of them.

A refinement by a machine whose starts reach a hole row is incomplete, not held: the steps the hole
stands for are not shown to refine anything. A rejection found on its rows stands. A Query that reads
through such a refinement is answered over the machine's rows; a violation it finds stands, and
otherwise its result is incomplete by those holes.

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
- Whether a member machine's monitors watch a composition is not defined. A reader refuses a Query over
  a composition a member of which names monitors as unsupported, rather than answer it unwatched.

## Assumptions

An assumption is a name every result of a check that relies on it carries. A machine's `assumes`
lists the ones every check of it relies on: that it stands for an opaque provider, or that it
includes a fault only the assumption allows. An assumption's `fair` actions are weakly fair under it:
a path a check of a progress claim considers does not keep one of their classes enabled at every
state of a suffix without taking it there.

## Compositions

A composition is one Model from machines of different entities: its state type is a record with one field per member; a member's own action class is keyed
`<field>_<class>` and steps its member alone; a sync pairs two members' actions into one step keyed by
the sync's name followed by each class's inputs, whose results are the product of the members'
results, with the first member's outcome and both members' facts, each keyed `<field>_<key>`; a
composed state is keyed by its members' state keys joined by `_`; only the rows reachable from the
members' starts belong to it; its `ends` is a predicate over the composed state; and its Definition
IDs hang off the owner `compose-<name>`.

Its starts are the product of every member's starts in member order, the last member varying fastest.
A claim of a composition reads a composed step as a step record whose `state` is the composition's
state record, one member state per field, and whose `outcome` and `facts` are strings: the composed
keys `<field>_<key>`.

A member that `replaces` a machine stands in for it within this composition: a detailed provider in
place of an opaque one. Its machine declares a refinement of the replaced machine (Machines 6) that
holds, and a check of the composition relies on none of the replaced machine's assumptions. Within
the composition that refinement also reads every start of the replaced machine: a start of it that no
start of the member reads as is a behavior the member does not provide. The member's own assumptions,
and every other member's, stay: an assumption the member declares is the composition's also where the
replaced machine declares one of the same name, since the replaced machine is no member and none of
its assumptions is read.

## Claims

A Property belongs to a machine or a composition. A same-step Property holds of the step record of
each step it is about: every step, the steps of one class (`when_class`), or the steps of every class
of one action (`when_action`). A transition Property (`transition`) holds of the state before a step
and the step record.

A Property of a composition is about composed classes. Its `when_class` is the class keyed by the
action's name followed by the inputs, which a sync named as the action it takes spells; its
`when_action` is every class whose key begins with that name, a sync's name or `<field>_<action>` for a
member's own action. A member's action a sync takes steps only with its pair, so it has no class of its
own and neither names it.

A Scenario belongs to a machine or a composition: a start state and a pinned schedule of classes, of
composed class keys for a composition, or `free`, any action at every step.

A Query asks whether its Property holds on every trace of its Scenario (`verify`) or finds one trace
on which it holds (`find`), within its Limits: `steps` bounds the depth, `actions` the length of a
pinned schedule, and `search` the product states the search may visit. It is answered as the
reader's search answers it, whose Limit Reached proves nothing. A Query with `through` reads a
Property of the machine the Scenario's machine refines through that refinement: a state by its map,
an outcome and facts by name. Two Properties, Scenarios or Queries under one key would share a
Definition ID.

A Query with `through` is answered only where that refinement holds: over a refinement that is
rejected its result is the rejection. Only a `verify` reads through a refinement, and a transition
Property is about every step, so a `find` with `through` and a transition Property with a `when` are
unsupported.

A Property's and a Scenario's Definition ID is formed from the family and the name alone, so two
machines of one family that each declare a claim of one name share it. A result names a claim by its
family, the machine or composition it is declared on, and its name, which keeps the two apart.

## Query totals

A Query's `total` is its author's count of its static combinations, which the reader recomputes and
holds the author to. For a pinned Scenario it is the Scenario machine's whole state catalog times the
scheduled slots within the step limit, `min(steps, scheduled classes or keys)`. For a free Scenario it
is the state catalog times the machine's finite action-class catalog, one class for every input
assignment of every bound action (a composition's: each member's classes no sync takes plus every
pair of classes each sync takes), times `steps`. A Query read `through` a refinement counts its
Scenario machine's states and classes, not the refined machine's. The count is taken before anything
is reached: unreachable states, disabled rows, repeated states and an early answer all still count,
and a named choice's alternatives are results of one class, not classes of their own. It is a review
and capacity figure, not a prediction of the paths the search visits or the steps a Run executes. A
step limit of 0, or an empty pinned schedule, counts 0. `total` is metadata: no table, fingerprint,
Definition ID, answer, lowering, Case or exploration identity reads it. It is unset only in IR lifted
before it existed; every Query a current Model declares states one, and an exploration candidate
whose schedule differs from its source Scenario's is counted again.

## Progress

A progress claim of a machine says that from every reachable state `from` accepts, a state `to`
accepts follows within `within` steps, on every path its assumptions admit. It is violated by a path
from such a state that reaches a state with no row before a `to` state (a deadlock); by a cycle
reachable from such a state, on which no state `to` accepts and every class a named assumption makes
fair that is enabled throughout the cycle is taken on it (a fair non-progress cycle); or by a path of
`within` steps from such a state with no `to` state after it (the deadline missed). These are reported
apart. A search that stops at a Limit before a witness is complete establishes none of them: a finite
prefix that ends open is not a counterexample.

A hole row is not a step and not a disabled pair: a state whose only pairs are hole rows is no
deadlock, and a cycle is no fair one while a fair class it does not take has a hole row at one of its
states. A kind of violation ruled out while the check read a hole row is incomplete.

## Realizations

A realization says how the find Queries of one machine run against a system. It declares:

- **roles**, the participants commands and activations address, with the environment bindings a run
  supplies for them;
- **learned values**, each a text or a handle, bound once by one command and read by the commands
  that depend on it;
- **observations**, each one protobuf message a run records;
- **kinds of evidence**, each the recorded data that confirms the facts the machine's evidence
  function names: a member of the history event's attributes, the elements of a repeated field a
  unary method returns, the one message at a path of a unary method's response, or the Run's own
  record of one command's events of one kind, where a guard over the event's payload holds, which,
  where it is what a worker reports of an activation, is declared the record of one attempt, by its
  number, of the activity one script runs; the
  field path that keys it to its operation, or, for the Run's record, the source's key, the run's own
  id or a path of the payload; what it commits to, what a caller was told or a durable commit of the
  receiver; the fields of the recorded data it keeps, each with the identity it names, if it
  names one: the operation, the attempt or the delivery; whether its source is exhaustive; and the
  steps of a path one piece of it confirms, where it names them, each the n-th step of a class,
  counted from one;
- a **correlation**: the fields that scope evidence to its run and name its operation, the
  observation that carries it, and the window a check keeps;
- **controls**, the actuators a run needs beyond its commands: one that holds the deliveries of a
  channel, or one that holds what a step of a class dispatched, which names the task-queue role
  whose deliveries a run holds it through;
- **scripts**, each an ordered list of items run by the controller or by a worker for each workflow,
  Nexus operation or activity that activates it. An activity's script names the classes the delivery
  of an attempt to its worker is.

A script item is a command every Case carries, a command with `when` classes, or a list of
performances, each the command that performs one class of an action. A command names what it does,
the commands of its script it runs after, a deadline, and whether it runs whatever became of those
commands. Without `after` it runs after the item before it; with it, commands that name the same
predecessors and not each other are independent branches. A command may name the exhaustive kinds of
evidence its read closes. A command holds what a control holds, or releases it: the hold is done once
the held thing is held, and the release once what it let go was delivered and what the receiver
committed for it is observed, which the release's own record carries. A message a command writes out names its protobuf type and the fields it
sets, and a field it does not name stays unset.

Scala writes the three modes of an item as `everyCase(command)`, a command;
`onPath(classes*)(command)`, a command with `when`; and `perform(step -> command, …)`, performances.
A command's id is the name of the `val` that declares it, in kebab case (`val pauseActivity` is
`pause-activity`), unless it is written out with `Command(id, …)`. `call.withFields { … }` is the call
with assignments appended, under the call's command id, so the commands of one call that
performances bind read as one name; commands of one script share an id only as performances, never
with a command that is not a performance. A declaration other declarations refer to by value, a
role, script, actuator, learned value, kind of evidence or command, is written as its id, and a fact
as its name. A status table, `statusTable(fact -> value, …)`, is read when the IR generator lifts: a
lookup `table(fact)` writes the value the table lists for the fact, and the table itself adds
nothing to the IR. None of these forms has an IR field of its own.

`Fault(role, admissionResponseLoss)` releases a previously held activity dispatch and loses one
committed admission response. The Driver waits for the same request's retry to confirm the response
replacement was consumed, then reports the original durable admission. Its successful outcome is one
`FAULT_INJECTED` event, with the instruction's causal coordinates; arming, refusal, cancellation and
uncompleted loss are not successful faults. Scala supplies the budget, allowed uncertain outcomes,
recorded evidence and expected assessment. Removing durable evidence does not turn a missing
response into a rejected admission.

An activity's script answers its attempts: its commands end an attempt with a result, fail it with a
failure, or answer it as canceled, and the commands a path places in it are the activity's attempts
in path order. The
delivery of an attempt is the script's activation, so a step of a class the script starts with is
performed by no command.

A realization may also carry the system's **API behavior** and its **server steps**
(.plans/API_BEHAVIOR_HINTS.md). The behavior is facts about the system's calls, not about the
machine: a **visibility** says when the effect of a write, a method or a kind of asynchronous cause
(an activity's answer, a workflow task, a Nexus handler's reply, a delivery, a timer), is visible to a
read, a method, at once or eventually within a wait bound; a **cause bound** says how long one kind of
cause may take, as an interval a wait looks at and the most it waits. A server step names a class of
the machine that no command performs, such as an activity's delivery or a timeout, with the kind of
cause it is and, for a timer, the deadline the realization set. Each hint has an id, derived from
what it relates (`visibility.<write>.<read>`, `cause.<kind>`), and its declaration's position. The
behavior also says how the system numbers an activity's attempts (from a positive first number, and
whether every attempt is of the activity's one run), the limits of an instruction that writes none,
and whether a run's record order is the causal order of one operation's evidence across sources;
the lowering writes each into the Case's Program as declared, and admission refuses a first attempt
number or a limit of zero or less. No table, ID, fingerprint, Query answer or Contract reads
either; they shape only a lowered Case's Program, and conformance reads whether attempts are of one
run. Admission refuses a hint with no id or one another hint took, a
visibility that names no write or no read or a pair another names, a cause bound of no known kind or
of a kind another bounds, a cause bound with no bound, a bound or interval of zero or less, an
interval greater than its bound, and a server step that names no class of the machine, a class
declared twice or one a command performs, a kind the realization bounds no cause of, a timer with no
positive deadline and a deadline on a step that is no timer; each at its declaration's position.
Whether the API binds a write to POST and a read to GET is read from descriptors where waits are
derived, by the lowering, which refuses a visibility that names otherwise at its position, and a
command that calls a method bound to neither.

The lowering derives a read's wait from the behavior. A read is a poll, or a call bound to GET that
reads its response; a write is a call bound to POST, or a command of a worker's script, which is the
kind of cause its activation is (an activity's answer, a workflow task, a handler's reply). A script
synchronizes at each read. A poll that writes no interval waits as derived: if a write of its own
script performs the step its evidence confirms, that write's visibility decides, and the poll reads
once when it is at once and polls within the visibility's bound when it is eventual; otherwise it
polls within the sum of the bounds of every asynchronous cause between its script's last
synchronization and that step (a timer's bound is its deadline and then its slack), plus the
visibility's bound where the write that performs the step is eventual, at the smallest interval
among them. The Case's wait names each hint with its position, and its timeout is their sum. The
lowering refuses, at the read, a write in that window with no declared visibility to the read's
method, naming both; a step no command performs and no server step declares; and a cause whose kind
the realization bounds not. A call that reads is checked the same way but never waits: one after an
eventually visible write is refused. A closing read and a poll that writes its own interval check
nothing; in a realization that declares a behavior, such a poll is lint's `explicit-wait` finding,
kept only where an acceptance records why no hint covers its wait. A poll left to derive its wait
that writes a deadline is refused. A realization that
declares no behavior has its calls that read taken as written.

A realization carries no Case and decides nothing a Query's path decides. Lowering a find Query
through it is the same for every realization:

1. The Query's witness is the path: the classes its pinned Scenario lists, with the steps the search
   found.
2. A script becomes one entrypoint. A plain command is carried by every Case; a command with `when`
   by the Cases whose path takes one of those classes; a performance by the Cases whose path takes
   its class, once per step that takes it, in path order, a class taken again under its ordinal. A
   step of the path that a party takes and neither a performance binds nor an activity script starts
   with is refused; a step of the system needs no command.
3. A command that reads a learned text runs only where every one it reads is bound.
4. A step of the path is confirmed by one kind of evidence, and one piece of a kind confirms each
   step once. A kind that names a step confirms it, and must record one of the step's facts. Any
   other step is confirmed by the kind that records a fact of its class, the first the path meets,
   among the kinds that name no step. A kind that names several steps confirms them all by one piece
   of evidence, with the steps between them that record nothing: it confirms every step it names or
   none, and no other kind's evidence lies between them. A class the path takes again, and a second
   step that records a fact another step's kind already confirmed, therefore each need a kind that
   names the step; a step that records a fact evidence names and is left with no kind of its own is
   refused, and so is a kind two steps would both be confirmed by. A step that records nothing the
   evidence names is confirmed with the next step that does, and listed as a Known Gap. A kind is
   carried where it confirms a step of the path or records another result of a row the path takes,
   and an exhaustive kind is carried by every Case: where it confirms nothing of the path it is
   declared, lifted by its closing read, and given no meaning in the Contract.
5. The Property is lowered to the clauses that say the same thing over the machine's table: over the
   steps its `when` admits, a state, an outcome and facts are fixed where every accepted step carries
   them and the predicate rejects every accepted step with them changed. Each clause is due within as
   many of the operation's transitions as the Scenario places before the Property's action. A
   predicate the clauses cannot carry exactly, and a transition Property, are refused.
6. The Contract's transition table is the machine's rows under the actions the evidence confirms,
   from the Scenario's start.

A verify Query is searched and realizes nothing. A find Query whose machine declares no realization
has none to be lowered through. A Query with no witness is no Case.

A value a command or a source computes is typed. A condition is a presence test, a comparison of
two values of one type, an order of two numbers, the negation of a condition, or a conjunction of
conditions, which is read left to right up to the first that does not hold. Presence is of a message
or of a oneof member: a scalar a message holds is present whatever its value, so a condition that
means a positive number or a text that is not empty says so. A path is of a message, to any depth: a path of a
path is read against the message the inner path reaches. A guard of the Run's own record is a
condition over the event's payload and reads nothing else: it writes out flags, numbers, texts and
enum values, and no name a Case binds, since a recorded Run holds no such binding. Its paths name
fields and oneof members, each one flag, text, enum value, signed integer or message. Its key is the
run's id or one text or integer of the payload. What a well-formed guard is has one reading
(`GuardProblem`, over `TypeOf`, in `tools/umpire/model`), which admission makes with no descriptor, the lowering
with the payload's, and the evaluation on a recorded Run with the payload's again, so a guard one of
them refuses is refused by all, in the same words, by the first that can know. A guard that cannot be evaluated on an event, because it reads a field the
payload's type does not have, compares or orders a value that is absent, or is no condition, is an
error at that event and is never read as a guard that does not hold. A value the payload does not
hold keeps the type its descriptor gives it, so what is wrong with a guard's types is an error on
every event, and only the absence of a value it compares depends on the event.

Scala authors name protobuf messages by generated types, unary methods by generated method
constants, and fields and enum values through typed selections. The IR generator writes their descriptor
names into the same IR paths and operands defined above: nested fields use dots, repeated elements
use `[*]`, and a oneof arm uses `<member>`. A constant message still writes only its declared
fields; a map key such as `Payload.metadata`'s `encoding` is data, not a field selection. A Run
Event's payload arrives dynamically, so its author names `InstructionOutcome` as the root of
its guard and key selections. These types constrain authoring but do not change the IR's meaning.
Scala does not instantiate or send Temporal messages. Go independently validates descriptors and
paths when it lowers an IR, including one edited by hand or emitted by an older IR generator, and Testpilot
executes the resulting Case.

What a Run's evidence says of the machine is the same for every reader that assesses a Run against
it:

- Evidence proves what it reports and nothing of what it does not: a step that records a fact no
  observation reports may still have happened. The one exception is declared. A kind of evidence is
  `exhaustive` when its source reports every occurrence of the facts it records, for the operations
  its closing read covers, and one command of the realization, which every Case carries, names the
  kind in `closes`. The Run's own record of a command is closed by that command, which may perform a
  step: a Case that does not carry it has no record of the kind, and nothing is inferred from it. On a Run that closed complete, where that command's last completion is a success
  and the ordinals of the kind's source are unbroken, a step that records a fact of a kind the Case
  carries has an observation of it, and a step taken without one did not happen. Short of any of
  that, and for a kind the Case does not carry, absence says nothing.
- A field with the role `attempt` or `delivery` names the attempt or the delivery its evidence
  belongs to: two observations that name different ones are not facts of one step. A field with the
  role `operation` repeats the operation the evidence is keyed to, and evidence whose field and key
  differ is crossed and is read as neither. A redacted field is carried without its value and names
  nothing.

A reader lowers a realization only into what its runtime can run. What a realization declares that
the runtime has no primitive for is reported with the declaration's position, and the Query has no
Case: nothing is lowered around it. A realization and a path are read whole first: what is wrong with
either is an error, every one of them, and is reported before any such gap: what the realization
writes against its descriptors, a step no command performs, a Query with no witness, a Property
that does not lower, and, of a realization sound in itself, everything the producer decides before
it writes a Case, such as a fact of the path no kind of evidence records. A Property whose
predicate cannot be read on a step, because it reaches a hole there, is not lowered: a step it cannot
be read on is not a step it rejects.

## Results

A check's result is one of these, which are kept apart:

- `admission-error`: the Model is rejected before any check ([Admission](#admission));
- `declaration-error`: a declaration cannot be read as the IR says it is. A function a declaration
  reads as a Boolean, a machine's or a composition's `ends`, a refinement's `visible` or
  `visible_outcomes`, or a claim's function, that returns another value is one, at that declaration,
  and so is a monitor state outside the monitor's state type. It is never read as `false`;
- `resource-limit`: a ceiling of the scope refused the work before any of it was done;
- `limit-reached`: a search's `search` Limit cut it; `unresolved`: a progress check's depth left
  reachable states unexplored. Neither proves anything;
- `refinement-rejected`: a refinement does not hold, with the rule it breaks and a witness;
- `counterexample`, and `found` for a `find`: a witness, which holes the check also read do not take
  away;
- `verified-within-limits`, and `not-found` for a `find`: the claim holds on everything the scope
  reaches, and the check read no hole;
- `incomplete`: the check found no witness and read a hole, which it names with the row. A hole no
  explored path reaches, one past the step bound and one a pinned Scenario does not schedule are not
  read;
- `unsupported`: the reader does not check the declaration. It is listed, and is no check;
- `replay-failed`: a witness did not replay through a second interpretation of the Model. It is an
  error, whatever the result said.

A result names the table it read by its Definition ID and Behavior Fingerprint, the limits it ran
within, the assumptions it relies on, and the work it took. No source position enters a result but in
where it points.

Where one declaration's result is made of several, the members of a composition, a search and the
refinement it reads through, a result and the replay of its witnesses, it is the one of the highest
precedence, the first of several: an error (`replay-failed`, `admission-error`, `declaration-error`),
then a violation (`refinement-rejected`, `counterexample`), then a limit (`resource-limit`,
`limit-reached`, `unresolved`), then `found`, then `incomplete`, then `unsupported`, then what held. It
names every hole any of them read and keeps every witness, and a `verified-within-limits` or
`not-found` beside a hole is `incomplete`. A `found` witness is realized and is not undone that way: it
stands beside a result a hole leaves incomplete, with that hole named.

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
- declares a Query total below 0, one that is not the Query's static combination count, or one for a
  Query whose count is more than an int64 holds;
- gives an example to an action that takes other than one input, or an example that is no member of
  that input's type;
- names the facts or the outcomes a refined machine sees on a machine that refines none, or a member replacement that
  names no member or whose member does not declare a refinement of what it replaces;
- pairs a Property with a Scenario of another machine, other than through the Scenario machine's
  declared refinement of the Property's machine;
- declares a Property of a composition about a class, or about an action, that no class of the
  composition is of, or a Scenario of a composition over a key that is none of its classes;
- declares two monitors of one name, or two different declarations under one Property, Scenario,
  Query or progress key;
- contains a function that calls itself, directly or through others;
- declares a realization with no id or no name, two under one id or one name, one of a machine it
  does not declare, or one with no correlation; a
  role, learned value, observation, kind of evidence, control, script or command with no id, or two
  of one kind under one id; a kind of evidence that names no recorded kind, no source, no operation
  key, nowhere it is recorded or no commitment, or two kinds for one recorded kind that both name no
  step they confirm;
- declares a kind of evidence that confirms a step of no class or of a class the realization's
  machine does not bind, a step counted from below one, or one step twice; two kinds that confirm one
  step; an exhaustive kind that names the steps it confirms; or a kind that names its steps and
  records a fact an exhaustive kind records, since an exhaustive kind is the one kind of its fact;
- declares evidence that is the Run's own record of no known kind of event, of a script or a command
  the realization does not declare, with a key that is neither the run's id nor a path of the
  payload, with a guard that reads anything but the payload or writes out a name a Case binds or a
  value that is no flag, number, text or enum value, or with an operation path beside its key; or
  polls such evidence;
- declares the Run's record of an attempt of no script, of a script it does not declare or that no
  activity activates, of a script that starts with no delivery, of an attempt counted from below one,
  or as an event that is no diagnostic, or declares a diagnostic the record of no attempt;
- declares a field of evidence with no id or no path, two fields of one kind under one id, a field
  whose role is none it knows, a role on a redacted field, or two fields of one kind with one role;
- declares an exhaustive kind of evidence that no command closes; or closes a kind that is not
  declared or not exhaustive, one kind by two commands, a kind by a command that does not read it (a
  history kind is closed by the read that lifts it, the Run's own record of a command by that
  command, any other by a poll of it), or, unless the kind is the command's own record, by a command
  some Cases do not carry;
- declares a control that holds what a step dispatched and names no class its machine binds, or no
  task-queue role;
- fails an attempt, or answers one as canceled, in a script no activity activates;
- names, in a realization, a role, learned value, observation, kind of evidence, control, channel or
  command it does not declare, a role of another kind than its use needs, or a class the
  realization's machine does not bind;
- binds a learned value by two commands or by none, reads a text as a handle or a handle as a text,
  reads a learned value in a command that runs whatever became of the commands before it, or
  performs one class by two commands, or by a command and the activation of an activity script;
- keys a realization's runs and operations by one field, lifts evidence into an observation the
  correlation does not read, or observes a value into the one it does;
- orders the commands of a script in a cycle;
- declares a script item that is neither a command nor performances, or both; a command with no
  instruction or a deadline below 0; an operand or a written value of no known kind, or a conjunction
  of no operand; or a written message with no name or a field set twice;
- orders what is written out as no number, negates or joins what is written out as no condition,
  compares two written values of two types, two enum values it writes out, or a message, reads a
  path of what is no message, polls until what is no condition, or guards the Run's own record by
  what is no condition or by a path written as no field or oneof member.

The IR generator refuses the ones it can see at the Scala line that declares them; a Model written some
other way meets the same rules at its reader.

## What the reader implements

The rules for the specimens' constructs are above: channels and the
redeliveries derived from them, monitors, assumptions, holes, scoped replacement in a composition, the
visible projection of a refinement, Claims and progress. An internal action (`internal`) is a step of
the system that is not a timer. `tools/umpire/model` implements all of them: `Build` interprets Values through
Machines, Channels and Holes, `Validate` applies the [Admission](#admission) rules, and `Check` binds
monitors, assumptions, compositions, Claims and progress claims to its private checker and
gives each declaration one of the [Results](#results). What `Check` does not answer it reports as
`unsupported`: a Query over a composition a member of which names monitors, a `find` with `through`,
and a transition Property with a `when`.

`tools/umpire/lower` lowers a find Query through its [realization](#realizations) into a Testpilot Case
with its private producer. The Query it gives the producer is the one `Check` answers,
from the same binding (the reader's `Realizer`): one Property read on one step record, its explanation
included, over the machine's table with its hole rows as unknown pairs. That table also carries the
state fields and Abstraction Claims. The realization is translated declaration for declaration. It checks what a realization writes and reads against the protobuf
descriptors it names, at the declaration: a message, field, enum value or method the descriptors do not have, a
value of another kind than its field, a message written where another belongs, a path that does not
resolve, and a value observed into an observation of another message. It reads the path selectors
`field`, `field[*]` and `oneof<member>` and refuses the others. It lowers an activity's script to
the attempts of an activity entrypoint, a failing one as an attempt failure and never as a result
that is a failure, and refuses, where they are written, what Testpilot would refuse: a command of
that script that is no answer to an attempt, a failure that is neither an application failure nor
one of no kind, and a poll's condition that reads the run, its environment or a learned value. It
types a poll's condition and the guard, the key and the fields of the Run's own record against the
descriptor of what they read, the instruction outcome for the Run's record, and refuses, as an error
at the declaration and before any gap is said, a condition that is none, an order of what is no
number, a comparison of two types or with a name an enum does not have, and a key that is no single
text or integer. It lowers evidence that is the Run's own record to a Run Event declaration under
the lowered guard, at the controller's instruction the source names and keyed by the Run or by the
payload's path, and refuses such evidence of a command no controller runs; evidence read from one
message to a single read; a kept field to the Program's field declaration and the Contract's retained
field, as the text, flag or unsigned integer its descriptor makes it; and an attempt answered as
canceled to the instruction of that name. It lowers the hold and the release of a control that holds
what a step dispatched to the Driver's delivery controls of the control's task-queue role, which a
Profile admits only where its environment supplies a delivery control; and evidence of a durable
commit where it is the Run's own record, of the release that observed it. A machine's monitors are no
part of a Case: the prepared assessment reads each beside the Contract. Six things have no task that
owns them and are named as limits of the prototype. A durable commit read back through an RPC or from
history: neither reports a commit of the receiver. A control that holds the deliveries of a channel,
and a command that holds or releases one: a Driver holds only what a step dispatched to a task queue.
A redacted field: a lift reads a value for every field its evidence declares, so none
carries a field without its value. An attempt of an activity the path starts and gives no answer: an
activity entrypoint's instructions are answers, and none waits out a deadline. And the record of an
attempt that a Run would record out of the path's order, which the Contract reads evidence in. And
the record of an attempt in a Case that runs two activities: a Run records an attempt at the command
that carries it, by its number and under no script's name, and the one carrier of a Case carries
every activity the Case gives an instruction, so neither the Case's guard nor `admits` can tell the
attempts of the two apart. When a
Run records a piece of evidence is read from declarations and never from the kind of event it is.
What a worker reports of an activation is always declared the record of an attempt, so no such
evidence is placed by a guess.
Evidence declared the record of an attempt reaches a Run once that attempt is answered: with the
step of the path that is the script's answer of that number, before that step's own evidence. Any
other evidence is recorded by an instruction of the controller and reaches the Run as the controller
runs it. A record that confirms steps before other evidence and is answered after it, or confirms
steps after other evidence and is answered before it, has no Case. Neither has a path confirmed by
two kinds that are the record of one attempt, which the Run records as one Run Event, evidence of
one kind. A record of an attempt the path
never starts is an error. The lowered guard of such a record also says the record is of the attempt
its source names, after the source's own guard, so the runtime selects the record by the declaration.
For the evidence the controller records, the Case itself is checked: each kind that confirms a step
has an instruction that records it, and the instruction of a later kind is the same one or runs after
the instruction of an earlier kind, by the order the script is written in and the instructions it
names to run after; anything else is an error. Two orders are Testpilot's and are assumed, since no
declaration fixes them: that the completion of the call that carries an attempt is recorded before
the attempt's record, and that an attempt's record is recorded before the status read after its
answer. A Run that breaks either carries evidence the Contract refuses, and is incomplete, never a
wrong Verdict. Each gap is named where it was written. A monitor, a kind of
evidence and a control are the realization's, and stand in the way of every Query of it. A command
Testpilot cannot run, an unanswered attempt, a late record and a record among two activities stand in
the way of the Queries whose path meets them and of no other: a command off a Query's path is listed as off the path, and the
Case's inventory accounts for it as a command the path does not perform. A path that takes one class
more than once is lowered like any other, each step confirmed by the kind that names it; what the
producer refuses of such a path is an error. The inventory of a Case checks what it carries of a kind
against the declaration, both ways: where it is recorded, the guard and the instruction of the Run's
own record, the fields it keeps, the steps it confirms, and the closing read of an exhaustive kind,
which every Case carries. A Case is over one operation.

`tools/umpire/conformance` assesses a Run against the Model through the Query the reader's `Realizer` binds, and
reads the Run's evidence as [Realizations](#realizations) says. Beside the fields a realization
gives a role, it reads the activity attempt a Run Event records with the evidence it carries, the
Run protocol's typed data, as that evidence's attempt and delivery: where a field and the event name
the same identity they must agree, and the evidence of one operation is of one activity run. Its
`admits` says whether a Run Event is an occurrence of the evidence a Run Event source declares, by
the event's kind, the command that recorded it, the number of the attempt the event records where
the source is declared the record of one, and the source's guard, which it checks by the one reading
above before it evaluates it. The script such a source names it cannot read off the event, which is
why a Case that runs two activities is not lowered. Evidence of a kind that is the Run's own record is read only
from a Run Event its source admits: on any other event it is refused at that event, and a guard that
cannot be evaluated there is an error. The assessment reads a kind by the fact it records and not by
the steps it names: a piece of evidence may be explained by any step that records its fact, which
keeps more executions than the Contract's reading of the same piece and rules out none that
happened.

The Scala framework's own composition starts from each member's first start only. The reader composes
every start, as [Compositions](#compositions) says; bringing the front end in line
is future work.

Two narrowings came from the specimens' evidence:

- **Machines 6.** A stutter records no fact the product sees only where the refinement names what the
  product sees. `activityProtocol` names nothing, so its 240 stutters that record a product fact
  (finding F3) are read as before.
- **Expressions, `match`.** A value no case matches is an undeclared hole, apart from the admission
  errors a reader rejects and from declared holes (finding F4).

## Generated Case expectations

A Query's optional `expected_run` is test metadata, not a transition, search assumption, refinement,
or evidence filter. Every Query that generates a live Case must supply it. It declares the Contract's
Verdict (`contract`: satisfied or violated), how the Run ends (`disposition`, as Testpilot's
RunDisposition names it) and how its cleanup ends (`cleanup`), the expected trace conformance and
the selected Property's outcome; each monitor attached to the Scenario's machine must be named exactly
once. Each is declared and none is inferred: an unset contract, disposition or cleanup is an
admission error, as are unknown statuses, missing monitor names and duplicate names. An
inconclusive or violated outcome names its reason, a satisfied one none; a reason is the judge's id
(`RunExpectation.Reason`), whose wording only the judge has. `conformance_reason` is the judge's
reason for the conformance, from the same ids, set exactly when conformance is not conformant: the
reader refuses one on a conformant expectation, none on an expectation that is not conformant, and
an unknown id, at the Query's position. Lowering refuses a contract and disposition
`testpilot.ConcludeVerdict` cannot conclude together, at the Query's position. The live checker
retains its own independent semantics; an expectation cannot change its result. A live Run is
compared with it value by value, each reason by id, never by the assessment's prose.

The version-1 Case manifest lists every Query and its lowering standing, names the canonical Case
file of each lowered Query, and carries its declared expected assessment. Readers reject unknown
versions/fields, trailing data, duplicate Queries or file names, unsafe paths, and incomplete or
invalid expectation metadata. Checking regenerates the complete inventory in memory and under a
temporary root before comparing it; publishing uses a lock and a complete staged directory, with
rollback if its final rename fails. A crash between the directory renames leaves `.previous` and
`.lock` for explicit recovery; subsequent publication refuses to overwrite that recovery state.

## Exploration and legal reductions

`Query.exploration` is optional metadata for a pinned, non-`through` find Query. Each variation
replaces one distinct prefix position with one finite action sequence, possibly empty. The final
Scenario action is preserved. Names are nonempty and unique within an axis and may not contain
`+`, the separator of the candidate tuple. Admission rejects unknown classes, empty domains,
invalid indexes, nonpositive Run budgets, negative edit budgets, and Cartesian products above
4096 combinations. Axes are ordered by position; candidates by decreasing summed priority and then
lexicographic tuple name. This exhaustively enumerates the declared finite domain, not arbitrary
parameter values, runtime schedules, or undeclared scenarios. Lowering refusals remain explicit
candidate rejections and are never counted as covered executions.

A candidate is a cloned IR model with the authored Scenario edits, rechecked by the reader and lowered
through `tools/umpire/lower`. A deterministic protobuf digest identifies that edited model; the Case
identity covers its exact canonical bytes. JSON protocol and proposal envelopes preserve those
bytes, including literal protobuf field-path angle brackets. The Run budget limits actual candidate
attempts. Runtime coverage reports selected, covered, violated, attempted, unrealizable and pending
targets separately, and remains sampled even when every finite candidate has been attempted.

`drop_prefix` permits a single reverse sweep of deletions before the final action, bounded by
`edits`. Every deletion re-enters the original checked lowering path: a missing prerequisite,
unsatisfied Query, invalid evidence mapping, dangling learned value, or broken script dependency
rejects the edit before a Run. The existing replay algorithm requires two fresh Runs with the same
failure key before retaining an edit. A proposal is emitted only after a complete uncapped sweep;
its embedded model, target and descending edit recipe must regenerate the exact model digest,
Case identity and Case bytes on read. This is bounded local minimization, with no global-minimality
claim. No candidate from discovery alone is promoted.

An expectation whose `contract` is violated denotes a negative control, and declares that its live
Run stops at the Contract monitor (`disposition` stopped by monitor). This changes test
expectations, not machine transitions, evidence interpretation or runtime policy.
