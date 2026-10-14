# Proposal: key-level bounded composition over named `*umpire.Table` members (Phase B; requires Phase A)

Status: proposed and unrun. No Flow ID is minted. This is **Phase B** of the consolidated L prerequisite in `orchestration-recommendation.md`. It is **sequential after Phase A (key-level query)** and not parallel with it. It consumes Phase A's `UnknownPair` table contract, `(*Table).Model()` and error wrapping. Run as a separate task, it would need `depends_on` Phase A and a Touches overlap on `model/go/umpire/table.go`.

## Title

Dynamic bounded composition of IR-built tables in `model/go/umpire`

## Description

IR compositions (`ir.proto:462-495`: `Composition`, `Member{field, machine, replaces}`, `Sync`, `ends`) cannot use `Compose[S]`. Its state must be a compile-time struct with `umpire:"field"` tags, found by reflection (`compose.go:287-310`, `:555-570`). Its `Ends` is `func(S) bool` (`:98`). Its replacement goes through the unexported `replacing` interface of typed machines (`:77-80`, `:195`). Phase B adds `ComposeTables`, a dynamic entry point that runs **the same composition algorithm** over named member `*Table`s, syncs and an error-aware ends predicate, under a mandatory work and state ceiling. The typed `Compose[S]` path keeps its behavior byte-for-byte. Both paths share one non-generic core, which removes the risk that two copies drift apart.

### Grounding

- **Starts.** Every composed start is the Cartesian product of member starts in member order, with the last member varying fastest (`compose.go:175-187`). Starts are listed deduplicated in that order (`:494-498`).
- **Actions.**
  - Synced classes are the product of the two members' classes. Each is keyed by the sync name followed by each class's inputs (`:332-337`).
  - Own classes are keyed `<field>_<class>` (`:339-346`).
  - All classes are sorted by composed key (`:347`).
  - A sync naming a missing member or action is an error (`:321-331`).
- **Results.**
  - A synced step takes the product of the members' results (`:371-402`).
  - The outcome is the first move's, as `<field>_<outcome>`. Facts are every move's, as `<field>_<fact>`, in member order.
- **Collision.** `remember` rejects two different member-state vectors that share one `_`-joined key (`:404-417`).
- **Exploration.** Breadth-first from all starts, with **no ceiling** (`:429-453`).
- **Layout.**
  - States are sorted by key and rows run states-major over sorted actions (`:457-536`).
  - Outcomes and facts are prefixed per member in member order (`:467-479`).
  - State fields follow `composedFields`, which drops a refining member's refined field (`:541-552`).
  - The owner is `compose-<name>` (`:155`).
- **Replacement.** `checkRefinement(true)`, i.e. cover-starts (`:200`). `dischargedBy` drops, from the replacing member's own assumptions, those named by the replaced table (`:233-246`). Assumptions merge by name (`:248-260`). Fair entries map to composed classes, and an unknown fair entry is an error (`:264-285`).
- **Refined field.** goir appends the refined product to `StateFields` (`goir/machine.go:331-335`), but a `NewTable` table leaves `refinedField` empty (`table.go:59`, `:265-279`; only typed machines set it at `machine.go:297`). Composing goir members through the unchanged algorithm would therefore keep the `back_store` state field that typed composition drops.
- **Front-end divergence.** Task 2's Scala `Compose.scala` composes only `tables.map(_.starts.head)` (task2-review-r2 `model/scala/umpire/Compose.scala:104`). Task 16 relocates this file and must preserve it unchanged. This phase does not touch Scala. It pins Go's all-starts behavior, and front-end parity correction stays explicit future work.

## Touches

`[model/go/umpire/**]`

Expected files: `compose.go` (extract a non-generic core; the typed `Composition[S]` delegates to it), `table.go` (`TableSpec.RefinedField` only), and new `composekeys.go` and `composekeys_test.go`. `compose_test.go` must pass unmodified.

## Proposed additive API (indicative)

```go
type ComposeMember struct {
    Field    string
    Table    *Table
    // Replaces, when set, is the table this member stands in for; Refinement reads Table as it.
    Replaces   *Table
    Refinement RefinementSpec // CoverStarts is forced true
}

type ComposeSync struct{ Name, FirstMember, FirstAction, SecondMember, SecondAction string }

type ComposeSpec struct {
    Family  Family
    Name    string
    Members []ComposeMember
    Syncs   []ComposeSync
    // Ends reads a composed state by its key and its member state keys in member order.
    Ends func(key string, parts []string) (bool, error)
    // Ceiling bounds composed states and state×action evaluations; required (>0).
    Ceiling ComposeCeiling // {States, Evaluations int64}
}

// ComposeTables builds the composed table, or a *ComposeLimitError, *RefinementError or *Error.
func ComposeTables(spec ComposeSpec) (*Table, error)
```

