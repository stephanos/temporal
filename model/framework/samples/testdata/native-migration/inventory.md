# Six-sample state maps and intended answers

Baseline and retained source paths are fixed by [the fixture guide](README.md). Answer labels
below preserve the samples' declarations and the existing sample check's expectations: ordinary
verify is Verified, find is Found, `refutes...` verify is Counterexample; declared Progress is
expected Verified separately. No answer was recomputed by this native source checkpoint.
All scenarios listed below are free schedules. Exact selectors, action/input order, initial
values, ends, facts, monitors and associations are additionally retained in the complete texts.

## Bank (.13 representation map complete)

Map every old `Balances` by reading the four named address fields and both `Coins` fields.
For each address, include a denomination exactly when its old amount is nonzero; include the
address exactly when its denomination map is nonempty. Native state is
`Balances(of: Map[Addr, Map[Denom, Int]])`. The inverse on this image reads absent accounts
and denominations as zero. This is an exact representation map on the old 0..2 field domain,
not a statement that every native Int state has an old preimage.

`withBalance` removes zero denomination entries, then empty accounts; missing zero writes
remain absent. Native state adds Int values outside that old domain, with no invented bound.
The initial image is king -> {banana -> 2, burger -> 2}, with no other accounts.
The normal send preserves refusal for empty/insufficient senders before its self-send shortcut,
then receiver overflow before debit/credit. Funded zero is accepted unchanged; empty zero,
including self, is refused unchanged. A self-send above balance is refused. Leaky independently
keeps self first, overflow second, credit without debit third. These are preserved behaviors,
not the older design illustration's proposed negative amounts or zero refusal.

The input map changes `Send.amount: UpTo[3]` to Int with explicit ascending domain 0..3;
field order remains fromAddr, toAddr, denom, amount (amount fastest, 128 classes). Addr and
Denom declaration order stays fixed. No global Finite[Int] or state catalog is added.
Model/Leaky, outcome names, facts=Nothing, end=true, default reach and source action identity
are retained. All Properties use final expect blocks.

| Query | Form, Property and owner | Scenario/start | Limits | Intended answer |
| --- | --- | --- | --- | --- |
| totalSupplyHolds | verify totalSupplyInv / Model | anySend / Model.init | short (4,4,65536) | Verified |
| failedSendChangesNothingHolds | verify failedSendChangesNothing / Model; across | anySend / Model.init | short | Verified |
| kingSendsPeachesBananas | find peachesGetsBananas / Model; exact king->peaches banana1 | anySend / Model.init | short | Found |
| refutesLeakyTotalSupply | verify leakyTotalSupplyInv / Leaky NegativeControl | anyLeakySend / Model.init | short | Counterexample |

## TwoPhaseCommit (.13 representation map complete)

Map old States.rm1/rm2/rm3 to the corresponding three keys in
`Map[ResourceManager,ResourceManagerState]`, including working values. Keep all three keys;
lookup uses the partial map operation and never silently supplies working for a missing key.
Map each old Members record to the subset of managers with true fields. Apply that map
independently to Transaction.tmPrepared and Messages.prepared. Map Messages.abort/commit
to Message.abort/commit members, and each true prepared bit to Message.prepared(rm).
tmState is unchanged. The inverse on the exact-three-key image reads memberships back into
the old bits and each map key back into its named field.

Messages is an ever-sent Set[Message], with no delivery consumption, loss or Inbox. Preparing
updates one manager and appends its prepared message; receiving updates only tmPrepared.
Repeated receipt is accepted and idempotent while TM is init. Decisions append their message
without removing old messages. Normal commit needs all three received managers; EagerCommit
needs any received manager. All working/abort-message and commit-message guards remain as
authored, including aborting a prepared/committed manager after an abort message and committing
a noncommitted manager without an added prepared guard. Initial state has three working managers,
TM init and two empty sets. End remains TM decided and all three managers committed or aborted.

Scalar prepare/abort/commit/addPrepared inputs keep manager declaration order rm1,rm2,rm3;
TM decisions keep no inputs. Existing per-manager wrapper/rule names and exact action identities,
outcome accepted, facts=Nothing, default reach and composition TODO scope are retained.

