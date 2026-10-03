---
satisfies: [R2, R3, R10]
---
# fn-117-type-the-temporal-api-in-the-models.3 Add typed schema and method declarations with generic metadata lowering

## Description
Build the core message/schema and unary method typing seam selected by tasks1-2. Keep the old forms only as a migration bridge until task8.

**Size:** M
**Files:** model/umpire/Action.scala; a generic typing module under model/umpire/realize; model/umpire/realize/Realize.scala; model/lifter/Declarations.scala and Realizations.scala; focused lifter fixtures/tests.
**Touches:** [model/umpire/Action.scala, model/umpire/realize/**, model/lifter/Declarations.scala, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, .flow/tmp/fn117-3/**]

### Approach
- Expose the smallest generic typed interface for message schema and method metadata; the request and response types travel with the method value. Reuse the generated metadata/descriptor mechanism from task2, with no handwritten Temporal method table or Temporal imports in the reusable DSL.
- Type .schema by protobuf message evidence, retaining ordered multi-schema actions (Nexus names two schemas). A non-message type must not compile.
- Type RPC assignments by the method request and response reads by its response; reuse task2's exact public spelling. Let task4 implement the full selector/operand operators behind this seam, using a focused nested-field example here. Poll evidence IDs remain Model-owned IDs; do not invent method strings for them.
- Teach the lifter to lower these author declarations to today's full message and /package.Service/Method names, preserving schema order, generic descriptor-driven emission, refusal positions and defaults. Keep request/response typing compile-time; the IR remains text.
- Add compile-negative fixtures for a non-message schema and a method/request/response root mismatch, and positive single/multiple schema and unary method fixtures that compare decoded IR with current equivalent string forms modulo positions only. A generated Scala class name is not presumed identical to the protobuf full name.

### Investigation targets
**Required:**
- model/umpire/Action.scala:71-72
- model/umpire/realize/Realize.scala:45-59,256-268
- model/lifter/Declarations.scala:44-45
- model/lifter/Realizations.scala:1-36,150-269
- model/lifter/test/Fixtures.test.scala:56-64,201-305
- model/temporal/nexuscaller/Model.scala:57
- tools/umpire/lower/descriptor.go:53-75

### Quick commands
Focused compile/lift fixtures through the existing harness; mise exec -- scala-cli test model/lifter; check-mode model gate and lint-model once ready. Reuse current full Go baseline unless relevant inputs change.

### Execution constraints
No Model migration yet, no Go lowering/schema/behavior change, no per-message emitter mapping. Preserve comments, user owns commits, and follow MILESTONES.md baseline/gate scoping.

### Typed read-evidence link
Extend Recorded.Read/Single and the realization evidence declaration to retain the method request/response types and the type selected by its projection. A typed evidence reference carries that metadata to Poll; erasing it to an unrelated String and manually selecting a root is not the author API. It still lowers to the existing evidence ID, with no IR fields or general Model ID/name-capture feature. Provide the core reference here and exercise complete poll selectors in task4.


## Acceptance
- [ ] Schemas and unary methods carry verified message/request/response types; invalid schema and wrong request/response roots fail compilation.
- [ ] Recorded.Read/Single and their evidence declarations preserve typed method/request/projection metadata through a typed reference, while lifting the original evidence ID.
- [ ] Typed declarations lift to the same schema/method names and order as the equivalent current declarations; no unrelated IR difference appears.
- [ ] Core interface, refusal fixtures and applicable scoped gates pass without adding Temporal-specific imports to the DSL.

## Done summary
Typed schema declarations now append ScalaPB descriptor names in order. Typed unary RPC, repeated and single response evidence, and poll references retain their request, response, and projected roots; the lifter derives method and field names from generated descriptors and emits the existing IR forms. The positive fixture compares typed and legacy realization IR after removing only positions and root IDs. Following a NEEDS_WORK review, reference carriers now have restricted constructors and no case-class copy, while direct typed enum constructors enforce the same roots as their public factories.

baseline: green via handoff (.flow/tmp/fn117-2/summary.md and .flow/tmp/fn117-2/review-fix-check-model.log); unchanged Go input checks reused from .flow/tmp/fn113-16-evidence.json.

Source scope: model/umpire/Action.scala; model/umpire/realize/Realize.scala; model/umpire/realize/Typed.scala (new); model/lifter/Declarations.scala; model/lifter/Realizations.scala; model/lifter/test/Fixtures.test.scala; model/lifter/testdata/lifts/project.scala; model/lifter/testdata/lifts/Typed.scala (new); model/lifter/testdata/typedInvalid/Invalid.scala (new). Existing-file preedit copies are in .flow/tmp/fn117-3/preedit/; lifts/project.scala was clean at HEAD before this task and its original is preedit/lifts-project.scala.

Verification after the review fix: `mise exec -- scala-cli test model/lifter --test-only umpire.lift.Fixtures` exit 0, 19 tests (.flow/tmp/fn117-3/review-fixtures.log). The negative fixture now has 11 compiler refusals: five original cases and six direct-constructor controls (.flow/tmp/fn117-3/review-negative-final.log); before the fix, those six compiled and only five refusals appeared (.flow/tmp/fn117-3/review-red-direct-constructors.log). `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` exit 0 (.flow/tmp/fn117-3/review-gen-model.log). All 30 prior IR, Case, manifest, and fixture artifact SHA-256 hashes match (.flow/tmp/fn117-3/review-artifacts-verify.log). `make lint-model` exit 0 (.flow/tmp/fn117-3/review-lint-model.log). Final `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` exit 0 (.flow/tmp/fn117-3/review-check-model.log); this gate reran the complete lifter suite. `git diff --check` exit 0. Earlier gate logs are superseded by this review-fix run.

Limitation: full symbolic selector and operand operators, including poll intervals beyond the current zero default, remain for task 4. Typed conditions here support a projected-root `present` field selector, which proves the evidence-to-poll type link.

No commits or staging, per user instruction. The same independent review session reached SHIP after the constructor fix; the conductor completes Flow tracking.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave-dispatch - ran (model: gpt-6-sol at high)
## Evidence
- Commits:
- Tests: baseline: green via handoff (.flow/tmp/fn117-2/summary.md), mise exec -- scala-cli compile model/lifter/testdata/typedInvalid, mise exec -- scala-cli test model/lifter --test-only umpire.lift.Fixtures, make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, sha256sum -c .flow/tmp/fn117-3/artifacts-before.sha256, make lint-model, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, git diff --check, Independent implementation re-review: SHIP; same session 01a1017d-e4f7-7631-a581-623be14f3e3d, Conductor verified all 60 pinned source/evidence hashes and 30 preserved artifact hashes; git diff --check passed
- PRs: