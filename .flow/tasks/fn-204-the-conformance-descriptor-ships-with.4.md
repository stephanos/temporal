---
satisfies: [R4, R5]
---
# fn-204-the-conformance-descriptor-ships-with.4 Bounded passive observation and monitor-only unbounded checks

## Description
Accept an event stream from a recorded Run or an external source mapped through the evidence roles; judge it with the same interpreter; report events with no role mapping; run monitor-only checks without a machine slice, unbounded.

**Files:** common/testing/testpilot/conformance/, a new `testpilot-observe` entry in the assess command, tests.

**Quick commands:** `go test -tags test_dep ./common/testing/testpilot/...`

## Acceptance
- A recorded Run judged through the stream path equals its Run-record verdict.
- Monitor-only check on a long synthetic stream completes without the slice.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: