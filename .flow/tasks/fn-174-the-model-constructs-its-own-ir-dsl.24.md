---
satisfies: [R3, R4, R6]
---

# fn-174-the-model-constructs-its-own-ir-dsl.24 Reconcile framework Class/default IR before deferred exact-IR baseline

## Description
**Recut 2026-10-18 (one-language series):** The exact-IR baseline is taken in-process (fn-201); no checked-in IR and no Go reading of it.

**Held framework IR checkpoint, not native DSL work.** Resolve the qualified old-reader/native Class/default discrepancy carried by fn-174.11 and .3 before the later fn-165 exact-IR preservation proof freezes its baseline. Native record-input authoring and its batch gate are already complete; this task never precedes or reopens them. Priority-0 fn-147.5 and any demonstrated unmet native DSL requirement come first. This is not the full signature switch or completion of .3, fn-141, or fn-174.

**Admission:** committed fn-188.4 fresh-artifact/discovery/boundary checkpoint; fn-174.10/.11/.13/.14 source checkpoints and committed receipts; final native expect/API surface fn-164.2/.11/.10 and fn-188.3 plus fn-147.5 closures. This replaces the circular requirement for full fn-188.2 successful-gate closure with its actual artifact prerequisite; fn-164's held IR/lowering proofs remain separate unfinished obligations, not missing native API work. Cross-spec task admissions are conductor-checked. This task remains independently held after .4 and requires a deliberate later admission; do not resume fn-174.2 or system checks.

**Size:** M
**Touches:** [model/framework/dsl/Signature.scala, model/framework/dsl/InputSource.scala, model/framework/lift/Claims.scala, model/framework/lift/Syntax.scala, model/framework/lift/Declarations.scala, model/framework/lift/Constants.scala, model/framework/lift/ConstructedSignature.test.scala, model/framework/lift/InputDefaults.test.scala, model/framework/lift/testdata/constructedSignature/**, model/framework/lift/testdata/inputDefaults/**]. Change only the owning conversion and reached regressions; this is not authority to activate all signature/Machine producers or redesign input authoring.

Read `.flow/tmp/fn165-framework-prerequisite-decisions.md`, the fn-174.11 receipt and its repaired source `26c7d076fa`, the qualification in fn-174.3, and `ConstructedSignature.test.scala`'s nested-default control. Native `Some(None)` and the historical reader output are explicitly not equal. Source presence and isolated Signature support do not resolve that qualification.

1. Reproduce the current discrepancy with fresh framework-only artifacts and distinguish live source export, native conversion and the frozen historical `original-defaults.json`. Preserve the historical output and its source/provenance; do not rewrite it to make equality pass.
2. Establish the intended Class/input type using an independently authored ordinary-core signature/Class oracle and the declared input domain. Cover nested `Some(None)`, `None`, ordinary enum values, false/zero, required and explicit/non-first defaults, omitted classes and explicit values. Compare complete Class/Action/type closure, not strings or selected fields. A declared Option and its None case must not be equated by normalization.
3. Repair the smallest owning path to one explicit contract. Record any semantic correction against the historical output separately from the later record migration. Preserve exact ordered Params, names, supplied finite witnesses, schema identity, default omission/refusal, native evaluation timing and metadata-only non-enumeration controls. No catalog-first default or eager `Finite.values` fallback.
4. Prove live reader, native conversion and the independent ordinary-core oracle agree on the corrected admitted cases. Keep historical mismatch evidence labelled historical; do not silently replace it or present it as the current contract. Include aliases/helper forwarding, saved-tree replay where the seam changes, and two-JVM deterministic output. Existing located refusals and no-output failure behavior remain effective.
5. Record the exact committed contract fn-165.1 may freeze. Remaining native Action/Class consumer activation, one-producer integration, shared-helper reconciliation, full signature/Machine switching, reader retirement, Temporal migration, Go/Cases/backend answers and original R5 remain open in their original owners.

Never read, restore, compile, lift or migrate `model/temporal`. Do not use `model/project.scala`, Temporal Model/API jars, Go, schemas or regeneration. Use fn-188's fresh artifact/ModelBuild contract with explicit `UMPIRE_FRAMEWORK_BUILD`, never its unset production fallback. Reuse fn-164.3's documented narrow artifact build recipe, not stale jars or a new permanent proof harness.

### Verification

Run focused `framework.export.ConstructedSignatureSuite` and `framework.export.InputDefaultsTest` with JUnitCore under the explicit fresh build environment. Run relevant native `framework.dsl.SignatureTest`, `InputDefaults` and selected-witness regressions if their seams change. Record actual commands, exits and positive executed counts. Source/capture/core comparison and two-JVM assertions belong in the normal focused test suites. Keep Scala builds serialized. The full `make check-framework` and independent review run at framework source-batch end; no system gate is credited.
## Acceptance

- [ ] Current discrepancy is reproduced and all three baseline forms (live reader, native, frozen historical output) are accurately distinguished; historical bytes and provenance remain unchanged.
- [ ] An independent ordinary-core oracle establishes the declared input/Class type contract, including nested Option, explicit/default/required values and complete referenced type/schema closure.
- [ ] The smallest owning conversion is repaired; complete live/native/core outputs agree without type/name/position erasure or regenerated historical goldens. Any historical semantic correction is explicit and precedes the fn-165 preservation baseline.
- [ ] Exact names, ordered Params, selected witnesses, non-enumeration, omission/refusal, evaluation timing and affected helper/replay controls are preserved. Focused tests and two-JVM determinism pass with fresh isolated artifacts.
- [ ] The receipt identifies the committed fn-165 baseline contract and explicitly defers missing generic consumer activation, full producer switches, reader retirement, system/Go/backend acceptance and broad batch gates. No full-spec closure is claimed.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: