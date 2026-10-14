---
satisfies: [R1, R2, R5, R6, R12]
---
# fn-114-state-every-scala-model-declaration-once.4 Convert the Nexus close-policy declarations to the final DSL in place

## Description
The close-policy Model is the largest remaining one (`Model.scala` 702 lines, `Claims.scala` 465 lines, 147 literals, 19 roots in `nexus-close.json`), so it is split across two tasks: this one converts declarations without moving files; task 5 splits the files. Depends on task 2 because it imports the caller's renamed declarations. It also defines the one construct this spec adds beyond the IR-file declaration, the monitor pattern `sticky` (spec API Contracts, R12), because both of its uses are this Model's `retainedOutcome` and `ownerAcknowledgment`.

**Size:** M
**Files:** `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`; `model/umpire/Syntax.scala` (`sticky`, `stickyAcross`, documented with the `monitor` form they stand for; `Monitor.scala` is not edited and imports nothing from it); `model/lifter/Syntax.scala` (their matching, lowering to the tree `monitorOf` at `model/lifter/Declarations.scala:365-405` builds for the `monitor[...](name, initial)(next)(violated)` form; `Declarations.scala` gains at most one hook); `model/lifter/testdata/lifts/` (a lifting fixture declaring both forms on a fixture machine beside their `monitor` spellings, expected JSON proving the two trees equal) and `Rejects.scala` (a predicate of the wrong state type, recorded as the compiler refusal it is); `model/lifter/test/Fixtures.test.scala` roots.
**Touches:** [model/temporal/nexuscaller/closepolicy/**, model/umpire/Syntax.scala, model/lifter/Syntax.scala, model/lifter/Declarations.scala, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-close.json]

### Approach
- Same surface as task 2: captured names (rename the `val` or the one explicit form for differing names), DefinitionScope pins per former owner, family `given`, machine derivation in place of copied machines, typed compositions and syncs, step helpers, captured action inputs, named choices, evidence exceptions only, refinement reads instead of `Reads.through`.
- `sticky(predicate)` and `stickyAcross(predicate)` (`.plans/TEMPORAL_PATTERNS.md` section 2.4): plain `def`s in `model/umpire/Syntax.scala` the lifter matches by name in `model/lifter/Syntax.scala`, the monitor's name taken from the `val` (R2). Lower to the existing `ir.Monitor` with Boolean state, `initial = false`, `next = or(m, not(call(p, after)))` (or `call(p, before, after)` for the two-state form), `violated` the identity, read after every step; this is the tree the two lambdas lift to today, so the `nexus-close.json` monitor entries and the monitor-agreement exports stay byte-identical. Rewrite `retainedOutcome` (`Model.scala:436`) and `ownerAcknowledgment` (:445) with it; `singleOutcome` and `cancelPrincipal` need history and stay hand-written.
- Core and sugar (fn-112's rule, this spec's API Contracts): the conversion uses only sugar forms fn-112 placed in `Syntax.scala` files and adds none outside them; `sticky` is the one sugar form this spec adds.
- Keep files where they are so diffs stay reviewable; record each construct that does not fit with its reason (R6).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`
- `model/temporal/nexuscaller/` after task 2 - the pattern
- `model/lifter/Declarations.scala:365-405` - `monitorOf`
- `model/umpire/Syntax.scala` and `model/lifter/Syntax.scala` (post fn-112) - where sugar and its matching live
**Optional:**
- `model/ir/nexus-close.json` - names and IDs to keep
- `model/lifter/testdata/lifts/CloseReset.scala` - fixture style

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir
make umpire-check-model
scala-cli test model/lifter
```

### Execution constraints
- `nexus-close.json` changes only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, and fn-112 R1's function projection); tables, IDs, fingerprints and answers exact; the two `sticky` monitors are tree-identical to today's.
- fn-122 (capabilities) may be running alongside: it adds `model/umpire/Capabilities.scala` and `model/lifter/Capabilities.scala` and does not edit `Monitor.scala`, `monitorOf` or the `Syntax.scala` files; whichever lands second rebases.
## Acceptance
- [ ] Every R2 declaration in closepolicy takes its name from its `val` or the one explicit form; evidence lists only exceptions; no `Reads.through` remains.
- [ ] Machines are derived rather than copied, compositions use typed selectors, branches use named choices, inputs use captured tokens; kept old forms are listed with reasons.
- [ ] `sticky` and `stickyAcross` exist in `model/umpire/Syntax.scala` as plain `def`s documented with their `monitor` form, matched in `model/lifter/Syntax.scala`; `retainedOutcome` and `ownerAcknowledgment` use them; the monitor entries of `nexus-close.json` are byte-identical; one lifting fixture proves both arities IR-equal to their `monitor` spellings and one refusal (wrong state type) is recorded at the compiler layer; `Monitor.scala` imports nothing from `Syntax.scala`.
- [ ] R1 goldens and fn-112.1 equivalence pass; model gate, lifter tests and lint-model pass.
## Done summary
Converted the Nexus close-policy declarations to the final DSL in place (files not moved), and added the monitor pattern `sticky`/`stickyAcross`. Commits: 6bd505d17a (sticky), 6269ce6172 (conversion), 61ba58cdc4 (diagnostics test), d4377cdd71 (review P3).

**What changed**
- **sticky (R12).** `sticky(p)` and `stickyAcross(p)` are plain defs in `model/umpire/Syntax.scala` with `Core form:` docs. `model/lifter/Syntax.scala` has the hook `stickyMonitor`, which lowers each to the `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !p(...))(broken => broken)` tree, read every step. `Declarations.monitorOf` calls it from its fallback arm (the one hook). `Monitor.scala` is untouched. `stickyAcross` was added to the gate's `SyntaxRule.sugarNames` (`sticky` was already there).
  - `retainedOutcome = sticky(outcomePreserved)` and `ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)`. Their `nexus-close.json` monitor entries differ only in position lines, and the function bodies only in parameter names, which the projection alpha-normalizes.
- **Names (R2).** Machines (the `*Design` vals were renamed to their IR names), actions, input tokens (`principal`, `result`), `internal`, monitors, Limits, and every local Property/Scenario whose val equals its name now take the name from their val. The family is a `given`. The entity val was renamed `request` → `nexusRequest`. Queries whose name equals the default `<machine>.<scenario>.<property>` drop their string (9 per shared def).
- **Constructs (R6).**
  - **Derivation:** eight designs derive from `rejectAfterClose` via `assuming`/`rebind`/`extend`.
  - **Step helpers:** `accept`/`disabled`/`in`/`records`/`implies`, and `once(isDone).keeps(_.handler)` in place of the `handlerEffectIsIrreversible` def.
  - **Named choices** (`taken`, `rejectedForNow`, `ackLost`, `refused`) on the three branching deliveries.
  - **Evidence** lists only `cancelRequested(_)` (a differing name) and `handlerFinished(_)` (a fact with fields needs a line). No `Reads.through` existed.
- **Literals (R11, `metrics-{before,after}.txt`).** The folder went from 1,207 lines and 148 literals to 1,011 and 58.

**Kept old forms, with reasons (R6 errors)**
- **Assumptions keep `assume("…")`.** All six IR names differ from their vals, and an assumption's Definition ID is built from its val's name, so renaming a val would change the ID.
- **Six Properties keep `m.property("…")` inside the shared defs** (`outcomePreserved`, `ackOnlyWhenKept`, `closedHistoryIsFrozen`, `knownIsTheHandlersOutcome`, `knowledgeIsFinal`, `noUnnecessaryWait`). A local val of that name would refer to the predicate def it reads. This is fn-112 R7's explicit form inside a def.
- **Computed Query names stay `s"${m.name}.…"`** where the name is not the default.
- **Progress claims keep `leadsTo("…")`.** Progress is not an R2 kind, has no captured form, and seven share the name `outcomeReachesOwner`.
- **The `refused` alternative writes its step out** rather than calling `dropped(s)`, because fn-120.1 refuses helper calls as choose alternatives. `dropped` still serves the two non-branching sites.

**Decisions taken autonomously**
- **The IR-equality fixture extends `lifts/Sugar.scala`, not a new `expected/*.json`.** It adds a `watched` machine with both sticky forms beside their `monitor` spellings, and a test asserting equal monitors. The fn-115 golden inventory refuses unknown expected files, as fn-112.2 recorded.
- **The wrong-state-type refusal is in `crossed/Sugar.scala:23:69`.** `Rejects.scala` must compile, and crossed/ is where sugar compiler refusals are recorded.
- **Go tests outside Touches:**
  - `nexus_close_baseline_test.go` now reads captured names (val-named machines, monitors, Properties, Scenarios and default-named Queries).
  - `nexus_close_test.go` expects the inert choice names.
  - `diagnostics_test.go` reads the refusal line from the Query's position instead of a pinned line 230.
- **Worktree build setup:** generated Go files, `proto/api.binpb` and the copied jars were given fresh mtimes so the gate's staleness checks pass. No content changed.

**IR delta.** `nexus-close.json` changes only in positions, function references and inventory (inherited lambdas and evidence keep the source machine's name, and `handlerEffectIsIrreversible` is now a pattern function), and inert choice names. Source roots are unchanged and no Case changed.

**Verification (gates.status).**
- gen-model; OriginalBaseline + Migration + NexusClose (golden/model/lower); check-model; lint-model.
- Full Go suite: one failure, the pinned diagnostics line, fixed in 61ba58cdc4; the model package was re-run green.
- lint-code-fast.
- After the P3 fix: gen-model, baseline/close/validate tests and lint-model again.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3 (the `dropped` duplication) and one FYI (`because` consistency). Both were fixed in d4377cdd71 after the verdict.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6bd505d17a, 6269ce6172, 61ba58cdc4, d4377cdd71
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, includes lifter tests), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration|NexusClose|Validate' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1 on a pinned diagnostics line, fixed in 61ba58cdc4; ./tools/umpire/model/... re-run exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: