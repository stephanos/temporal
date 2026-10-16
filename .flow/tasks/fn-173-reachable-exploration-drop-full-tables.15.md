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
Cache authors now store native Int values and each passive monitor retains its own immutable ordered Event/Option[Int] history. Shared pure observation preserves write priority, repetitions and silent expiry; history verdicts preserve sticky read staleness and strict write monotonicity. Kettle stores signed Celsius directly under r-40, preserving heat<100/failover>=100, button effects and every Property/Query identity. Existing Expect truth-table cases remain, with constructors mapped to native values.

Baseline commit c6f6d4649b54f942c196ae13e198a8c49bab50a5 is persisted in .flow/tmp/base_commit and .flow/tmp/fn17315-native-baseline. Source commit 3d47b0cd5caa8759d971a5bbdc5508985ba1e570. Tier: session (jev-unavailable(no_key)). Both disjoint source children completed and joined without compiler/Git/Flow authority. No executed-model identifier is available from host metadata.

Baseline was green with SampleExpectTest6 and scoped lint RuleSuites29. New NativeHistoryTest initially failed with two runtime comparisons, actual Kettle60 versus expected20 and CacheSeen versus expectedNil, before production edits. No setup error was counted as a semantic red. The first green History run passed11. Final frozen-source suites passed NativeHistoryTest11, SampleExpectTest6, NativeMapsTest12, NativeVotingTest13, NativeDataTest9, RulesTest26 and scoped lint RuleSuites29, for106 unique tests, all terminal rc0. Changed Scala fmtcheck, git diff --check and all six fixed-baseline snapshot comparisons returned rc0. Complete commands, counts, logs and terminal session statuses are in .flow/tmp/fn17315-native-run-evidence.json; logs are retained under .flow/tmp/fn17315-*.log. All source/test writes were frozen during every compiler epoch.

Literal histories pin repeated misses/L1 reads, normal L2 fill, nine-step write-expire-read, successive writes and both owners' fourth-write refusal, real initial mutant miss, sticky failure after later good writes, separately supplied older-Some fallback, None/equal/decreasing writes and one observation with write priority. Actual selected Steps feed both monitor callbacks. Kettle controls pin -40/20 and 99->100, exact button behavior, beeping negative Property and supplied101 failover. These examples establish no search or universal answer.

The durable native-migration inventory now completes all six state maps and all25 intended Query answer rows. The six .scala.txt snapshots remain byte-identical to 53f05f9bce8434ee069ec7717212700217358bbc. Cache history-to-old-Seen is many-to-one; no monitor-product graph bijection is claimed. Voting's guard/projection and Broadcast's interpreter-owned dequeue distinctions remain explicit. Original .6/R7 graph and Query proofs, checked-int64, lifting, IR/Go/backend work and Temporal migration stay held.

flowctl classification returned rc1 FULL for unmatched model/README.md. Verification follows the owner's scoped native checkpoint exception; no held full Model/mini, catalog, exporter or regeneration gate was run. Only .15 closes. .16 and the parent remain open. Formal review and broad native gates stay deferred until after .16. No Temporal source was read, compiled or restored. MILESTONES.md remains conductor-owned.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers formal review until after .16)

stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 3d47b0cd5caa8759d971a5bbdc5508985ba1e570
- Tests: /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeHistoryTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.SampleExpectTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeMapsTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeVotingTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeDataTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.RulesTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/samples/TwoLayeredCache.scala model/framework/samples/Kettle.scala model/framework/samples/Expect.test.scala model/framework/samples/NativeHistory.test.scala, git diff --check, for sample in Bank TwoPhaseCommit Voting ReliableBroadcast TwoLayeredCache Kettle; do git show "53f05f9bce8434ee069ec7717212700217358bbc:model/framework/samples/$sample.scala" | cmp - "model/framework/samples/testdata/native-migration/$sample.scala.txt" || exit; done
- PRs: