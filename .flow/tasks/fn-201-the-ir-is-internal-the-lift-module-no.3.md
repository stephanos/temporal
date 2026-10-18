---
satisfies: [R3]
---
# fn-201-the-ir-is-internal-the-lift-module-no.3 IR proto under model/framework/proto; ScalaPB jar from there

## Description
Move ir.proto under the framework proto directory and generate ir-scalapb.jar from the new path; a stale jar fails naming the generation target.

**Files:** proto/internal/temporal/server/api/umpire/v1/ir.proto -> model/framework/proto/ir.proto, the jar generation (currently in the disabled Gate; move the generation recipe to model/framework/check/Tools.scala or a script), Makefile MODEL_PROTO_JARS.

**Quick commands:** `make gen-model-api`; `make check-framework`

## Acceptance
- Jar generates from the new path; framework compiles against it.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
