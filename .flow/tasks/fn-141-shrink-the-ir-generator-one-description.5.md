---
satisfies: [R7, R8, R9, R15]
---
# fn-141-shrink-the-ir-generator-one-description.5 Capture points and function lookup by span (proof for Part B)

## Description
Part B's early proof. The framework learns to record where a declaration is written and where a function is written. The lifter learns to lift the function found at a recorded span. Nothing is migrated yet: one fixture shows that a Property declared this way gives the IR the tree path gives.

**Size:** M
**Files:** capture data/macros and tests in `model/framework/dsl`, span lookup and binding in `model/framework/export` and `inspect`, dedicated proof/refusal fixtures and coverage, `.plans/DSL_OPERATORS.md`, `.plans/SCALA.md`, `model/SEMANTICS.md`.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/export/**, model/framework/inspect/**, model/framework/lint/**, model/framework/check/**, model/temporal/check/**, model/SEMANTICS.md, .plans/DSL_OPERATORS.md, .plans/SCALA.md]
**Activation (2026-10-13):** .3 and .4 are committed and done; .4 receipt is `58bcd2debc`, with a fresh 610-test scoped pass. Current homes replace old `umpire`, `irgen` and `check` paths below. Re-read before editing. This is the complete Part B proof, not a metadata-only spike. Stay on `umpire`, one worker writer; review none and planSync=false under the source-batch policy. Real activity/Nexus, generated payloads, schema, Go, tooling and Makefile edits are outside scope. Independent review and mini IR/Go propagation remain batch obligations.

**Existing owner edits:** `model/README.md` and `model/framework/samples/Kettle.scala` are dirty at activation and are outside Touches. Preserve their exact working content and staging state; do not edit, revert or commit them. Root owns MILESTONES. Exclude unrelated owner edits from every task commit, including any newly observed ones. If an edit overlaps this task, report the exact overlap rather than overwriting it.

### Approach
- Naming context: a context parameter the framework supplies from the enclosing `val` or object. It carries the name, the qualified path the Definition ID is made of, and file and line. Hold it to the lifter's `capturedName`, `qualifiedName` and `familyOf`, which define today's values.
- Recorded function: the function, its source span, and the values it closes over, by name. A def reference and a lambda both convert at the call site.
- Carry forward fn-141.3's selector boundary: composition `records` currently captures a direct typed member selector at its thin call-site macro. Prove reusable recorded selector data can be passed through helpers and reused in both native Scala and lifting, retaining exact member identity and canonical fact typing. Do not substitute function identity, compiler-generated names, reflection or state equality for recorded data. Opaque raw selector functions remain refused until this explicit data path is proven.
- These are the only `inline` or macro uses in the framework. Weigh `sourcecode` against a small macro of our own and record both line counts (fn-113 R25).
- Lifter: find the function whose tree has the recorded span, lift it under the liftable subset, and bind each captured value. A finite Model value becomes the IR value it is; a recorded function becomes a call. The spike of 2026-10-06 proved the capture and did not prove this lookup.
- Refuse a captured value that is neither, naming it and its type, and a span with no function.
- Revise rule 5 of `.plans/DSL_OPERATORS.md` to say where `inline` and macros are allowed (R15).
- Reuse the shared typed Index, generic expansion and subset admission from .1/.3/.4. Thin capture macros record metadata only; the exporter lifts function bodies. No function identity, runtime reflection, compiler-generated name or equality-based lookup. Preserve exact names/IDs/positions, finite bindings, strict/default/contextual evaluation, caller-source provenance and all frozen IR/refusal contracts.
- Prove span lookup consumes recorded data: missing, mismatched and ambiguous spans are explicit refusals, rather than silently falling back to call-shape recognition. A helper/comprehension and reusable recorded function/selector must use this same path. Existing declaration sugar and reserved Bindings oracles remain assigned to .9-.14 until constructed-value migration; do not make a new permanent exception.

### Investigation targets
**Required:**
- `model/irgen/Context.scala` (`capturedName`, `qualifiedName`, `familyOf`, `definitionId`, `pos`)
- `model/irgen/Expressions.scala` (`lambda`, `function`, `parameters`, `forwardedDef`, `boundValues` use)
- `model/irgen/Constants.scala` and `Declarations.scala` (`literalValue`): a Model value as an IR value
- `model/umpire/Syntax.scala` (`Rules.on`, `codeOf`, `writtenAction`): the existing inline exception
- `.plans/DSL_OPERATORS.md` rule 5; fn-113 R15 and R25; `.plans/SCALA.md`, How Scala produces the IR

## Acceptance
- [ ] munit: name, ID and line are recorded from a `val`, an object, a helper def with the context, and a comprehension; a declaration with nothing to name it does not compile or is refused at its position.
- [ ] munit: a recorded function carries its span and captured values; a def reference and a lambda both convert.
- [ ] A fixture Property declared through the capture points exports the same IR as its tree-lifted twin.
- [ ] Reusable recorded composition selector data works through helpers in native Scala and lifting; equal-state/different-member and wrong-fact-type controls retain exact member/fact identity.
- [ ] Reject fixtures: a captured value of no finite Model type; a function body outside the liftable subset is refused with today's text at today's position.
- [ ] The `sourcecode` weighing and both line counts are in the done summary.
- [ ] If the lookup or the binding cannot be made to work, stop: Part B does not continue (see the spec's Early proof point).
- [ ] Meaningful red/green native and export tests prove capture, lookup, finite/recorded-function binding, refusal and reusable-selector parity. Move rewritten construct tests into dedicated suites and register coverage.
- [ ] A fresh terminal `timeout 1200s make model-check-changed BASE=<original-task-base>` passes, including all samples, mini step-table equivalence and selected suites. Frozen fixture IR, names, positions and refusals are unchanged. Preserve unrelated owner edits; diagnose any inherited out-of-scope failure separately. Strict generated IR/full/Go gates remain held for propagation, with no regeneration here.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
