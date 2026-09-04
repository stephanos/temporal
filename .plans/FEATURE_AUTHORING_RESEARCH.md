# Feature authoring: language and tooling research

Research date: 2026-09-04. This note supports evaluation of `model/Temporal/Feature`; it does not
select or implement a replacement authoring language. Recommendations below are design hypotheses,
not results of a user study.

## Conclusion

Lean can support a substantially friendlier authoring interface without requiring feature authors
to implement metaprograms. The important choice is which concepts and obligations authors must
handle. Replacing punctuation does not remove the need to understand nondeterminism, state changes,
quantification, bounded claims, and unsupported behavior.

The strongest first candidate is a small typed Lean API with routine infrastructure hidden behind
stable interfaces, specific diagnostics, checked examples, and a generated reading view. Compare it
with a narrowly scoped syntax extension only after identifying tasks that remain difficult. Neither
candidate should be called the most intuitive until representative authors have used it.

## Constraints from this repository

The following are requirements in [UMPIRE4_SPEC.md](UMPIRE4_SPEC.md), not limitations of Lean:

- SCP-03 and SEM-01/02 make handwritten Lean Model Definitions the behavior authority. An external
  YAML, Quint, or Alloy model that generates the authoritative behavior is an architectural change.
  External Portable Plans have a distinct, deliberately narrower authority under SEM-11/12.
- AUT-07 requires the existing Property, Behavior, and Query languages to remain the public
  declaration paths. AUT-08 explicitly prohibits the finite adapter from introducing another
  Behavior, Property, Query, Scenario, or macro language. A proposed new DSL needs an explicit
  compatibility analysis; a design that contradicts these rules requires their deliberate revision
  or an approved exception before implementation. Ordinary functions over the existing language
  data do not inherently create a second semantic authority.
- AUT-05 requires portable material to be serializable data. Offering arbitrary Lean predicates or
  callbacks as a convenient property interface does not by itself provide a portable interpreter.
- AUT-08 requires ordered domains, encoders, enumerators, domain-closure evidence, and executable
  Action evidence. Moving or mechanically discharging these obligations is different from removing
  them. The authoritative enumerator must not be confused with an implementation that needs a
  separate equivalence proof against an independently stated relation.
- AUT-02/03/04/06 require explicit meaning, checking, stable IDs, and explicit composition. SEM-09
  requires a Limit and unit for progress. A convenient default must not hide a behavioral choice.
- [LEAN_GUIDELINES.md](LEAN_GUIDELINES.md) prefers idiomatic Lean with a readable API boundary and
  checked examples. It also requires honoring proof-trust boundaries when automating obligations.

## What Lean can provide

### Verified version and scope

[model/lean-toolchain](../model/lean-toolchain) pins `leanprover/lean4:v4.33.1`.
[model/lakefile.toml](../model/lakefile.toml) uses Batteries `v4.33.0`.
The installed compiler reports Lean 4.33.1, commit
`819816b2e0a3bf405af45ae5c7af2491d8f5bee6`.

The exact-version web manual URLs could not be retrieved through the browser tool. The general
explanations below use the current official manual; the concrete APIs were checked in the installed
4.33.1 source. This establishes API availability, not a working Umpire authoring prototype or editor
quality. No model source was changed and no model build was required for this research note.

### Typed APIs before new grammar

Keep domain enums, records, functions, field access, and ordinary pattern matching where these
already express the feature clearly. Small semantic constructors can hide representation details
while retaining ordinary Lean name resolution and type checking. A design can accept already typed
expressions without exposing the feature author to the implementation of those constructors.

This is an engineering recommendation. The relevant feasibility boundary is that Lean elaborators
can process syntax with an expected type and access the local context. Availability was confirmed
in installed `Lean/Elab/Term/TermElabM.lean` at `TermElab` and `elabTermEnsuringType`.
See the [official elaborator reference](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/Elaborators/).

### Macros versus elaborators

Macros transform syntax into syntax; they are suitable for predictable desugaring. They do not
provide the type-directed context of a term elaborator. A custom elaborator can examine types and
issue domain-specific errors, but requires framework expertise to maintain. Adding a grammar is
therefore feasible; producing a reliable authoring experience is a larger task.
See the [official macro reference](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/Macros/)
and [elaborator reference](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/Elaborators/).

A portable expression surface must lower to the existing serializable language, or to a deliberately
extended one with semantics and evaluation. Accepting arbitrary Lean terms and evaluating them is
not equivalent to reifying them. Reject unsupported terms at their source. This follows from
AUT-05 and SEM-14; macros do not erase that obligation.

### Diagnostics and editor behavior are separate deliverables

Lean syntax retains original or synthetic source locations. Generated syntax can retain a reference
to the author’s input. That makes a diagnostic on the wrong field or unsupported expression
technically possible without reporting only a generated definition.
See [source positions in the official syntax reference](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/Defining-New-Syntax/).

