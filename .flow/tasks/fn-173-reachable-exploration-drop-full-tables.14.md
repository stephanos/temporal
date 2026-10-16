---
satisfies: [R2, R7]
---
# fn-173-reachable-exploration-drop-full-tables.14 Native Voting and ReliableBroadcast with explicit guard and channel deltas

Touches: [model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Continue the source-tail contract from committed .13 (source d45444ba6064f093ed7ad1a404a2d04350f5b820, receipt 67aaec4a49d1846a179427529e86059a0809dc2c), consuming its retained six-sample bundle, shared Expect fixtures and inventory convention. Full R2/R7 engine and graph proofs remain held.

**Size:** M.
**Files:** model/framework/samples/Voting.scala, ReliableBroadcast.scala, Expect.test.scala, a focused NativeVoting.test.scala; retained migration maps/inventory and local guides.

### Approach
- Follow approved Voting native state in design section 8.3: chosen-value set, acceptor-to-vote sets and native ballot map with the documented None sentinel. Use final finite record-field input domains; preserve quorums, action/Property identity and negative controls.
- At pinned bac173188b, inv guards increaseMaxBal (Voting.scala:173) and normal voteFor (:182). Remove those enabling restrictions as approved, retaining the invariant Property. voteUnsafe already lacks inv; do not invent a removal there. Record the inherited increaseMaxBal change for UnsafeVoting separately from its existing unsafe vote.
- Keep the remaining conditions and authored refusal results, including Outcome.refused, refinement visibleOutcomes/unobservable metadata and voteUnsafe's othersAgree check. Do not copy the proposed design snippet's refusal-to-disabled conversion or broaden the unsafe effect beyond the approved inv delta.
- The chosen-value Set retains both values on inconsistent supplied states; old Decision None/Some maps only to empty/singleton sets. Preserve decides' existing true case when both values are chosen: the old chosen helper preferred v1, so this predicate tests v1 membership, not equality to a singleton. Record the larger inconsistent-state refinement image as a delta, not arbitrary all-state equivalence.
- Pin real input-guard refusals, ballot progression, quorum behavior, selected refinement projections and a two-value UnsafeVoting consistency failure independently. Selected projections do not prove refinement validity or all-row equivalence.
- Remove ReliableBroadcast's manual whole-state and catalog-local Inbox Finite assembly (pinned :43-47), retaining Inbox fields, exact channel ownership/policies, message input order, faults and native effects. Never put Inbox inside a Map/collection. Preserve relay-before-delivery, crash and naive controls.
- Associate Model's rooted network field with the exact wire.network Channel through the final typed Machine.inbox API. Naive inherits that state-field ownership through its actual derivation. Pin valid supplied state admission and a crossed same-configuration Channel refusal through public illustration attachment; names, supplied Inbox self-consistency and callback discovery are not ownership evidence. Native receive callbacks do not dequeue Inbox deliveries; keep interpreter-owned channel eligibility/removal separate from concrete callback examples.
- Extend the durable maps and expectation inventory from the prior task, preserving old source fixtures unchanged. Record structural unordered-channel ordering versus old catalog order and Voting's intended guard delta; native smoke does not prove arbitrary old/new transition equality or all Query answers.

### Investigation targets
Required: final sample/domain/guard documentation; model/design/reachable-exploration.md section 8.3 and Phase 5; samples/Voting.scala, ReliableBroadcast.scala, Expect.test.scala; prior retained migration bundle. Refresh after the previous source task.

### Verification
Run focused native sample/expect suites with nonzero counts, strict warnings, formatting and relevant lint. No graph, lift-and-Go Samples harness, whole-state tables, exporter/schema/Go/Temporal or regeneration.

## Acceptance
- [ ] Voting maps/sets and finite ballot inputs retain declared identities; the exact approved inv removals, remaining conditions/refusal outcomes, unsafe-control behavior and invariant Property are independently pinned. Existing conflicting-state decides truth is preserved while the chosen Set retains both values.
- [ ] Broadcast admits structural state without manual whole-state/Inbox catalogs while preserving channel ownership, capacity, duplicates, order and native crash/relay/naive controls.
- [ ] Existing Expect cases and all intended Query/monitor/progress expectations remain inventoried; state maps record guard/channel deltas without an isomorphism claim. Prior retained fixtures remain unchanged.
- [ ] Focused native gates pass; complete R2/R7 graph, refinement and checked-answer obligations stay open.

## Done summary
Implemented native Voting vote-set/max-ballot maps and chosen-value sets, retaining all three acceptor keys, ballot input domains/order, refused Steps, remaining vote conditions and all sample Property/Query identities and bounds. Removed exactly the normal increaseMaxBal/voteFor inv conjuncts and recorded UnsafeVoting's inherited max-ballot change. Native chosen retains both inconsistent values while decides retains v1-membership truth. ReliableBroadcast retires only its state/Inbox catalog assembly and declares exact network field ownership inherited by actual Derived Naive. Authored crash classification/reset maps, guards/effects, relay order and end-only nonsticky agreement remain.

Baseline commit 8f80edd293c5e299b6388cc258ae985d6a4f785c. Source commit e9fdec5b0f5999e76585b35e05cac0ab9ee11155. Tier: session (jev-unavailable(no_key)); judge unavailable no_key. REVIEW_MODE:none; no review verdict. Both disjoint source children completed and joined without compiler/Git/Flow authority.

Baseline terminal rc0 counts were SampleExpectTest6, NativeMapsTest12, NativeDataTest9 and RulesTest26 (53). An initial unused-import compile failure was setup-inconclusive and corrected under unchanged Werror. Before production edits, two independent normal supplied inv-false dispatch tests failed at runtime with actual refused versus expected accepted Steps. Final terminal rc0 counts were NativeVotingTest13, SampleExpectTest6, NativeMapsTest12, NativeDataTest9, RulesTest26, IllustrationSuites23 and scoped lint RuleSuites29 (118 unique tests). The strengthened NativeVoting rerun isolates max-ballot refusal independently from prior votes. Final changed-Scala formatting check, git diff --check and all six fixed-baseline snapshot comparisons returned rc0. Exact commands, logs, rc/counts and terminal handles are in .flow/tmp/fn17314-native-final-evidence.json.

The fixture guide is model/framework/samples/testdata/native-migration/README.md and maps/deltas/all25 intended Query answers are in its inventory.md. All six old texts remain byte-identical to 53f05f9bce8434ee069ec7717212700217358bbc. Supplied inv-false states are intentional local relation deltas; selected projections and native callback examples are not reached-graph or full refinement proofs. Broadcast ordinary receive dispatch does not dequeue the Inbox; eligibility/removal/loss/redelivery remain interpreter-owned.

Gates use the authorized scoped native checkpoint. flowctl classification returned rc1 FULL (unmatched model/README.md); no broad/system gate was substituted. R2/R7 graph/Query proofs, checked-int64, collection lifting/capture, full refinement, interpreted progress/fairness, IR/Go/backend and Temporal migration remain held. No Temporal source read/compiled/restored and no schema/exporter/generated/regeneration work. Only .14 closes; .15/.16 and the parent remain open. Formal review and broad native gates remain deferred until after .16.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers formal review until after .16)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e9fdec5b0f5999e76585b35e05cac0ab9ee11155
- Tests: /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.NativeVotingTest', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SampleExpectTest', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.NativeMapsTest', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.NativeDataTest', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.RulesTest', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.*Illustration*Suite', /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeVotingTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/samples/Voting.scala model/framework/samples/ReliableBroadcast.scala model/framework/samples/Expect.test.scala model/framework/samples/NativeVoting.test.scala, git diff --check, for sample in Bank TwoPhaseCommit Voting ReliableBroadcast TwoLayeredCache Kettle; do git show "53f05f9bce8434ee069ec7717212700217358bbc:model/framework/samples/$sample.scala" | cmp - "model/framework/samples/testdata/native-migration/$sample.scala.txt" || exit; done
- PRs: