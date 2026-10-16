---
satisfies: [R7]
---
# fn-173-reachable-exploration-drop-full-tables.15 Native cache history and signed Kettle temperatures

Touches: [model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Complete the six-sample source bundle using the parent's explicit cache-history contract, consuming prior fixture/map/expectation and shared Expect updates.

**Size:** M.
**Files:** model/framework/samples/TwoLayeredCache.scala, Kettle.scala, Expect.test.scala, a focused native history/temperature test; durable migration maps/inventory and local guides.

### Approach
- Convert Kettle's offset UpTo reading to native signed Celsius using old reading + floor. Preserve heating/beeping semantics, source identities, negative controls and independent -40/20/100 boundary cases (pinned Kettle.scala:22-30,45-47,59-62,78-80).
- Replace Cache Seen monitor state with an immutable ordered history of observed cache events and their Option[Int] values, following the parent contract. Keep the existing protocol guards/bounds, facts, actions and end condition; no new silent-expiry log event or history cap. Repeated reads/writes stay represented.
- Each passive monitor retains its own history, using one shared pure history-update helper and its own old verdict logic expressed over that history. Preserve after-every-step observation; do not add a global mutable log or a new monitor/checking interface.
- Independently pin miss/L1-hit/L2-fill/write/expiry sequences, repeated identical reads, successive writes and the L1-only stale-read mutant. Preserve readAfterWrite/writeAfterWrite identities and refutesWriteThroughL1's monitor-driven intended failure even though buggyWritesRecorded remains true.
- Complete the durable six-sample mapping and intended-answer inventory without editing the retained old sources. Explicitly record history-to-summary as many-to-one, monitor-product scope and unresolved R7 comparison proof; a projection/example cannot satisfy isomorphism.

### Investigation targets
Required: parent Native source-tail/cache contract; model/design/reachable-exploration.md section 4.7 and Phase 5; samples/TwoLayeredCache.scala, Kettle.scala, Expect.test.scala; dsl/Monitor.scala and prior migration bundle. If implementation would require a new semantic DSL interface, return for re-scoping, not an implicit sample feature.

### Verification
Run native callback/history/Property cases with nonzero counts under strict flags, scoped format/lint. No witness search, state enumeration, Scala explorer, lift-and-Go Samples harness, IR/schema/Go/Temporal or regeneration.

## Acceptance
- [ ] Signed Kettle boundary/heating/beeping and negative-control cases preserve the explicit old-to-new temperature map.
- [ ] Cache history records ordered payload-bearing observations, preserves repetitions and existing protocol/monitor timing, and independently detects the stale-read control without a summary surrogate or new history bound.
- [ ] All six retained sources/maps/Query expectation inventories are complete; Cache's non-bijection and Voting's guard delta remain explicit unresolved graph-proof work.
- [ ] Focused native checks pass without whole-state enumeration or search; checked-int64, universal Query answers and full R7 proof remain held.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
