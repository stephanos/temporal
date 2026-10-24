---
satisfies: [R2]
---
# fn-201-the-ir-is-internal-the-lift-module-no.2 Delete checked-in IR, lift-and-settle, equivalence dumper and the Go sample check

## Description
Remove model/ir, every generated/ir.json, the lift-and-settle steps of the model gate, check/equivalence InterpreterTables and the external dumper, and the Go program embedded in check/Samples.scala. Rename the surviving gate steps under the framework check.

**Files:** model/ir/, model/**/generated/ir.json, model/framework/check/Samples.scala, model/framework/check/equivalence/, model/framework/check/ChangeCheck.scala, Makefile (umpire-check-model, umpire-capture-baseline, umpire-check-equivalence).

**Quick commands:** `make check-framework`

## Acceptance
- No Make target reads or writes model/ir.
- Framework check passes without the deleted steps.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: