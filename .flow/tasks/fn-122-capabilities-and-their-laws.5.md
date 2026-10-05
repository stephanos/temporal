---
satisfies: [R7]
---
# fn-122-capabilities-and-their-laws.5 Add the law lint kinds and write waiver reasons into the accepted-findings file

## Description
Connect the law sidecar to fn-120's accepted-findings file and add the four law lint kinds, so intended gaps have one source and the gate fails on a waiver that names no law.

**Cross-spec entry gate:** start only after fn-120.3 (model lint, accepted-findings file) is done; fn-120.3 itself waits for fn-114 to close. Verify with `flowctl show fn-120-adopt-what-quint-does-well-named.3`.

**Size:** M
**Files:** `tools/umpire/model/laws.go` (new: sidecar reader) and `tools/umpire/model/lint*.go` (four kinds: law waived with no reason, reason naming no law, capability parameter with no citation, law with one instantiating machine); the gate step that forwards sidecar waivers into the accepted-findings file in fn-120.3's format; lint fixtures (one triggering, one not, per kind); `model/gate/**` wiring.
**Touches:** [tools/umpire/model/**, model/gate/**, model/ir/**]

### Approach
- Reuse fn-120.3's finding record and acceptance file; add kinds, never a second acceptance file. Each `except`/`overriding` reason in a sidecar becomes an acceptance entry keyed by `<machine>.<law>`; lint fails on an acceptance whose law the sidecar's catalog list no longer brings.
- Citation presence, `promises`/`doesNotPromise`, positions and the catalog's instantiating machines are all read from the sidecar (spec Architecture "The law sidecar"); the IR is not changed and no text is parsed from Scala.
- One-instance detection counts instantiating machines per law across all sidecars the gate reads, with the spec's one definition of an instantiating entity.

### Investigation targets
**Required:**
- `.flow/tasks/fn-120-adopt-what-quint-does-well-named.3.md` and the lint command it produced
- `model/ir/*.laws.json` (post tasks 3 and 4) - the sidecar shape
- `tools/umpire/model/checking.go` - reader indexes to reuse
**Optional:**
- `model/README.md` lint section (post fn-120.3)

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/model/...
make umpire-check-model
```

### Execution constraints
- No IR schema change; no Case change.
## Acceptance
- [ ] Each `except`/`overriding` reason appears in the accepted-findings file keyed by `<machine>.<law>`, forwarded from the sidecar; a reason naming no catalog law fails the gate.
- [ ] The four lint kinds report kind, machine, message and Scala position read from the sidecar, each with a triggering and a non-triggering fixture; a malformed sidecar is a reader error with no findings.
- [ ] The gate runs lint over every IR file; first-run findings and what was done about each are in the done summary.
## Done summary
Connected the law sidecar to fn-120.3's accepted-findings file and added the law lint kinds (R7).

**What changed**
- **Lint kinds** (`tools/umpire/lint/laws.go`; they read only `model/ir/<file>.laws.json`, at the sidecar's Scala positions, and name no capability or law):
  - `waived-law`: an `except`/`overriding` with a reason; its acceptance is forwarded.
  - `law-waived-without-reason`
  - `reason-names-no-law`: a waived law the sidecar's catalog does not list.
  - `parameter-without-citation`
  - `law-with-one-instance`: counted across every sidecar of the linted directories, one entity per state type (owner `catalog`).
- **Forwarding.** `lint.Forward` replaces every `waived-law` acceptance with one per valid sidecar waiver, keyed `<machine>.<law>`, with the sidecar's reason, after the author's entries.
  - `umpire-lint --update` writes it; the gate's update passes `--update` (Gate.scala, Gate.test.scala).
  - A check fails on a `<file>.lint.json` that does not carry the waivers, and a vanished waiver leaves a stale acceptance.
  - An acceptance of `law-waived-without-reason` or `reason-names-no-law` is an error of the file: only the declaration fixes either.
- **Reader** (`tools/umpire/model/laws.go`): strict decode of claims, waivers and catalog. An unknown field, missing identity, a claim of an unlisted law or a duplicate waiver is a reader error with no findings.
- **Sidecar additions** (Scala, no IR change):
  - `Law(…, parameters = Seq(...))` names the parameters where entities differ on purpose; `closedIsRejectedUniformly` lists `rejected`.
  - Core `cited(value, "<server file>")`: the lifter unwraps it and records `cites` per claim. Refused at its line: no citation, a blank one, a non-literal one, or a `parameters` entry its `apply` does not take.
  - Catalog entries carry `parameters`, `position`, and instantiating `{machine, state}`.
  - Lifter fixtures and three refusal fixtures added.
- **Docs:** model/README.md documents the kinds, forwarding, `cited` and `parameters`.

**First-run findings and what was done**
- `parameter-without-citation` × 3: activityProduct `rejected` in activity.json and activity-system.json, and nexusOperation `rejected`. Fixed by citing activity.go:106 (NotFound) and nexusoperation/operation.go:55 (ErrOperationAlreadyCompleted).
- `waived-law` × 8: the 7 `except closedIsRejectedUniformly` on the admission designs and compositions (fn-122.3), and the nexusOperation override (fn-122.4). Forwarded into activity-system.lint.json and nexus-operation.lint.json.
- No `law-with-one-instance`: every catalog law has ≥2 state types across model/ir. No `reason-names-no-law`, no `law-waived-without-reason`.
- Reproduction: `.flow/tmp/fn122-5/firstrun.txt`.

**Decisions**
- **Citations are declared, not parsed.** No text is read from Scala, so citations come from `cited`, and the law marks which parameters need one.
- **Positions kept.** The citations were placed so that no IR file and no Case changed: `notFoundCode` val at the end of the activity file, doc comments tightened. Only the three sidecars and two lint files changed in model/ir.
- **reason-names-no-law** tests membership in the sidecar's catalog list (the task's definition), not per-machine bringing; the lifter already refuses the per-machine case.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3 and FYIs.
- The P3 (docs reshaped to keep line positions) is kept as decided, since the Case bytes are frozen; the Close.scala doc is restored to name "SEMANTICS.md, Claims".
- FYIs fixed in 34d1061971: forward once, duplicate waivers refused, waiver-fault acceptances refused.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: dfbb79889a, a67fe13bea, 34d1061971
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; gate tests pass), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, after review fixes), go test -tags test_dep -p 2 ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (all ok except TestFrameworkNamesNoTemporal, fixed in a67fe13bea; tools/umpire/model, lint, cmd/umpire-lint rerun exit 0), go test -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), scala-cli test model/lifter (exit 0), make lint-model (exit 0), make lint-code-fast (exit 0)
- PRs: