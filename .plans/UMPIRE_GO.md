# A comparative Go implementation of the Umpire model layer

Plan, 2026-09-29. It proposes an experiment that builds the Umpire model layer in Go 1.27 beside
the Lean one, ports the same Models, and measures both against the same evidence. It changes no rule
and approves no design. Its result is input to a GOV-02 decision on SCP-03, which today mandates
Lean for the Behavior Model. Background is in [UMPIRE_OUTSIDE_THE_BOX](lean/UMPIRE_OUTSIDE_THE_BOX.md)
and the ten-language comparison under `../cmp/` (`EVAL.md`, `SPEC.md`, and the uncompiled Go
sketch in `cmp/go/`).

## 1. The question and the answer we need

Can a Go model layer reproduce every answer the Lean model gives for the Nexus caller and the
standalone activity, emit the same Case bytes, and do it with a faster loop, a smaller maintenance
surface, and error feedback at least as useful to an agent author?

The ten-language comparison could not answer this: nothing in it was compiled, and every non-Lean
sample left search unwritten. This experiment compiles, runs, and compares. It ends with
`model/go/RESULTS.md` and a recommendation under the decision rule in section 9.

## 2. Scope

In scope:

- A Go `umpire` package covering finite domains, machines, tables, reachability, stuck states,
  refinement, composition, Properties, Scenarios, Limits, Queries, Sets, bounded search, exploration
  targets, Definition IDs, and Behavior Fingerprints.
- Three Models: the worker, the Nexus caller ported from
  `model/lean/Temporal/Feature/Nexus/Caller/Model.lean`, and the standalone activity from the revised
  Model 2 in `cmp/SPEC.md`.
- Pin parity, row-level parity against dumped Lean data, and byte parity for the seven
  `nexusCallerTests-*-case.json` fixtures.
- Four generated views.
- A twelve-mistake error corpus and an agent authoring trial, run on both implementations.

Out of scope:

- Any change to `model/lean/`, `common/testing/testpilot/`, `tools/umpire/`, the Makefile, or the
  checked-in fixtures. The experiment reads them and writes only under `model/go/`,
  apart from three `tool` directives in `go.mod` (section 3).
  One exception, requested by the owner on 2026-09-29: the Lean workspace moved from `model/` to
  `model/lean/` and this experiment from `experiments/umpire-go/` to `model/go/`, so the two
  implementations sit side by side. That move rewrote paths in the Makefile, the umpire workflow,
  `tools/umpire`, the Testpilot tests and the plans' links, and changed no behavior.
- The topology layer, holes, focus, and derived faults. The experiment compares like with like.
- The negative control Model (`Nexus/Control`) and its Case. It adds a derivation command the
  comparison does not need.
- Replacing Lean. Both implementations exist side by side until the decision.

## 3. Layout

`model/go/` sits inside the server module, with no `go.mod` of its own. The Case
producer then imports the Testpilot protobuf types and `common/testing/testpilot` directly, and the
code runs under the server's own linters, so Go is measured with the checks it would really run
under.

```
model/go/
  README.md                 how to run each phase; what each script measures
  run.sh                    build, lint, test, parity, views; exits non-zero on any failure
  measure.sh                phase-0 and phase-7 measurements; writes results/<side>-<commit>.json
  umpire/                   framework (section 5)
    finite.go machine.go table.go refine.go compose.go property.go scenario.go
    query.go search.go set.go coverage.go ids.go fingerprint.go canonical.go
    *_test.go               framework unit tests, one per check and one per rejection
  worker/                   worker Model and pins
  nexuscaller/              Nexus caller Model, pins, realization, producer test
  standaloneactivity/       standalone activity Model and pins
  caseproducer/             Go Case producer over umpire model data (phase 4)
  views/                    renderers; testdata/ holds golden views
  parity/                   Lean-versus-Go comparison tests; testdata/lean/ holds the dumps
  leandump/
    Dump.lean               writes the Lean dumps (section 6)
    dump.sh                 runs Dump.lean against the built Lean model in model/lean
  corpus/
    NN-<slug>/lean.patch go.patch expected.json
    run.sh                  applies each patch, measures, reverts (section 8)
  trial/
    TASK.md                 the agent task text, identical for both sides
    log/                    one file per run
  results/                  measurement JSON, committed
  RESULTS.md                the report
```

`run.sh` is the one entry point. A developer or agent runs it after every edit:

```sh
model/go/run.sh            # go vet, linters, go test, parity against committed dumps
model/go/run.sh --views    # also regenerates views and diffs them against goldens
model/go/leandump/dump.sh      # refreshes the Lean dumps; needs a built model
```

The inner commands are the ones the server repo already uses:

```sh
go vet ./model/go/...
go tool exhaustive ./model/go/...
go tool go-check-sumtype ./model/go/...
go test -tags test_dep ./model/go/...
```

`exhaustive`, `go-check-sumtype`, and `enumer` are not in the server's `go.mod` today; the only
`tool` directive there is `benchstat`. T2 adds the three as `tool` directives. That is the only
change outside `experiments/`, and the report lists it.

## 4. Ground truth

The Lean model is the oracle.

| Source | What it pins |
| --- | --- |
| `cmp/lean/Pins.lean`, the real Nexus pins, 104 `#guard`s | Counts, step results, reachability, refinement rows, query outcomes, set bindings, targets, Case instruction ids, evidence kinds |
| `cmp/lean/ActivityPins.lean`, 40 `#guard`s | The same groups for the standalone activity |
| `model/lean/Temporal/Feature/Nexus/Caller/Fixtures/CallerExploratoryCoverage.json` | The 889 exploration targets, as Lean renders them |
| `model/lean/Temporal/Feature/Nexus/Caller/Fixtures/CallerRetryPlan.json` | The `retry` query's Plan, including its witness |
| `model/go/parity/testdata/lean/*.json` | Everything else, dumped by `leandump/Dump.lean` (section 6) |
| `tests/testcore/testpilot/testdata/nexusCallerTests-*-case.json` | The seven Case fixtures, byte for byte |

The activity Model's Lean file does not fully elaborate: its `case` block names a missing
realization, and its status observations assume a catalog rule. Parity for that Model covers what
Lean can answer, and the report lists the rest.

## 5. The framework

### 5.1 Authoring surface

Go 1.27 generic methods let the authoring surface be builder chains. A `verify` Query across a
refinement takes the refinement as an argument, so the compiler checks the pairing that the
evaluation found every other sample getting wrong.

```go
// Domains
type Phase uint8                                  // enumer generates Values() and String()
type Reply interface{ isReply() }                 //sumtype:decl
type HandlerError struct{ Retryable bool }        // one struct, two classes

// Machines
var NexusProtocol = umpire.NewMachine[ProtocolState, ProtocolOutcome, ProtocolFact]("nexusProtocol").
	For(operation).
	Starts(ProtocolState{Phase: Unscheduled}).
	Ends(terminal).
	Timers(backoff, scheduleToClose, scheduleToStart, startToClose).
	Unobservable(backoff).
	Evidence(NexusOperationScheduled{}, "nexusOperationScheduled").
	Step(schedule, scheduleStep).                  // generic method infers the input types
	Step(handlerReply, protocolHandlerReplyStep)

var protocolRefinesProduct = NexusProtocol.Refines(NexusProduct, productOf)

// Claims
var syncSucceeds = NexusProtocol.Property("syncSucceeds").
	When(handlerReply.With(SyncSuccess{})).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == Succeeded && s.Records(NexusOperationCompleted{})
	})

var syncReplied = NexusProtocol.Scenario("syncReplied").
	Starts(ProtocolState{Phase: Unscheduled}).
	Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(SyncSuccess{}))

var syncCompletion = syncReplied.Find(syncSucceeds).Within(two)
var terminalHolds = asyncThenSucceeded.Verify(terminalIsFinal, protocolRefinesProduct).Within(four)
```

Arity still splits, because Go has no variadic type parameters: `Step` is overloaded by name as
`Step0`, `Step1`, `Step3` if the probe in task T2 shows one generic method cannot cover all three.

### 5.2 Porting rules

Each rule below is taken from the Lean source and must hold exactly, because Definition IDs,
fingerprints, witnesses, and Case bytes depend on it. Rules marked "confirm" are inferred from pins
and fixtures and are checked by the Phase 3 dumps before anything depends on them.

| Topic | Lean rule | Source |
| --- | --- | --- |
| Member order | Declaration order for `enum` constructors; `Bool` and `Fin (n+1)` in value order; a structure is the product of its fields | `Umpire/Command/Finite.lean` module doc |
| Row order | States-major, then actions. Fingerprint-visible | same |
| Field that varies fastest | Confirm from the dump | Dump |
| State key | Field values in declaration order joined by `-`, e.g. `scheduled-0-unset-unset-unset` | Pins and fixture ids |
| Action class key | Action name then class values joined by `-`, e.g. `handlerReply-async`, `handlerReply-handlerError-true`, `complete-canceled` | `Temporal/Case/Realization/Nexus.lean` keys, pins |
| Action catalog order | Confirm: pins show `["backoff", "complete-canceled"]` first, which looks lexicographic | `Pins.lean:60` |
| Row key | State key, `-`, action class key | `Pins.lean:124` |
| Definition ID | `temporal` root, family from the namespace below `Temporal.Feature` in lowerCamel segments (`nexus.caller`), then `<kind>.<owner>.<member>` | `Umpire/Id.lean`, `Umpire/Command/Authoring.lean:31-55`, `Temporal/Case/Conventions.lean` |
| ID examples | `temporal.nexus.caller.target.nexusProtocol`, `.behavior.syncReplied` for a Scenario, `.query.syncCompletion`, `.property.syncSucceeds`, `.property.retrySucceeds.fact-nexusOperationCompleted`, `.state.nexusProtocol.scheduled-0-unset-unset-unset`, `.state-field.nexusProtocol.phase`, `.action.handlerReply`, `.action.nexusProtocol.backoff`, `.fact.nexusProtocol.pendingAttempts`, `.outcome.nexusProtocol.accepted`, `.evidence.scheduled`, `.source.history`, `.scope.run`, `.projection` | Fixture `nexusCallerTests-retry-case.json` |
| Case ids | `temporal.case.<set>.<query>`, with `.program` and `.contract` suffixes | same |
| Fingerprint | `"sha256:" + hex(sha256("umpire.behavior-fingerprint/v1" + "\n" + canonical))`, standard SHA-256 over UTF-8 | `Umpire/Fingerprint.lean:180-185` |
| Target canonical content | `targetSemanticJson` over id, definitions, required capabilities, providers, connectors, machine metadata, and the behavior table with sorted, de-duplicated terminal conditions | `Umpire/Model/Check.lean:380-415` |
| Fingerprint check values | `nexusProtocol` target `sha256:b3864750…af14`; `syncReplied` `sha256:2e707f73…7de5`; `syncCompletion` `sha256:120e8f7e…6992`; `syncSucceeds` `sha256:a4708e68…0d26` | `nexusCallerTests-syncCompletion-case.json` |
| Search space | Product of model state, Scenario progress, Property monitor states, and a fired-clause bitset | `Umpire/Search/Product.lean` module doc |
| Search order | Breadth-first. Roots by sorted setup then initial index. Successors by action index then outcome index. First-discovery parent, so the witness is the shortest and ties go to the lower index | same, and `Search/Backend/Veil.lean:27-48` |
| Limits | `steps` is the depth bound; `search` caps visited product states; exceeding it is `limit-reached`, never "not found" | `Search/Backend/Veil.lean:45-48` |
| Exploration targets | Rows whose source is within `steps − 1` of a start, in table order; then outcomes those rows reach, in catalog order; then class claims those rows' actions make, in claim order; per goal in the set's order; cut at `search` | `Umpire/Command/Coverage.lean:34-60` |
| Target JSON | `{"kind","key","state","action","results"}` for rows, `{"kind","outcome"}` for results | `Umpire/Command/Coverage.lean:62-70` |
| Case JSON | Canonical ProtoJSON: keys in proto declaration order, fields without presence elided | `Testpilot/ProtoJSON.lean` module doc |
| Fixture bytes | Canonical compact form, re-indented with two spaces, one trailing newline | `tools/umpire/internal/casefile/casefile.go` |