Installed 4.33.1 source confirms these specific mechanisms:

| Mechanism | Source under the installed `src/lean/` | Relevant behavior |
| --- | --- | --- |
| `Lean.throwErrorAt` | `Lean/Exception.lean:84` | Uses supplied syntax as the diagnostic reference. |
| `Command.withMacroExpansion` | `Lean/Elab/Command.lean:451` | Records original and expanded syntax in the information tree and macro stack. |
| `Term.addTermInfo'` | `Lean/Elab/Term/TermElabM.lean` | Associates terms with syntax for hover and navigation. |
| `TermInfo`, `CompletionInfo` | `Lean/Elab/InfoTree/Types.lean` | Represents information needed by language-server interactions. |

The local source root is
`/Users/stephan/.elan/toolchains/leanprover--lean4---v4.33.1/src/lean/`.

Inference: a new DSL needs acceptance checks for incomplete input, completion, hover, navigation,
error recovery, and source attribution. Successful batch compilation alone does not establish any
of these. Reusing ordinary typed terms where possible should reduce the custom tooling surface;
the actual benefit requires a prototype comparison.

## Useful contrasts

### Quint: familiar expressions, explicit modeling modes

Quint uses familiar functions, records, and expression syntax, while separating pure expressions,
state access, actions, and nondeterministic choice. Its documentation makes this distinction
explicit and describes both simulation and branching over possible choices. It also excludes
recursive operators and recursive types, illustrating that an approachable surface can deliberately
restrict expressiveness to support analysis.
See [Quint language basics](https://quint.sh/docs/language-basics).

Quint’s language reference distinguishes Action and Temporal modes and describes its type system.
Its authors report that unclear expression levels caused confusion in TLA+; this is their design
rationale, not a controlled usability result for Temporal engineers.
See [Quint language reference](https://quint.sh/docs/lang).

Design inference for Umpire: keep state predicates, transition conditions, trace properties, and
query intent visibly distinct. Familiar braces and function syntax can help recognition but cannot
teach those differences automatically. Borrow the separation principle; translating Umpire to
Quint would introduce a separate semantic and tooling boundary.

Quint also provides a workflow for literate specifications that mixes explanatory Markdown with
specification code. This is evidence that reading and executable authoring can be presented
together, not proof that product owners can author correct formal models.
See [Quint literate specifications](https://quint.sh/docs/literate).

### Alloy 6: concise relations plus visible counterexamples

Alloy distinguishes model facts, predicates, assertions, and analysis commands. Analysis scopes
bound signature sizes. A concise declarative formula still requires the reader to understand what
was assumed and what was checked.
See [Alloy language specification](https://alloytools.org/spec.html).

Alloy 6 adds mutable state, next-state notation, temporal operators, and a trace visualizer. Its
finite lasso representation denotes infinite traces. It offers bounded time horizons and complete
checking over finite state scopes; the latter can remain computationally expensive.
See [Alloy 6 documentation](https://alloytools.org/alloy6.html).

Design inference for Umpire: borrow adjacent-state comparison and explicit scenario exploration.
Do not copy Alloy’s temporal words without their semantics. Umpire’s finite Execution and bounded
progress rules are not interchangeable with Alloy’s infinite-trace interpretation. A diagram or
concrete trace can supplement source review, but must expose state changes, scope, and assumptions.

## Evaluate designs against real authoring work

The sources establish language capabilities and design choices. They do not establish which
interface is most intuitive for this team. A small comparative exercise should hold behavior
constant and change only the interface. Suggested candidates are the current surface, a simplified
typed API, and a limited syntax prototype if policy permits it.

Give developers unfamiliar with Lean a short onboarding and ask them to:

1. Explain an allowed trace and a rejected trace, including all nondeterministic outcomes.
2. Add an ordinary transition and identify which fields remain unchanged.
3. Write a safety property, distinguish it from a Scenario constraint, and find a counterexample.
4. Write bounded progress with a unit and explain whether the trigger step counts.
5. Add a variant or boundary value and repair resulting authoring errors without framework help.
6. Identify an overconstrained Scenario, an unsupported case, and a Limit Reached result.

Measure semantic correctness, completion time, assistance needed, error-recovery time, and ability
to explain the result. Record errors that compile successfully, not just syntax errors. Vary task
order to reduce learning effects. A handful of participants gives useful formative feedback, not a
statistically established universal winner.

For the product-owner stretch goal, evaluate reading separately: show a generated behavior summary,
a transition table, and example/counterexample traces linked to the authoritative declarations.
Ask reviewers to detect a deliberately wrong requirement. Do not use subjective preference for
natural-language-looking syntax as the sole success criterion.

Before committing to an implementation, require equivalent behavior on representative existing
models, explicit serialization coverage, no weakened trust checks, source-specific diagnostics,
reasonable incremental feedback time, and preservation of stable IDs and behavior fingerprints.
