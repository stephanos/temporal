---
satisfies: [R13, R14]
---
# fn-120-adopt-what-quint-does-well-named.5 Close the Quint-inspired tools

Touches: [model/gate/**, model/README.md]

## Description
Owner decision 2026-10-05: the IR explorer (former fn-120.4) is removed entirely with R8-R10; R13, which that task carried, moves here.

Close the spec once the choice rollout and lint are settled. ITF interchange (former Part D, R11 and R12) was withdrawn by the owner on 2026-10-04 and is not built.

**Size:** S
**Files:** model gate and README.

### Approach
- Run the closing model, Scala lint, Go tooling and fast Go lint gates once.
- Make sure the model's README tells an author how to use lint.
- Write R13 into `model/SEMANTICS.md`: the semantic levels and which declaration takes which, that any temporal operator Umpire adds takes its meaning from TLA, and the "Modalities" paragraph under Machines.

## Acceptance
- [ ] R13 is in `model/SEMANTICS.md`.
- [ ] R14 full gates pass and README tells authors how to use lint.

## Done summary
Closed fn-120's R13 and R14. Commits: aa00195dfd (SEMANTICS levels and modalities, lifter level refusal and fixtures, README), b478b83dd0 (review P3: precondition named in the refusal, with a fixture).

**What changed**
- **SEMANTICS "Levels" (R13).**
  - It names the five levels: value, reading of a state, step, reading of a step, claim over a path.
  - A table says which level each place of a declaration takes.
  - It states the rule that only a step makes a step record.
  - It lists the wrong levels the Scala types already rule out, which get no fixture:
    - a value reading a state or a step;
    - a state reading reading a step;
    - any expression reading a claim's truth.
  - It records that any temporal operator Umpire adds takes its meaning from TLA. A progress claim is TLA leads-to, cut to `within` steps, under WF. `eventually` is reserved for TLA's `<>`.
- **SEMANTICS "Modalities"**, one paragraph under Machines:
  - a row is permission with fixed results;
  - a disabled pair is prohibition for a timer or internal action, and silence for a party action;
  - obligations are same-step Properties, progress claims and fairness;
  - refinement narrows permission and does not by itself preserve obligation.
- **Lifter (R13, fixture half).** The lifter refuses, at its line, an expression that makes a step anywhere except in a function that gives steps. That covers `Step(...)`, `accept`, `stay`, `choose`, `copy` of a step and a call of a step function. The refusal applies in a function that gives anything else, in a precondition, and in a declared value.
  - It is implemented as `Context.making` and `Expressions.wrongLevel`/`giving`.
  - There are 12 `level*` fixtures in `Rejects.scala`, with matching lines in `rejects.txt`: start, ends, evidence, refinement, monitor, precondition, same-step Property, transition Property, claim pattern, progress claim, composition ends and Scenario start.
  - No checked-in Model tripped the refusal: `model/ir` and `model/cases` are unchanged.
- **README (R14).**
  - A "Writing a Model" paragraph on model lint: `make umpire-check-lint`, `umpire-lint --tables`, `.lint.json` acceptances (kind/owner/subjects/because) and how the gate fails.
  - A sentence on the level rule in the lifter's step-function bullet.
- **Explorer mentions.** No live doc or Go comment mentions the explorer, so there was nothing to remove. The fn-120 spec body still has historical explorer lines (Part C note, API example, edge cases); I left them as the owner's record.

**Decisions (own recommendation, pre-authorized)**
- **Lifter change in scope.** I implemented R13's refusal-fixture clause, although the task's Touches named only `model/gate/**` and README. R13 requires it, and closing the spec without it would leave R13 half met.
- **One enforced rule.** The rule is "only a function that gives steps makes a step", keyed by Scala type. The other wrong levels are type-impossible and are listed in SEMANTICS instead of given fixtures, as R13's Errors clause allows.

**Gates** (`.flow/tmp/fn120-5/gates.status`, heavy lock, all exit 0):

| Gate | Time |
|---|---|
| `make lint-model` | 152 s |
| `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | 372 s |
| OriginalBaseline | 370 s |
| full Go suite (`-json -p 2`) | 242 s |
| `make lint-code-fast` | 80 s |

- The slowest Go tests were TestOriginalBaselineModel (49 s), TestOriginalBaselineCases (48 s) and TestMigrationProjectionPreservesSemantics (47 s).
- After the P3 fix, lint-model and the model gate ran again and passed. The fix changed no Go code or IR, so the Go suite, the baseline and lint-code-fast results stand.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. The writer and the reviewer are the same family (Opus).
- Round 1: SHIP with one P3. The refusal in a precondition named "a function that gives Boolean"; it was fixed in b478b83dd0.
- Round 2: SHIP, no findings.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aa00195dfd, b478b83dd0
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -json -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: