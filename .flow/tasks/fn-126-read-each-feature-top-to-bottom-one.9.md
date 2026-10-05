---
satisfies: [R21]
---
# fn-126-read-each-feature-top-to-bottom-one.9 Lint a reachable state that is no end and enables nothing (stuck-state)

## Description
Implements R21: the `stuck-state` lint kind (owner request, 2026-10-05).

A reachable state of a machine or composition that is not an `end` state and in which no action class has an enabled row is a finding: machine, state, and a shortest path to it as the witness, at the machine's position. It is on by default, like `silent-rejection` and `never-enabled`. Reachability and stuck states are already computed by the reader (`tools/umpire/model/machine.go:158`); this adds the lint pass, the kind, its message and its fixtures.

Runs in parallel with fn-126.5: it touches `tools/umpire/lint/**`, its testdata, `tools/umpire/lint/testdata/coverage.golden` if affected, the README's lint table and, if today's Models have stuck states, their acceptances in `model/ir/*.lint.json` with a reason each (or a reported Model bug; do not change a Model here). Whichever of this and fn-126.5 lands second rebases the acceptance files.

## Acceptance
- [ ] `stuck-state` is a default lint kind; it reports every reachable non-end state with no enabled row, with a shortest path, and nothing else.
- [ ] One fixture produces exactly the expected finding; one passing fixture produces none; the README lists the kind.
- [ ] Every finding on today's Models is either accepted in its `*.lint.json` with a reason, or reported to the host as a Model bug (no Model edited here).
- [ ] `./tools/umpire/lint`, `./tools/umpire/model`, the model gate, `make umpire-check-cases` and lint-code-fast pass.


## Done summary
# fn-126.9: the stuck-state lint (R21)

### The kind
`stuck-state` is a default lint kind in `tools/umpire/lint` (`StuckState`, run by `stuckStates` in
`kinds.go`, listed after `never-enabled`). It reports a reachable state of a machine that is not in
the machine's `ends` and in which no action class has a row with a result. That includes timers and
internal steps.
- **Subject:** the state key. **Position:** the machine's declaration. **Population:** the
  reachable states. It has no coverage count, like `never-enabled`, so `coverage.golden` is
  unchanged.
- **Message:** `<state> is reachable, is no end and enables no action class: no action can happen
  in it, so a timer or an internal step may be missing a rule; if it is meant to be final, declare
  it in the machine's ends; reached by <start> -<action>/<outcome>-> ... <state>`.
- **Holes (host decision, commit ec0fda47a9):** a state with a hole row is not stuck. A hole
  declares that unmodeled behavior may happen in that state; it is not a forgotten rule. This is
  how the progress check reads it (SEMANTICS.md "Progress"). The kind's doc comment and the
  README's lint row say so.
- **Witness:** the reader now exposes `Table.PathTo(state) *Trace`. It is a 6-line exported wrapper
  over the checker's existing breadth-first `pathTo` and `trace`, in
  `tools/umpire/model/internal/checker/replay.go`. Lint computes no path of its own. A test
  replays the path against the table.

### Composition decision
The kind lints machines only, not composed tables, matching `never-enabled`. Reasons:
- Lint builds no composed table today.
- `Realizer.Composition` is the check's expensive build. In the lint tests the composition probe
  took about 36s.
- Some compositions deliberately fail to build. These are negative controls such as
  `currentOverForgetful` and `currentOverVolatile`, whose refinement fails.

A probe of every composition in today's `model/ir` found no stuck composed state. All seven
compositions that build declare `ends`.

### Fixtures (`TestStuckState`, `tools/umpire/lint/kinds_test.go`)
- **Finding fixture:** `captured.json`'s `putOnly`, which is `disk.restrict(client.put)`. It drops
  the flush, an internal step. It produces exactly one finding: `putOnly` / `staged`, at
  `putOnly`'s position, with path `empty -put/accepted-> staged`. The full message is pinned, and
  the path replays against the table.
- **Passing fixture:** `activity.json`. Its four machines have 9, 238, 2 and 2 reachable states,
  and none is stuck. In `captured.json` itself, `disk` (the same machine with its flush) is not
  reported.
- **Hole case (passing):** `declarations.json`'s disk with its flush binding removed. `staged` is
  reachable, is no end and has no row; its one pair is the `crash` hole. The test asserts that this
  gives no finding (population 2).
- **README:** `model/README.md` gains a lint table with `never-enabled`, `silent-rejection` and
  `stuck-state`, plus a line on the composition choice.

