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

## Voting (.14 representation map complete)

Map old Votes.a1/a2/a3 to exact acceptor keys in both native maps. An old Ballots
at0/at1 Some(v) contributes Vote(0,v)/Vote(1,v); None contributes nothing to that
acceptor's Set[Vote]. Keep empty vote sets present. Old maxBal None maps to -1;
Some(0/1) maps to 0/1. Required lookups remain partial: a missing acceptor is not
silently initialized. The inverse covers the old two-ballot image, not arbitrary
native Int/maps/sets. Initial maps contain all three acceptors, empty votes and -1.

Decision None maps to Set.empty and Some(v) to Set(v). Native chosen retains both
values on inconsistent supplied vote states; old chosen preferred v1 there. The
larger Decision(Set(v1,v2)) projection has no old Decision preimage. Accordingly,
decides tests v1 membership and retains its true conflicting-state case and exact
a2/0/v1 selector. This is an explicit inconsistent-state projection delta.

IncreaseMaxBal(a,b) and VoteFor(a,b,v) preserve their field order and acceptor/value
order. Native b uses explicit ascending 0..1 domains (6 and 12 classes); -1 remains
state metadata only. Quorums stay the three acceptor pairs. Exactly inv(votes) &&
is removed from normal increaseMaxBal and voteFor effect conditions. All remaining
ballot, one-vote, othersAgree and safe-quorum conditions remain, as do always rules
and unchanged-state Outcome.refused Steps. UnsafeVoting's vote already lacked inv;
it still checks othersAgree, maxBal and one vote per ballot. Its inherited normal
increaseMaxBal changes separately with that guard removal.

Supplied invariant-false tests demonstrate these local relation deltas, not normal
reachability. The Model-owned invariant Property remains; no new refinement invariant
registration is claimed. Consensus remains the refinement target, toProduct remains
typed, visibleOutcomes remains false and increaseMaxBal remains unobservable.
chosenStays remains Consensus-owned once(nonempty).keeps(chosen). All ends are true,
all scenarios remain free, default reach and everyBallot remain unchanged, with no
Progress or authored fairness assumption. The six Query rows below retain intended
answers; selected projections and callback paths prove no complete refinement.

## ReliableBroadcast (.14 representation map complete)

Keep origin, p1, p2 and network's actual Delivery(message,redeliveries) sequence and
exact wire.network association. Remove only the manual Finite[Broadcast] and its
catalog-local Finite[Inbox[Msg]] assembly. The direct Inbox field is not a collection
element. Model declares inbox(_.network, wire.network); actual Derived Naive inherits
that ownership. Public illustration controls admit a valid supplied state and refuse
a crossed Channel with the same configuration, for both owners. Attachment checks
one value, not reachability, satisfaction or an old/new graph correspondence.

The channel retains capacity4, unordered, reliable and duplicates0. Its selected
message input order remains p1/hello, p1/goodbye, p2/hello, p2/goodbye. Native unordered
storage uses structural message order rather than a custom catalog order; for this
derived Msg catalog they coincide. Repeated equal independently sent messages remain
separate Delivery entries at redeliveries0. No arbitrary custom-channel IR key parity
is established. Capacity guards, relay before local delivery and ignore-if-busy rules
remain. Ordinary network.deliver dispatch runs receive without removing any delivery;
tests label supplied post-channel states and callbacks explicitly. Eligibility,
dequeue, loss and redelivery belong to the interpreter and are not simulated here.

fault.crash remains classified crash on Model's authored originSending guard/effect.
The four sending origins map by inMemory to crashed with the matching payload;
p1/p2/network remain durable. No generated crash wrapper or budget is added. Naive
still rebinds only receive. The agreement monitor recomputes disagreement each step,
is not sticky and is readAtEnds (readAt=None, atEnds=true); an intermediate violation
is not a completed-run failure. Model/Naive Property owners, survivesCrash's exact
p2-deliver selector, scenarios/starts/end/default reach and broadcastLimits remain.
everyoneDelivers still means someoneDelivered to everyoneDelivered within9, intended
Verified, with no authored fairness assumptions. The four Query rows below retain
intended answers; native cases establish no channel graph or interpreted crash proof.

## TwoLayeredCache (.15 representation map complete)

Map old Cache.l1/l2 by preserving None and each contained 0..2 value as Int;
map num to the same 0..3 integer. The inverse applies on this old image only.
Initial state is (None,None,0); write remains enabled only for num<3, including
the effect-only WriteThroughL1 rebound. Read is always enabled; expiry requires
L1 and clears only L1 with no fact. End remains l1==l2. Normal writes update both
layers, mutant writes only L1; read keeps L1, fills it from L2, or misses in order.
Actions, accepted outcome, original Event names and default reach are unchanged.

