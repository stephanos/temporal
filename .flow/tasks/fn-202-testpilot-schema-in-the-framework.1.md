---
satisfies: [R1]
---
# fn-202-testpilot-schema-in-the-framework.1 Move the Testpilot protos under model/framework/proto; generate Go and Scala from there

## Description
Move the eleven proto files, add the framework path to the protoc proto path and directory list, generate Go into the existing output package and Scala into api-scalapb.jar, and point the protocol comment check at the new paths.

**Files:** proto/internal/temporal/server/api/testpilot/v1/*.proto -> model/framework/proto/testpilot/v1/, Makefile (PROTO_FILES, PROTO_DIRS, TESTPILOT_PROTOCOL_PROTOS, umpire-check-testpilot-protocol), the API jar generation recipe.

**Quick commands:** `make protoc`; `make umpire-check-testpilot-protocol`; `make gen-model-api`

## Acceptance
- Bindings generate from the new path; comment check passes.
- `go build ./...` passes.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: