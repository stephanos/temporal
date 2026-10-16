---
satisfies: [R2, R7]
---
# fn-173-reachable-exploration-drop-full-tables.14 Native Voting and ReliableBroadcast with explicit guard and channel deltas

Touches: [model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Continue the source-tail contract, consuming the prior retained six-sample bundle, shared Expect fixtures and inventory convention. Full R2/R7 engine and graph proofs remain held.

**Size:** M.
**Files:** model/framework/samples/Voting.scala, ReliableBroadcast.scala, Expect.test.scala, a focused NativeVoting.test.scala; retained migration maps/inventory and local guides.

### Approach
- Follow approved Voting native state in design section 8.3: chosen-value set, acceptor-to-vote sets and native ballot map with the documented None sentinel. Use final finite record-field input domains; preserve quorums, action/Property identity and negative controls.
- At pinned bac173188b, inv guards increaseMaxBal (Voting.scala:173) and normal voteFor (:182). Remove those enabling restrictions as approved, retaining the invariant Property. voteUnsafe already lacks inv; do not invent a removal there. Record the inherited increaseMaxBal change for UnsafeVoting separately from its existing unsafe vote.
- Pin real input-guard refusals, ballot progression, quorum behavior, selected refinement projections and a two-value UnsafeVoting consistency failure independently. Selected projections do not prove refinement validity or all-row equivalence.
- Remove ReliableBroadcast's manual whole-state and catalog-local Inbox Finite assembly (pinned :43-47), retaining Inbox fields, exact channel ownership/policies, message input order, faults and native effects. Never put Inbox inside a Map/collection. Preserve relay-before-delivery, crash and naive controls.
- Extend the durable maps and expectation inventory from the prior task, preserving old source fixtures unchanged. Record structural unordered-channel ordering versus old catalog order and Voting's intended guard delta; native smoke does not prove arbitrary old/new transition equality or all Query answers.

### Investigation targets
Required: final sample/domain/guard documentation; model/design/reachable-exploration.md section 8.3 and Phase 5; samples/Voting.scala, ReliableBroadcast.scala, Expect.test.scala; prior retained migration bundle. Refresh after the previous source task.

### Verification
Run focused native sample/expect suites with nonzero counts, strict warnings, formatting and relevant lint. No graph, lift-and-Go Samples harness, whole-state tables, exporter/schema/Go/Temporal or regeneration.

## Acceptance
- [ ] Voting maps/sets and finite ballot inputs retain declared identities; the exact approved inv removals, unsafe-control behavior and invariant Property are independently pinned.
- [ ] Broadcast admits structural state without manual whole-state/Inbox catalogs while preserving channel ownership, capacity, duplicates, order and native crash/relay/naive controls.
- [ ] Existing Expect cases and all intended Query/monitor/progress expectations remain inventoried; state maps record guard/channel deltas without an isomorphism claim. Prior retained fixtures remain unchanged.
- [ ] Focused native gates pass; complete R2/R7 graph, refinement and checked-answer obligations stay open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
