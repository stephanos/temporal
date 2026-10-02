# Decide whether the Umpire IR's expressions become CEL

## Goal & Context
<!-- scope: business -->

The Umpire IR carries every step function, Property and monitor as an expression tree of its own design: `Expr` with field access, calls, constructors, `copy`, unary and binary operators (`OP_EQ` and its siblings), `if`, `match` with patterns, `let`, lists and lambdas. Go evaluates that tree with a hand-written evaluator of 671 lines, and `SEMANTICS.md` defines what each node means because no outside standard does.

The owner asked on 2026-10-01 why the lifter does not translate Scala expressions to CEL instead. CEL is a specified, non-Turing-complete expression language with maintained evaluators in Go (`cel-go`) and Java (`cel-java`) and a protobuf form of its syntax tree (`cel.dev/expr`, already an indirect dependency of this module). If the IR carried CEL, the project would stop owning an expression language, its evaluator and its semantics document, and any JVM or Go program could evaluate a Model's expressions with a library.

This spec is a spike. It answers whether that trade is worth making, with a measured prototype, and it changes no production code path. The reader it serves is the owner, who decides after reading its report.

## Architecture & Data Models
<!-- scope: technical -->

**What CEL covers directly.** Comparison, boolean and arithmetic operators, the conditional, field selection, list literals, concatenation and membership, message construction against protobuf descriptors, and bound variables through the `cel.bind` extension.

**What the lifter would have to compile away.** CEL has none of these, so each becomes a translation rule.

| IR construct today | In CEL |
| --- | --- |
| `match` with case, binding, alternative and wildcard patterns | nested conditionals over a case test, with field selections for the bindings |
| a call of another Model function | inlined at the call site, or a host function the evaluator registers per Model function |
| `copy` with updated fields | construction of the whole message with the other fields copied |
| an enum case with fields | a protobuf message with a oneof, so the Model's types become descriptors built from the IR |
| a declared hole | a host function that reports the hole |
| a precondition (`require`) | a separate CEL expression beside the body |

**What does not change.** The lifter still walks typed Scala trees; it would emit a CEL syntax tree where it emits an `Expr` today. No text is parsed in either design. The IR still declares types, actions, machines, claims and realizations in its own messages. The Quint and P exports still translate an expression tree to their own syntax, from CEL's tree in place of the IR's.

**Where the saving would be.** The Go expression evaluator and the expression half of the IR validator, and the Expressions section of `SEMANTICS.md`. The spike measures the size of each.

## API Contracts
<!-- scope: technical -->

The spike's output is a report and a prototype branch of the reader. The report states, for the Nexus caller Model and the standalone activity Model:

- the count of IR expressions and how many translate to CEL without a rule from the table above;
- the size of the CEL form against the size of the `Expr` form, in bytes of ProtoJSON, for the largest step function with every call inlined;
- the lines of Go the prototype deletes and adds;
- whether `cel-go` evaluates every translated step function to the same result on every state and class as the current evaluator, by the baseline goldens;
- the time to derive every table with each evaluator.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Running the Model in Scala needs no CEL.** A step function is already executable Scala. The benefit on the JVM is for a consumer that has only the IR.
- **Two evaluators can disagree.** `cel-go` and `cel-java` share a conformance suite and are separate implementations. The spike runs the translated step functions through both on every state and class and reports any difference.
- **Integers.** CEL integers are 64-bit and overflow is an error. Model counters are bounded, so a result outside its range must stay the model error it is today and not become a CEL error with a different message.
- **Errors in `&&` and `||`.** CEL's logical operators may absorb an error on one side. The current evaluator reads left to right. The report says whether any Model expression can tell the difference.
- **Source positions.** Every IR expression carries its Scala position, and model errors are reported at that line. The report says how a CEL tree keeps positions (its `SourceInfo`) and whether a failing subexpression is still reported at its own line.
- **Inlining and size.** Inlining every helper call may grow the IR. The report gives the measured growth.
- **Dependency.** `cel-go` becomes a direct dependency of the server module if adopted. The report states its transitive additions.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A prototype translates every expression of the Nexus caller and standalone activity IR files to CEL and evaluates it with `cel-go` behind the reader's existing interface, on a branch that is not merged. The translation is written in Go, from the IR's `Expr` trees, so the spike touches neither the lifter nor the checked-in IR and can run while the Scala cleanup specs are in progress. Errors: an expression that has no translation is listed with its Scala position and the construct that blocks it.
- **R2:** The prototype derives the same tables, Definition IDs, refinement rows, fingerprints and Query answers as the current evaluator, checked by the baseline goldens of fn-115 R2. Errors: each difference is listed with the expression that causes it; the spike does not hide a difference by changing a golden.
- **R3:** The same translated step functions are evaluated with `cel-java` on every state and class of both Models, and the results equal `cel-go`'s. Errors: each disagreement is listed with the expression and both results.
- **R4:** A report committed under `.plans/` gives the measurements listed under API Contracts and a recommendation with its reasons: adopt, adopt for a subset (for example Properties and monitors only), or keep the IR's own expressions. Errors: a measurement the spike could not take is stated as not taken, with the reason.
- **R5:** The report names what adoption would cost beyond the prototype: the lifter's new translation rules, the Quint and P exports, `SEMANTICS.md`, the IR schema change and the regeneration of every checked-in IR file (no error surface).

## Boundaries
<!-- scope: business -->

- No change to the IR schema, the lifter, the checked-in IR or any consumer on the main branch. Adoption is a later spec the owner opens after reading the report.
- No CEL in the Scala Models. They stay ordinary Scala.
- No replacement of the IR's declarations (types, machines, claims, realizations) by CEL. Only expressions are in question.
- The Testpilot Case's expression language is a separate question and is not evaluated here.

## Decision Context
<!-- scope: both — conditionally substructured -->

The idea is sound in direction: the project should not own an expression language if a standard one fits. Two things keep it from being an obvious win, which is why this is a spike and not a migration.

First, the claimed saving in parsing does not exist, because the IR is already a tree and nothing parses text. The real saving is the evaluator and its semantics, a few hundred lines.

Second, CEL lacks pattern matching, user functions and record update, and Model step functions are mostly made of those three. The lifter would gain translation rules of about the size the Go evaluator loses, and the IR could grow where calls are inlined.

The spike is worth its cost because the measurements are cheap to take once the baseline goldens exist, and because a "subset" outcome is plausible: Properties, monitor verdicts and realization operands are small boolean expressions that fit CEL as it is.

It runs after fn-115, which writes the goldens it needs and settles where the reader lives.

## Parked unknowns

- Whether the Testpilot Case's own expression language should also be CEL. It is out of scope here and worth asking if this spike recommends adoption.