### Today's Models
There are 15 findings, all in `nexus-close.json`. Each is accepted in `model/ir/nexus-close.lint.json`
with a reason. No Model was changed, and none of these is a Model bug.
- **`rejectAfterClose` (N1):** 5 states `resetOpen-…-done-…-none-none-none`.
- **`ackByOriginal` (N2):** the same 5 states.
- **`truncatesOnReset` (N4's mutation):** the same 5 states.

These three are deliberate faulty controls. In each, a reset successor owns an outcome that
nothing holds. `settled` (the machine's `ends`) refuses that state, and its own comment says so:
"Any other state with no step is a lost outcome". `tools/umpire/model/nexus_close_test.go`
already pins these states as stuck: `TestNexusCloseDeadlockIsTheLostOutcome` and
`TestNexusCloseResetAfterAcknowledgment`. The `*WithDeadline` variants give the state a step (N8)
and produce no finding. No activity, Nexus caller or control machine has a stuck state.

### Gates (logs under `.flow/tmp/fn126-9/`)
I reran these after the hole decision (ec0fda47a9), and all pass:
- `go test -tags test_dep ./tools/umpire/lint ./tools/umpire/cmd/umpire-lint` (`go-test-lint-holes.log`).
- `umpire-lint` over all IR exits 0 with 15 stuck-state findings, all accepted, and no stale
  acceptance (`lint-holes.log`).
- The model gate (`model-gate-holes.log`).
- lint-code-fast reports 0 issues (`lint-code-fast-holes.log`).

The earlier runs follow.
- **Go tests:** `go test -tags test_dep ./tools/umpire/lint ./tools/umpire/model/... ./tools/umpire/cmd/umpire-lint` passed (`go-test.log`, `go-test-lint.log`).
- **lint-code-fast:** `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast` reported 0 issues (`lint-code-fast.log`).
- **Model gate:** `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` passed (`model-gate.log`).
- **Cases:** `make umpire-check-cases` passed (`check-cases.log`, empty on success).
- **Lint, all IR:** `go run ./tools/umpire/cmd/umpire-lint` failed on 15 unaccepted findings before the acceptances (`lint-first.log`) and passes after them (`lint-accepted.log`).
- **Merge:** the gates ran after `git merge umpire` (a2e4d6d8de, up to date). A later
  `git merge umpire` (f1045d51a1, docs and flow only) touches no code.

### For the owner
- **Composed tables:** compositions are not linted directly. A stuck composed state that no
  member explains, such as a sync mismatch, would go unreported. Today there are none.
- **Worktree gotcha:** the first model-gate run in this worktree failed with "api/umpire/v1/ir.pb.go
  is older than the IR schema". This is an mtime artifact of the checkout: the content is the
  committed one, identical to lane-a's. `make protoc` cannot run here because `.bin` links to the
  host's `.bin`, and its `goimports` fails with "exec format error". The failed run also deleted 42
  generated files, which I restored with `git restore`. I ran `touch api/umpire/v1/ir.pb.go`, which
  changes no content.
- **fn-126.5:** if it changes `nexus-close`'s close-policy machines or their state keys, the 15
  acceptance subjects follow the new state keys.
- **Pinned populations:** `TestStuckState` pins the activity machines' reachable-state counts, as
  `TestNeverEnabled` pins class counts.

Host decision (for the owner): a state with a hole row is not stuck. A hole declares unmodeled behavior, not a forgotten rule, which matches the progress check. Applied in ec0fda47a9; all 15 findings remain, none was hole-only.

Review: done by the host directly (claude-opus-5-5), not by a separate agent, at the owner's request to return to a single lane. Same family as the writer. SHIP, no findings. Checked:
- `stuckStates` counts only rows with results, plus hole sources.
- `PathTo` wraps the reader's existing `pathTo` and adds no search of its own.
- Each acceptance cites the Go test that already pins those deadlocks as deliberate (`TestNexusCloseDeadlockIsTheLostOutcome`).
- Compositions are not linted; this is a documented gap, and no composition is stuck today.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: da78170aa3, 332ba194a3, 0f09dbb3a8, ec0fda47a9
- Tests: go test -tags test_dep ./tools/umpire/lint ./tools/umpire/model/... ./tools/umpire/cmd/umpire-lint, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, make umpire-check-cases, go run ./tools/umpire/cmd/umpire-lint
- PRs: