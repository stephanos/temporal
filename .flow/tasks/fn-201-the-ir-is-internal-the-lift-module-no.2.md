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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
