---
satisfies: [R4]
---
# fn-203-cases-are-produced-in-scala.4 Managed Case trees and the byte-equality gate

## Description
Generate, check and sync the model, functional and canary trees with manifest and pins; before fn-201's recut every Scala-produced Case must equal the Go-produced one byte for byte; after it, regenerate once and stop running the Go producer.

**Files:** model/framework/export/testpilot/Trees.scala, model/cases/, tests/testcore/testpilot/testdata/generated/, tools/canary/casebinding/testdata/, Makefile (umpire-gen-cases, umpire-check-cases, fixtures, canary targets).

**Quick commands:** `make umpire-check-cases` (Scala-backed)

## Acceptance
- Byte equality before the recut; one regeneration after.
- Stale pin fails naming the Case.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
