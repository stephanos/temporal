---
satisfies: [R3]
---
# fn-191-samples-follow-the-machine-sections-and.2 Samples lint stage in checkFramework

Touches: model/framework/check/FrameworkLoop.scala, model/framework/check/FrameworkLoop.test.scala, model/framework/lint/Structure.scala, model/framework/lint/Order.scala, model/README.md

## Description
Add a stage to checkFramework between the plugin build and the test modules: compile model/framework/samples against the fresh framework jar with the plugin into a scratch output, index the TASTy with the inspector, run the structure and order lints over it with the feature-folder rules exempt (an exemption switch, not a rule change), print `PASS samples lint: <n> files` or one `FAIL samples lint: <file>:<line>: <text>` per finding, and fail the check on any finding, on a compile failure (with the compiler output) or on zero indexed files. Focused test: a scratch sample with a misnamed section and one with sections out of order fail with the lint's text; the six samples pass. Document the stage in model/README.md's framework-only loop section.

**Size:** M

### Verification
- checkFramework passes end to end with the new stage and prints the PASS line.
- The focused FrameworkLoop test covers both failing scratch samples and the passing six.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