- **Result provenance.** Each composed `Result.Step` is a `ComposedStep{Parts []string; Moves []MemberMove{Member int; Row string; Result int}}`. It is not serialized and not fingerprinted (`table.go:27`). A caller such as goir can build the composed step record from member values without re-deriving it. The typed path keeps its `Step[S, string, string]` (`compose.go:531`).
- **`(*Table).Parts(key)`** returns the member state keys of a composed state, and the Phase A `Model()` handle applies.
- **`TableSpec.RefinedField string`** (additive). `NewTable` copies it into `refinedField`, and only `composedFields` and `FieldValues` read it. goir setting it is task 3's work. No existing fingerprint reads it.
- **Unknown propagation (Phase A contract).**
  - A composed pair is unknown when at least one member move is an unknown pair and no member move of that composed action is disabled.
  - Disabled dominates unknown, because a sync needs both moves. Unknown dominates enabled.
  - Composed unknown pairs go into the composed table's `Unknown` with the member cause wrapped. Exploration does not pass through them.
  - Reachability through known rows is unchanged.
- **Ceilings before allocation.**
  - `Ceiling` must be positive; zero or less is a declaration error.
  - The start product size is computed overflow-safe before the start list is built, and `> States` is refused.
  - Each newly reached composed state and each state×action evaluation is counted before it is stored.
  - Exceeding a ceiling returns `*ComposeLimitError{Composition, Resource ("states"|"evaluations"), Ceiling, Needed int64 (at least), Overflow bool}` and **no table**. A truncated table is never returned.
- **Replacement.**
  - `RefineTables(member.Table, member.Replaces, spec with CoverStarts=true)` runs before exploration. Its `*RefinementError` (Kind, Witness, ProductWitness) is returned unwrapped.
  - Discharge and merge rules are exactly those of `compose.go:210-285`, with the replaced table's `Assumptions` as the discharge set.
  - A member naming `Replaces` without a `MapState` is a declaration error.
- **Unchanged checks.** Collision (`remember`), sync validation and fair-class mapping errors keep their current messages. Member-field validation replaces the struct-tag lookup. Duplicate or empty fields are a declaration error, and the typed path keeps its tag error.
- **Monitors.** The dynamic entry point takes no monitors and applies none. Whether a composition member's monitors watch a composed Query is unspecified (SEMANTICS task2 R2 lines 166-182 cover only machines). Task 3 refuses such Queries with a located `unsupported` receipt.

## Acceptance

- [ ] Every existing `compose_test.go` test and every typed-composition answer is unchanged: table, IDs, `TargetFingerprint`, starts, assumptions, and collision and replacement errors.
- [ ] Parity twins: for each typed composition in `compose_test.go`, `ComposeTables` over `NewTable` copies of its members yields equal:
  - `States`, `Actions`, `Outcomes`, `Facts`, `StateFields`, `Starts`, `Ends`, `Rows` (keys, order, results and facts), `Assumptions`;
  - `IDs()` and `TargetFingerprint()`.

  The twins cover a refining member with `RefinedField` set.
- [ ] Composition starts are the full Cartesian product of every member `Table.Starts`, with the last member fastest. The replacement check covers every opaque start (twin of `compose_test.go:162`), and a violating provider is rejected with a replayable witness (twin of `:149`).
- [ ] With a tight `States` or `Evaluations` ceiling, a `*ComposeLimitError` carries the ceiling and count, and a start product above the ceiling is refused before its list is allocated. The tenfold probe completes under a sufficient ceiling and is limit-refused under an insufficient one, never truncated.
- [ ] Unknown propagation: a member unknown pair yields a composed unknown pair. A sync whose other move is disabled stays disabled. Exploration does not pass through unknown pairs. A Phase A key-level verify over the composed table becomes `Incomplete()` only when that composed pair is explored.
- [ ] `Ends` callback errors propagate with their type (`errors.As`).
- [ ] Separator collisions in member keys are rejected as today (twin of `compose_test.go:200`).

## Focused tests (proposed)

- `TestComposeTablesMatchesTypedComposition` (table-driven twins)
- `TestComposeTablesStartsInEveryMemberStart`
- `TestComposeTablesReplacementCoversEveryOpaqueStart` and `TestComposeTablesViolatingProviderFails`
- `TestComposeTablesDischargesOnlyTheReplacingMembersAssumptions`
- `TestComposeTablesCeilingBeforeAllocation` (states, evaluations, start product, overflow)
- `TestComposeTablesPropagatesUnknownPairs` (own, sync-with-disabled, sync-with-enabled)
- `TestComposeTablesRejectsCollidingKeys`
- `TestComposeTablesEndsErrorKeepsItsType`

## Prerequisites and ordering

Phase A of the same prerequisite, which in turn depends on task 14. Task 3 depends on this phase for the IR compositions `pair` and `detailedPair` (Declarations fixture, task2 R2 lines 122-131) and their Queries. `detailedPair.bothPut` stays `unsupported` while member monitors are undefined, because `disk` names monitors (fixture lines 97-107).

## Out of scope

- IR reading of compositions and composed step records (task 3).
- The composed step record's IR typing. SEMANTICS Compositions keys outcomes and facts as `<field>_<key>`, and model/go and Scala both read them as strings (`compose.go:531`). Task 3 records this in SEMANTICS as a clarification.
- Scala `starts.head` parity (future front-end work).
- Composition monitors.
- New collision rules for `<field>_<class>` action keys (the current engine does not check them; preserve).
