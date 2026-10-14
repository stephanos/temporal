# Quint versus FizzBee

Two off-the-shelf specification languages with a simulator or checker and a model-based-testing
story. Facts below were read from the repositories and READMEs on 2026-09-29; nothing was run.

## Maturity

| | Quint | FizzBee |
|---|---|---|
| Repository created | May 2021 | March 2024 |
| License | Apache-2.0 | Apache-2.0 |
| Stars, forks | about 1,750, 147 | about 350, 23 |
| Commits | about 5,300 | about 316 |
| Contributors | one dominant author with about 2,950 commits, then 557, 327, 216, 122 | one author with 302 of 316 commits |
| Latest release | v0.33.0 on 2026-09-28, releases roughly monthly | v0.5.3 on 2026-08-24 |
| Open issues | 249 | 30 |
| Owner | Originated at Informal Systems (Apalache team), now a small "Quint core team" under quint.sh | One person, fizzbee.io |
| Install | npm package, VS Code extension, LSP, REPL | brew, Docker image, online playground; building from source needs Bazel plus g++ |
| Implementation | TypeScript CLI and simulator, Rust evaluator, Apalache (Scala, JVM, Z3) for `verify` | Go model checker, stock go.starlark.net, ANTLR Python parser emitting a protobuf AST |

Both are young and small. Quint has the longer history, a team rather than a person, and a
versioned changelog. FizzBee is a one-person project in its third year with a pre-1.0 version.

## Language and semantics

| | Quint | FizzBee |
|---|---|---|
| Surface syntax | TypeScript-like, statically typed with inference, sum types, records, sets, maps, `match` | Python-like: Starlark bodies inside `role`, `action`, `func`, `atomic`, `serial`, `parallel`, `oneof`, `any`, `require`, `always`, `eventually`, `assertion`, `refine`, `compose` |
| Semantics | TLA+ in the logic: `init`, `step`, `action`, `nondet ... oneOf`, `pure def`, temporal operators, `next` | Interleaving semantics with explicit yield points inside non-atomic actions; crash and message loss explored at yields |
| Modularity | Modules, imports with parameters, `import M(x = 1) as I` for instantiation | Roles as classes, `compose`, `refine` |
| Types | Static, with a type checker and an effect system for state updates | Dynamic (Starlark) |
| Refinement | Not built in; written as an abstraction function plus an invariant, or checked via TLC or Apalache on the mapped spec | `refine` keyword exists in the grammar and docs |
| Liveness and fairness | Temporal properties, `weakFair`, `strongFair`; checked by Apalache or TLC backend | `fair`, `eventually always`, `always eventually`, checked via Markov-chain reachability |
| Probabilistic and performance | No | Yes: probabilities on choices, Markov-chain analysis for expected cost and latency |
| Symmetry reduction | No | Yes, via `symmetric role` |
| Visualization | Trace explorer TUI, ITF traces | State graph, sequence diagrams, interactive "whiteboard" explorer in the playground |

Quint is a stricter language with types and an effect system, which suits a model that agents
write and humans read. FizzBee is a looser scripting language whose checker does more out of the
box for distributed-systems concerns: crashes at yield points, fairness, probabilities, symmetry.

## Checking

| | Quint | FizzBee |
|---|---|---|
| Random simulation | `quint run`, seeded, `--max-steps`, `--invariant`, `--out-itf` | Not the main mode; the checker is exhaustive |
| Exhaustive checking | `quint verify` through Apalache (symbolic, bounded), or `--backend tlc` (explicit, needs the TLA+ tools) | Explicit-state model checker in Go, bounded by `max_actions` and per-action options in YAML front matter |
| Deadlock detection | Through Apalache or TLC | Built in |
| Determinism of traces | Seeded simulator; ITF traces are plain JSON | Seeded random walks in the MBT server |
| Test blocks | `run name = init.then(...).expect(...)` inside the spec | Not a first-class construct; checks are assertions and invariants |

## Model-based testing against a real system

| | Quint | FizzBee |
|---|---|---|
| Shape | Generate traces from the spec, replay each step against the implementation through an adapter, compare state after each step | Server replays random walks over the checked state graph through an adapter, sequentially and in parallel, with linearizability checking, seeded for replay |
| Adapter languages | Rust only, via quint-connect (v0.1.1, May 2026, 88 stars) | Go first, with generator templates for Go, Java, Rust and TypeScript in the repository's `mbt/` directory; the shipped agent skill `fizz-mbt` is about writing Go adapters |
| Go support | None from the project. ITF is a small JSON format; a Go reader is a few hundred lines. The only Go ITF library, itf-go, has not been pushed since 2023 and is a no-go | Native, since the checker and the MBT server are Go |
| Status | quint-connect is v0.1 | The FizzBee site describes parts of MBT as work in progress; the feature exists in the repository |

For Umpire, this is the sharpest difference. FizzBee's MBT is Go-native and already has the
random-walk-against-implementation loop that the exploration item of the vision wants. Quint's is
Rust-only, and you would write the Go trace replayer yourself, using quint-connect as the design
to copy.

## Fit with the Umpire vision

| Vision item | Quint | FizzBee |
|---|---|---|
| Single behavior model, readable by many, written by agents | Good. Static types and the effect system catch mistakes before a run. Agent skills ship in the repo | Fair. Python-like syntax is readable, but dynamic typing defers errors to a run. Agent skills ship in the repo |
| Deterministic regression plans as artifacts | Good. ITF traces are canonical JSON | Fair. Seeded walks are reproducible, but the artifact is the seed plus the graph, not a checked-in trace |
| Faults as first class | Weak. Faults are actions you write | Strong. Crashes and message loss are explored automatically at yield points |
| Exploration to find unknown bugs | Simulator plus invariants finds spec bugs; against the implementation only through your own replayer | The MBT server does this directly |
| Clock skew, timing | Neither models wall time; both abstract to steps |
| Compose feature models into a system model | Modules and instantiation; no built-in refinement | `compose` and `refine` keywords; no static types to hold the composition together |
| Export to other checkers | TLA+ output, TLC backend, Apalache | None; the checker is the only consumer |

## Bottom line

Quint is the better specification language and the safer bet as a team-maintained project. FizzBee
is the better model-based-testing tool for a Go shop today, and a riskier bet as a single-author
project. If the priority is a typed source of truth that other tools can consume, prototype Quint
and budget for a Go ITF replayer. If the priority is running random walks with automatic fault
exploration against a Temporal server this quarter, prototype FizzBee and accept the bus factor.
Neither gives you refinement between a feature-level and an integration-level model with static
checking; that is something you would build in either case.
