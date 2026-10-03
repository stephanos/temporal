---
satisfies: [R3, R4, R5, R10]
---
# fn-117-type-the-temporal-api-in-the-models.4 Type all existing path roots and symbolic operands

## Description
Complete the typed selector and operand module behind the method seam, covering every existing realization path context.

**Size:** M
**Files:** model/umpire/realize typing module and Realize.scala; model/lifter/Realizations.scala; focused compile/lift fixtures and Fixtures.test.scala.
**Touches:** [model/umpire/realize/**, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, .flow/tmp/fn117-4/**]

### Approach
- Keep one field-selection abstraction parameterized by root message and value type; normalize accessor spelling through generated metadata, including optional nested fields, repeated elements and oneof arms. Preserve exactly plain, [*] and oneof<member> IR path text. Reject a field/root/oneof mismatch at compile time.
- Apply this abstraction to assignments, RPC responses, recorded reads/single values, polls, evidence fields, operation keys and guards. Roots come from method request/response or named evidence where known; explicitly name InstructionOutcome for Projected run-event payloads and HistoryEvent where required. Model-owned role/evidence/command IDs remain as they are.
- Carry types through symbolic environment, run and learned-value operands, literal/equality conditions and any conversions needed by today's realizations. Reject a value or symbolic operand with the wrong field type. Genuinely unknown dynamic values must be narrowly listed with their reason and existing Go validation coverage; never erase all operands to Any.
- Keep metadata-only field lambdas from introducing extra IR functions, types, expressions or declaration order changes. No symbolic operand builds or sends a message. Preserve unsigned/range and other runtime validation in Go rather than claim Scala native scalar types prove it.
- Add negative fixtures for misspelled/wrong-root field, wrong literal and symbolic type, invalid oneof selection; add positive nested/repeated/oneof/contextual/explicit-root and poll/operation/guard paths. Compare typed and existing outputs modulo positions only. Map-entry reads remain outside Go's supported path grammar; task5 types the existing constant-message maps.

### Investigation targets
**Required:**
- model/umpire/realize/Realize.scala:50-78,256-315
- model/lifter/Realizations.scala
- tools/umpire/lower/realization.go:134-350,695-830
- tools/umpire/lower/descriptor.go:90-135
- model/temporal/standaloneactivity/Realization.scala
- model/temporal/nexuscaller/Realization.scala
- model/lifter/test/Fixtures.test.scala

### Quick commands
Focused negative/positive compiler/lifter fixtures; full lifter suite, check-mode model gate and lint-model once ready; reuse unaffected Go and artifact coverage per MILESTONES.md.

### Execution constraints
No new map-read path language, Go/schema changes, runtime API-hint metadata or script helper DSL. Preserve comments, no staging/commits/worktrees/push; serial gate ownership.

### Poll typing contract
Poll takes the typed read-evidence reference built by task3. Its assignments use that evidence source method's request message; its condition uses the projected value selected by Recorded.Read/Single, including repeated-element unwrapping. Authors cannot pick a different poll root manually. Add two compile-negative fixtures: evidence from method A with an assignment field of request B, and evidence projecting value A with a condition field of unrelated message B. A positive poll fixture must preserve the exact existing evidence ID, request assignments and until operand in IR. Explicitly named payload roots remain for unknown-origin run-event/guard contexts, not an escape from known poll evidence types.


## Acceptance
- [ ] Every existing path context has a typed root/value selection; repeated and oneof forms lift to the existing text grammar, and explicit dynamic-origin roots work.
- [ ] Poll takes a typed read-evidence reference and derives request and projected-value roots from Recorded.Read/Single; wrong-root poll assignments and conditions fail compilation, while valid polls lift the unchanged evidence ID.
- [ ] Literal and symbolic type mismatches fail compilation; any intentionally unknown operand is listed with a narrow reason and retained Go check.
- [ ] Positive/negative fixtures and scoped gates pass; lifted meaning and generated outputs change only in positions.

## Done summary
Task fn-117-type-the-temporal-api-in-the-models.4 implements typed field paths and typed operands across the existing realization contexts while preserving the emitted IR. The positive typed fixture matches the legacy realization after positions and realization IDs are removed. Wrong roots, values, fields, oneof arms, and direct carrier construction fail compilation.

Review round 1 fixes: `Recorded.read` now tracks the terminal selected protobuf field and refuses a mapped selector ending in a singular message with a positioned lifter diagnostic before writing IR. The Go read-source contract requires a terminal repeated message, so `executions[*].execution` is not a valid read source. Only `Operand.Projected.as[T]` can name an unknown message origin; `Operand.Run.as[T]` fails compilation, and the valid projected form still lifts. Constructor privacy remains source-level enforced with `@publicInBinary` for the inline expansion.

baseline: green via task-3 handoff (.flow/tmp/fn117-3/summary.md) at the same HEAD. Exact seven source paths: .flow/tmp/fn117-4/touched-files.json. The six pre-existing source files, including task-3 untracked files, have mirrored pre-edit copies under .flow/tmp/fn117-4/preedit/<repo-relative-path>; dynamicInvalid/Invalid.scala is new. All pre-existing source comments in those files remain byte-for-byte present.

Verification: review-red mapped fixture failed as intended (.flow/tmp/fn117-4/review-red-mapped.log); standalone non-Projected inline probe failed as intended (.flow/tmp/fn117-4/review-probe-dynamic.log). Final focused Fixtures suite 21/21, exit 0 (.flow/tmp/fn117-4/review-fixtures-3.log). Final full check-mode model gate exit 0, including lifter suite and IR/Case comparison (.flow/tmp/fn117-4/review-check-model.log). Final lint-model exit 0 with zero current compiler error diagnostics (.flow/tmp/fn117-4/review-lint-model.log); it retains the inherited caught JDK 27 Scalafix NoSuchFieldException trace. Format check exit 0 (.flow/tmp/fn117-4/review-format-check.log), all 30 artifact hashes unchanged (.flow/tmp/fn117-4/review-artifacts.log), and git diff --check exit 0. No Go input changed; existing fn-113.16 Go evidence remains applicable.

Intentionally dynamic origins are the explicitly named HistoryEvent attribute and InstructionOutcome RunEvent payload, validated by the existing Go history/payload walkers. Numeric Scala types establish kind; Go retains numeric range checks. Environment and learned bindings are constrained to String. No source staging or commit. The same independent review session reached SHIP; the conductor completes Flow tracking.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave-dispatch - ran (model: gpt-6-sol at high)
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test model/lifter --test-only umpire.lift.Fixtures (exit 0; 21/21; .flow/tmp/fn117-4/review-fixtures-3.log), mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; .flow/tmp/fn117-4/review-check-model.log), mise exec -- make lint-model (exit 0; zero current compiler errors; .flow/tmp/fn117-4/review-lint-model.log), mise exec -- scala-cli fmt --scalafmt-conf /Users/stephan/Workspace/temporal/umpire/model/.scalafmt.conf --check <7 task Scala sources> (exit 0; .flow/tmp/fn117-4/review-format-check.log), sha256sum -c .flow/tmp/fn117-3/artifacts-before.sha256 (exit 0; 30/30; .flow/tmp/fn117-4/review-artifacts.log), git diff --check -- <7 task Scala sources> (exit 0), Independent implementation re-review: SHIP, same session 01a101ad-57f4-7973-bd1f-80811d429130, Conductor verified all 51 pinned hashes and30preserved artifacts; final lint no compiler errors and git diff --check passed
- PRs: