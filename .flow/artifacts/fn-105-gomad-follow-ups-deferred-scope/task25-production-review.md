# Round 1: service/matching change (reviewed diff: fn105-d25-reviewed-diff.patch)

No BLOCKER, MAJOR, MINOR, or NIT findings.

1. Cache keys: [matching_engine.go:1500](/Users/stephan/Workspace/temporal/gomad/service/matching/matching_engine.go:1500) isolates poller/stat shape, build ID, and task-queue type. Fixed prefixes prevent collisions with `dtq_default:`/`dvtq:`; build-ID punctuation is harmless because keys are not parsed and suffixes are unambiguous.
2. Aliasing: after a miss, [matching_engine.go:1535](/Users/stephan/Workspace/temporal/gomad/service/matching/matching_engine.go:1535) removes cached values before merging. Thus `append` and stats merging operate only on fresh request-local protos; no cached or earlier response is subsequently mutated.
3. Concurrency: the LRU synchronizes access. Concurrent misses may duplicate fan-out, but request-local maps prevent cross-request merging; different shapes use different keys.
4. Cardinality: up to four variants per build-ID/type is a bounded performance tradeoff. The existing 10,000-entry per-root cap and TTL remain intact.
5. Callers: false/false requests correctly receive empty physical information; reachability remains per-response. Frontend defaults and worker-deployment poller-only/stat-only callers remain compatible.
6. Compatibility: there is no wire or persisted-state change. Caches are process-local, so mixed-version nodes do not share incompatible entries.
7. Partial hits: clearing inner maps is correct because fan-out re-fetches every requested build-ID/type. Existing outer build-ID entries remain, and partition-returned IDs are added normally.
8. Tests: [matching_engine_describe_enhanced_test.go:219](/Users/stephan/Workspace/temporal/gomad/service/matching/matching_engine_describe_enhanced_test.go:219) covers all 64 ordered flag combinations, cache reuse, both request orders, prior-response/cache immutability, expiry, and build-ID/type isolation. It uses `require` and proto equality without sleeps, suites, relaxed assertions, or production seams. The functional test is unchanged.
9. No additional merge blocker found. Duplicate task-queue types and unusual `AllActive` behavior are pre-existing and outside this diff.

VERDICT: SHIP
# Round 2: test readiness wait, manifest entry, milestones bullet (reviewed delta: fn105-d25-reviewed-delta.patch). The one NIT (zero TTL clamped to 1ns) was applied after the review.

1. Test change — No finding. All assertions/messages and the 3s/500ms Await bounds remain intact; no sleep or expiry wait was added. Requiring the worker identity under both defaulted task-queue types is sufficient, native-valid, and does not mask the cache bug.

2. Manifest — No finding. Removing the failure overrides follows the generator convention; the reason accurately limits observed qualification to darwin/arm64 and explicitly says Linux was not run.

3. **NIT** — [GOMAD_MILESTONES.md:140](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:140): “the 1ns cache TTL the suite configures” is slightly inaccurate—the suite configures zero, which the server clamps to 1ns. Fix: “the suite’s zero TTL is clamped to 1ns, which never elapses…”

4. Other hunks — No findings or blockers.

VERDICT: SHIP