| Query | Form, Property and owner | Scenario/start | Limits | Intended answer |
| --- | --- | --- | --- | --- |
| consistencyHolds | verify consistencyInv / Model | anyRun / Model.init | transaction (10,10,1048576) | Verified |
| successTest | find everyoneCommits / Model; exact rm.commit(rm3) | anyRun / Model.init | transaction | Found |
| refutesEagerCommit | verify eagerConsistencyInv / EagerCommit NegativeControl | anyEagerRun / Model.init | transaction | Counterexample |

`fairCommits = assume.fair(rm.commit)` and `commitReachesEveryone` remain owned by Model:
from TM committed to allCommitted, within 3 under fairCommits; intended Progress answer Verified.
This native checkpoint pins concrete successful and conflicting paths, not fairness evaluation.

## Remaining pairs (retained declarations; state-map work belongs to .14/.15)

The following inventory begins the shared proof input; it does not claim the later maps complete.
Voting/ReliableBroadcast maps and actual guard deltas are owned by .14. Cache/Kettle maps and
history distinctions are owned by .15. The original .6 must reconcile graph scope and every
finite Query bound before claiming R7. In particular, cache history separates previously merged
monitor states: no blanket product-graph bijection is implied by a projection.

| Sample | Query | Form / Property owner | Scenario/start; Limits | Intended answer |
| --- | --- | --- | --- | --- |
| Voting | chosenStaysInConsensus | verify chosenStays / Consensus | anyDecision / Consensus.init; everyBallot (12,12,4194304) | Verified |
| Voting | chosenStaysInVoting | verify chosenStays / Consensus through Model refinement | anyVote / Model.init; everyBallot | Verified |
| Voting | consistencyHolds | verify consistency / Model | anyVote / Model.init; everyBallot | Verified |
| Voting | invariantHolds | verify invariant / Model | anyVote / Model.init; everyBallot | Verified |
| Voting | valueChosen | find decides / Model | anyVote / Model.init; everyBallot | Found |
| Voting | refutesUnsafeVoting | verify unsafeConsistency / UnsafeVoting NegativeControl | anyUnsafeVote / Model.init; everyBallot | Counterexample |
| ReliableBroadcast | safetyHolds | verify safety / Model | anyRun / Model.init; broadcastLimits (11,11,1048576) | Verified |
| ReliableBroadcast | reliabilityHolds | verify reliability / Model; across | anyRun / Model.init; broadcastLimits | Verified |
| ReliableBroadcast | deliveredDespiteCrash | find survivesCrash / Model | anyRun / Model.init; broadcastLimits | Found |
| ReliableBroadcast | refutesNaiveAgreement | verify naiveSafety / Naive NegativeControl | anyNaiveRun / Model.init; broadcastLimits | Counterexample |
| TwoLayeredCache | readFillsL1Holds | verify readFillsL1 / Model | anyRun / Model.init; history (9,9,65536) | Verified |
| TwoLayeredCache | readAfterExpiry | find servedFromL2 / Model | anyRun / Model.init; history | Found |
| TwoLayeredCache | refutesWriteThroughL1 | verify buggyWritesRecorded / WriteThroughL1 NegativeControl | anyBuggyRun / Model.init; history | Counterexample |
| Kettle | beepsOnlyWhenOffHolds | verify beepsOnlyWhenOff / Model | fromAnyTemperature / coldest; toBoiling (144,144,65536) | Verified |
| Kettle | warmsOnlyWhenOnHolds | verify warmsOnlyWhenOn / Model; across | fromAnyTemperature / coldest; toBoiling | Verified |
| Kettle | failsOverAtBoilingHolds | verify failsOverAtBoiling / Model | fromAnyTemperature / coldest; toBoiling | Verified |
| Kettle | boilsAndBeeps | find failsOverAtBoiling / Model | fromRoomTemperature / Model.init; toBoiling | Found |
| Kettle | refutesNeverBeeps | verify neverBeeps / Model | fromRoomTemperature / Model.init; toBoiling | Counterexample |

Other retained associations: Broadcast agreement monitor is read at ends; everyoneDelivers
Progress goes from someoneDelivered to everyoneDelivered within 9. Cache readAfterWrite and
writeAfterWrite monitors observe after every step; agreement Progress goes from !agreed to
agreed within 1. Both Progress expectations are Verified. Voting keeps Consensus refinement,
chosenStays and all four native expect Properties; its forthcoming removal of actual inv guards
must be recorded as a deliberate reached-invariant premise. Kettle retains its unmodified
failover selector and neverBeeps negative Property. Exact starts, bounds and associations in
the retained sources govern any future reconciliation.