### 5.3 Checks the framework runs

`umpire.Check(t, decls...)` runs every semantic check the Lean elaborator runs, as one test that
fails with the declaration named and the offending row shown:

- every step lands inside the finite domain;
- no stuck state: every reachable non-end state has a row;
- every declared end is reachable;
- every recorded fact has an evidence line;
- refinement, under the stricter rule the real Lean checker applies (a protocol row stutters under
  the map, or matches a product row with the same outcome whose facts all appear among the protocol
  row's facts);
- composition agrees with its members' tables;
- a canary set names no silent step;
- Queries: `find` is found, `verify` holds within limits, and `limit-reached` is its own result.

## 6. The Lean dumper

`leandump/Dump.lean` imports the built production modules and writes canonical JSON. It reads only
public names that the pins already use, so it needs no change to `model/lean/`.

```sh
# leandump/dump.sh
cd model
mise exec -- lake build Temporal
mise exec -- lake env lean --run ../model/go/leandump/Dump.lean \
  ../model/go/parity/testdata/lean
```

On macOS the script sets `SDKROOT` and the clang path the way the Makefile's `LEAN_LAKE` does.

Files it writes, one per machine or query:

| File | Content | Lean names read |
| --- | --- | --- |
| `machine-<name>.json` | Ordered state keys, action class keys, transition rows with results (outcome, next state, facts), starts, ends, reachable, stuck, state-field ids, every Definition ID, target fingerprint | `<machine>.table`, `.transitions`, `.actionKeys`, `.starts`, `.ends`, `.stuck`, `.stateFieldIds`, `reachableFrom` |
| `refinement-<name>.json` | Every refinement row key and the product row it maps to, or null for a stutter | `<machine>.refinement.rows` |
| `query-<name>.json` | Outcome, witness trace as row keys, fingerprints of target, scenario, property, query | the elaborated query value |
| `targets-<set>.json` | Exploration targets | `<set>.targets`, `CoverageTarget.json` |

The dumps are committed so `run.sh` works without a Lean build. `dump.sh` is rerun only when the
Lean model changes.

## 7. Tasks

One Flow-Next spec, one task per row, in order. Each task's done summary carries the command it ran
and its output. A task that cannot meet its acceptance stops the spec and records why.

| Task | Deliverable | Depends on | Acceptance |
| --- | --- | --- | --- |
| **T0 Baseline** | `measure.sh`; `results/lean-<commit>.json` | none | JSON holds cold build seconds, edit-loop seconds for file and downstream library, line counts per category, pin counts, and the Lean message for each corpus mistake |
| **T1 Lean dumper** | `leandump/Dump.lean`, `dump.sh`, committed dumps | none | Two runs produce identical bytes; the dumped counts equal the pins (table below) |
| **T2 Framework core** | `umpire/` finite, machine, table, ids, fingerprint, canonical | T1 | Unit tests pass; a probe confirms whether one generic `Step` covers every arity |
| **T3 Refinement and composition** | `refine.go`, `compose.go` | T2 | Unit tests cover the stutter, match, outcome mismatch, and missing-fact cases |
| **T4 Claims and search** | property, scenario, query, search, set, coverage | T2 | Unit tests cover `find`, `verify`, `limit-reached`, and target cut at the budget |
| **T5 Worker and Nexus caller** | `worker/`, `nexuscaller/` | T3, T4 | Compiles, passes linters, `umpire.Check` accepts every declaration |
| **T6 Nexus parity** | pins as tests; `parity/` comparisons | T1, T5 | Every translated pin passes; every dumped row, refinement row, witness, target, id, and fingerprint compares equal |
| **T7 Standalone activity** | `standaloneactivity/` with pins | T5 | Compiles, `umpire.Check` passes, `ActivityPins` translated; parity where Lean answers |
| **T8 Case producer** | `caseproducer/`, Nexus realization, producer test | T6 | `syncCompletion` byte-identical first, then all seven; each passes `testpilot.Prepare` over its derived Profile |
| **T9 Views** | `views/` with goldens for both Models | T6, T7 | Two runs identical; a diff view for adding one activity class |
| **T10 Error corpus** | `corpus/` twelve cases, both sides | T0, T7 | Filled table (section 8) |
| **T11 Agent trial** | `trial/TASK.md`, six logs | T7, T9 | Three runs per side logged with wall time, iterations, review result |
| **T12 Report** | `RESULTS.md`, `results/go-<commit>.json` | all | Every metric in section 9 filled, with a recommendation under the decision rule |

T0 and T1 run in parallel. T2 to T4 can overlap once T2's types exist.

### 7.1 Nexus parity values T1 and T6 must reproduce

| Value | Lean | Pin |
| --- | --- | --- |
| Product states, ends | 6, 4 | `Pins.lean:24-25` |
| Product action classes | 12 | `Pins.lean:29` |
| Protocol states | 192, which is 8 phases × 3 attempt counts × 2 × 2 × 2 deadlines | `Pins.lean:53` |
| Protocol ends | 96 | `Pins.lean:54` |
| Protocol action classes | 22 | `Pins.lean:59` |
| Protocol transitions | 1,152 | `Pins.lean` exploration group |
| Reachable from the start | 158 | `Pins.lean:109` |
| Stuck states | none for both machines | `Pins.lean:41,104` |
| Refinement | no rejection; one row per transition | `Pins.lean:119-120` |
| Exploration targets | 889: 885 rows, 2 results, 2 class members | `Pins.lean` exploration group |
| Query outcomes | seven found; `terminalHolds` verified within limits | `Pins.lean` query group |
| Case instruction ids | e.g. `retry` controller: `start-workflow`, `await-scheduled`, `pending-attempts`, `await-close`, `history`; handler: `respond-error-retryable`, `respond-sync` | `Pins.lean` Case group and the fixture |

### 7.2 Task notes

**T2.** Start from `cmp/go/umpire/umpire.go` and replace every `panic("sketched")` body. Port
`targetSemanticJson` literally; do not redesign it. The fingerprint test compares the Go value for
`nexusProtocol` against `sha256:b38647500819c04cd82e179972ed23c76609d69e99a424748596295ce067af14`.
If it differs, diff the canonical string against one printed by a one-off Lean `#eval` before
changing any Go code.

**T4.** Implement the product state and visited set exactly as `Search/Product.lean` describes, so
witnesses match without special cases. Record `limit-reached` separately from "not found".

**T5.** Keep the Lean file's section order and its comments that explain semantics. Keep spec names
as declaration names; use type prefixes only where Go constants would collide, and record each
prefix in the package doc.

**T8.** Mirror `Umpire/Case/Producer.lean` and `Temporal/Case/Realization/Nexus.lean` in that
order: identity and provenance, Program roles, slots, observations, entrypoints, cleanup, evidence
declarations, then the Contract's correlated rules. Render with `protojson.Marshal`, then pass the
bytes through `casefile.Persisted`. Compare against the fixture with a byte diff; on mismatch, print
the first differing JSON path. Go's `protojson` emits fields in declaration order, which matches the
Lean encoder, but map key order and `Any` handling must be checked against the fixture before
assuming they match.

**T9.** Four views, all from model data, never from Go source:

- transition table grouped by phase, unspecified cells marked, optional `Because` column;
- Mermaid state diagram;
- behavior summary in a compact declarative form close to the Lean surface;
- behavior diff between two model revisions: rows, facts, and targets added or removed.

## 8. The error corpus and the agent trial

### 8.1 Corpus

Each case is a directory with a patch per side and the expected catch point. `corpus/run.sh`
refuses to run on a dirty tree, applies one patch, runs the side's check, records the result,
and reverts with `git checkout --` on the patched file. The Lean side rebuilds only the patched
module and its dependents.

| # | Mistake | Expected catch in Lean | Expected catch in Go |
| --- | --- | --- | --- |
| 1 | A step names an undeclared action | compile | compile |
| 2 | A step's input type disagrees with the action's input | compile | compile |
| 3 | A Scenario passes an input of the wrong type or arity | compile | compile |
| 4 | A match or switch misses a case | compile | lint |
| 5 | A step lands outside the finite domain | compile | test |
| 6 | A recorded fact has no evidence line | compile | test |
| 7 | A protocol row has no product counterpart under the map | compile | test |
| 8 | A `verify` Query pairs a product Property with a protocol Scenario and no refinement | compile | compile, through the typed `Verify` |
| 9 | A Query's claim is unreachable within its limits | compile | test |
| 10 | A machine has a stuck state | compile | test |
| 11 | A canary set names a silent step | compile | test |
| 12 | Two declarations collide on a name | compile | compile or test, depending on scope |

For each case and side the harness records: where it was caught, seconds from save to message,
whether the message points at the author's line, and whether it names the fix. The expected
columns are hypotheses; the report shows what happened.

### 8.2 Agent trial

`trial/TASK.md` gives one task, identical for both sides except for file paths:

> Add a heartbeat timeout to the standalone activity Model. Add a `heartbeat` action class the
> worker performs on a started attempt, a `heartbeat` timer the system owns that fires only on a
> started attempt whose schedule set a heartbeat timeout, the rows for both, a Property that the
> timer times the attempt out with the heartbeat timeout type recorded, a Scenario and a `find`
> Query for it, and pins for the new rows. Run the check command until it passes.

The agent gets the repository, the check command (`run.sh` or the Lean build of the Model and its
tests), and nothing else. Run three times per side, each in a fresh session. Record wall time,
number of check runs, the final diff, and whether a reviewer accepts it without changes.

## 9. Metrics and decision rule

| Metric | How it is measured |
| --- | --- |
| Pin parity | Share of translated Lean pins that pass in Go |
| Row-level parity | Rows, refinement rows, witnesses, targets, ids, and fingerprints equal, per Model |
| Case parity | Nexus caller fixtures byte-identical in persisted form, out of seven |
| Edit loop | Seconds from saving a one-line step-function edit to the first pass or fail |
| Cold build and test | Seconds, same machine, per side |
| Lines | Framework, Models, pins, generated code, counted separately per side |
| Error quality | The section 8.1 table |
| Agent authoring | Wall time and check runs, three runs per side, every run shown |
| Dependencies | New Go modules and tools; toolchains per side |

The report recommends amending SCP-03 under GOV-02 and moving the model layer to Go when all of
these hold:

- pin, row-level, and Case parity are complete;
- the Go edit loop is under ten seconds;
- Go catches every corpus mistake no later than `go test`, with a message at or near the author's
  line;
- the agent trial takes no more check runs on the Go side than on the Lean side.

It recommends keeping Lean when parity fails for a reason Go cannot express, such as a check only a
proof can make, and names that reason. Anything in between is reported as it is, with the trade
stated, and the decision goes to a human.

## 10. Risks

- **Byte parity depends on Lean details.** Id spelling, fingerprint canonical content, witness
  order, and ProtoJSON key order must all match. T1's dumps and T2's fingerprint check exist to find
  mismatches before T8.
- **The dumper needs a built production model.** Each refresh costs a Lean build. It runs only when
  the Lean model changes.
- **The activity Model does not fully elaborate in Lean.** Parity there is partial, and the report
  says which parts.
- **Drift into a rewrite.** The scope excludes the topology layer and every production change. A
  finding that wants more scope becomes a note in the report.
- **Noisy agent trials.** Three runs per side, every run reported, no averaging away of outliers.
- **Corpus patches touching the production model.** The harness refuses a dirty tree and reverts
  after each case. It never commits.
