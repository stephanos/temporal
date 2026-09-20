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
- [x] the canary set over Queries 1 and 2 with `handler: observed` admits; a canary set naming a Query with a white-box Known Gap rejects in place naming both, pinned by `#guard_msgs`
- [x] the exploratory set enumerates its coverage targets deterministically; the golden is checked by `make umpire-check-goldens` and rendering twice is byte-identical
- [x] `lake build TemporalModelTests` green; `make umpire-check-goldens` exit 0


## Done summary

Done 2026-09-19; self-review. Commit 4f47809.

### Where the canary check runs

Decided: the Temporal `case … realizes <set>` block admits a canary set, not a sibling block and
not the Umpire `set` command. A set is Umpire's and the Cases it produces are the platform's, and
the white-box question -- can a deployment close every gap this Case carries -- is answered only
by the produced Case, so the block that produces Cases is where it is asked. For a canary set the
block emits the same realization, identity, evidence and Case definitions a functional set gets,
registers nothing (no fixture, no `Registry.recordCase`, so `umpire-case --list` still prints the
seven functional Queries), and reads each Case's white-box gaps through
`Temporal.Case.whiteBoxGaps` (the `capability` and `interpretation` kinds of the produced
provenance; an `input` gap is a parameter the deployment binds, a `claim` gap the Model's own) with
an elaboration-time evaluator in the pattern of the machine command's stuck-state check. A gap
rejects at the set reference naming the Query and the gap: "Query
'Temporal.Feature.Nexus.Caller.retry' cannot be a canary: its Case carries the white-box Known Gap
'temporal.nexus.caller.action.nexusProtocol.backoff.unobserved' (capability), a step of its path
that no observation confirms; …". A Case that does not produce rejects naming the construct. An
exploratory set is still refused by the block, with the message reworded to say what each purpose
does. `nexusCallerCanary` (caller and worker `driven`, handler and network `observed`, Queries 1
and 2) is admitted by `case nexusCallerCanaryCases`; both its Cases carry no gap, and the Approach's
`…atConcurrencyLimit.unbound` concern is moot since `.10` removed the setup parameter.

### Coverage targets

`Umpire.Command.CoverageTarget` (`Records.lean`): `row key state action results`, `result outcome`
and `classMember member action field className exampleValue`, each by Definition ID;
`SetDeclaration` gains `machine : Option DefinitionId` and `targets : List CoverageTarget`. The
`set` command takes `machine:` for an exploratory set (rejected on a functional or canary set,
required on an exploratory one, and it must resolve to a `machine` declaration), checks `budget:`
resolves to a `Umpire.Limits` constant, binds the parties of the machine's actions, and emits
`targets := coverageTargets <machine> (classClaims <machine> [actions]) [goals] <limits>`.
`Umpire.Command.Coverage` (new, imported by `Syntax` and the `Umpire.Command` facade) enumerates:
the rows an exploration within the budget's `steps` of a start can take (a row taken at step k
needs its source within k − 1 steps, walked by the table's own rows as `reachableFrom` does), in
table order; the result values those rows reach, in catalog order; the claims of the classes
those rows' actions make, in claim order; per goal in the set's order, the whole list cut at the
budget's `search` count. `coverageJson` renders a set's coverage as ordered `CanonicalJson` (set,
purpose, machine, cover, budget, targets). `nexusCallerExploration` over `nexusProtocol` with
`cover: rows | results | classMembers` and `budget: four` enumerates 885 rows of the table's 1152,
two results (`accepted`, `notFound`) and the two claimed handler-error classes (889 targets);
`Temporal.Tool.Goldens` writes it to `Temporal/Feature/Nexus/Caller/Fixtures/CallerExploratoryCoverage.json`
(322 KB), the directory is in `UMPIRE_GOLDEN_DIRECTORIES`, and two renderings are byte-identical
(`cmp` of two `--output-root`s and of the checked-in file). Importing the Caller Model into the
writer made `query` a keyword there, so its binder is respelled.

### Pins and specimens

`Caller/Tests.lean` "The sets": the canary's purpose, bindings and gap-free Cases; a `set
canaryRetry` and the `#guard_msgs` rejection of `case canaryRetryCases`; the exploration's
purpose, machine id, goals, budget, 1152 table rows, 889 targets, kinds in order, 885 rows, the
search bound, the two results and the two class members. `Success/Tests.lean`: `exploration` names
`machine: lifecycle`, and four new rejections (no `machine:`, a Query as machine, a Query as budget,
`machine:` on a functional set) plus the reworded block message. DESIGN.md carries a `.12`
amendment after `.11`'s.

### Gates

`lake build` (all 626 jobs, `TemporalModelTests` included) green; `make umpire-gen-goldens` then
`make umpire-check-goldens` exit 0 with every other golden unchanged; `umpire-case --list`
unchanged; conformance fixtures regenerated unchanged; inventory, retired vocabulary, protocol,
authoring and regression-view checks exit 0; `LEAN_NUM_THREADS=1 make lint-model` at the `.11` baseline (41 warnings, none new);
`go test ./tools/umpire/... ./tests/testcore/testpilot/...` ok; `make umpire-check-regression`
exit 0 with 29 passing live identities (29 before; no live test changed).

## Evidence
- Commits: 4f47809
- Tests: `lake build`; `make umpire-gen-goldens && make umpire-check-goldens`; `cd model && lake exe umpire-case --list`; `make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-regression-views`; `LEAN_NUM_THREADS=1 make lint-model`; `go test -count=1 -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/...`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