Each monitor now owns List[Observation(event: Event, value: Option[Int])], initially
Nil. Both use the shared pure observe helper. A wrote step appends once with write
priority; otherwise one actual read/miss event appends once. Payload is after.state.l1.
Silent expiry adds nothing; repeated equal observations remain, with no history cap.
readAfterWrite scans earlier writes and retains every prior stale read, with None
older than any Some and nothing older than no write. writeAfterWrite requires each
write strictly greater than its preceding write, so equal writes and a None write
violate. Both verdicts remain after every step (readAt=None, atEnds=false), on Model
and inherited by WriteThroughL1. No global mutable history or new checking interface
is introduced. agreement remains Model's !agreed-to-agreed Progress within1, without
authored fairness assumptions; its intended answer is Verified.

The old Seen(lastWrite,broken) summary and new history are not bijective. For either
monitor, folding history to its old lastWrite/broken is a many-to-one projection:
repeated reads and different read-event kinds can erase to the same summary. Repeated
reads grow the monitor product even at unchanged machine state. The underlying machine
representation map does not establish product-graph isomorphism. Original .6/R7 must
reconcile underlying/product graph scope, finite Query bounds, erased history and
intended answers; this checkpoint claims no graph bijection or universal answer.

Literal callback paths pin miss repetitions, repeated L1 reads, L2 fill, silent expiry,
three increasing writes and both owners' fourth-write refusal. The real mutant init
path write0/expire/read has [wrote Some(0),notFound None]: readAfterWrite breaks while
buggyWritesRecorded remains true. Mutant init never fills L2. The separate older-Some
fallback example starts with explicitly supplied Cache(Some(0),Some(0),1), not a claimed
reachable mutant witness. A later good write/read cannot erase an earlier stale read.
All three Property owners/selectors, both free scenarios, three Query forms/names/order
and history Limits(9,9,65536) remain as listed below.

## Kettle (.15 representation map complete)

Map old Snapshot.reading r to native Celsius r-40, preserving heatingOn/beeping.
Old readings 0,60,140 become -40,20,100; inverse r=c+40 covers the old image -40..100.
Other native Int values have no old preimage; there is no added global bound.
temperature reads signed reading directly; heat adds one. Model.init is (20,false,false)
and coldest is (-40,false,false). Preserve heat while heatingOn && temperature<100,
failover while heatingOn && temperature>=100 (including supplied >100 controls),
press while off and depress while on, their exact beep resets, accepted/no-fact Steps
and end=true. Constants boiling100/freezing0/floor-40 and Fahrenheit conversion,
veryCold=-40 and veryHot=104 remain. All four Model-owned Properties and selectors,
both free scenarios, Limits(144,144,65536), and five Query rows below remain.
neverBeeps is the negative Property on Model; no separate NegativeControl is added.
Literal cold/room paths and supplied 99->100/failover steps establish those callbacks,
not a 140-step search or any checked-int64 result.

## Complete Query inventory

The seven Bank/TwoPhaseCommit rows above and eighteen rows below retain all 25 intended
Query answers for all six samples as shared proof input. The original .6 must reconcile graph scope and every
finite Query bound before claiming R7. In particular, cache history separates previously merged
monitor states: no blanket product-graph bijection is implied by a projection.

| Sample | Query | Form / Property owner | Scenario/start; Limits | Intended answer |
| --- | --- | --- | --- | --- |
| Voting | chosenStaysInConsensus | verify chosenStays / Consensus | anyDecision / Consensus.init; everyBallot (12,12,4194304) | Verified |
| Voting | chosenStaysInVoting | verify chosenStays / Consensus through Model refinement | anyVote / Model.init; everyBallot | Verified |
| Voting | consistencyHolds | verify consistency / Model | anyVote / Model.init; everyBallot | Verified |
| Voting | invariantHolds | verify invariant / Model | anyVote / Model.init; everyBallot | Verified |
| Voting | valueChosen | find decides / Model (fn-190.4: decides, a find target, is declared in Model.queries) | anyVote / Model.init; everyBallot | Found |
| Voting | refutesUnsafeVoting | moved (fn-190.4) to NativeVoting.test as assertRefuted of the consistency invariant on UnsafeVoting NegativeControl, a negative control being a test | Model.init; default budget | Counterexample |
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
chosenStays and all four native expect Properties; the actual inv guard removals above
are a deliberate reached-invariant premise. Kettle retains its unmodified
failover selector and neverBeeps negative Property. Exact starts, bounds and associations in
the retained sources govern any future reconciliation.

Kettle gains (fn-191) the boils Progress, owned by `HeatingOnly`, a `Derived` restriction of
Model to pressButton, heat and failover: from heatingOn to beeping within 81 (a press at room
temperature, 80 heats and the failover), with no assumption. depressButton is restricted away,
since a user who switches the kettle off before it boils would refute it; pressButton stays,
since from Model.init nothing heats until the element is on, and without it the claim would hold
vacuously. Intended Progress answer Verified.
