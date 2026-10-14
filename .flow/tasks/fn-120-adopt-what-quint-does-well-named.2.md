---
satisfies: [R2, R3]
---
# fn-120-adopt-what-quint-does-well-named.2 Finish named-choice rollout and refuse unnamed branches

Touches: [model/temporal/**, model/lifter/**, model/umpire/**, model/ir/**]

## Description
Complete Part A only after fn-114 has migrated all Models and fixtures to the settled syntax. Fn-112 and fn-114 own their conversions; this task audits every remaining consumer and turns on the final refusal. Before work, verify fn-114 is closed. Flowctl permits only same-spec task dependencies, so this is an explicit cross-spec entry gate.

**Size:** M
**Files:** remaining model/temporal and fixture branch declarations, model/lifter refusal fixtures, model/umpire author surface.

### Approach
- Inventory branches after fn-114's rollout; identify any alternative nobody can name for owner disposition without changing its behavior.
- Refuse a multi-result unnamed list at its Scala line only after all current consumers use named choices.
- Reuse task 1's original-baseline harness and its single choice-name-only allowance; do not define another projection or enlarge the accepted delta during retirement.

## Acceptance
- [ ] fn-114 is closed before this task starts; every current Model and lifter fixture with multiple results names each branch.
- [ ] The lifter refuses an unnamed multi-result list at the source; one-result steps remain valid without choose.
- [ ] Task 1's unchanged harness proves all behavioral, identity and Case outputs remain at the original baseline under its precise choice-name metadata allowance.

## Done summary
Finished the named-choice rollout. The lifter now refuses unnamed branching, and a `choose` alternative may call a function. Commits: 8d6d4ffd23 (lifter, DSL, fixtures, docs), 5e90d11ce5 (Models and IR), 4b139bc95f (review P3s).

**What changed**
- **Refusal (R2).** The lifter refuses a step function's several results written without `choose`, at their line. That means a `List(Step(...), Step(...))` of two or more steps, or step lists joined with `++`, in a step function or any function it calls. A one-result step needs no `choose`. Inside a `choose` alternative, the choose's own "not one step" message still wins (`Context.choosing`).
- **Helper-call alternatives (decision).** An alternative may call a function of the lifted sources that gives no step or one written-out step in each branch: `forged -> Protocol.completeStep(s, Resolution.succeeded)`. Where that function gives no step, the alternative is not taken.
  - The IR calls a copy, `<function>$<choice>`, whose every step carries the name. Nested calls get their own copies, and the function keeps its unnamed steps at its other calls.
  - The choose lifts as the concatenation of its alternatives in the order written. An all-written-out choose is still one list, the same IR as fn-120.1.
  - The runtime `choose` now allows at most one step per alternative. The lifter still refuses an alternative written as `Nil`/`disabled`/`if`/two steps, a called function that gives several steps or a step it does not write out, and the rest of fn-120.1's refusals.
- **Models.**
  - Nexus control `forgedComplete` now uses `choose(forged -> ..., sent -> ...)` over two `completeStep` calls. This was fn-114.2's unnamed `++`.
  - The close policy's `refused` alternative calls `dropped(s)` again, instead of fn-114.4's written-out copy.
  - No other Model or fixture had unnamed multi-result branching: the refusal now runs on every lift, and gen-model and the lifter tests passed. No alternative needed owner disposition.
- **Fixtures.**
  - `Choices.scala` retired the `*Unnamed`/`unchosen` twins (fn-114.6).
  - It added `retry`, whose alternatives call `admitted` directly and through `redeliveredStep`, plus an unnamed `resume` call.
  - The test holds each copy equal to its function apart from names.
  - `Rejects.scala`: `choiceHelper` now refuses a called function that gives two steps; new refusals are `choiceKeptHelper`, `unnamedList`, `unnamedJoin` and `unnamedInHelper`. `rejects.txt` was regenerated.
- **Docs.** SEMANTICS "Named choices", the README named-choice section, and the spec API contract (the two rules settled here).

**IR / R3.**
- Only `nexus-control.json` and `nexus-close.json` changed: function copies, call targets, inert names and positions.
- `model/cases` and `lifts/expected/*.json` (other than `rejects.txt`) are unchanged.
- `original.json` and the harness are unchanged. Copies sit inside the existing Functions projection.
- Nexus-control's conformance Model identity moves, as any function refactor does. SEMANTICS now says so.

**Gate (conductor decision).** The conductor started this task before fn-114 closed. fn-114.1-.7 and .11 were done; .8/.9/.10/.12 are cleanup.

**Verification** (`gates.status`, heavy lock):
- gen-model --skip-go-checks (includes lifter and framework tests)
- OriginalBaseline / Migration / NexusClose / Choice Go tests
- lint-model, the full Go suite and lint-code-fast
- the lifter tests again after the P3 commit

All passed.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. The writer and the reviewer are the same family (Opus).
- Round 1: SHIP with two P3s. One was SEMANTICS' identity claim, the other the missing copy-equality test; both were fixed in 4b139bc95f.
- Round 2: SHIP.

**Shared files** (other lanes):
- `model/lifter/{Expressions,Context}.scala`, `model/lifter/test/Fixtures.test.scala`
- `model/lifter/testdata/lifts/{Rejects.scala,expected/rejects.txt,Choices.scala}`
- `model/umpire/Machine.scala`, `model/README.md`, `model/SEMANTICS.md`

CloseReset.scala (fn-114.10) was not touched. It still writes the `refused` step out, as copied text.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8d6d4ffd23, 5e90d11ce5, 4b139bc95f
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; includes lifter and framework tests; model/ir changes only nexus-control.json and nexus-close.json, model/cases unchanged), scala-cli test model/lifter (exit 0, after 4b139bc95f), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration|NexusClose|Choice' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower ./tools/umpire/export ./tools/umpire/explore (exit 0), make lint-model (exit 0); make lint-model-lifter lint-model-lifts after 4b139bc95f (exit 0); scala-cli fmt --check (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: