---
satisfies: [R4]
---
# fn-191-samples-follow-the-machine-sections-and.3 Samples model-check stage in checkFramework: lift the GoHeld hold and compare answers with the inventory

## Description
**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after fn-190 phase 1 (fn-190.7) against the final DSL, batched with the other IR/plugin work; one full framework gate at the end. The samples model-check stage needs the lifted IR, fn-173.4's native-collection schema and the owner's release of the Go hold; it lands with fn-190.10.

Lift the hold on the two `GoHeld` cases in model/framework/check/Samples.test.scala (reason today: "runs the Go toolchain, held in the framework-only loop") and run the same path as a stage of checkFramework after the samples lint: lift the six samples, build and run the generated Go checker, compare every query answer and progress verdict with samples/testdata/native-migration/inventory.md, print `PASS samples check: <n> queries, <m> progress claims` or one `FAIL samples check: <sample>.<query>: expected <answer>, got <answer>` per difference, and fail the check on any difference, on a Go build or run failure (with its output) or on a sample with no queries. Confirm first that the hold is only about the Go toolchain and nothing Temporal-shaped is reached by the generated checker; record the confirmation in the task receipt. Add the Kettle `boils` claim's intended answer (Verified) to the inventory once fn-191.1 adds the claim. Document the stage in model/README.md's framework-only loop section.

**Size:** M

### Verification
- checkFramework passes end to end with the new stage and prints the PASS line with the current counts.
- A deliberately flipped inventory answer fails the stage with the FAIL line naming it; restored before completion.
## Acceptance
- [ ] TBD

## Done summary
Superseded by fn-201 (one-language series, 2026-10-18): no Go sample check; native sample suites are the check

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: