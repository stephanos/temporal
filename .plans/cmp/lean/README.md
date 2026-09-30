# Lean 4 (the current implementation)

`NexusCaller.lean`, `Worker.lean` and `Pins.lean` are verbatim copies of the real files under
`temporal/model/Temporal/Feature/`. `StandaloneActivity.lean` is new, written in the same command
DSL against the spec in `../SPEC.md`; it has not been compiled. `ActivityPins.lean` holds the
spec's pins for it in the style of the real tests. Two parts are known not to elaborate today: the
`case` blocks name a realization that does not exist, and the nine status observations assume the
evidence catalog accepts several observations over one field.

## DSL mechanism

Custom Lean 4 syntax categories and command elaborators (`Umpire/Command/Syntax.lean`, about 3,900
lines, 10 `declare_syntax_cat` across the two syntax files, 19 `elab`s). `entity`, `enum`, `action`, `observation`,
`machine`, `compose`, `property`, `scenario`, `limits`, `query`, `set` and `case` are real
commands. Each elaborates to plain records, then runs work at elaboration time:

- `enum` derives `BEq`, `DecidableEq`, `Repr`, `Finite`.
- `machine` enumerates the whole finite state table from the step functions.
- `refines:` and `compose` emit theorems proved by `decide +kernel` over the tables and reject
  `sorryAx`.
- `query` runs the bounded search while the file compiles and reports failure at the `find:` or
  `verify:` token.
- `case` produces the Case fixtures.

## Where checks happen

Everything above is compile time. Nothing is deferred to a test run, which is what makes the loop
slow: a one-line edit to `NexusCaller.lean` re-elaborates 3 min 23 s for the file alone and about
5 min 24 s for the downstream library. The protocol machine's 192-state table already needs
`maxRecDepth 65536` and `maxHeartbeats 1000000`; the elaborator refuses larger machines.

## Author errors

Curated, English, at the right token, pinned by `#guard_msgs`, for example:

```
error: 'awaitFinish' is not an action declared by an `action` command; a `steps:` line names the
action its function steps on
```

A non-exhaustive `match` in a step function is Lean's own error, which is also good. An error the
elaborator did not anticipate lands the author inside a 3,900-line elaborator using `evalExpr`.

## Libraries leveraged

| Library | Status (2026-09) | Role |
|---|---|---|
| Lean 4 core, Batteries | maintained | language, `Finite`, decidability |
| Veil (verse-lab) | maintained, pinned commit, needs Node/npm | BFS backend over the product |
| Lean-zh/protobuf | pinned commit, small | proto codec for Cases |

No stateful property-testing library exists for Lean; search, fingerprints and canonical JSON are
all hand-written in `Umpire/`.

## Easy and awkward

Easy: the declarative surface is the cleanest of the eight, no proofs or tactics appear in a
feature Model, exhaustiveness and finiteness are native, and the compile-time query makes a wrong
Model fail to build. Awkward: the loop measured above, a 3.4 GB build directory (4.6 GB with the dependency packages), 240 MB lint
binaries, hand-bounded state spaces with `Fin`, hand-computed `search:` budgets, a vocabulary only
the specs define, and a maintenance surface roughly ten times the size of the Temporal behavior it
models.
