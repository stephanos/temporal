---
status: research
accessed: 2026-10-09
---

# Gobra and Goose/Perennial for Umpire: verifying Go concurrency, and what to copy

This note looks at two verifiers for Go concurrency. Gobra is ETH Zurich's Viper-based verifier.
Goose/Perennial is MIT PDOS's tool chain: a Go-to-Rocq/Lean translator plus an Iris-based
concurrent separation logic, with the Grove extension for distributed systems. For each, the note
asks whether it could check parts of this repository and whether it could connect to the Umpire IR.
It then asks which of their ideas Umpire should copy even if neither tool is adopted.

The facts come from the tools' repositories, papers and CI, all read on 2026-10-09. Statements
marked **(unverified)** were not confirmed in a primary source. Statements marked **(estimate)** are
this note's own extrapolation.

## Summary

**Recommendation: do not adopt Gobra or Goose/Perennial for Temporal server code now. Copy five of
their ideas into Umpire and into Go-side checks (listed below).** If the owner wants one data point
backed by a proof, a 2-week Gobra spike on `common/locks.PrioritySemaphoreImpl` is the cheapest one,
but it is optional. Revisit Goose/Perennial once its Lean port settles, no earlier than 2027-04.

Why neither tool, in short:

| | Gobra | Goose/Perennial (with Grove) |
| --- | --- | --- |
| How proofs are found | Automatic (SMT via Viper/Silicon/Z3); the user writes `// @` annotations | Interactive (Iris Proof Mode in Rocq, now Lean 4); the user writes tactic proofs |
| Annotation or proof size | 0.3 to 3.1 annotation lines per Go line (CAV 2021); 2.8 (VerifiedSCION); 6.5 (WireGuard) | 11x to 19x proof lines per Go line (Tulip 11x, Grove 12x, vMVCC 13x, GoJournal 19x) |
| Go coverage | No generics: its Go grammar has no type parameters, and PR #671 has been open since 2023-08. `select` is not verified (issue #902 is open). Stubs exist only for `sync.Mutex` and `sync.WaitGroup` | The new Goose translates generics, `select`, `defer`, closures and type switches. It also translates the real `sync`, `sync/atomic` and `context` packages, which are then proven. `unsafe` pointer-to-integer conversions are not modeled |
| Largest real Go use | VerifiedSCION router: 4,700 lines of Go, 13,400 annotation lines, about 2.5 person-years, 3 h verification on a laptop. AWS SSM Agent (Diodon): a core of about 1% of 100k+ lines, under 3 person-months | Grove vKV and others: 2,435 verified Go lines, 28,077 proof lines. Tulip: about 4,000 Go lines, about 42,000 proof lines. Selected etcd functions are translated and partly proven in the repository; no paper on this was found |
| Concurrency | Lock and channel invariants, fractional permissions, goroutines. Iris-style atomics and invariants were merged on 2026-08-05 | Full Iris: logical atomicity, ghost state, prophecy, lock and condition-variable invariants |
| Crashes and distribution | None | Crash and recovery refinement (Rocq branch only); Grove handles lossy, duplicating networks, node crashes and leases |
| Liveness | Termination measures only; no deadlock freedom or progress | Safety only (Grove and Tulip say so) |
| Link to an abstract model | I/O specifications, with precedent: Igloo, VerifiedSCION (Isabelle), Arquint et al. (a tool generates Gobra specifications from Tamarin) | Refinement to a spec transition system (Perennial); state-machine specs with ghost permissions (Tulip PSM) |
| State on 2026-10-09 | Active: v26.02 released 2026-03-01; commits in 2026-09 | In transition: the default branch moved to a Lean 4 port on 2026-10-02 that has no crash reasoning; agent notes call the Rocq sources "frozen" |
| License | MPL-2.0 (some files Go BSD or CC0) | MIT |

In this repository, Gobra cannot parse most candidate modules because of generics. Its lack of
`select` blocks the lock helpers. Goose/Perennial can translate them, but each proof costs months of
Iris work. The framework is also mid-migration, so a proof started today risks having to be ported.
Neither tool proves liveness, and neither says anything about allocation counts. Lost wake-ups and
progress across composed machines are the properties Umpire plans to cover, and these tools would not
check them.

