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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: