---
satisfies: [R12, R7]
---
# fn-85-model-side-effects-as-typed-actions-and.12 Canary and exploratory sets admitted with coverage targets

## Description
Admit the two remaining set purposes over the Nexus Model (R12, the rest of R7): a canary set over Queries 1 and 2 with `handler: observed` admits, a canary set naming a Query whose Case carries a white-box Known Gap rejects, and an exploratory set over the protocol machine enumerates its coverage targets (rows, result values, members of claimed classes) deterministically under its budget, pinned by a golden. Running them stays with fn-70, fn-29 and fn-33.

**Size:** S
**Files:** `model/Umpire/Command/Syntax.lean:2337-2490` (the `set` command .7 landed: exploratory coverage-target enumeration; the canary white-box check goes where a Case is produced, see the dated note), `model/Umpire/Command/Records.lean` (`CoverageTarget`; `PartyBinding`, `SetPurpose`, `CoverageGoal` and `SetDeclaration` with `cover`/`budget` exist since .7), `model/Temporal/Case/Syntax.lean:183-260` (the set-realizing `case` block, which rejects a non-functional set today), `model/Temporal/Feature/Nexus/Caller/Model.lean` (the two sets), `model/Temporal/Feature/Nexus/Caller/Tests.lean` (`#guard_msgs` for the rejections), `model/Temporal/Feature/Nexus/Caller/Fixtures/CallerExploratoryCoverage.json` (new golden, written by `Temporal.Tool.Goldens`), `model/Temporal/Tool/Goldens.lean`, `Makefile` (`UMPIRE_GOLDEN_DIRECTORIES` if a new directory)
**Touches:** [model/Umpire/Command/**, model/Temporal/Feature/Nexus/Caller/**, model/Temporal/Tool/Goldens.lean, Makefile]

### Approach
- White-box Known Gaps are the `capability` or `interpretation` kinds a Case records from an `unobservable` row or an unbindable setup parameter; the canary admission walks the would-be Case's gaps and rejects naming the Query and the gap.
- Coverage targets: enumerate rows by position, result values by enum, and class members by the product of constructor fields; order is the declaration order (AUT-09) so the golden is stable; the budget is a `limits` name and bounds the enumeration.
- Goldens change only through `make umpire-gen-goldens`; add the new file to the checked directories.
- Adjusted 2026-09-19 after .16, .5 and .7 landed. **Gap kinds moved:** an unbindable setup
  parameter is an `input` Known Gap coded `<parameter>.unbound` (.5), not `capability` or
  `interpretation`, and `unobservable:` names timers only (.16; there is no unobservable row), so
  the first Approach line reads: white-box gaps are the `capability` and `interpretation` kinds an
  `unobservable:` timer records; an `input` gap is not white-box. That matters because every Case
  over `nexusProtocol` carries `…atConcurrencyLimit.unbound` (no dynamic-config key bounds pending
  Nexus operations) and the canary set over Queries 1 and 2 must still admit; say so in the
  specimen and pin the rejection on a Query whose path fires an `unobservable:` timer. **Where the
  canary check runs:** .7 decided a set is Umpire's and the Cases it produces are the platform's,
  so `set` (Umpire) cannot walk a would-be Case's gaps; the check belongs in the Temporal
  `case … realizes <set>` block (`Temporal/Case/Syntax.lean:183-260`), which today rejects any
  non-functional set with "only a functional set compiles to Cases", extended to admit a canary set
  by producing each Query's Case, rejecting on a white-box gap and registering no fixture -- or a
  sibling block; decide and record. **Coverage targets:** a class is one member of its domain (.2),
  so `classMembers` enumerates the claimed classes -- those with an `examples:` line, which
  `Umpire.Command.classClaims` (`Command/Claims.lean`) already lists per machine -- and the members
  an exploration tries are outside the Model. **An exploratory set names no machine:** the `set`
  command scopes an exploratory set's parties to the actions declared beside it and rejects
  `queries:`, and `budget:` is only checked to resolve as a constant, so the enumeration over the
  protocol machine needs a machine to read rows and results from; add a key (`machine:` under the
  fn-83 reading rule) or derive it, and check `budget:` names a `limits` declaration. Rows are
  enumerated from the checked table, results from the outcome domain, in the canonical catalog
  order the `machine` command emits (.16).

### Investigation targets
**Required:**
- `model/Umpire/KnownGap.lean:8-68` — kinds and canonical set; `model/Umpire/Case/Producer.lean:311-320,615-616` — the `input` gap an unbound setup parameter records
- `model/Umpire/Command/Syntax.lean:2337-2490` and `Records.lean:148-210` — the `set` command and its records; `Command/Claims.lean` — `classClaims`
- `model/Temporal/Feature/Nexus/Success/Tests.lean` (the `### Sets` section .7 added) — the specimen style for set rejections
- `model/Temporal/Tool/Goldens.lean:41-116` — the golden writer
- `Makefile:85-89,588-600` — golden directories and `umpire-check-goldens`

**Optional:**
- `.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md:28-40` — what fn-33 will read from the targets

### Key context
- Single-value classes carry no claim and produce no class-member target.

## Acceptance
- [ ] the canary set over Queries 1 and 2 with `handler: observed` admits; a canary set naming a Query with a white-box Known Gap rejects in place naming both, pinned by `#guard_msgs`
- [ ] the exploratory set enumerates its coverage targets deterministically; the golden is checked by `make umpire-check-goldens` and rendering twice is byte-identical
- [ ] `lake build TemporalModelTests` green; `make umpire-check-goldens` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