**Copy these, in order of value per cost** (details in [section 2](#2-transferable-insights)):

1. **Shadow refinement monitor (the I/O specification idea, checked at run time).** Every
   `chasm.Transition.Apply` already emits a `chasm.transition` span event (source, destination,
   event type, component) when `telemetry.DebugMode()` is on (`chasm/statemachine.go`), but nothing
   in the repository consumes it. A test-side span processor could map each event through an
   abstraction map declared in the Model, then check it against the machine's table in
   `tools/umpire/interp`. Cost: 1 to 2 weeks.
2. **Static transition-table conformance.** The 18 `chasm.NewTransition` values in
   `chasm/lib/activity/statemachine.go` are data (`Sources`, `Destination`). A Go test can check that
   every code edge is carried by a model row under the abstraction map, and report rows that no code
   edge realizes. Cost: 2 to 4 days.
3. **Executable lock invariants plus `checklocks`.** `matcherData` states its lock invariant in a
   comment ("all pollers and tasks in these data structures have matchResult == nil and queued ==
   true"). Make the invariant a function checked at unlock in test builds (the repository already has
   `softassert`), and annotate guarded fields with gVisor `+checklocks`. Cost: about 1 day to wire
   up, then hours per struct.
4. **Model-driven interleaving tests for lock helpers.** Write a small Umpire machine for a helper
   such as `PrioritySemaphore`, with 2 to 3 goroutines. Explore its schedules, then replay each
   schedule against the Go code under `testing/synctest` through test-only cut points. This covers
   bounded no-lost-wakeup, a property neither prover checks. Cost: 2 weeks for the harness, then days
   per helper.
5. **Crash and recovery as a Model derivation (from Perennial).** Declare which state fields are
   durable, derive a `crash` action that keeps only those, and check that crash plus recovery
   refines a single crash step of the product machine. Cost: 1 to 2 weeks in Scala, with no new IR.

On the owner's broader question: can models describe a Go module's goroutines, locks, channels,
ownership and allocations safely and efficiently?

- **Protocol** (who holds a lock, who waits, what is in a channel, who may write a field): Umpire
  can already describe this as finite machines and compositions, and explore it exhaustively for
  small bounds. Umpire channels already have the loss, duplication and ordering knobs that Grove
  and Igloo assume.
- **Memory safety and data races** (aliasing, heap ownership, fractional read sharing): this needs
  a separation-logic prover such as Gobra or Perennial. Umpire models have no heap. Go-side
  approximations are `checklocks`, the race detector and executable invariants.
- **Linking model and code**: proofs (Igloo-style I/O specifications) cost person-years at the
  router scale. Run-time and test-time conformance (items 1, 2 and 4 above) costs weeks.
- **Allocations and efficiency**: neither tool reasons about allocation counts or throughput.
  Perennial's Lean port bounds execution steps with "time receipts", not memory. Use benchmarks
  (`testing.AllocsPerRun`).

## 1. Leverage as-is

### 1.1 Gobra

**What it is.** "Gobra is an automated, modular verifier for Go programs, based on the Viper
verification infrastructure" ([README](https://github.com/viperproject/gobra)). The tutorial says
it "verifies memory safety, crash safety, data-race freedom, and partial correctness based on
user-provided specifications" ([tutorial](https://github.com/viperproject/gobra/blob/master/docs/tutorial.md)).
It translates annotated Go into Viper. Silicon (symbolic execution) is the default backend; Carbon
(via Boogie) is optional. Both use Z3.

**Maturity and maintenance** (GitHub API, 2026-10-09):

- Created 2020-09-16; last push 2026-10-08; 202 stars. The open count of 181 includes pull
  requests (163 issues and 18 PRs per the repository page).
- Releases: v22.10, v23.02, v24.02, v25.02, v25.09 (2025-09-25), and v26.02 (2026-03-01). v26.02
  moved Docker images to OpenJDK 21 and added `rel` expressions (hyperproperties). Commits from
  2026-08 and 2026-09 upgraded Z3 to 4.16.0, added a cancellation API and fixed interface dispatch.
- The README still calls it "a prototype verifier".
- Notable 2025 to 2026 features: package invariants (v25.09), "Hyper Gobra" (v25.09), ghost
  pointers and fields (v25.02), and physically atomic functions with Iris-style invariants
  (PR #983, merged 2026-08-05).

**Supported Go subset.**

- Supported, per the regression-test feature folders and the release notes: interfaces, closures
  (v22.10), `go` statements including closures (v24.02), channels, `sync.Mutex`,
  `sync.WaitGroup`, `defer`, maps, slices, strings, globals, type switches, labels, variadics,
  ADTs and ghost code, termination measures, and optional overflow checks.
- **Generics are unsupported.** Gobra parses with its own ANTLR grammar
  (`src/main/antlr4/GoParser.g4`), which has no type-parameter rules. PR #671 ("Parsing and
  type-checking generics") has been open since 2023-08-24 with no update since 2023-08-25.
- **`select` is not verified.** The grammar parses it, but issue #902 ("Add support for `select`
  statement") has been open since 2025-03-31.
- **Standard-library stubs** exist for `bytes`, `encoding`, `errors`, `fmt`, `net`, `strconv`,
  `strings`, `sync` and `time`. The `sync` stubs are only `mutex.gobra` and `waitgroup.gobra`: no
  `RWMutex`, `Cond` or `Once`. VerifiedSCION wrote its own specifications for `context`,
  `crypto/*`, `encoding/binary`, `golang.org/x/net`, `gopacket`, `prometheus` and others
  (`verification/dependencies`).
- **Channel close** is only partly supported. A regression test notes that `c.Closed()` "cannot be
  proved with the current version of Gobra"
  (`src/test/resources/regressions/features/channels/channel-simple9.gobra`).
- No `unsafe` tests or issues were found **(unverified that it is rejected)**. cgo is out of scope.
- No Go version is stated. Inference: the VerifiedSCION fork's `go.mod` says `go 1.18`, while
  upstream `scionproto/scion` says `go 1.26.4`.

**Annotation style.** Annotations live in `.go` files as `// @ requires …` lines and inline
`/*@ ghost x T @*/` parameters, which the Go compiler ignores. Separate `.gobra` files hold
predicates and ghost code. With `only_files_with_header`, a file is verified only if it carries a
`// +gobra` header (VerifiedSCION `gobra-mod.json`, `router/dataplane.go`). The concurrency
vocabulary works like this:

- `acc(x)` and `acc(x, 1/2)` are permissions, the second a fractional read share.
- `pred` declares a predicate.
- `m.SetInv(inv)` binds a lock invariant. `Lock()` hands the invariant to the caller and `Unlock()`
  takes it back.
- `c.Init(sendInv, PredTrue{})` binds a channel. `SendGivenPerm` and `RecvGotPerm` name what the
  sender gives up and what the receiver gains.

The full syntax is in the [tutorial](https://github.com/viperproject/gobra/blob/master/docs/tutorial.md),
sections Concurrency, Mutex and Channels.

**Toolchain.** The README lists Java 64-bit (tested with 11 and 15), sbt 1.4.4, and Z3 4.16.0 (the
CI version). Z3 4.15.0 or newer needs glibc 2.38 or newer. Boogie and Mono are needed only for
Carbon. CI uses `viperproject/gobra-action` (Docker).

**Performance on real code.**

- CAV 2021: 14 examples took 1 to 58 s each on a MacBook Pro i9. The concurrent ones took longest
  (53 s and 58 s for 35 and 31 Go lines with 94 and 98 annotation lines). The test suite of 407
  tests (10,030 lines) took 14.9 min.
- VerifiedSCION (arXiv v1, 2024-05):
  - Size: 4,700 lines of Go and 13,400 annotation lines. The annotation count includes 900 lines
    for the I/O specification; another 2,400 lines are trusted library specifications. That is 2.8
    annotation lines per Go line.
  - Effort and run time: about 2.5 person-years for the code, plus 2 to 3 person-years for the
    Isabelle protocol work. Verification takes 3 h on a laptop.
  - Coverage: 332 functions in 12 packages. Twelve rely on unproven lemmas, and three are only
    partly verified "because of performance problems of Gobra".
  - Code changes needed: a type that combined interfaces with delegation was rewritten, compound
    expressions were split, and some range loops became plain `for` loops.
- VerifiedSCION CI: the router step has a 6 h timeout. Of the last five workflow runs
  (2026-09-06 to 2026-09-08), the three that passed took 49 to 89 min of wall-clock time and the
  two that failed took 79 and 97 min.
- The CCS 2025 version of the paper (DOI 10.1145/3719027.3765104) was not read. Its numbers may
  differ **(unverified)**.

**CI integration.** VerifiedSCION verifies every push and PR with one `gobra-action` step per
package. Per-package `gobra.json` files set the options; the router uses `chop: 10`,
`mce_mode: on` and `conditionalize_permissions` ([gobra.yml](https://github.com/viperproject/VerifiedSCION/blob/master/.github/workflows/gobra.yml)).
The repository-wide config sets `"overflow": false`, so integer overflow is not checked there.

**License.** MPL-2.0. The `sync` stubs and builtins carry the Go BSD license; tests and `project/`
are CC0.

### 1.2 Goose / Perennial / Grove

**What it is.** Goose translates Go into GooseLang, a lambda calculus with heap and concurrency.
Perennial is an Iris-based program logic for proving the translated code. In its Rocq form it
covers crash safety: crash invariants, recovery helping, versioned memory and "concurrent recovery
refinement" ([Perennial SOSP 2019](https://www.chajed.io/papers/perennial:sosp2019.pdf)). Grove
extends it to distributed systems: RPC over a network that may lose or duplicate messages, node
crashes, TrueTime-like clocks and leases ([Grove SOSP 2023](https://pdos.csail.mit.edu/papers/grove:sosp23.pdf)).
The translator and the Go semantics are trusted.

**Maturity and maintenance** (GitHub API, 2026-10-09):

- `goose-lang/goose` (MIT; created 2019-02-09; last push 2026-04-07) now says "Development for this
  repository has moved to https://github.com/mit-pdos/perennial". Its README describes the old
  translation and is stale: it says "Assignments are not supported, only bindings".
- `mit-pdos/perennial` (MIT; 246 stars; last push 2026-10-09):
  - The **default branch is now `lean`**. It is a Lean 4 (v4.34.1) port on iris-lean, begun
    2026-10-02, with 129 commits in its first week. Of the latest 100 commits, 38 are authored as
    "Upamanyu Sharma's AI agent".
  - Its README states "No crash logic … proofs reason about executions without crashes". It adds
    step-bounded "time receipts".
  - Its `CLAUDE.md` tells agents not to consult the Rocq sources: "They are frozen and out of date,
    and the Lean statements are authoritative."
  - On `master` (Rocq), the last substantive commit was 2026-08-18 ("Re-goose etcd code"). The
    scheduled CI runs on 2026-10-07, 08 and 09 failed.
  - No public announcement of the migration was found **(unverified beyond the repository)**.

**Supported Go subset (new Goose).** Read from `perennial/goose/goose.go` on `master` and the
`goose/testdata/examples/unittest` file list:

- **Translated:** generics (with type parameters; `unnamed type parameters` is rejected),
  goroutines, `defer`, closures, `select`, type switches, interfaces, maps, slices, panics and
  globals.
- **Channels** come from a Go model ("ChanLib", `goose/model/channel`) that is itself translated. A
  2026-03-16 commit in goose-lang says "Clarify channel cv is unverified".
- **Not modeled or rejected:** method expressions, anonymous structs with fields, function type
  declarations, and some conversions. `unsafe.Pointer`/`uintptr` conversions are "stuck", meaning
  not modeled (lean README).
- **Standard library:** the real `sync` package (Mutex, RWMutex, Cond, Once, WaitGroup, semaphores),
  `sync/atomic` and `context` are translated, and proofs exist in `new/proof/sync_proof/*` and
  `new/proof/context.v`.
- **Selective translation:** a per-package `.v.toml` lists which declarations to translate and
  which imports to keep. The etcd config translates only `EtcdServer.Put`,
  `EtcdServer.processInternalRaftRequestOnce` and a few helpers, and keeps imports such as
  `etcdserverpb`, `context`, `otel/trace` and `prometheus`. This is how a large real codebase was
  brought in without translating all of it.
- The module requires Go 1.26 (`perennial/go.mod`).

**Proof style and burden.** Proofs are interactive separation-logic proofs in Iris Proof Mode, not
SMT. Published ratios:

| System | Go lines | Proof lines | Ratio | Source |
| --- | ---: | ---: | ---: | --- |
| GoJournal | 1,345 | 25,797 | 19x | [OSDI 2021](https://www.chajed.io/papers/gojournal:osdi2021.pdf) |
| vMVCC | 827 | 11,117 | 13x | [OSDI 2023](https://pdos.csail.mit.edu/papers/vmvcc:osdi23.pdf) |
| Grove (vKV and libraries) | 2,435 verified + 170 trusted | 28,077 | 12x | Grove Fig. 10 |
| Tulip | about 3,956 | about 42,000 | 11x | [SOSP 2026](https://people.csail.mit.edu/nickolai/papers/chang-psm.pdf) |

Grove trusts a 120-line network library and a 50-line filesystem library. Grove's lease extension
changed five components while the earlier protocol proofs "remained the same". Grove: "Grove
cannot verify liveness properties". Tulip: its proof "does not capture liveness".

**Toolchain.** The Go translator (`goose`) emits Rocq on `master`, or Lean 4 plus iris-lean on
`lean`. Builds use opam or nix on master and lake on lean. No published build or proof-check times
were found for the full development **(unverified)**.

**CI.** `master` has scheduled CI (about 40 min per run in the observed failures). The Goose
translator has gold-file tests and a semantics test package that runs the same Go functions under
`go test` and under a verified interpreter ([Waddle](https://pdos.csail.mit.edu/papers/gibsons-meng.pdf)).

### 1.3 Candidate targets in this repository

Three targets were inspected. A fourth, `common/locks.IDMutex`, is the friendliest for Gobra (no
generics, no `select`), but nothing outside its package constructs it, so it is not worth proving.

Repository-wide note: by a rough grep, at least 111 of the 1,818 non-test Go files under `common/`,
`service/` and `chasm/` declare generic types or functions, and many more use them. Gobra cannot
parse any of those files.

| Target | Size | Go features used | Gobra blockers | Goose/Perennial blockers | Dependency surface needing trusted specs | Properties worth proving | Estimated cost (estimate) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `common/locks/priority_semaphore_impl.go` (`PrioritySemaphoreImpl`; used by the history workflow-context lock and the shard IO semaphore) | 228 lines, about 150 of code | `sync.Mutex`, `select` with `default`, `close(chan)`, `container/list` holding `any` with a type assertion, `context`, `defer`, `panic` | `select` (#902) in `Acquire`; closing a channel to transfer tokens has no reasoning support; needs a `container/list` spec | None in the language; `container/list` must be translated or modeled | `context` (VerifiedSCION has a stub), `container/list`, `serviceerror` | Accounting: `0 <= cur <= size`, and `cur` equals the summed weight of holders (ghost). A canceled acquire returns its tokens exactly once. No double unlock | Gobra: 2 to 3 person-weeks and about 400 to 500 annotation lines, after wrapping each `select` in a trusted helper. Goose: 1 to 2 person-months and about 1,700 to 2,900 proof lines. Neither proves "no waiter stays blocked while tokens are free" |
| `service/matching/matcher_data.go` (`matcherData`, the sync-match core of `priTaskMatcher`) | 766 lines; `pri_matcher.go` adds 704 | `sync.Mutex`, `sync.Cond` whose `L` is the matcher lock, a `context.AfterFunc` closure that takes the lock, `defer`, an intrusive doubly linked list (`pollerList`), generic `tidwall/btree.BTreeG`, `unsafe.Pointer` to `uintptr` in `taskBTreeLess`, timers, a rate limiter | Generics (the `btree.BTreeG` field) fail at parse; `unsafe`; no `sync.Cond` stub; closures inside `AfterFunc` need specs | `unsafe` tie-breaker is stuck; `tidwall/btree` must be translated or modeled | `tidwall/btree`, `context.AfterFunc`, `clock`, `log`, `softassert`, `util`, `time` | The stated invariant (queued means `matchResult == nil && queued`); `wake` at most once per waiter; a task is matched to at most one poller; no race on `matchResult` | Either tool first needs refactors: a sequence-number tie-breaker instead of `unsafe`, and the B-tree behind a non-generic interface (Gobra only). Then Gobra: 2 to 4 person-months, about 1,700 annotation lines. Goose: 4 to 8 person-months, about 7,000 proof lines |
| `chasm/lib/activity/statemachine.go` | 771 lines, 18 `chasm.NewTransition` values | Generic `chasm.Transition[S, SM, E]`, protobuf messages (`activitypb`, `historyservice`, `workflowservice`), `chasm.MutableContext` task scheduling, metrics. No local concurrency (the CHASM engine serializes access) | Generics in `chasm/statemachine.go` | None in the language; `MutableContext`, protobuf types and metrics would be trusted models (etcd shows protobuf packages can be translated, at size: `etcdserverpb.v` is 384 KB) | `chasm`, generated protobuf code, `metrics`, `timestamppb` | Each transition's effect matches the Model row it realizes (functional refinement) | Not recommended as a proof target. The concurrency risk is low and the refinement is checkable more cheaply by the static table test and the shadow monitor ([section 2](#2-transferable-insights), rows 1 and 2) |

**An observation from the read-through (not run):** `PrioritySemaphoreImpl.Release` defers
`s.mu.Unlock()` and also calls `s.mu.Unlock()` before `panic("semaphore: released more than
held")`. If that panic fires, the deferred unlock runs on an unlocked mutex. Go treats that as a
fatal error that cannot be recovered, so the intended recoverable panic becomes a process crash.
Gobra's `Unlock` precondition (`m.UnlockP()`) would reject the second unlock. A two-line fix is
possible, but it is outside this note's scope; this file changes no code.

### 1.4 Connecting either tool to the Umpire IR

**Precedent exists, and it is strongest for Gobra.**

- **Igloo** ([OOPSLA 2020](https://arxiv.org/abs/2010.04749)) links an Isabelle/HOL event system
  to code through *I/O specifications* (from Penninckx et al., ESOP 2015).
  - An I/O permission `bio(t, v, w, t')` allows one I/O operation from place `t` to place `t'`.
    A `token(t)` marks the current place.
  - Each model event becomes a co-recursive predicate that holds the permission for each allowed
    output and the predicate again for the updated model state. Internal events become ghost I/O
    actions.
  - The translation is formalized in Isabelle. Moving it into the verifier's syntax was manual.
  - Trusted: the tools, that manual step, and the faithfulness of the environment model.
  - Case studies used VeriFast (Java) and Nagini (Python). The verified properties are safety
    only.
- **VerifiedSCION** uses Igloo with Gobra. The authors "extended Igloo's tooling to automatically
  generate the I/O specification from the routers' models within Isabelle/HOL, along with a
  correctness proof showing its trace equivalence with the router model". Only "a small, mostly
  syntactic step, where we manually translate the I/O specification from Isabelle to Gobra syntax,
  is unverified". The I/O specification and its definitions take 900 lines.
- **Arquint et al.** ([IEEE S&P 2023](https://arxiv.org/abs/2212.04171)) *automatically generate*
  Gobra (and VeriFast) I/O specifications from Tamarin protocol models
  ([tool](https://github.com/viperproject/protocol-verification-refinement)). For WireGuard, 1,241
  of 3,936 annotation lines (32%) were generated, for 608 lines of Go. Verification took 148 s and
  138 s per role. A deliberately faulty implementation fails because "the I/O permissions do not
  permit sending this payload".
- **Diodon** ([IEEE S&P 2026, arXiv 2507.00595](https://arxiv.org/abs/2507.00595)) applied this to
  the production AWS SSM Agent (100k+ lines). Gobra verified a core of about 1% of the code in
  under 3 person-months, with a 1.17 min run. Static analyses (Argot taint analysis, Capslock)
  showed that the rest of the code cannot break the core's assumptions.
- **Perennial** states correctness as refinement between two transition systems, the code and a
  specification, including crash transitions. **Tulip** (SOSP 2026) writes TLA-style
  state-machine specifications whose modules interact only through separation-logic
  "permissions" ("permissioned state machines").

**What an Umpire exporter would emit.** An Umpire machine is a finite, labeled transition system:
it has state, outcome and fact catalogs, action classes, and rows from (state, class) to results.
That is the shape Igloo starts from. Two exporters are possible:

- **Umpire IR to Gobra I/O specification:**
  - an ADT per state, outcome and fact catalog;
  - a pure `step(s State, c Class) seq[Result]` lowered from the IR step functions (the Quint
    exporter already lowers these, in 1,069 lines of `tools/umpire/export/quint.go`);
  - one abstract I/O permission per action class, guarded by `step` being non-empty;
  - places as Gobra `Place` values.
  The Go code would take an `io.IOToken(place)` and consume one permission per observable
  operation. Cost: about 2 to 3 weeks for the exporter (estimate). The proof work it enables would
  still cost person-months per module.
- **Umpire IR to a Perennial specification:**
  - a Lean (or Rocq) `def step : State → Class → List Result`;
  - a ghost "authoritative model state" resource;
  - one logically atomic specification per Go entry point, each committing exactly one model
    step.
  This matches Perennial's refinement style. Because the Lean port started a week ago, its
  interfaces will move.

**Why the link does not pay off today.**

1. **Level mismatch.** Today's Umpire machines describe product and system behavior (the activity
   record, the task queue). Gobra- and Perennial-sized targets are lock helpers and the matcher
   core, which no Umpire model describes yet. A module-level Model would come first, and then an
   abstraction function from Go state (protobuf status, attempt counters, lists) to model state.
   Today the Models declare refinements between machines (SEMANTICS.md, Machines 6), not from Go
   state.
2. **Granularity.** Umpire's realization boundary is RPCs and history events, observed by
   Testpilot. An I/O specification needs every observable operation inside the module, such as
   each persistence write and each matching dispatch, to be a permission-guarded call. The repo's
   hold/release actuators (`ControlKind.HoldDispatched`) sit at the same kind of cut point, but
   they are test controls, not specifications.
3. **What carries over.** Igloo, Grove and Tulip carry only safety (trace inclusion). Umpire's
   progress claims and planned bounded liveness across compositions
   (`.flow/specs/fn-150-bounded-liveness-across-composed.md`) would not carry over. Holes would
   have to become trusted or unknown I/O permissions.
4. **"One meaning, one source."** An exported Gobra or Lean specification would be a second opinion,
   as Quint is today (`tools/umpire/export/README.md`). It would need its own agreement receipt
   showing that the exported `step` equals the reader's table. That is feasible: the Quint
   agreement compares 42,506 state and class pairs.

The run-time form of the same idea (section 2, row 1) gets most of the bug-finding value without a
prover, because it checks the same "every code step is a model step" claim on executed traces.

## 2. Transferable insights

| # | Insight (source) | Mapping onto Umpire or Go | Cost (estimate) | Value |
| --- | --- | --- | --- | --- |
| 1 | **I/O specifications as the refinement link** (Igloo, VerifiedSCION, Arquint) | **Shadow refinement monitor.** The Model declares an abstraction map in the realization: CHASM component type, source status and event type map to a model class and state. A span processor collects `chasm.transition` events from functional tests (already emitted when `telemetry.DebugMode()` is on). The checker replays each event through `tools/umpire/interp` and fails on a pair the table disables or a result the row does not list. `p-org/PObserve` ("Monitoring P Specifications on Traces") is the industrial analogue | 1 to 2 weeks: a processor in test setup, a map in Scala, and an interp lookup. The abstraction map is Model knowledge, so it is declared in Scala, not in Go | High: it checks every test run, not only generated Cases, and catches a code edge the Model forbids |
| 2 | **The transition relation is data** (Perennial spec transition systems, Igloo events) | **Static table conformance.** A Go test reads the 18 activity `Transition` values (`Sources`, `Destination`, event type) and the activity IR. It checks that each code edge maps, through the same abstraction map, to some row, and lists rows that no code edge realizes. The same applies to other CHASM libraries (`chasm/lib/nexusoperation` already enriches its telemetry) | 2 to 4 days | High: cheap, deterministic, and runs in `go test` |
| 3 | **Lock invariants** (Gobra `SetInv`/`LockInv`, Iris lock invariants) | **Go:** for each struct whose comment says a lock "covers everything below", add a `checkInv()` and call it before each unlock in test builds through `softassert` (for example a `lockWithInv` helper). Add gVisor `checklocks` (`+checklocks:lock` on fields, run as a vet tool). Limits: `checklocks` cannot annotate closures (such as the `AfterFunc` callbacks), and fields of `waitableMatchResult` guarded by `matcherData.lock` cannot name that lock from their own declaration, so they need `+checklocksignore` or a restructure | About 1 day to wire up `checklocks`, then hours per struct | Medium to high: turns prose invariants into failing tests and static errors |
| 4 | **Permission accounting and ghost state** (fractional permissions, ghost fields in Gobra; Iris ghost resources) | **Go:** test-only ghost counters behind a build tag, for example the summed weight held per semaphore or wakes per waiter, checked at the points a proof would assert them. **Umpire:** in module Models, model ownership as an owner field per resource over a finite set of actors, and read sharing as a bounded count | Hours per counter; Model work is part of row 6 | Medium |
| 5 | **Channel invariants** (Gobra `SendGivenPerm`/`RecvGotPerm`; Actris-style channel protocols appear in recent Goose commits, unverified depth) | **Umpire:** an optional message invariant on a channel declaration (SEMANTICS.md, Channels), checked by interp whenever a step appends to or delivers from the channel, so one declaration replaces a Property per sender. **Go:** document ownership transfer on send ("after `c <- p` the sender does not touch `*p`") and back it with race-detector tests | 2 to 3 days in IR, reader and Scala | Medium |
| 6 | **Logical atomicity and linearization points** (Perennial, Grove, vMVCC) | **Umpire realizations:** name, per action class, the cut point at which its model step takes effect, for example the durable commit before the response. Generate Cases that hold before and after that point and check observations on both sides. This extends `HoldDispatched` and the `admissionResponseLoss` fault. Lock helpers then get module Models whose steps are one critical section each, explored over small goroutine counts and replayed under `testing/synctest` (GA since Go 1.25; already used in `service/matching/pri_matcher_test.go`) through test-only hooks | 1 week for the realization field; 2 weeks for the synctest replay harness, then days per helper | High for lock helpers: it covers bounded no-lost-wakeup, which neither prover checks |
| 7 | **Crash and recovery refinement** (Perennial crash invariants, recovery helping; Grove node crashes) | **Umpire DSL:** a derivation that takes a durable projection of the state, adds a `crash` action that drops volatile fields, and checks that crash plus recovery steps refine one crash step of the product machine. This generalizes the hand-written `VolatileQueue`, `ForgetfulQueue` and `LossyMatchingQueue` providers into one rule | 1 to 2 weeks in Scala and model tests; no new IR (it lowers to rows) | Medium to high for persistence-backed machines (task backlog, history mutable state) |
| 8 | **Network, clock and lease models** (Grove: lost and duplicated messages, TrueTime-like `GetTimeRange`, time-bounded invariants) | Umpire channels already have `lossy` and `duplicates d`. Add **time-bounded facts** (a lease is valid until a timer action fires), for example for task-queue partition ownership through persistence range IDs | Days of Model work per use | Medium |
| 9 | **Explicit trusted boundary** (Gobra stubs, Perennial `TrustedCode`, Grove's 120-line trusted network library, VerifiedSCION's 2,400 trusted lines) | Umpire already has opaque providers with `assumes` and `replaces`. Add a receipt that lists every trusted provider or assumption a verdict rests on, and require one adversarial negative-control Case per assumption | Days | Medium |
| 10 | **Small deep core plus cheap checks on the rest** (Diodon: Gobra on 1%, static analyses on 99%) | Choose a few concurrency cores (the semaphore, `matcherData`, CHASM transitions) for rows 1 to 6, and run automatic analyzers (`checklocks`, `go vet`, the race detector in CI) over everything else | Ongoing | Medium: sets scope honestly |
| 11 | **Differential testing of a trusted semantics** (Goose's semantics tests run under both `go test` and a verified interpreter) | Umpire's Quint agreement already does this for the IR. Any future Gobra or Lean exporter needs the same receipt | Included in any exporter cost | Required for any exporter |
| 12 | **Permissions between composed modules** (Tulip PSM) | Umpire compositions already restrict interaction to syncs. A later step could declare which member may fire which sync, as tokens, and lint compositions where a member steps another's state outside a sync | Unclear | Low to medium now |

Allocations: no row addresses them. Neither tool models allocation counts or performance, and an
Umpire model should not either. Use `testing.AllocsPerRun` benchmarks and pprof.

## Risks and open questions

- **Perennial's moving target.** The Lean port became the default branch one week before this
  note was written, drops crash reasoning, and is partly written by an AI agent. Whether crash
  logic returns, and whether the Rocq `master` stays maintained, is unknown. The only evidence is
  the `CLAUDE.md` statement that the Rocq sources are "frozen".
- **Gobra and generics.** Nothing shows generics support is coming: the PR has been stale since
  2023. Temporal's use of generics grows with each Go release. Unless that changes, most of
  `service/` and `chasm/` stays out of Gobra's reach.
- **Cost figures transfer poorly.** The ratios come from code that verification experts wrote or
  chose. Temporal code was not written for verification, and the estimates in section 1.3 extrapolate
  from those ratios without any trial **(estimate)**.
- **Liveness.** The defects Umpire most wants to catch in matching and history are lost wake-ups,
  stuck tasks and missed timers. All of these are progress properties, and none of the surveyed
  proofs (Gobra, Grove, Tulip, Igloo) covers them. Bounded checking in Umpire and synctest replay
  remain the tools for them.
- **Abstraction map ownership.** Rows 1 and 2 need a map from Go state to model state. Under
  "Models are the only smart component" it belongs in Scala realizations. A new IR field and its
  admission rules are needed. Open question: is the CHASM telemetry (status strings, event type
  names) a stable enough contract for that map?
- **Telemetry gating.** The transition event is emitted only when `telemetry.DebugMode()` is on.
  Test environments must enable it, and production must be unaffected.
- **Unverified items:**
  - the CCS 2025 numbers for VerifiedSCION;
  - whether Gobra rejects `unsafe`;
  - how far the Goose channel model and `context.AfterFunc` are proven;
  - full Perennial build times;
  - whether a paper on the etcd work exists;
  - the scope of the Huawei EuroSys 2026 paper the Gobra page lists. Its link to Gobra was not
    confirmed.

## Sources

All accessed 2026-10-09.

Gobra and VerifiedSCION:

- Gobra repository and README: https://github.com/viperproject/gobra
- Gobra releases (v22.10 to v26.02): https://github.com/viperproject/gobra/releases
- Gobra tutorial: https://github.com/viperproject/gobra/blob/master/docs/tutorial.md
- Gobra `sync` stubs: https://github.com/viperproject/gobra/tree/master/src/main/resources/stubs/sync
- Gobra Go grammar: https://github.com/viperproject/gobra/blob/master/src/main/antlr4/GoParser.g4
- Gobra PR #671 (generics): https://github.com/viperproject/gobra/pull/671
- Gobra issue #902 (`select`): https://github.com/viperproject/gobra/issues/902
- Gobra PR #983 (atomics and invariants): https://github.com/viperproject/gobra/pull/983
- Gobra LICENSE: https://github.com/viperproject/gobra/blob/master/LICENSE
- Wolf et al., "Gobra: Modular Specification and Verification of Go Programs", CAV 2021,
  extended version: https://arxiv.org/abs/2105.13840
- ETH Gobra project page and publication list: https://www.pm.inf.ethz.ch/research/gobra.html
- VerifiedSCION repository, README, `gobra.yml`, `gobra-mod.json`, `router/dataplane.go`:
  https://github.com/viperproject/VerifiedSCION
- Pereira et al., "Protocols to Code: Formal Verification of a Next-Generation Internet Router",
  arXiv v1 2024-05-09: https://arxiv.org/abs/2405.06074. The CCS 2025 version, DOI
  10.1145/3719027.3765104, was not read. Artifact: https://zenodo.org/records/16891070
- Sprenger et al., "Igloo: Soundly Linking Compositional Refinement and Separation Logic for
  Distributed System Verification", OOPSLA 2020: https://arxiv.org/abs/2010.04749
- Arquint et al., "Sound Verification of Security Protocols: From Design to Interoperable
  Implementations", IEEE S&P 2023: https://arxiv.org/abs/2212.04171. Tool:
  https://github.com/viperproject/protocol-verification-refinement
- Arquint et al., "The Secrets Must Not Flow: Scaling Security Verification to Large Codebases"
  (Diodon), arXiv 2507.00595, to appear at IEEE S&P 2026: https://arxiv.org/abs/2507.00595

Goose, Perennial, Grove and Tulip:

- Goose repository (moved notice): https://github.com/goose-lang/goose
- Perennial repository, `lean` branch (default) README and `CLAUDE.md`, and `master` README:
  https://github.com/mit-pdos/perennial
- New Goose translator source: https://github.com/mit-pdos/perennial/tree/master/goose
- Goose channel model: https://github.com/mit-pdos/perennial/tree/master/goose/model/channel
- etcd translation configs:
  https://github.com/mit-pdos/perennial/tree/master/new/code/go_etcd_io
- Chajed et al., "Verifying concurrent, crash-safe systems with Perennial", SOSP 2019:
  https://www.chajed.io/papers/perennial:sosp2019.pdf
- Chajed et al., "GoJournal: a verified, concurrent, crash-safe journaling system", OSDI 2021:
  https://www.chajed.io/papers/gojournal:osdi2021.pdf
- Chang et al., "Verifying vMVCC", OSDI 2023: https://pdos.csail.mit.edu/papers/vmvcc:osdi23.pdf
- Sharma et al., "Grove: a Separation-Logic Library for Verifying Distributed Systems", SOSP 2023:
  https://pdos.csail.mit.edu/papers/grove:sosp23.pdf
- Chang et al., "Verifying a high-performance distributed transaction system using permissioned
  state machines" (Tulip), SOSP 2026:
  https://people.csail.mit.edu/nickolai/papers/chang-psm.pdf
- Gibson, "Waddle: A proven interpreter and test framework for a subset of the Go semantics":
  https://pdos.csail.mit.edu/papers/gibsons-meng.pdf

Alternatives:

- gVisor checklocks: https://pkg.go.dev/gvisor.dev/gvisor/tools/checklocks
- Go 1.25 release notes (`testing/synctest` GA): https://go.dev/doc/go1.25
- Dafny Go compilation: https://dafny.org/latest/Compilation/Go
- PObserve: https://github.com/p-org/PObserve; the P language: https://github.com/p-org/P

In this repository:

- `chasm/statemachine.go`, `chasm/lib/activity/statemachine.go`, `service/matching/matcher_data.go`,
  `service/matching/pri_matcher.go`, `common/locks/*.go`
- `model/SEMANTICS.md`, `tools/umpire/export/README.md`,
  `model/temporal/features/activity/standalone/system/Realization.scala`,
  `.flow/specs/fn-150-bounded-liveness-across-composed.md`
- `.plans/archive/UMPIRE4_RESEARCH.md`, the earlier Grove/Perennial entry
