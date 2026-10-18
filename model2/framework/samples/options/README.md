# model2: Kettle design options in Rust

`model2` is the Rust prototype of the framework. Its scope is exactly what
[`model/framework/samples/Kettle.scala`](../../../model/framework/samples/Kettle.scala) uses today:
vars with a `oneOf` start, actors with actions and internal steps, transitions with guards and
effects, invariants, postconditions with `before` and `after(action)`, one scope (`without`), one
progress claim (`whenever … eventually … in scope`), and the native questions the Scala test asks
(`explore`, `verifyAll`, `refuted`). Nothing else.

Each file here is the same Model in one authoring style. None compiles yet; the framework they
assume is the thing the prototype builds. Read them side by side with `Kettle.scala`.

| File | Kind | One line | Macro effort | Editor support inside the Model | Lifting |
| --- | --- | --- | --- | --- | --- |
| `Kettle.grammar.rs` | custom A | one `model!` block with its own grammar; reads like the Scala | highest: a parser over tokens, own diagnostics | none inside the block | free, the macro has the tokens |
| `Kettle.macro_rules.rs` | custom B | plain items plus four `macro_rules!` helpers; bare var names by destructuring `self` | low: declarative macros, no proc-macro crate for the helpers | full: rust-analyzer expands `macro_rules!` | pass-through `#[machine]` attribute records the impl |
| `Kettle.attrs.rs` | custom C | plain functions with pass-through attributes (`#[on]`, `#[invariant]`, …), statig's mechanics | low-medium: attributes that only register | full | same |
| `Kettle.trait.rs` | custom D | a trait and plain Rust, no macros at all | none | full | `syn` pass over source in `build.rs` |
| `Kettle.statig.rs` | library E | statig 0.4 handlers as the surface, claims bolted on | none for authoring | full | would read statig's impl |
| `Kettle.smlang.rs` | library F | smlang 0.8 transition table | none for authoring | table only; guards and effects are functions elsewhere | table is already data |
| `Kettle.stateright.rs` | library G | stateright 0.31 `Model` trait as surface and engine | none | full | `syn` pass |

What the comparison shows:

- The libraries model control flow with named states; Umpire's Models are var-shaped with claims
  as the point. statig and smlang force `heating_on` and `beeping` into state names and lose the
  claims' vocabulary; stateright has the right `next_state` shape but only state predicates. They
  are references, not foundations. stateright's last commit is July 2025.
- Among the custom options the spread is between A (prettiest, most to build, no editor help) and
  D (plainest, nothing to build, most noise). B and C sit between: B keeps the Scala look with the
  language's own declarative macros; C keeps every item a plain function with attributes that
  register rather than rewrite.
- All four custom options share the same expansion: a `State` struct, an exhaustive `Action` enum,
  `step(&mut self, Action) -> Outcome` with `_ => Refused`, claim functions, scope values, progress
  triples. The engine does not care which surface is chosen.

Layout the prototype follows, mirroring `model/framework`:

```
model2/
  framework/
    dsl/        State, Action, Machine, Claim, Scope, Progress, the macros
    checker/    explore, verify_all, refuted; BFS with a shared visited set
    samples/    Kettle in the chosen style, with its test
```
