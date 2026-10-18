---
satisfies: [R4]
---
# fn-202-testpilot-schema-in-the-framework.4 Rename the package to umpire.testpilot.v1 and the Go package path

## Description
**Owner decision 2026-10-18:** rename to umpire.testpilot.v1 in the same change as .2 so one fixture regeneration covers both; schedule this task with .2.

Rename the proto package and go_package, regenerate, and update every Go import mechanically; live tests pass.

**Files:** model/framework/proto/testpilot/v1/*.proto, api/testpilot/v1 -> new path, all Go importers.

**Quick commands:** `go build ./...`; `make umpire-check-live-tests` (when a deployment is available)
## Acceptance
- No reference to the old package remains; build and tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
