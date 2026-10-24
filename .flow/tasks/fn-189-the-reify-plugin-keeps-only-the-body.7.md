---
satisfies: [R2, R7, R9]
---
# fn-189-the-reify-plugin-keeps-only-the-body.7 Counts receipt after R1 to R7

Touches: model/README.md, .flow/tmp

## Description
**Recut 2026-10-18 (one-language series):** The R2/R7 full-mode proofs and the IR-equivalence obligation are proved on the in-process internal IR (fn-201): no dumper, no checked-in IR.

**Phase 3 admission:** run after fn-190.9 supplies plain-Machine var/effect lowering and the framework export fixtures are re-anchored to the final DSL. This task also seals the full-mode proofs retained by the native-source checkpoints fn-189.3 and fn-189.5. Those checkpoint closures do not satisfy the full parent R2 or R7 obligations.

Retained R2 proof: a Model with effects in its Machine and a helper in a Lifted object in another file lifts in a full compile and a warm Bloop session; changing only the helper body recompiles the dependent Model and changes its IR. Retain the exact refusal for an uncovered cross-file helper and a marked helper compiled without the plugin. Compile and lift all six samples, not only realization-mode headers or pure-helper fixtures.

Retained R7 proof: all six samples and the Facets fixture lift with the required semantics and generic role evidence; diagnostics retain the sugar name and call-site location. Check the original IR-equivalence obligation explicitly. If accepted compiler-generated strict argument Let nodes change the old shape, prove their semantic equivalence rather than regenerating a baseline and calling it unchanged. The historical DefinitionExpansion, DeclarationSugar, Sugar, Facets and ReifySamples proof obligations must be restored or discharged by equivalent final-DSL tests, not silently omitted.

Record in the task receipt and in model/README.md's plugin section: plugin phases, plugin modes, annotations in the DSL's reify vocabulary, and plugin lines outside the body-lifter files (reify/body), with the command used to count. Expected at this point: store and capture phases plus the still-present RuntimeCapture and retention phases (R8 pending), one mode, two annotations. The line count is reported, not gated. Compare against the pre-spec baseline of about 5,000 non-lifter lines.

**Size:** M

### Verification
- Full-mode R2 Model-with-effects compile, warm helper edit, dependent IR change and refusal controls pass on the final plain-Machine lifter.
- R7 full-mode samples/Facets and equivalent final-DSL historical exporter controls pass, with the original equivalence obligation accounted for.
- The counts are reproducible from the recorded command on the committed source.
## Acceptance
- [ ] Parent R2's full plain-Machine effect/incremental proof is verified, beyond the native-source checkpoint.
- [ ] Parent R7's full sample/Facets lifting, diagnostics and equivalence proofs are verified, beyond the native-source checkpoint.
- [ ] Phases, modes, annotations and non-lifter line count are recorded with their command.
- [ ] One mode and exactly two reify annotations (liftedIr, capture) remain.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: