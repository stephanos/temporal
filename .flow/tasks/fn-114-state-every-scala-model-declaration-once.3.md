---
satisfies: [R1, R4, R6, R11]
---
# fn-114-state-every-scala-model-declaration-once.3 Rewrite the Nexus caller realization by value with the shared script helpers

## Description
Finish the Nexus realization that fn-112.9 migrated only far enough to consume the shared kit: every reference to its own roles, learned values, observations, evidence kinds, controls and commands becomes a value reference, and its private string constants go.

**Size:** M
**Files:** `model/temporal/nexuscaller/Realization.scala` (687 lines on 2026-10-03, 28 private string constants at :42-74, :311, :362; evidence ids re-spelled at :106, :114, :134-166, :256-260 - re-locate after fn-112.9), `model/temporal/realize/**` only if a missing generic helper is found (else record it as a finding for fn-112 or a later spec), focused lifter refusal fixture.
**Touches:** [model/temporal/nexuscaller/Realization.scala, model/lifter/testdata/realizationRefusals/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-caller.json, model/ir/nexus-control.json, model/cases/**]

### Approach
- Mirror the finished `standaloneactivity/Realization.scala` from fn-112.9: `script`/`perform`/`onPath`/`always`, no `Item(` constructor, facts named by value, no `Control as _` import.
- An id the IR needs as text (e.g. `temporal.nexus.caller.evidence.started`) is written once on its declaration; every other mention references that declaration. The evidence-id vector order must stay `started, completed, failed, canceled, timedOut` (module map, migration goldens).
- Leave the literal waits (250 ms polls, `timeoutMs = 5000`) untouched: fn-118 owns them. Budget literals stay as data.
- Add a refusal fixture proving a reference to a non-existent declaration fails to compile or is refused by the lifter at its line (R4 errors).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/Realization.scala`
- `model/temporal/standaloneactivity/Realization.scala` (post fn-112.9)
- `model/temporal/realize/` (fn-112.9 kit)
**Optional:**
- `model/lifter/Realizations.scala` - reference resolution
- `tools/umpire/lower/testdata/migration/oracles/nexus`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model
```

### Execution constraints
- Realization IDs, evidence catalogs, script order/modes and every Case byte equal the baseline; no wait/interval/timeout change (fn-118 boundary).
## Acceptance
- [ ] `Realization.scala` has no private string constant referenced by spelling elsewhere; each IR text id is written once on its declaration.
- [ ] No `Item(` constructor or `Control as _` import remains; the shared helpers are the only form, or exceptions are listed with reasons.
- [ ] A refusal fixture proves a reference to a missing declaration fails at its source.
- [ ] Literal waits are untouched and listed as fn-118's; all IR and Case bytes match the baseline; model gate passes.
## Done summary
Rewrote the Nexus caller realization by value with the shared script helpers. Commit 3f642f6a87.

**What changed**
- **Realization.scala (645 lines / 60 literals -> 525 / 30).** The private id constants are gone: the 3 source ids, 7 evidence ids, the observation id, the learned id and the command ids. Two vals remain, `service` and `operation`. Each is written once and referred to by value, in both the handler activation and the schedule attributes. The commit message's "28 constants" is the task file's count from before fn-112.9; at the base there were 16.
  - Commands are vals named in kebab case. `rpc {}`, `await`, `command(...)`, the kit's `controller`, and `script`/`perform`/`onPath`/`always` are the only forms. There is no `Item(`, `Performance(`, `Script(` or `Control as _` left.
  - Evidence ids come from `evidenceId(kind)` and `sourceId(name)`. `pending` uses `evidenceId(ProtocolFact.pendingAttempts)`. Facts are named by value. The history command closes `Vector(started, completed, failed, canceled, timedOut)`, in the order the task requires.
  - Learned value, observation and command ids are referenced as `completionAuthority.id`, `historyEvent.id`, `correlated.id` and `startNexusOperation.id`. A misspelling fails to compile.
  - The positional `schedule(unset, …)` calls became `schedule()` and `schedule(Inputs.x := expires)`. `(using CallerFamily.family)` became `import CallerFamily.given`.
- **R4 refusal fixture** `model/lifter/testdata/referenceInvalid/Invalid.scala`, with a test in Fixtures.test.scala. It has 7 misspelled references, one per form: command, perform, learned, observed, closes, command id and role. Each is a compiler error at its line and column.
- **tools/umpire/lower/lower_test.go:434** now accepts the kit file for the poll-condition case (`requireDeclaredIn`). This follows fn-112.9's activity precedent: `await` places the poll at Kit.scala.

**IR and Cases.** nexus-caller.json and nexus-control.json differ only in positions. The controller script and the two polls are now placed at Kit.scala, and the golden merges already cover that. model/cases is unchanged, umpire-check-fixtures and canary-check-case pass, and the canary identity did not move, so there was no re-pin. Golden config and original.json are untouched.

**Exceptions (R6) and findings for fn-112 or a later spec**
- **`history` keeps `Instruction.rpc(assign, reads)` with `Assignment.typed`.** The `rpc {}` scope cannot read a response. So the four history request fields are written both there and in `awaitClose` (reviewer P3, accepted as a finding). The fix would be a generic `.reading(...)` helper.
- **`start-nexus-operation` stays a written-out `Command(id, …)` inside `scheduling`.** Three deadline variants share the id and there is no `setting` for workflow commands. `"start-nexus-operation"` is written once.
- **Ids are read as `.id`.** Widening `AwaitLearned`, `AwaitCommand`, `NexusReply.binds`, `NexusCompletion.handle` and `Target.*` to `String | X` would remove the `.id`. That is a framework change outside this task's Touches.
- **The metrics counter calls the evidence kinds "own name".** That covers `"started"` and the others, where the val equals the kind. Each is an id written once; no construct names an evidence kind after its val.
- **Kit positions (reviewer P2, pre-existing).** Polls and the controller are located at Kit.scala, not the call site.

**Literal waits, untouched, fn-118's:** `timeoutMs = 5000` in `replying` (5 replies) and in `finishWorkflow`, and the kit's 250 ms `await` interval used by `awaitScheduled` and `pendingAttempts`.

**Deviation from Touches:** the fixture lives in its own dir, not under `realizationRefusals/`. That dir is packaged as a jar and must compile.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3, recorded above.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3f642f6a87
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-check-fixtures && make canary-check-case (exit 0), make lint-model (exit 0), scala-cli test model/lifter (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1 in tools/umpire/lower only: kit position in lower_test.go:434; fixed, then go test ./tools/umpire/lower/... exit 0; all other packages ok), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0, twice)
- PRs: