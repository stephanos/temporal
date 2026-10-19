# The path! pattern: progressions for claims, witnesses and examples (model)

## Conversation Evidence

> user (2026-10-19): "is there a delcarative option; something like a list where you can say [state_A, ..., state_B] and the `...` means evntually? somethig that reads like what a human not knowing temporal logic would write? and its reads like the state progression"
> user (2026-10-19): "is there value in haveing a general purpose way to describe a partial path through the model? for instane, maybe in tests or exploration?" — yes; then "yes" to defining `path!` as the general pattern with chains allowed, `#[progress]` refusing more than two waypoints, and `find`/`Run::along` in fn-213.

## Goal & Context

A developer with no temporal logic reads a list with gaps as a state progression: `path![self.heating_on, .., self.beeping]`, Rust's slice-pattern `..` standing for "any number of steps" and `..=n` for "at most n". The list is a pattern over paths; whether it is a promise about every admitted path, a request for one witness, or a story to fill in is decided by the question asked with it, not by the list: `#[progress]` (all paths), `find` (some path), `Run::along` (this path, gaps supplied by the checker). A survey of every progress claim in the archived Scala (Kettle, TwoLayeredCache, TwoPhaseCommit, ReliableBroadcast, five Nexus protocol Models, the activity-mini protocol) found each to be one leads-to, optionally bounded, under zero to two assumptions, so a claim needs exactly two waypoints; witnesses and examples use more.

## Architecture & Data Models

- `Path<S>` in `umpire-dsl`: an ordered list of waypoints separated by gaps; a waypoint judges a state and is built from a `bool` (inside a Model method, evaluated per state as `Progress` is today), a closure `Fn(&S) -> bool` (in tests), or a concrete state (equality); a gap is `Any` (`..`) or `AtMost(n)` (`..=n`, `n` any expression). `path!` is a `macro_rules!` producing it; it accepts two or more waypoints and refuses a list with no gap between two waypoints (`write [p, ..=1, q] for the next step, or [p, .., q]`) and a trailing or leading gap, by catch-all arms with those messages.
- `Progress` wraps a two-waypoint `Path` plus `assumption`; `path![p, .., q]` and `path![p, ..=n, q]` are the canonical spelling of a claim and expand to what `whenever(p).eventually(q)` / `.within(n)` build (fn-214), which stay public. `#[machine]` refuses a `#[progress]` method whose path has more than two waypoints: `a progress claim has one trigger and one target; ask a longer progression with find or Run::along`.
- fn-213's suite consumes the same `Path`: `find(&space, path) -> Result<Found, Refusal>` finds the shortest admitted path through the waypoints in order, leg by leg (BFS from each reached waypoint state to the next, honouring `..=n`), and `Run::along(path) -> Result<Run, Refusal>` is `find` from the first waypoint (which must be a concrete state) followed by replay, failing by naming the first gap it cannot bridge. Counterexample matching (`Path::matches(&trace)`) is a later addition when a test needs it.
- Reading rules in `README.md`, beside Kettle: in a claim `..` means on every admitted path; in `find` and `Run::along` it means some path, the shortest; `..=n` bounds the gap in steps; a pattern says nothing about what happens between or after its waypoints.

## API Contracts

- `path![w, .., w]`, `path![w, ..=n, w]`, and longer alternations of waypoints and gaps `-> Path<S>`; waypoint = `bool` | `impl Fn(&S) -> bool` | `S`.
- `Progress: From<Path<S>>` for exactly two waypoints; `Progress::assuming(..)`, `.within(n)` unchanged (fn-214).
- fn-213: `find(&Space<M>, Path<M::State>) -> Result<Found, Refusal>`; `Run::along(Path<M::State>) -> Result<Run<M>, Refusal>`.

## Edge Cases & Constraints

- The macro adds no dependency and no proc-macro work; rust-analyzer expands it. R30 budgets hold.
- A chain in a claim is a compile-time refusal from `#[machine]` (it sees the expression's waypoint count only when written as a literal `path![..]`; a `Path` built elsewhere is checked at registration and refused with the same message).
- `find` through waypoints is shortest per leg, which need not be the globally shortest path through all waypoints; the README says so.

## Acceptance Criteria

- **R1:** `path!` builds a `Path` from two or more waypoints with `..`/`..=n` gaps, waypoints being `bool`, closures or states. Errors: no gap between waypoints, or a leading/trailing gap, is a compile error with the messages above, pinned by trybuild `.stderr` fixtures.
- **R2:** A `#[progress]` claim is a two-waypoint `path!` (optionally `.assuming(..)`); Kettle's `boils` is written that way, all Kettle goldens are byte-identical, and the `machine` macro fixture uses it. Errors: more than two waypoints in a claim are refused with the message above, pinned by a trybuild fixture.
- **R3:** `README.md` has the reading rules beside the Kettle example; DESIGN.md's authoring comparison shows the claim in Scala and in `path!`. No error surface.
- **R4:** (delivered in fn-213.3 and .2) `find` through waypoints returns the shortest admitted path leg by leg with its actions spelled as the checker's paths; `Run::along` fills a milestone story and yields a replayable `Run`. Errors: an unbridgeable gap is a `Refusal` naming the waypoint index and the last reached state; a first waypoint that is not a concrete state is refused for `Run::along`.

## Boundaries

- Expressive ceiling (owner, 2026-10-19): a `path!` is a regular expression over state sequences with concatenation and bounded wildcards only (`..` is `.*`, `..=n` is `.{0,n}`). Branching between *states* at a waypoint is a predicate (`self.beeping || !self.heating_on`) and needs no syntax; in a claim every branch must satisfy it, in `find`/`Run::along` the search takes the shortest. Not in this spec, each its own construct if a sample ever needs it: alternation of sub-sequences (`(b, .., c)` or `(d, .., e)`), repetition, negative patterns ("no x between a and b"), monitors, "always eventually", "until". SystemVerilog sequences (`##[1:3]`, `or`, `[*n]`) are the reference for that rung; it stays a graph search, not LTL.
- Planned first extension, not in .1: an *action* as a waypoint (`path![room(), Heater::Heat, .., |k| k.beeping]`, "then the heater heats"), for `find` and `Run::along` where tests speak in actions; added when the first `tests.rs` wants it.
- No `Path::matches` on counterexamples until a test needs it.
- Does not start before fn-214 (it needs `.within(n)` and `.assuming(..)`); fn-213.2/.3 carry R4 and must not start before this spec's R1.

## Decision Context

2026-10-19, owner: the list-with-gap notation is adopted as the canonical spelling of a progress claim because it reads as a state progression to a developer with no temporal logic; the builder remains as its expansion. Then generalized the same day: the list is a pattern over paths, and the question (`#[progress]`, `find`, `Run::along`) decides universal versus existential, so the neutral name `path!` stands (after the checker's `paths/` goldens, which show exactly such progressions); `leads_to!` was considered and rejected because it is wrong for `find`. Chains are allowed in the pattern and refused only in a claim, where the universal reading of a chain is ambiguous and no real claim needs one; adjacency without a gap is refused rather than meaning "next step".

2026-10-19, naming: `Progress` as the claim value type in this spec is `LeadsTo` (`Progress` is the attribute `#[progress]`, not a type); `LeadsTo: From<Path>` for two waypoints, `.assuming(..)` and `.within(n)` unchanged.
