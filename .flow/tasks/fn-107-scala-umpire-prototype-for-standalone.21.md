---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.21 Carry declared fields, off-path kinds and repeated classes through the generic producer; lower the activity Cases

## Description
**Touches:** [model/go/caseproducer/**, model/go/umpire/**, model/scalav2/goir/testpilot/**, model/scalav2/goir/conformance/**, model/scalav2/goir/load.go, model/scalav2/goir/*_test.go, model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/lifter/**, model/scalav2/ir/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/SEMANTICS.md, model/scalav2/README.md, model/scalav2/specimens/**, model/scalav2/run.sh]

Carry what the Scala realization declares through the generic Go producer into the Case, so the standalone activity's completion and retry Queries lower to admitted Cases and `syncCompletion` concludes. Review text with the constraints: `.flow/tmp/fn-107/task19/t19-r1.md` (judgments c, d, e).

**Size:** L

### Approach
- **Declared fields and off-path exhaustive kinds.** `model/go/caseproducer` carries only the kinds on a path and no evidence fields. Carry declared fields (path, type, disposition, role) and the exhaustive kinds a Case's closing read lifts even when the path does not record them, with field paths, types, dispositions and off-path rules inside the projection fingerprint. A realization that declares none of this produces byte-identical Cases; the Nexus caller Cases change because they gain declarations.
- **A class taken more than once on a path.** The producer confirms a class once per path, so `retry` (two `attemptStart` steps) is refused with `evidence.action-repeated`, and `pauseResume` would put one kind in two projection rules. "One rule per kind" is not sound: a confirmed rule's outputs are sequential effects of every occurrence, not alternatives. Give the generic producer a discriminant that makes each occurrence its own confirmation: distinct evidence kinds per occurrence, guarded projection outcomes, or occurrence- and state-aware projection, whichever is smallest and keeps unique-kind admission, transition authorization, Known Gap behaviour and deterministic fingerprints. Existing non-repeated Cases are unchanged.
- **Two things the task 19 review found waiting here.** `pauseResume` and `retry` each put the `statusScheduled` kind in two projection rules, which `testpilot.Prepare` rejects; an occurrence discriminant alone does not create the second stable scheduling observation the retry declaration lacks, so the realization must declare one (for example the retry's own typed attempt fact). Both must be proven with ordinary `testpilot.Prepare`. A gap that belongs to a command off a Query's path (the canceled answer, for one) must not block that Query: make unsupported commands path-specific without losing the declaration inventory. `startToCloseTimeout` needs an attempt that gives no answer and no instruction waits: record it as a limit, do not build it.
- **What task 20 delivered and its two constraints.** Testpilot now has a single-message read source, a Run Event evidence source under a typed guard (`RunEventSource` with `instruction` and `run_keyed`), distinct kinds sharing one dense source, and a canceled-answer instruction arm (handover: `.flow/tmp/fn-107/task20/summary.md`). One Run Event carries one evidence Observation: the activity realization's `delivered("statusStarted")` and `delivered("attemptCount")` cannot both lift from one attempt event; declare one kind with two projection outputs, or disjoint guards. A canceled answer reaches the server only for a delivery the server asked to cancel, so a Query that cancels needs a controller step that requests it. Lift the task-19 `unsupported` entries that named fn-107.20 by lowering to these primitives.
- **Stable activity evidence.** Re-key the standalone activity realization as the review requires: scheduling from the start instruction's outcome, attempt start and identity from the typed `activity_attempt` Run fact through task 20's Run Event source and guard, terminal states from single-message Describe reads. No transient-state poll remains.
- **Result.** The activity Model's completion and retry Queries lower to Cases `testpilot.Prepare` admits under a derived Profile; `syncCompletion` concludes on its witness Run and stays inconclusive on a history that holds a started event; the other six Nexus Properties stay listed as inconclusive with their reasons (action-class evidence, driven-party closed worlds, exhaustive retry attempts and enum discriminants are recorded limits, not work for this task).
- No feature policy in Go; typed and existing key-level behaviour of `model/go/umpire` and `caseproducer` unchanged apart from what the fingerprint must now cover.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task19-summary.md`, task 20's handover; `model/go/caseproducer/{producer,program,correlated,build,localize}.go`; `model/scalav2/goir/testpilot/{lower,realization}.go`; `model/scalav2/scala/temporal/standaloneactivity/Realization.scala`; `model/scalav2/goir/conformance/evidence.go`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; `GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; scoped `make lint-code` over `./model/scalav2/goir/... ./model/go/caseproducer ./model/go/umpire` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] The generic producer carries declared evidence fields and off-path exhaustive kinds, covered by the projection fingerprint; a realization that declares neither produces byte-identical Cases.
- [ ] A class taken more than once on a path is confirmed per occurrence by a generic discriminant; unique-kind admission, transition authorization and Known Gap behaviour hold; the comparative Go Model's existing Cases are unchanged.
- [ ] The standalone activity realization uses no transient-state poll; its completion, retry and pause/resume Queries lower to Cases ordinary `testpilot.Prepare` admits under a derived Profile, the failing attempt as `activity_attempt_failure`; identical inputs give identical bytes.
- [ ] `syncCompletion` concludes on its witness Run and is inconclusive on a history holding a started event, live and replayed alike; the six other Nexus Properties are listed as inconclusive with their reasons and none is narrowed.
- [ ] Inventories close: everything a realization declares is lowered or is a located `unsupported` entry naming its owner.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
