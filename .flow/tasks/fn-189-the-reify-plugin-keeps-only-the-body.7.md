---
satisfies: [R9]
---
# fn-189-the-reify-plugin-keeps-only-the-body.7 Counts receipt after R1 to R7

Touches: model/README.md, .flow/tmp

## Description
Record in the task receipt and in model/README.md's plugin section: plugin phases, plugin modes, annotations in the DSL's reify vocabulary, and plugin lines outside the body-lifter files (reify/body), with the command used to count. Expected at this point: store and capture phases plus the still-present RuntimeCapture and retention phases (R8 pending), one mode, two annotations. The line count is reported, not gated. Compare against the pre-spec baseline of about 5,000 non-lifter lines.

**Size:** S

### Verification
- The counts are reproducible from the recorded command on the committed source.

## Acceptance
- [ ] Phases, modes, annotations and non-lifter line count are recorded with their command.
- [ ] One mode and exactly two reify annotations (liftedIr, capture) remain.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
