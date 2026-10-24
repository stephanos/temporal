---
satisfies: [R3]
---
# fn-191-samples-follow-the-machine-sections-and.2 Samples lint stage in checkFramework

Touches: model/framework/check/FrameworkLoop.scala, model/framework/check/FrameworkLoop.test.scala, model/framework/lint/Structure.scala, model/framework/lint/Order.scala, model/README.md

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after fn-190 phase 1 (fn-190.7) against the final DSL, batched with the other IR/plugin work; one full framework gate at the end. The samples lint stage reads the samples through the lint/inspector and lands with the final sections and the plugin.

Add a stage to checkFramework between the plugin build and the test modules: compile model/framework/samples against the fresh framework jar with the plugin into a scratch output, index the TASTy with the inspector, run the structure and order lints over it with the feature-folder rules exempt (an exemption switch, not a rule change), print `PASS samples lint: <n> files` or one `FAIL samples lint: <file>:<line>: <text>` per finding, and fail the check on any finding, on a compile failure (with the compiler output) or on zero indexed files. Focused test: a scratch sample with a misnamed section and one with sections out of order fail with the lint's text; the six samples pass. Document the stage in model/README.md's framework-only loop section.

**Size:** M

### Verification
- checkFramework passes end to end with the new stage and prints the PASS line.
- The focused FrameworkLoop test covers both failing scratch samples and the passing six.
## Acceptance
- [ ] The samples lint stage runs after plugin build and before test modules with the explicit feature-folder exemption, without weakening lint rules.
- [ ] The six samples pass with a nonzero indexed-file count and deterministic PASS summary.
- [ ] Misnamed/out-of-order scratch samples, compilation failure and zero indexed files fail with actionable diagnostics.
- [ ] Document the stage and record focused plus end-of-phase gate results only after explicit resumption and path revalidation.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: