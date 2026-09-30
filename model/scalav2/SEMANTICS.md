# Semantics of the Umpire IR

The IR's meaning is defined here, not by the lifter that writes it or the Go interpreter that reads
it. `proto/internal/temporal/server/api/modelir/v1/ir.proto` is the schema; `goir/` is one evaluator of these rules, and the parity
tests check it against the Lean Model.

## Values

A value is a Boolean, an integer, a string, an enum value `C(v1, …, vn)` of one case `C` of an enum
type with its fields, a record value `(v1, …, vn)` of a record type, a list `[v1, …, vn]`, or an
anonymous function. Equality is structural. The step record `umpire.Step` is a built-in record with
the fields `outcome`, `state`, `facts` (a list) and `because` (a string).

## Catalogs

A finite type has its members in catalog order:

- `bool` is `false, true`;
- an integer range `low..high` is its integers in increasing order;
- an enum is its cases in declaration order, each case once per assignment of its fields;
- a record, and a case's fields, are the product of the fields in declaration order, with the last
  field varying fastest.

## Keys

`key(v)` spells a value in state, action, row and Definition keys:

- `key(true) = "true"`, `key(n)` is `n` in decimal, `key(s) = s` for a string;
- `key(C) = "C"` for a case without fields, and `key(C(v1, …, vn)) = "C-" + key(v1) + "-" + … + key(vn)`;
- `key((v1, …, vn)) = key(v1) + "-" + … + key(vn)` for a record.

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
| `match s { p1 if g1 => b1; … }` | `E(bi, σi)` for the first case whose pattern matches `E(s, σ)`, extending `σ` to `σi`, and whose guard holds under `σi`; a value no case matches is an error, a hole in the Model |

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
   has one result per step record, and a result state outside the catalog is an error.
4. Its starts are its start expressions' values, and its ends the states its `ends` function accepts.
5. Its evidence is, for each fact in catalog order whose case has no line yet, the case name and
   `E(call(evidence, fact))`.
6. A refining machine's refinement is checked under `Umpire.Command.deriveRefinement`'s rule, as
   model/go applies it: every outcome is a product outcome of the same name, every start maps to a
   product start, and every row result is carried by a product row from the mapped source to the
   mapped target with the same outcome whose facts are among the result's (preferring the product
   action of the row's own name), or else its source and target map to one product state and it is
   a stutter.

Reachability, stuck states, Definition IDs and the Behavior Fingerprint are then those of model/go's
`umpire.Table` over the derived keys.

## Not defined here yet

The [fn-107 specimens](specimens/README.md) need constructs these rules do not define. An IR that uses one
has no meaning under this document until a rule for it is added here:

- passive monitors;
- channels and the faults derived from them;
- opaque providers and scoped replacement;
- visible-result projection;
- named assumptions and conditional progress;
- declared holes;
- non-timer system actions.

Two rules above are narrower than the specimens need:

- **Machines 6.** A result whose source and target map to one product state is a stutter, whatever
  facts it records. `activityProtocol` has 240 such results that record a product fact.
- **Expressions, `match`.** A value no case matches is called both an error and a hole, where the
  spec distinguishes admission errors, declared holes and undeclared holes.

`specimens/README.md` gives the evidence for both (findings F3 and F4).
