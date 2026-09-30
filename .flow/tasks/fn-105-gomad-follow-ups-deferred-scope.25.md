---
satisfies: [R25]
---
# fn-105-gomad-follow-ups-deferred-scope.25 D25: fix enhanced DescribeTaskQueue caching of report flags

## Description
Required production fix approved on 2026-09-30. Enhanced DescribeTaskQueue caches physical task-queue information by build ID and task-queue type without distinguishing the requested poller/stat fields. A cached poller response can leak pollers into a reachability-only response, and an incomplete cached response may omit subsequently requested fields. Make cache reuse and response construction respect request report flags using the established matching-cache patterns. Preserve the existing functional assertion and obtain a dedicated production code review; this is ordinary server work, not a Gomad-only fix.

## Acceptance
- Reproduce back-to-back requests within the cache lifetime and retain focused regression coverage for report-flag combinations and both request orders, including poller-enabled then reachability-only and initially omitted then requested fields.
- Each response omits unrequested poller/stat information and includes requested information when available, with reachability handling unchanged except as needed to honor its existing contract.
- Preserve cached-data integrity across differently shaped requests; response filtering cannot mutate shared cached values or change earlier responses. Preserve valid cache reuse, expiry, and build-ID/task-queue-type isolation.
- Preserve TestDescribeTaskQueueEnhanced_ReportFlags assertions. Verify focused matching/cache tests, the native functional case, and Gomad seeds 11 and 17 with retained commands, platform identity, and outcomes.
- Obtain a dedicated production correctness review. After verification, update qualification findings and expected target-failure dispositions attributable to this bug; preserve unrelated findings and reflect the actual remaining workload outcomes.
- A relaxed report-flag assertion, cache-expiry wait, Gomad-only seam, classification, or unresolved production behavior cannot close this required fix.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
