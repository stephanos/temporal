---
satisfies: [R4]
---
# fn-243-programmed-workers-the-model-issues.3 A raw executor in another language follows the program file

## Description
After .2. A raw executor in one language other than Go follows the same program file with no connection to Umpire, if a gRPC stack for that language is available offline (checked 2026-10-24: the sandbox has the Go SDK in its module cache only; no Python SDK or gRPC). If none is available, record what is missing and how the Go raw executor's tests would be reused, and close the task as recorded rather than proven.
## Acceptance
- [x] Non-Go raw executor follows the program file and passes the complete and fail scenarios, or the gap is recorded with what is missing
## R4 evidence

Recorded, not proven. The refreshed offline inventory, missing language runtimes/bindings and
Go raw/live CLI reuse points are recorded in
[the R4 gap record](../artifacts/fn-243-programmed-workers-the-model-issues/non-go-offline-gap.md).
Protoc 29.5 is available through mise; a usable non-Go gRPC stack was not found in the checked
locations. No non-Go execution passed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
