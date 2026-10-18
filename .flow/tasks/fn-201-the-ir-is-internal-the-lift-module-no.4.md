---
satisfies: [R4]
---
# fn-201-the-ir-is-internal-the-lift-module-no.4 Fingerprints from Lifted.canonical; record first values

## Description
Move the canonical encoder into the lift module, define `Lifted.canonical` as the byte-stable input for fingerprints and Case identities, pin it with a test, and record the first values of the samples' fingerprints for fn-203's one-time regeneration.

**Files:** model/framework/lift/Canonical.scala, tests, a receipt under model/design or .flow/artifacts.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lift --require-tests`

## Acceptance
- Byte-stability test pins the encoding.
- First values recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
