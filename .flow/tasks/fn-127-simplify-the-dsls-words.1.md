---
satisfies: [R1, R3, R4]
---
# fn-127-simplify-the-dsls-words.1 Rename the DSL words that collide with Temporal's

## Description
Implements R1 and the word half of R3. This is a mechanical rename with zero change to the IR, Cases or Contracts. Words: `accept` → `enter`, `Accepted` → `Ok`, the realization `poll` → `readUntil`, `.setting` → `.withFields`, `always` → `everyCase`, and `umpire.realize.Outcome` → `PropertyOutcome`.

**Cross-spec entry gate:**
- fn-114, fn-118 and fn-122 are closed (fn-118.5 has rewritten the waits in every `Realization.scala`).
- Not concurrently with fn-124.8.
- fn-124.3 also edits realization surfaces; whichever lands second rebases.
- fn-126 does not start until this spec closes.

**Size:** M
**Files:**
- `model/umpire/Syntax.scala`, `model/umpire/realize/{Scripts,Realize}.scala`;
- `model/temporal/realize/*`;
- every Model and `Realization.scala` under `model/temporal`;
- `model/irgen/{Syntax,Realizations}.scala` and the lifter fixtures;
- `model/check/SyntaxRule.scala`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/DSL_OPERATORS.md`.

**Touches:** [model/umpire/**, model/temporal/**, model/irgen/**, model/check/**, model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md]

### Approach
- Rename each definition in place, keeping its `Core form:` doc (update the doc's spelling). `Ok[O]` replaces `Accepted[O]` as the given that `enter` and `stay` read.
- Rewrite the call sites mechanically (sed, or a scalafix rule if one is cheaper), then format. Rename `umpire.realize.Outcome` everywhere it is imported, and drop the `Outcome as RunOutcome` alias in `standaloneactivity/admission/Queries.scala:6`.
- Update the lifter wherever it matches these names by spelling (`model/irgen/Syntax.scala` for the sugar, `Realizations.scala` for `poll`, `setting` and `always`), and its fixtures. Update `SyntaxRule.sugarNames`.
- Leave each Model's `enum Outcome` and its case `accepted` alone. R1 says why: the case is in IR type catalogs, fingerprints and Case bytes.
- Docs: replace the words in `model/README.md` and `model/SEMANTICS.md`. In `.plans/DSL_OPERATORS.md`, add one entry per rename with its reason (the Temporal word or the reserved operator it collided with).

### Investigation targets
**Required:**
- `model/umpire/Syntax.scala:1-40`
- `model/umpire/realize/Scripts.scala:40-110`, `model/umpire/realize/Realize.scala:470-490,595-605`
- `model/irgen/Syntax.scala`, `model/irgen/Realizations.scala` (grep `"poll"`, `"setting"`, `"always"`, `"accept"`)
- `model/check/SyntaxRule.scala:30-60`
**Optional:**
- `.plans/DSL_SIMPLIFICATION.md` section 4a

### Quick commands
```bash
grep -rn "accept(\|Accepted\[\|\.setting {\|always(\|\bpoll(" model --include=*.scala | grep -v /testdata/migration
make umpire-check-model && make lint-model
git diff --stat model/ir model/cases    # empty after --update
```

### Execution constraints
- Zero IR, Case, Contract or lint-finding change. If regeneration shows any diff in `model/ir/**` or `model/cases/**`, stop and find the name match that changed meaning.

## Acceptance
- [ ] `enter`, `Ok`, `readUntil`, `.withFields`, `everyCase` and `PropertyOutcome` replace the old words in the framework, kit, lifter, Models, fixtures and tests. The old definitions are gone, and each sugar keeps its `Core form:` doc.
- [ ] Each Model's `enum Outcome` keeps its case `accepted`.
- [ ] After `make umpire-check-model --update` and `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`, `git diff model/ir model/cases tests/testcore/testpilot/testdata` is empty.
- [ ] The grep of R3 over `model/` and the live docs finds no old word. `.plans/DSL_OPERATORS.md` records each rename with its reason.
- [ ] The model gate, `make lint-model`, the Umpire Go tests, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case` and `make lint-code-fast` pass.


## Done summary
Renamed the DSL words that collide with Temporal's, or with an operator the DSL reserves. Nothing they lower to changed: `model/ir`, `model/cases` and `tests/testcore/testpilot/testdata` are byte-identical to `umpire` at aabb1d14ef.

**Renames (R1)**
- `accept(state, facts*)` became `enter`, and `given Accepted[O] = Accepted(o)` became `given Ok[O] = Ok(o)` (`model/umpire/Syntax.scala`). Both keep their `Core form:` docs. Each Model's `Outcome.accepted` stays.
- The realization's `poll` became `readUntil`, in both forms: the script helper in `Scripts.scala` and `Instruction.poll` in `Realize.scala`. The kit's `await` now calls `readUntil`. The IR record and its proto field keep the name Poll.
- `call.setting { … }` became `call.withFields { … }`, and `always(command)` became `everyCase(command)`.
- `umpire.realize.Outcome` became `PropertyOutcome`. Both `Outcome as RunOutcome` aliases are gone (`admission/Queries.scala` and `nexusoperation/Queries.scala`).
- The lifter's name matches were updated:
  - `irgen/Syntax.scala`: `enter`, and the companion check `umpire.Ok`.
  - `irgen/Realizations.scala`: `everyCase`, `withFields`, `readUntil` and the factory `Instruction.readUntil`.
  - The refusal messages in `Expressions.scala` and `Syntax.scala`.
- `SyntaxRule.sugarNames` lists `enter` where it listed `accept`, and now also lists `Ok` (review P3, commit 4ad5f251c9).
- Every Model, Realization, lifter fixture and test was updated, including `SyntaxRule.test.scala`, `Choices.test.scala` and the `Fixtures.test.scala` names and columns. The fixture `ComputedAccepted` became `ComputedOk`.

**Decisions**
- **No line shifts in the Models.** IR positions record source lines, so no line in a Model file could move. One import would have reflowed past 100 columns: `nexuscaller/Queries.scala` with `PropertyOutcome`. It was split into its own import line, and `import Control.inspect, Timeout.expires` was joined into one line to keep the line count. A lifter fixture already uses this comma-import form (`lifts/Scripts.scala:16`). After every `fmt-model`, a `git diff --numstat` check found no line-count change in any Model file.
- **Fixture changes.**
  - `lifts/Capabilities.scala` imports `Conformance`, `PropertyOutcome` and `RunExpectation` instead of qualifying them with `realize.`, so its line does not reflow.
  - In `lifts/Rejects.scala`, one fully qualified `MonitorExpectation` line reflowed by one line. So `expected/rejects.txt` moves by one line for the refusals after line 727, and its two refusal messages read `enter` / `Ok`.
  - The `Overlap` test then found 8 lines `lifts/Realizations.scala` shared with `admission/Queries.scala`. Before the rename, `Outcome.` and `RunOutcome.` had kept them apart. Writing `property = PropertyOutcome.satisfied` as a named argument in `heldByName` breaks the run, and its expected IR is unchanged.
  - Two expected columns in `Fixtures.test.scala` moved: `crossed/Sugar.scala:16:71`, where the fixture `acceptForeign` became `enterForeign`, and `referenceInvalid/Invalid.scala:36:32`.
- **Docs (R3, words).**
  - `model/README.md` and `model/SEMANTICS.md` use the new words.
  - `.plans/DSL_OPERATORS.md` gained a "Words renamed (fn-127, 2026-10-05)" table: one row per rename with the Temporal word or reserved operator it collided with. The old words appear there as bare names, so the R3 grep stays empty.
  - Candidates 5 and 6 and the R5 quote use the new words.
  - Not edited: older dated research notes (`.plans/QUINT_MODULE_LAYOUT.md`, `MODALITIES.md`, `TEMPORAL_PATTERNS.md`), outside this task's touch list. The code pointer in `.plans/API_BEHAVIOR_HINTS.md` now names `Instruction.readUntil(...)` (review P3). Also not edited: `.plans/DSL_SIMPLIFICATION.md`, whose ranks are marked done in task .2.
- **R3 grep.** `grep -rnE "\baccept\(|Accepted\[|\.setting \{|\balways\(|\bpoll\(" model .plans/DSL_OPERATORS.md` (Scala and Markdown) finds only `def poll(j: Job)` in `lifts/Capabilities.scala`. That is a fixture step function named after its Pollable action, not the realization word.

**Gates (R4), each run under the heavy-suite flock; logs in `.flow/tmp/fn127-1/`**
- Regeneration: `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` (gen-model.log), then `make umpire-gen-cases umpire-gen-fixtures canary-gen-case` (gen-cases.log). Afterwards `git diff model/ir model/cases tests/testcore/testpilot/testdata` is empty.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: exit 0, 123s (model-gate.log).
- `make lint-model`: exit 0 (lint-model.log). The scalafix runs print `java.lang.NoSuchFieldException: path` stack traces, which are scalafix classpath reflection on JDK 27. They do not fail the run.
- `go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/...`: 17 packages passed, wall 278s (go-test.log). `tools/umpire/export` was killed by `signal: killed` (OOM on the shared machine) in `TestAgreementReadsAFaithfulDump/nexus-close`. Rerun on its own: `go test -count=1 -json -tags test_dep -p 1 -timeout 30m ./tools/umpire/export/...`, ok in 83s (go-test-export-rerun.log). The OriginalBaseline tests in golden, model and lower all pass. The slowest are `TestOriginalBaselineModel` at 71s and `TestOriginalBaselineCases` at 71s.
- `make umpire-check-cases`: exit 0. `make umpire-check-fixtures`: exit 0. `make canary-check-case`: exit 0.
- `make lint-code-fast`: exit 2. **Pre-existing, unrelated:** this task changes no Go file (`git diff aabb1d14ef HEAD -- '*.go'` is empty), so the Go lint input is the same as on `umpire`.
  - With the default base, the worktree's local `main` (6875191ef7, Feb 2026) gives about 700 upstream findings in chasm/, service/, tests/ and tools/flakereport (lint-code-fast.log).
  - With `GOLANGCI_LINT_BASE_REV=aabb1d14ef`: one finding, `forbidigo` on `common/searchattribute/sadefs/encode_value_test.go:405` (lint-code-fast-umpire-base.log).
  - The target runs golangci-lint with `--fix`. It rewrote 7 upstream Go files, which I restored with `git checkout --`; the tree is clean.

**Review follow-up (P3s, commit 4ad5f251c9)**
- `"Ok"` added to `SyntaxRule.sugarNames`.
- The `lifts/ScriptRejects.scala:103` comment says `readUntil`. It stays one line, so no position shifts.
- The script-helpers paragraph in `model/README.md` is reflowed.
- The `.plans/API_BEHAVIOR_HINTS.md:56` pointer says `Instruction.readUntil(...)`.
- Results:
  - `make lint-model`: exit 0 (lint-model-p3.log).
  - `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: exit 0 (model-gate-p3.log). This covers the gate's own tests, including SyntaxRuleSuite, and the lifter fixtures against unchanged expected files.
  - No Model line counts changed, and `model/ir`, `model/cases` and the testpilot testdata are still identical to aabb1d14ef.

**Environment notes**
- The fresh worktree first failed the gate on `api/umpire/v1/ir.pb.go` being older than `ir.proto`, which is checkout mtimes. I touched it after confirming its content matches HEAD.
- `proto/api.binpb` was missing, so I ran `make proto/api.binpb`. It is untracked.

Subagents: 0.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP, no P1/P2. Four P3s were applied in 4ad5f251c9: `Ok` in `sugarNames`, a fixture comment, a README reflow, and the `API_BEHAVIOR_HINTS.md` pointer. Two P3s were accepted as they are: the comma-joined import in `nexuscaller/Queries.scala` that keeps line counts (fn-126's layout move normalizes it), and the named argument in `lifts/Realizations.scala`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7e6e94104c06be249a42a52d5f477599644b9940, 381195441a0d21c905111252147fe9d39e3d51dd, 4ad5f251c97ff3d7961ca6147f4e5b5bfca2564a
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks && make umpire-gen-cases umpire-gen-fixtures canary-gen-case (git diff model/ir model/cases tests/testcore/testpilot/testdata empty), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... (17/18 packages pass; export OOM-killed, rerun -p 1 ./tools/umpire/export/... ok), make umpire-check-cases (exit 0), make umpire-check-fixtures (exit 0), make canary-check-case (exit 0), make lint-code-fast (exit 2, pre-existing upstream findings; task changes no Go file), make lint-model after review P3s (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks after review P3s (exit 0)
- PRs: