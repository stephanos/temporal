# Where the samples read worse than their Quint originals

Status: review, 2026-10-13, revised against the samples as they stand now. No source was changed.
Paths are relative to `model/` unless they start with `tools/`. "Ours" refers to
`framework/samples/*.scala`; "Quint" to the `*.qnt.test` file beside it. Code marked
**(proposed)** uses syntax that does not exist yet. Where it overlaps `design/bounded-collections.md`
(the "draft"), it uses the draft's spelling: `Set[K]`, `Total[K, V]`, `Total.fill`, `Total.by`,
`t(k)`, `t.updated`, `incl`, `forall`/`exists`/`sumOf`, and `where((s, i) => ...)`.

**Changes since the first version.**
- The samples no longer write `enter[...]`/`stay[...]` type arguments, so finding 5 now covers only
  the qualified `X.Outcome` and the state renames.
- ReliableBroadcast is finished. It now says that the agreement monitor refutes its negative
  control; finding 16 is updated.
- TwoLayeredCache and Voting are new. Their findings are 3, 4, 12, 17 and 22, and both appear in
  several others.
- TwoPhaseCommit's progress claim now sits at the top level, which `check/Samples.scala` lifts. The
  remark about mixed claim placement is gone.

**Method.**
- I read each pair line by line, together with:
  - SEMANTICS.md §1 and §3 to §21;
  - the DSL sources `Syntax.scala`, `Machine.scala`, `Domain.scala`, `Channel.scala` and
    `Claims.scala`;
  - the lifter's refusals in `export/Declarations.scala`, `export/Types.scala` and
    `export/Capabilities.scala`;
  - the lint in `lint/Conventions.scala`.
- In the first round I confirmed two claims by experiment, in `/tmp` copies:
  - the samples compile and lift without the explicit type arguments;
  - Scala 3.9 rejects a name that is inherited inside `object Model` and also defined in the
    enclosing sample object ("Reference to State is ambiguous"), but accepts it when the name is
    declared at package level.

**Size.** Rough lines spent on the same behaviour. The counts exclude the Quint REPL transcript,
long comment banners and Bank's unported keepers.

| Pair | Ours | Quint (ported part) | Ours has, Quint lacks | Quint has, ours lacks |
| --- | --- | --- | --- | --- |
| Kettle | 104 | 94 | 4 properties, 5 queries | `initNondet` |
| Bank | 158 | ~115 | negative control, 2 properties | negative amounts, `sendTest`, accounts, params |
| TwoPhaseCommit | 294 | 157 | negative control, bounded progress | parametric RMs, pinned `successTest` |
| ReliableBroadcast | 257 | 197 | negative control, progress, crash as fault | inductive invariant, sent log, self-send |
| TwoLayeredCache | 135 | ~110 code | negative control, bounded agreement | history log, 10 client ids, stutter |
| Voting | 234 | ~100 code | checked refinement to Consensus, negative control | unbounded ballots, quorum parameter, `ShowsSafety` |

---

## Findings

Each finding gives where it is, what is worse, the cause, a fix with a rough cost (S, M or L), and a
before/after.

### 1. Records and match ladders stand in for maps and sets

- **Where.**
  - Bank 23-57: `coinsOf`, `amountOf`, `withAmount`, `withBalance`, `sumForDenom`.
  - TwoPhaseCommit 27-85: `States`, `Members`, `Messages` and seven helpers.
  - ReliableBroadcast 53-59 and 95-107.
  - Voting 25-46 and 67-123. Its quorums are unrolled into three disjuncts each.
  - Quint uses `Addr -> Coins`, `str -> State`, `Set[str]`, `Set[Message]` and
    `Quorum.exists(Q => Q.forall(...))`.
- **Worse.**
  - About 35 lines in Bank, 50 in TwoPhaseCommit, 25 in ReliableBroadcast and 60 in Voting, each
    standing in for one operator.
  - Domain sizes are wired into code: a fifth address touches six places in Bank.
  - A ladder arm that writes the wrong field still type-checks.
- **Cause.** The DSL and lifter.
  - A state field must be an enum, a case class, `Option`, `UpTo` or `Boolean` (SEMANTICS §1).
  - Lists are refused (`export/Types.scala:112-116`).
  - There is no map access or fold in the liftable subset (SEMANTICS §20).
- **Fix.** `Set[K]` and `Total[K, V]`, as the draft proposes. Cost **L**.

Before (ours), Bank 29-33, 46-52, 55-57:
```scala
  def coinsOf(balances: Balances, addr: Addr): Coins = addr match
    case Addr.king       => balances.king
    case Addr.donkeykong => balances.donkeykong
    case Addr.peaches    => balances.peaches
    case Addr.mario      => balances.mario
  def withBalance(balances: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
    addr match
      case Addr.king       => balances.copy(king = withAmount(balances.king, denom, amount))
      // ... three more arms
  def sumForDenom(balances: Balances, denom: Denom): Int =
    balanceOf(balances, Addr.king, denom) + balanceOf(balances, Addr.donkeykong, denom) +
      balanceOf(balances, Addr.peaches, denom) + balanceOf(balances, Addr.mario, denom)
```
Before (Quint), bank 28 and 64; bankTest 55-64:
```quint
  type Coins = Denom -> Int256
    balances: Addr -> Coins,
  def sumForDenom(denom: Denom): Int256 = {
    balances.keys().fold(0, (sum, addr) => { ... sum + coins.get(denom) ... })
  }
```
After **(proposed, draft §7.2)**:
```scala
  type Coins = Total[Denom, UpTo[2]]
  final case class Balances(of: Total[Addr, Coins])
  def balanceOf(b: Balances, addr: Addr, denom: Denom): Int = b.of(addr)(denom)
  def withBalance(b: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
    b.copy(of = b.of.updated(addr, b.of(addr).updated(denom, UpTo(amount))))
  def sumForDenom(b: Balances, denom: Denom): Int = b.of.sumOf((_, coins) => coins(denom))
```
Before (ours), Voting 72-75:
```scala
  def chosenAt(votes: Votes, b: Int, v: Value): Boolean =
    chosenBy(votes, Acceptor.a1, Acceptor.a2, b, v) ||
      chosenBy(votes, Acceptor.a1, Acceptor.a3, b, v) ||
      chosenBy(votes, Acceptor.a2, Acceptor.a3, b, v)
```
Before (Quint), Voting 64-66:
```quint
def ChosenAt(b, v) = Quorum.exists( Q =>
                        Q.forall( a => VotedFor(a, b, v) )
                      )
```
After **(proposed)**:
```scala
  val quorums: Set[Set[Acceptor]] = Set(Set(a1, a2), Set(a1, a3), Set(a2, a3))
  def chosenAt(s: Votes, b: Ballot, v: Value): Boolean =
    quorums.exists(q => q.forall(a => s.votes(a)(b) == Some(v)))
```

### 2. One rule and one effect wrapper per input class

- **Where.** TwoPhaseCommit 177-188 (12 wrappers) and 209-244 (12 rules). ReliableBroadcast
  198-201 and 208-215. Quint TwoPhaseCommit has three parameterised actions (34-38, 60-84).
- **Worse.**
  - About 50 lines of copy-paste.
  - A slip such as `on(rm.abort(rm2))(where(s => mayAbort(s, rm1)) ~> effects.abortRm2)` compiles.
- **Cause.** The DSL and lifter.
  - A guard is `S => Boolean` (`Syntax.scala:515`, `585-589`).
  - `on(Class)` gives `Firing[..., EmptyTuple]`, so an effect cannot read the input
    (`Syntax.scala:758-766`).
  - An effect may never give `Nil` (`Declarations.scala:1217`).
  - A rule's effect must be a def of `effects` (`Declarations.scala:1150-1181`).
- **Fix.**
  - Input-reading guards, as in draft §3.4 and Phase 1. Cost **S-M**.
  - Until then, a lint that flags an `on(a(X))` whose guard names a different literal. Cost **S**.

Before (ours), TwoPhaseCommit 177-179 and 209-217:
```scala
      def prepareRm1(state: Transaction) = prepared(state, ResourceManager.rm1)
      def prepareRm2(state: Transaction) = prepared(state, ResourceManager.rm2)
      def prepareRm3(state: Transaction) = prepared(state, ResourceManager.rm3)
      on(rm.prepare(ResourceManager.rm1))(
        where(s => mayPrepare(s, ResourceManager.rm1)) ~> effects.prepareRm1
      )
      // ... and the same for rm2 and rm3, then abort, commit and addPrepared
```
Before (Quint), two_phase_commit 60-64 and 89-90:
```quint
  action RM::prepare(resourceManager: str): bool = all {
    RM::states.get(resourceManager) == Working,
    RM::states' = RM::states.set(resourceManager, Prepared),
    messages' = messages.union(Set(RMPrepared(resourceManager)))
  }
      nondet rm = resourceManagers.oneOf()
```
After **(proposed, draft §3.4)**:
```scala
      def prepare(s: Transaction, rm: ResourceManager) = prepared(s, rm)
    on(rm.prepare) {
      where((s, r) => mayPrepare(s, r)) ~> effects.prepare
    }
```

### 3. Voting turns enabling conditions into refusals

- **Where.** Voting 165-185 and 198-199: `always ~> effects.voteFor`, whose effect ends in
  `else reject(Outcome.refused, votes)`. Quint `IncreaseMaxBal` and `VoteFor` (178-201) are
  disabled when their conjuncts fail.
- **Worse, and less precise.**
  - The modality changes. Quint says "acceptor a cannot vote here". Ours says "acceptor a may try,
    and the system answers `refused`". Every class is enabled in every state, so the table carries
    a refusal row for each.
  - `refused` exists only to fill that role (29-30).
  - The refinement must declare refusals invisible (160).
  - A stuck state can never appear, so lint loses its `stuck-state` and `never-enabled` signals.
- **Cause.** Both.
  - The DSL forces it: the enabling conditions read the inputs `a`, `b` and `v`, a guard cannot
    (finding 2), and an effect may not give `Nil`.
  - Authoring: the author could write `3 × 2 × 2` per-class rules instead, at the cost of
    finding 2.
- **Fix.** Input-reading guards (S-M). The conditions then become guards, and the `refused`
  outcome and the `visibleOutcomes` line disappear.

Before (ours), Voting 176-185 and 199:
```scala
      def voteFor(votes: Votes, a: Acceptor, b: UpTo[1], v: Value)
          : List[Step[Votes, Voting.Outcome, Nothing]] =
        if inv(votes) && maxBalAtMost(votes, a, b) && didNotVoteAt(votes, a, b) &&
          othersAgree(votes, b, v) && someQuorumShowsSafeAt(votes, b, v)
        then enter(cast(votes, a, b, v))
        else reject(Outcome.refused, votes)
      on(voter.voteFor)(always ~> effects.voteFor)
```
Before (Quint), Voting 192-201:
```quint
action VoteFor(a, b, v) = all {
    maxBal.get(a) <= b,
    votes.get(a).forall( vt => vt._1 != b ),
    Acceptor.exclude(Set(a)).forall( c => ... ),
    Quorum.exists( Q => ShowsSafeAt(Q, b, v) ),
    votes'  = votes.setBy(a, old => old.union(Set((b, v))) ),
    maxBal' = maxBal.set(a, b),
}
```
After **(proposed)**:
```scala
  def mayVote(s: Votes, a: Acceptor, b: UpTo[1], v: Value): Boolean =
    maxBalAtMost(s, a, b) && didNotVoteAt(s, a, b) && othersAgree(s, b, v) &&
      someQuorumShowsSafeAt(s, b, v)
      def voteFor(s: Votes, a: Acceptor, b: UpTo[1], v: Value) = enter(cast(s, a, b, v))
    on(voter.voteFor) {
      where((s, a, b, v) => mayVote(s, a, b, v)) ~> effects.voteFor
    }
```

### 4. Voting guards every step with its own invariant, because refinement is checked on unreachable rows

- **Where.** Voting 125-129 (the comment and `inv`), 171 and 182 (`if inv(votes) && ...`), and 231
  (`invariantHolds`). Quint has no such conjunct. TLA+'s proof assumes the invariant instead:
  `<1>2. Inv /\ [Next]_<<votes, maxBal>> => [C!Next]_chosen` (Voting 235).
- **Worse.**
  - **The transition relation is edited to make a check pass.** Every step now depends on a
    30-line invariant, and the reader must accept a three-part argument:
    1. `inv` blocks steps only from non-`inv` states;
    2. `invariantHolds` shows no such state is reachable;
    3. so the reachable behaviour equals Quint's.
  - **The argument holds only within the query's limits.** `everyBallot` happens to cover the whole
    graph.
  - **The guard leaks into the negative control.** `UnsafeVoting`'s `increaseMaxBal` keeps it
    (202), so once the unsafe design reaches a non-`inv` state it can no longer move maxBal. The
    control explores a slightly different machine than intended.
  - **It costs time.** `inv` is evaluated on every row of every class.
- **Cause.** The DSL and the reader.
  - Refinement is defined over every row of the table, not just reachable rows: "every row result
    is carried by a product row" (SEMANTICS:741; `tools/umpire/internal/engine/refine.go:10`,
    `110`).
  - An unreachable state that has, say, two values half-chosen can step to a different `chosen`
    and break the refinement.
- **Fix.** Pick one:
  - (a) Check the refinement only on rows reachable from the starts. Reachability is already
    computed from the whole table (SEMANTICS §7) and is not bounded by `Limits`, so nothing is
    lost. Cost **M**, in Go.
  - (b) Let a refinement name the invariant it relies on. The reader then:
    1. checks the invariant on every reachable state;
    2. checks refinement rows only from states that satisfy it.

    This mirrors the TLA+ proof step. Cost **M**.

Before (ours), Voting 125-129 and 171:
```scala
  // Voting's invariant, VotesSafe and OneValuePerBallot (TypeOK is the state's type). The checks
  // read a refinement on every row of the table, reachable or not, so, as the proof of `Spec =>
  // C!Spec` does, Voting takes its steps where the invariant holds: `invariantHolds` shows every
  // reachable state is one.
  def inv(votes: Votes): Boolean = votesSafe(votes) && oneValueAt(votes, 0) && oneValueAt(votes, 1)
        if inv(votes) && maxBalAtMost(votes, a, b) && !maxBalAtLeast(votes, a, b) then
```
Before (Quint), Voting 233-235:
```quint
THEOREM Spec => C!Spec
<1>1. Inv /\ Init => C!Init
<1>2. Inv /\ [Next]_<<votes, maxBal>> => [C!Next]_chosen
```
After **(proposed, option b)**:
```scala
    object refinement extends Refinement(Consensus):
      def toProduct(votes: Votes): Decision = Decision(chosen = chosen(votes))
      // checked on every reachable state, and assumed for the rows the refinement reads
      val assuming = inv
        if maxBalAtMost(votes, a, b) && !maxBalAtLeast(votes, a, b) then
```

### 5. Qualified `Outcome` and states that cannot be called `State`

- **Where.**
  - Return types and monitor type arguments inside `Model`: Bank 94 and 119, TwoPhaseCommit 191
    and 200, ReliableBroadcast 155, TwoLayeredCache 59 and 69, Voting 160, 169, 181 and 192.
  - Every sample renames its state type: `Snapshot`, `Balances`, `Transaction`, `Broadcast`,
    `Cache`, `Votes`.
  - Outside `Model` the name is bare (TwoPhaseCommit 122).
- **Worse.**
  - Qualification appears in some places and not others, for no visible reason.
  - The samples read unlike SEMANTICS §4, where the state type is `State`.
- **Cause.** The DSL.
  - `Declares` defines `type State = S` (`Machine.scala:68`), and `Machine` defines
    `type Outcome = O` (`Machine.scala:405-406`).
  - Scala 3 reports such a name as ambiguous when it is both inherited inside `Model` and defined
    in the enclosing object.
  - `Machine.scala:64-67` says outcomes are deliberately not named "inside an object", which lines
    405-406 contradict.
- **Fix.** Pick one:
  - (a) Give each sample its own package, with its types at the top level (verified to compile).
    `check/Samples.scala` hard-codes `samples.$name$.queries` and needs to accept the new path.
    Cost **S**.
  - (b) Rename or drop `Machine`'s `Outcome`/`Fact` aliases. Cost **S-M**.

Before (ours), Bank 1, 11, 24 and 94; Voting 160:
```scala
package samples
object Bank:
  final case class Coins(banana: UpTo[2], burger: UpTo[2])
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
      val visibleOutcomes = (_: Voting.Outcome) => false
```
After **(sample rewrite, option a)**:
```scala
package samples.bank
final case class State(king: Coins, donkeykong: Coins, peaches: Coins, mario: Coins)
enum Outcome:
  case accepted, invalidCoinsOrInsufficientFunds, overflow
object Bank:
      ): List[Step[State, Outcome, Nothing]] =
```

### 6. `UpTo` starts at zero, so values are stored offset or wrapped in `Option`

- **Where.**
  - Kettle 22-30, 47 and 80 store the temperature 40 degrees up.
  - Voting 23-25 spells Quint's `maxBal = -1` as `None` and needs `maxBalAtLeast`/`maxBalAtMost`
    (57-65) to unwrap it.
  - Bank 71 cannot express Quint's negative amounts.
- **Worse.**
  - Magic numbers.
  - Since `UpTo[N] <: Int`, `state.reading >= boilingTemperature` compiles and silently means 60°C.
  - Voting spends 9 lines on a sentinel Quint writes as `-1`.
- **Cause.** The DSL.
  - `UpTo[N]` is `0..N` (`Domain.scala`).
  - The lifter reads opaque ranges only from `Finite.upTo(n)` (`export/Types.scala:199-201`).
  - The IR and Go already carry a low bound (`Types.scala:37`;
    `tools/umpire/interp/limits.go:108`).
- **Fix.** `Between[Lo, Hi]` or `Finite.range(lo, hi)`. Cost **S**.

Before (ours), Kettle 24-30 and 47; Voting 25 and 57-60:
```scala
  val floor = -40
  final case class Snapshot(reading: UpTo[140], heatingOn: Boolean, beeping: Boolean)
  def temperature(state: Snapshot): Int = state.reading + floor
    val init = Snapshot(reading = UpTo(60), heatingOn = false, beeping = false)
  final case class Ballots(at0: Option[Value], at1: Option[Value], maxBal: Option[UpTo[1]])
  def maxBalAtLeast(votes: Votes, a: Acceptor, b: Int): Boolean =
    ballotsOf(votes, a).maxBal match
      case None    => false
      case Some(m) => m >= b
```
Before (Quint), kettle 22 and 32; Voting 42 and 168:
```quint
    var temperature: int
      temperature' = 20,
var maxBal: str -> int
  maxBal' = Acceptor.mapBy(_ => -1),
```
After **(proposed)**:
```scala
  final case class Snapshot(temperature: Between[-40, 100], heatingOn: Boolean, beeping: Boolean)
    val init = Snapshot(temperature = Between(20), heatingOn = false, beeping = false)
  final case class Ballots(at0: Option[Value], at1: Option[Value], maxBal: Between[-1, 1])
  def maxBalAtLeast(s: Votes, a: Acceptor, b: Int): Boolean = ballotsOf(s, a).maxBal >= b
```

### 7. A single start replaces Quint's nondeterministic start

- **Where.** Kettle 78-80 and 94: `coldest`, with a comment arguing that it reaches every other
  start. Quint `initNondet` (87-92). ReliableBroadcast puts Quint's random initial receivers
  (100-104) into an origin process instead.
- **Worse.** The reader has to check a proof written in a comment.
- **Cause.** The DSL: one `init` (`Machine.scala:409`) and one Scenario start (`Claims.scala:71`).
- **Fix.** `scenario.startsWhere(pred)` or `startsAny(...)`. Cost **M**.

Before (ours), Kettle 78-80 and 94:
```scala
  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Snapshot(reading = UpTo(0), heatingOn = false, beeping = false)
  val fromAnyTemperature: Scenario[Snapshot] = Model.scenario.starts(coldest).free
```
Before (Quint), kettle 87-92:
```quint
    action initNondet = all {
      heatingOn' = false,
      nondet temp = oneOf((-40).to(40))
      temperature' = temp,
      beeping' = false,
    }
```
After **(proposed)**:
```scala
  val fromAnyTemperature: Scenario[Snapshot] = Model.scenario
    .startsWhere(s => s.temperature <= 40 && !s.heatingOn && !s.beeping).free
```

### 8. Every match must name every case, so Boolean tests become ladders

- **Where.** ReliableBroadcast 62-92: `broadcast`, `deliveredValue`, `relaying`, `relayed`,
  `originSending`. Quint writes `| _ => false` (67-71). TwoLayeredCache keeps Quint's
  `isWrite`/`isRead` out by using facts instead.
- **Worse.** About 30 lines in which the reader scans for the one `true`.
- **Cause.** The DSL lint R4 `enum-wildcard` (`lint/Conventions.scala:505-511`; SEMANTICS §5).
- **Fix.** Pick one:
  - Relax R4 for a Boolean-valued def whose wildcard arm is a literal.
  - Derive case tests, `is[Case]`.

  Cost **S**.

Before (ours), ReliableBroadcast 74-78:
```scala
  def relaying(local: Local): Boolean = local match
    case Local.idle         => false
    case Local.relaying(_)  => true
    case Local.relayed(_)   => false
    case Local.delivered(_) => false
```
Before (Quint), reliablebc 67-71:
```quint
  pure def send_in_progress(l) =
    match l.s {
      | SendToAll(_) => l.send_to > 0
      | _ => false
    }
```
After **(lint relaxation, or proposed `is`)**:
```scala
  def relaying(local: Local): Boolean = local match
    case Local.relaying(_) => true
    case _                 => false
  // or, with derived case tests (proposed):
  def relaying(local: Local): Boolean = local.is[Local.relaying]
```

### 9. Unreachable `stay` arms pad effects and can hide bugs

- **Where.** ReliableBroadcast 130-132, 137-139, 173-175 and 181-183. The guards at 205-215
  already exclude every one of these arms.
- **Worse.**
  - Enabledness is written twice.
  - If a guard is weakened by mistake, the action becomes an enabled no-op instead of a disabled
    pair, and lint can no longer see it.
- **Cause.** Both. A missing case is an undeclared hole (SEMANTICS §12) and effects may not give
  `Nil`. `stay` itself is the author's choice; `require` would be the honest spelling (SEMANTICS
  §5).
- **Fix.**
  - Rewrite with `require`. Cost **S**.
  - Treat a partial match in an effect as a precondition that the guard must imply. Cost **M**.

Before (ours), ReliableBroadcast 176-183:
```scala
      def crash(state: Broadcast) = state.origin match
        case Origin.sending(payload, _) =>
          enter(state.copy(origin = Origin.crashed(payload)))
        case Origin.idle       => stay(state)
        case Origin.done(_)    => stay(state)
        case Origin.crashed(_) => stay(state)
```
After **(sample rewrite with `require`, then proposed partial match)**:
```scala
      def crash(state: Broadcast) =
        require(originSending(state))
        enter(state.copy(origin = Origin.crashed(broadcast(state).get)))
      // proposed: the guard `originSending` implies the match, so the lifter adds the precondition
      def crash(state: Broadcast) = state.origin match
        case Origin.sending(payload, _) => enter(state.copy(origin = Origin.crashed(payload)))
```
(`.get` is outside today's subset. The second form is the target.)

### 10. Effects and rules are split, so each action is read in two places

- **Where.** Kettle 51-65 against 68-76, and the same pattern in every sample. Quint keeps the guard
  and the update in one `action`.
- **Worse.** In Kettle, every action is split across two blocks 15 lines apart.
- **Cause.** Mostly the DSL. The split is deliberate, so that `rebind` can swap effects
  (SEMANTICS §6), and the lifter refuses an effect that is not a def of `effects`
  (`Declarations.scala:1150-1157`).
- **Fix.** Allow an inline effect lambda, named after its action, for actions that no derivation
  rebinds. Cost **S-M**.

Before (ours), Kettle 53-54 and 69:
```scala
      def pressButton(state: Snapshot) =
        enter(state.copy(heatingOn = true, beeping = false))
      on(user.pressButton)(where(!_.heatingOn) ~> effects.pressButton)
```
Before (Quint), kettle 45-50:
```quint
    action pressButton = all {
      not(heatingOn),
      heatingOn' = true,
      beeping' = false,
      temperature' = temperature,
    }
```
After **(proposed lifter relaxation)**:
```scala
      on(user.pressButton)(where(!_.heatingOn) ~> (s => enter(s.copy(heatingOn = true, beeping = false))))
```

### 11. Outcome boilerplate and `Nothing` in samples that never reject

- **Where.** Kettle 32-35, TwoPhaseCommit 103-106, ReliableBroadcast 109-112 and TwoLayeredCache
  20-23. Each also writes `Nothing` for facts, except TwoLayeredCache.
- **Worse.** Four lines and a type argument that tell the reader nothing.
- **Cause.** Both. The DSL requires an outcome type. `framework.dsl.outcomes.Outcome` already ships
  with its `Ok` (`Syntax.scala:24-27`), so the per-sample enum is an authoring choice.
- **Fix.**
  - Use the shared outcomes. Cost **S**.
  - Add a `Machine[S, O]` alias with `F = Nothing`. Cost **S**.

Before (ours), Kettle 32-35 and 45:
```scala
  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)
  object Model extends Machine[Snapshot, Outcome, Nothing]:
```
After **(sample rewrite; `Machine[S, O]` proposed)**:
```scala
  import framework.dsl.outcomes.Outcome
  object Model extends Machine[Snapshot, Outcome]:
```

### 12. Consensus's effect takes inputs it does not read

- **Where.** Voting 5 and 143: `import scala.annotation.unused` and
  `def choose(decision: Decision, @unused a: Acceptor, @unused b: UpTo[1], v: Value)`.
- **Worse.** An import and two annotations exist only to satisfy an arity rule. A reader wonders
  why Consensus mentions acceptors at all.
- **Cause.** The DSL and lifter.
  - `~>` is typed per arity: the state alone, or the state and all of the inputs (`Syntax.scala`
    `readsOne` ... `readsSix`).
  - The lifter binds by position (`Declarations.scala:1162-1170`).
  - Consensus must bind Voting's own `voteFor` so that the refinement compares like with like.
- **Fix.** Bind inputs by name, so that an effect declares only the inputs it reads. Cost **S-M**.

Before (ours), Voting 142-147:
```scala
    object effects:
      def choose(decision: Decision, @unused a: Acceptor, @unused b: UpTo[1], v: Value) =
        enter(decision.copy(chosen = Some(v)))
    object rules extends Rules:
      on(voter.voteFor)(where(_.chosen == None) ~> effects.choose)
```
After **(proposed)**:
```scala
    object effects:
      // binds voteFor's input `v` by name; `a` and `b` are not read
      def choose(decision: Decision, v: Value) = enter(decision.copy(chosen = Some(v)))
```

### 13. `end`, `Limits` and `total` are bookkeeping done by hand

- **Where.**
  - `def end(...) = true`: Kettle 49, Bank 82, Voting 140 and 155.
  - Hand-tuned `Limits`: Kettle 96, Bank 152, TwoPhaseCommit 289, TwoLayeredCache 130, and
    Voting 226 with `search = 1 << 22`.
  - ReliableBroadcast 244-256 writes `total 4074840` four times. Quint uses CLI flags.
- **Worse.**
  - `steps` is always equal to `actions`.
  - `search` is found by trial.
  - The total is a seven-factor product that any edit breaks.
- **Cause.** Both.
  - The lifter's `countedTotal` cannot count `Inbox` contents (`export/Capabilities.scala:1230-1244`).
  - `Limits` has three fields (`Claims.scala:83-90`).
- **Fix.** Count channels in `staticTotal`; add `Limits.depth(n)` with a derived `search`. Cost
  **S**.

Before (ours), ReliableBroadcast 247 and 249:
```scala
  val broadcastLimits = Limits(steps = 11, actions = 11, search = 1 << 20)
    query("safetyHolds") verify safety in anyRun limits broadcastLimits total 4074840,
```
After **(proposed)**:
```scala
  val broadcastLimits = Limits.depth(11)
    query("safetyHolds") verify safety in anyRun limits broadcastLimits,
```

### 14. A channel's catalog given must be written in an exact form, outside the machine

- **Where.** ReliableBroadcast 38-47 (`object wire` and a `given Finite[Broadcast]`), re-exported
  at 153. Quint declares `var network: Set[NetworkMsg]` (97).
- **Worse.** Eight lines of plumbing, and two names for one channel.
- **Cause.** The DSL and lifter.
  - The lifter matches the given's block shape syntactically (`export/Types.scala:191-198`).
  - `Machine` takes `Finite[S]` in its constructor (`Machine.scala:400-403`), which pushes the
    channel outside `Model`, presumably to avoid an initialisation cycle.
- **Fix.** Derive `Finite[Inbox[M]]` from a `given Channel[M]`. Cost **S-M**.

Before (ours), ReliableBroadcast 39-47 and 153:
```scala
  object wire:
    val network: Channel[Msg] =
      channel[Msg](capacity = 4, order = Order.unordered, loss = Loss.reliable)
  object Broadcast:
    given Finite[Broadcast] =
      given Finite[Inbox[Msg]] = wire.network.contents
      new Finite[Broadcast]:
        def values: IndexedSeq[Broadcast] = Finite.derived[Broadcast].values
      val network = wire.network
```
After **(proposed)**:
```scala
  // the Inbox[Msg] field finds its catalog from this given; Broadcast derives as usual
  given network: Channel[Msg] = channel(capacity = 4, order = Order.unordered, loss = Loss.reliable)
```

### 15. A rename forced by a case clash in generated class names

- **Where.** ReliableBroadcast 114. The actor is called `sender` because an object `origin` would
  sit beside the enum `Origin` (30), and their class files collide on a case-insensitive file
  system.
- **Worse.**
  - In Quint, `sender` is a field of every message (31), so the new name misleads.
  - Nothing explains the error when it happens.
- **Cause.** The JVM and macOS. No lint under `framework/` checks for it.
- **Fix.** A lint that refuses names differing only in case. Cost **S**.

Before (ours), ReliableBroadcast 30 and 114-116:
```scala
  enum Origin:
  object sender extends Actor:
    val broadcast = action(this).input[Payload]("payload")
    val send = action(this)
```
After **(lint, then a sample rewrite)**:
```scala
  // lint: "object origin and enum Origin differ only in case; their classes collide on macOS"
  enum OriginState:
  object origin extends Actor:
```

### 16. Negative controls are refuted by monitors their query does not name

- **Where.**
  - ReliableBroadcast 222-224 and 254-256 (`refutesNaiveAgreement verify naiveSafety`), refuted by
    the `agreement` monitor at 152-157.
  - TwoLayeredCache 116-119 and 134: `buggyWritesRecorded` is true by construction, and the
    `readAfterWrite` monitor refutes it.
- **Worse.**
  - Both samples now say so in a comment, but each query names a Property that cannot fail.
  - Every `Model` query also checks the monitors silently: a violated monitor in a `verify` is a
    counterexample (`tools/umpire/internal/engine/monitor.go:125-126`).
  - ReliableBroadcast's monitor holds the negation of what its comment says (154-156).
- **Cause.** Both. The DSL has no way to name a monitor as a query's subject and no
  "holds at ends" Property form, so the author picks a trivially true Property.
- **Fix.**
  - Let `verify` take a monitor. Cost **S**.
  - Add sugar `property.atEnds(p)`. Cost **S**.

Before (ours), TwoLayeredCache 116-119 and 134:
```scala
  // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
  // monitor, which a read of a value older than the last write breaks.
  val buggyWritesRecorded: Property[Cache] =
    WriteThroughL1.property.when(client.write) holds (after => after.records(Event.wrote))
    query("refutesWriteThroughL1") verify buggyWritesRecorded in anyBuggyRun limits history
```
After **(proposed)**:
```scala
    query("refutesWriteThroughL1") verify WriteThroughL1.monitors.readAfterWrite in anyBuggyRun limits history
```

Before (ours), ReliableBroadcast 155-157:
```scala
      val agreement = monitor[Broadcast, ReliableBroadcast.Outcome, Nothing, Boolean](false)(
        (_, _, after) => !agreed(after.state)
      )(_ == true).readAtEnds
```
After **(proposed)**:
```scala
      // where a run stops, the processes delivered the same messages
      val agreement = property.atEnds(agreed)
```

### 17. TwoLayeredCache replaces the history log with facts and monitors, and drifts

- **Where.** TwoLayeredCache:
  - 16-18 (`Event` with no value);
  - 33-38 (`Seen`, and `readValue`, which recovers a read's value from layer 1 *after* the read);
  - 56-78 (two near-identical monitors);
  - 105 (`where(_.num < 3)`);
  - 107 (expiry only when layer 1 is non-empty);
  - 124-125 (`leadsTo(... within = 1)`).

  Quint:
  - 32 (`Read(int) | Write(int)`) and 58 (`history: List[HistoryEntry]`);
  - 174-178 (expiry may also leave layer 1 alone);
  - 184-189 (`stutter`) and 200 (`history.length() < 15`);
  - 214-237 (pairwise invariants over the history);
  - 245-249 (`eventually`).
- **Worse.**
  - **The read's value is reconstructed, not recorded.** A reader must check that layer 1 after
    every read equals what the read returned.
  - **Two copy-paste monitors** of 10 lines each, each with four explicit type arguments.
  - **The bound changed form.** Quint bounds the history; ours disables writes after three, a
    guard the original does not have.
  - **The claim changed.** Quint checks `eventually` agreement from the start, where `get` on two
    empty maps is undefined. Ours checks "every disagreeing state agrees within 1 step". That is
    stronger, and it holds only because, after an expiry, every enabled action happens to restore
    agreement. Without Quint's `stutter` and no-op expiry, the claim needs no fairness.
- **Cause.** Both.
  - DSL: no bounded list for the history; the draft's `ListUpTo` is a plain buffer, not a log
    with quantifiers over index pairs.
  - Authoring: the `Event` cases carry no value (a fact with fields is allowed, SEMANTICS §4
    "evidence"); the write bound; the dropped stutter.
- **Fix.**
  - Put the value in the fact. Cost **S**.
  - One parameterised monitor helper. Cost **S**.
  - Comment the write bound and the stronger agreement claim. Cost **S**.

Before (ours), TwoLayeredCache 17-18, 37-38 and 98-101:
```scala
  enum Event:
    case wrote, readL1, readL2, notFound
  def readValue(after: Cache): Option[UpTo[2]] = after.l1
      def read(cache: Cache) =
        if cache.l1 != None then enter(cache, Event.readL1)
        else if cache.l2 != None then enter(cache.copy(l1 = cache.l2), Event.readL2)
        else enter(cache, Event.notFound)
```
Before (Quint), two_layered_cache 32 and 128-134:
```quint
    type HistoryEntry = Read(int) | Write(int)
    action handleL2Found(): bool = {
        val value = l2.get(DefaultKey)
        all { writeL1(value), l2' = l2, history' = history.append(Read(value)) }
    }
```
After **(sample rewrite)**:
```scala
  enum Event:
    case wrote(value: UpTo[2])
    case read(value: Option[UpTo[2]], from: Layer)
      def read(cache: Cache) =
        if cache.l1 != None then enter(cache, Event.read(cache.l1, Layer.l1))
        else if cache.l2 != None then enter(cache.copy(l1 = cache.l2), Event.read(cache.l2, Layer.l2))
        else enter(cache, Event.read(None, Layer.none))
```

### 18. Bank drifts: negative and zero amounts, the order of the self-send check, the scale of "overflow"

- **Where.** Bank 21, 66-71 and 97-100. Quint bankTest 49; bank 142-151, 158 and 164-166.
- **Worse, and less precise.**
  - **Negative amounts are gone.** Quint deliberately sends -10..supply. A negative amount
    debits the receiver, which is the robustness case in `SendCoins`.
  - **Zero has collapsed into absence.** A zero send in Quint leaves a receiver key holding 0,
    which breaks the "no zero balances" invariant (bank 147). Ours cannot show that.
  - **The self-send check moved.** Ours stays on a self-send *before* checking overflow, because
    with the cap scaled down to `supply = 2`, Quint's order would refuse king→king 2 as an
    overflow. That conflates `MAX_INT256` with the supply, so `overflow` is unreachable in
    `Model`.
  - **Dropped:** `supply` as a variable, `accounts`, `params`, the keepers, and the runs
    (bankTest 72-115).
- **Cause.** Both: there is no signed range (finding 6) and no map with absent keys (finding 1). The
  rest is the author's choice.
- **Fix.**
  - Restore both cases once `Between` and `Total[K, Option[V]]` exist. Cost **S** after those.
  - Port `sendTest` as a pinned scenario. Cost **S**.

Before (ours), Bank 71 and 97-100:
```scala
      .input[UpTo[3]]("amount")
        if fromBalance == 0 || fromBalance < amount then
          reject(Outcome.invalidCoinsOrInsufficientFunds, balances)
        else if fromAddr == toAddr then stay(balances)
        else if toBalance + amount > supply then reject(Outcome.overflow, balances)
```
Before (Quint), bankTest 49 and 100-114:
```quint
    nondet amt = (-10).to(supply.get(denom)).oneOf()
    init
      .then(send("king", "donkeykong", Map("banana" -> 2_000)))
      .expect(and { balances.get("king").get("banana") == 8_000, ... })
      .then(send("donkeykong", "peaches", Set("burger").mapBy(d => 1_000)))
      .expect(_lastError == "invalid coins or insufficient funds")
```
After **(`Between` proposed; the pinned scenario exists today)**:
```scala
      .input[Between[-1, 3]]("amount")
  val sendTest = Model.scenario.starts(Model.init).actions(
    client.send(Addr.king, Addr.donkeykong, Denom.banana, Between(1)),
    client.send(Addr.donkeykong, Addr.peaches, Denom.burger, Between(1)))
  val donkeyHasNoBurgers: Property[Balances] =
    Model.property.when(client.send(Addr.donkeykong, Addr.peaches, Denom.burger, Between(1)))
      holds (_.outcome == Outcome.invalidCoinsOrInsufficientFunds)
    query("sendTest") find donkeyHasNoBurgers in sendTest limits short,
```

### 19. TwoPhaseCommit adds a guard, fixes the RM set and loosens its success test

- **Where.**
  - 95-97 ("once", `!rmIs(state, rm, committed)`); Quint `RM::commit` (80-84) has no such guard.
  - 18-19: three fixed RMs, against Quint's `const resourceManagers` (8, 117).
  - 271-274 and 292: a free `find`, against the pinned `successTest` (Quint 152-156).
- **Worse.**
  - The guard exists so that `commitReachesEveryone`'s `within = 3` (278-284) holds; otherwise a
    repeated commit could spend the bound. Nothing in the sample says this.
  - The free `find` shows that *some* order of 10 steps commits all three RMs, not the canonical
    schedule.
- **Cause.** Both.
  - DSL: bounded progress counts self-loops as steps (SEMANTICS §11), and `witness` needs a fact
    (`Claims.scala:189-193`).
  - Authoring: `scenario.actions(...)` exists (`Claims.scala:75`) and is not used.
- **Fix.**
  - Comment the guard. Cost **S**.
  - Pin the success test. Cost **S**.
  - A fact-less `witness(...).ends(p)`. Cost **S**.

Before (ours), TwoPhaseCommit 95-97 and 292:
```scala
  // A resource manager commits because of a Commit message, once.
  def mayCommit(state: Transaction, rm: ResourceManager): Boolean =
    state.messages.commit && !rmIs(state, rm, ResourceManagerState.committed)
    query("successTest") find everyoneCommits in anyRun limits transaction,
```
Before (Quint), two_phase_commit 80-84 and 152-156:
```quint
  action RM::commit(resourceManager: str): bool = all {
    messages.contains(Commit),
    RM::states' = RM::states.set(resourceManager, Committed),
    messages' = messages
  }
  run successTest = init
    .then(preparation("rm1")).then(preparation("rm2")).then(preparation("rm3"))
    .then(commitment)
```
After **(sample rewrite; `witness ... ends` proposed)**:
```scala
  // A resource manager commits because of a Commit message. Unlike TwoPhase.tla, it commits only
  // once, so that a repeated commit cannot spend commitReachesEveryone's three steps.
  val successTest = Model.witness(
    rm.prepare(rm1), tm.addPrepared(rm1), rm.prepare(rm2), tm.addPrepared(rm2),
    rm.prepare(rm3), tm.addPrepared(rm3), tm.decideOnCommit,
    rm.commit(rm1), rm.commit(rm2), rm.commit(rm3)).ends(allCommitted)
```

### 20. TwoPhaseCommit's message set becomes Boolean flags

- **Where.** TwoPhaseCommit 37-38, 93, 97 and 101. Quint 12-15, 22 and 36.
- **Worse.** The encoding is exact, but the TLA+ `Message` type is gone, and the reader has to
  rebuild "message set" from three fields.
- **Cause.** The DSL: no set type (finding 1).
- **Fix.** `Set[Message]`, as in draft §7.1. Covered by finding 1.

Before (ours), TwoPhaseCommit 38 and 101:
```scala
  final case class Messages(abort: Boolean, commit: Boolean, prepared: Members)
    state.tmState == TransactionManagerState.init && contains(state.messages.prepared, rm)
```
Before (Quint), two_phase_commit 12-15 and 36:
```quint
  type Message =
    | Abort
    | Commit
    | RMPrepared(str)
    messages.contains(RMPrepared(resourceManager)),
```
After **(proposed, draft §7.1)**:
```scala
  enum Message:
    case abort, commit
    case prepared(rm: ResourceManager)
    s.tmState == TransactionManagerState.init && s.msgs.contains(Message.prepared(rm))
```

### 21. ReliableBroadcast changes the message model

- **Where.**
  - 36-41 and 207: a consuming channel with no sender field.
  - 185-190: a busy receiver drops the message.
  - 160-183: the origin is an actor with a cursor and a crash fault.
  - 125-129: relays go only to `other(p)`.
  - 226-231: `reliability` becomes a transition property.

  Quint:
  - a never-consumed set with senders (30-34, 97);
  - initial receivers chosen at random (100-104);
  - each process sends to every process, itself included (86-94);
  - `IndInv` (156-196).
- **Worse, and less precise.**
  - **The subset {p2} of first receivers is unreachable**, because the origin always sends to p1
    first.
  - **`reliability` checks the code, not the protocol.** It now reads "delivered implies relayed
    before", which restates `deliverAt`'s guard. Quint's "p has sent |Proc| messages" cannot be
    stated once messages are consumed.
  - **Dead guards.** The `!isFull` guards (205, 209, 212) can never be false at capacity 4.
  - **The inductive invariant is gone**, so the result is `verified-within-limits` at 11 steps,
    not an unbounded proof.
- **Cause.** Both. The DSL has no persistent message set and no inductive mode (SEMANTICS §20,
  "Results"). The origin design, the relay targets and the dropping are the author's choices.
- **Fix.**
  - Keep a sent log, `sent: Set[NetworkMsg]`, beside the channel. Cost **M** after finding 1.
  - Let the origin send in either order. Cost **S**.
  - Add a list of deviations. Cost **S**.

Before (ours), ReliableBroadcast 226-231:
```scala
  val reliability: Property[Broadcast] = Model.property holdsAcross { (before, after) =>
    (deliveredValue(after.state.p1) == None || deliveredValue(before.p1) != None ||
      relayed(before.p1)) &&
    (deliveredValue(after.state.p2) == None || deliveredValue(before.p2) != None ||
      relayed(before.p2))
  }
```
Before (Quint), reliablebc 144-148:
```quint
  def reliability =
    Proc.forall(p =>
      Domain.forall(v =>
        system.get(p).s == Delivered(v) implies
          network.filter(m => m.sender == p).size() == Proc.size()))
```
After **(proposed, with a sent log)**:
```scala
  final case class NetworkMsg(sender: Node, receiver: Proc, payload: Payload)
  // in Broadcast: sent: Set[NetworkMsg], every message ever sent, never consumed
  def reliable(s: Broadcast): Boolean =
    Proc.values.forall(p => deliveredValue(s.local(p)) == None ||
      s.sent.count(_.sender == Node.proc(p)) == Proc.values.size)
  val reliability: Property[Broadcast] = Model.property holds (after => reliable(after.state))
```

### 22. Voting specialises Quint's generic definitions by hand

- **Where.** Voting:
  - 14-18 (fixed acceptors and values);
  - 23-25 (ballots 0 and 1, at most one vote each);
  - 71-75 (quorums as three hard-coded pairs);
  - 84-90 (`showsSafeAt` rewritten for `b ∈ {0, 1}`);
  - 103-117 (`SafeAt` only for ballot 1);
  - 119-123.

  Quint:
  - 7-18 (`Value`, `Acceptor` and `Quorum` as constants, plus `QuorumAssumption`);
  - 28 (`Ballot = Nat`);
  - 98 (`SafeAt` over `0.to(b - 1)`);
  - 139-152 (`ShowsSafeAt` and the `ShowsSafety` check).
- **Worse, and less precise.**
  - **The hand specialisation is where a port can go wrong.** `showsSafeAt`'s comment (84-86)
    summarises Quint's nested `exists`/`forall` over `-1..b-1` and must be checked against it
    case by case.
  - **`QuorumAssumption` is not stated**, so a quorum set without pairwise intersection would not
    be caught.
  - **`OneVote` is enforced by the type** (one `Option` per ballot), so it can never fail.
  - **`ShowsSafety` is dropped.** It is a checkable invariant in Quint that links the definitions.
  - **The ballot bound is fixed at 2**, and the sample does not say whether interesting Paxos
    behaviour needs 3.
- **Cause.** Both.
  - DSL: no sets of quorums, no quantifier over a range, no maps (finding 1); `UpTo` starts at
    zero (finding 6).
  - Authoring: dropping `ShowsSafety` and `QuorumAssumption`.
- **Fix.**
  - With the draft's types, write `ShowsSafeAt` generically over `catalog[Ballot]`. Cost **L**,
    via finding 1.
  - Today, add `ShowsSafety` as a Property and the quorum assumption as a checked constant.
    Cost **S**.

Before (ours), Voting 87-90:
```scala
  def showsSafeAt(votes: Votes, x: Acceptor, y: Acceptor, b: Int, v: Value): Boolean =
    maxBalAtLeast(votes, x, b) && maxBalAtLeast(votes, y, b) &&
      (b == 0 || (didNotVoteAt(votes, x, 0) && didNotVoteAt(votes, y, 0)) ||
        votedFor(votes, x, 0, v) || votedFor(votes, y, 0, v))
```
Before (Quint), Voting 139-146:
```quint
def ShowsSafeAt(Q, b, v) = and {
  Q.forall(a => maxBal.get(a) >= b),
  (-1).to(b - 1).exists(c => and {
        (c != -1) implies Q.exists(a => VotedFor(a, c, v)),
        (c + 1).to(b - 1).forall(d => Q.forall(a => DidNotVoteAt(a, d)))
    }
  )
}
```
After **(proposed: `Set`, `Total`, `Between`, a range catalog)**:
```scala
  type Ballot = UpTo[1]
  def showsSafeAt(s: Votes, q: Set[Acceptor], b: Ballot, v: Value): Boolean =
    q.forall(a => s.maxBal(a) >= b) &&
      catalog[Between[-1, 1]].exists(c => c < b &&
        (c == -1 || q.exists(a => s.votes(a)(UpTo(c)) == Some(v))) &&
        catalog[Ballot].forall(d => d <= c || d >= b || q.forall(a => s.votes(a)(d) == None)))
  // ShowsSafety, checkable today as a Property over the hand-written definitions
  val showsSafety: Property[Votes] = Model.property holds (after => !inv(after.state) ||
    quorums.forall(q => ... showsSafeAt(after.state, q, b, v) implies safeAt(after.state, b, v)))
```

### 23. Kettle keeps dead definitions, and claim parameters are asymmetric

- **Where.** Kettle 11-20: `freezingTemperature`, `fahrenheit`, `veryCold`, `veryHot` are never
  read; they belong to Quint's REPL tutorial. Kettle 85-87: `before` is a state, `after` is a step.
- **Worse.**
  - Dead code in a teaching sample.
  - `before.x` against `after.state.x` is a trap in every transition Property.
- **Cause.** Authoring (dead code); the DSL (`holdsAcross`, SEMANTICS §8).
- **Fix.**
  - Drop the dead definitions. Cost **S**.
  - Forward the state's fields on the step record. Cost **M**.

Before (ours), Kettle 85-87:
```scala
  val warmsOnlyWhenOn: Property[Snapshot] = Model.property holdsAcross { (before, after) =>
    before.heatingOn || temperature(after.state) <= temperature(before)
  }
```
After **(proposed field forwarding, plus `Between`)**:
```scala
  val warmsOnlyWhenOn: Property[Snapshot] = Model.property holdsAcross { (before, after) =>
    before.heatingOn || after.temperature <= before.temperature
  }
```

### 24. Smaller noise

- **`action(this)` on every action.** Inside `Actor`, `action` could default its owner. Cost **S**.
- **Input names are strings that repeat the effect's parameter names.** The lifter binds by
  position, so a mismatch is silent. Cost **S** for a lint. See Bank 68-71 against 89-93.
- **The sample gate infers "must be refuted" from the name prefix `refutes`.** It should use the
  `NegativeControl` marker instead. Cost **S**.

Before (ours), Bank 65-71 and 88-93:
```scala
  object client extends Actor:
    val send = action(this)
      .input[Addr]("fromAddr")
      .input[Addr]("toAddr")
      .input[Denom]("denom")
      .input[UpTo[3]]("amount")
      def send(balances: Balances, fromAddr: Addr, toAddr: Addr, denom: Denom, amount: UpTo[3])
```
After **(proposed default owner, and a lint on names)**:
```scala
  object client extends Actor:
    val send = action.input[Addr]("fromAddr").input[Addr]("toAddr")
      .input[Denom]("denom").input[UpTo[3]]("amount")
  // lint: "effects.send's parameter `to` does not name client.send's input `toAddr`"
```

---

## Where Umpire is better (counterweight)

- **A refinement that is checked.** Voting 157-161 checks `Spec => C!Spec` against a Consensus
  machine and its `chosenStays` Property. The Quint original can only state it in a comment
  (Voting 232-243). Finding 4 is the price of this.
- **Typed outcomes, and no sentinel values.**
  - Bank 60-61 against `_lastError: str`.
  - TwoLayeredCache's `Option[UpTo[2]]` against `NotFound = -98` and `Expired = -99` (Quint 44-45).
  - Quint reliablebc returns `recv: -1, payload: ""` from an "impossible" branch (93).
- **Derived negative controls, checked to fail.** `Leaky`, `EagerCommit`, `Naive`,
  `WriteThroughL1` and `UnsafeVoting` are one line each, and the marker check refuses a control
  that nothing refutes (SEMANTICS §14).
- **Bounded progress with explicit assumptions.** `commitReachesEveryone` (with fairness),
  `everyoneDelivers` and TwoLayeredCache's `agreement` each state their bound. Quint
  TwoPhaseCommit has no liveness, and reliablebc has only a comment.
- **History laws as finite monitors.** TwoLayeredCache's monitors check its two history laws in
  constant state. Quint's are quadratic over a list capped at 15.
- **Declared faults and channels, no frame conditions, and static checks.** These are the actor
  modalities, disjoint rules, strict equality, and lint for stuck, silent and never-enabled states.

---

## Top 10 DSL changes, by readability payoff per cost

Overlap with `design/bounded-collections.md` is noted for each item.

**1. Guards that read action inputs.** Cost S-M. Fixes findings 2, 3 and 12, in TwoPhaseCommit,
ReliableBroadcast and Voting. This is the same as draft §3.4 and Phase 1.
```scala
      // before: TwoPhaseCommit 209-217, three blocks like this per action
      on(rm.prepare(ResourceManager.rm1))(
        where(s => mayPrepare(s, ResourceManager.rm1)) ~> effects.prepareRm1
      )
      // after (proposed)
      on(rm.prepare) {
        where((s, r) => mayPrepare(s, r)) ~> effects.prepare
      }
```

**2. End the type-member clash.** Cost S. Use per-sample packages, or rename `Machine`'s aliases.
Fixes finding 5 in all six samples. Not in the draft.
```scala
      // before: Bank 94
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
      // after: package samples.bank, with State and Outcome at the top level
      ): List[Step[State, Outcome, Nothing]] =
```

**3. Relax R4 `enum-wildcard` for Boolean classifiers, or derive `is[Case]`.** Cost S. Fixes
finding 8 (ReliableBroadcast). Not in the draft.
```scala
  // before: ReliableBroadcast 80-84
  def relayed(local: Local): Boolean = local match
    case Local.idle         => false
    case Local.relaying(_)  => false
    case Local.relayed(_)   => true
    case Local.delivered(_) => false
  // after (proposed)
  def relayed(local: Local): Boolean = local.is[Local.relayed]
```

**4. `Between[Lo, Hi]` ranges.** Cost S. Fixes findings 6 and 18 in Kettle, Voting and Bank. The
draft's `ListUpTo` mirrors `UpTo`; ranges are a sibling it does not cover.
```scala
  // before: Kettle 27 and 30
  final case class Snapshot(reading: UpTo[140], heatingOn: Boolean, beeping: Boolean)
  def temperature(state: Snapshot): Int = state.reading + floor
  // after (proposed)
  final case class Snapshot(temperature: Between[-40, 100], heatingOn: Boolean, beeping: Boolean)
```

**5. Channel catalog from a `given Channel`, and channels counted in `staticTotal`.** Cost S-M.
Fixes findings 13 and 14 (ReliableBroadcast). The draft reuses channel catalogs and `channelSize`;
share that code.
```scala
  // before: ReliableBroadcast 43-47 and 249
  object Broadcast:
    given Finite[Broadcast] =
      given Finite[Inbox[Msg]] = wire.network.contents
      new Finite[Broadcast]:
        def values: IndexedSeq[Broadcast] = Finite.derived[Broadcast].values
    query("safetyHolds") verify safety in anyRun limits broadcastLimits total 4074840,
  // after (proposed)
  given network: Channel[Msg] = channel(capacity = 4, order = Order.unordered, loss = Loss.reliable)
    query("safetyHolds") verify safety in anyRun limits broadcastLimits,
```

**6. Refinement over reachable rows, or an `assuming` invariant.** Cost M. Fixes finding 4
(Voting). Not in the draft.
```scala
        // before: Voting 182
        if inv(votes) && maxBalAtMost(votes, a, b) && didNotVoteAt(votes, a, b) &&
        // after (proposed): the refinement reads reachable rows only, or declares `val assuming = inv`
        if maxBalAtMost(votes, a, b) && didNotVoteAt(votes, a, b) &&
```

**7. `Set[K]`, `Total[K, V]` and lifted quantifiers.** Cost L. Fixes findings 1, 18, 19, 20, 21
and 22, in Bank, TwoPhaseCommit, ReliableBroadcast and Voting. This is the draft itself.
```scala
  // before: TwoPhaseCommit 28-35 and 69
  final case class States(rm1: ResourceManagerState, rm2: ResourceManagerState, rm3: ResourceManagerState)
  final case class Members(rm1: Boolean, rm2: Boolean, rm3: Boolean)
  def all(members: Members): Boolean = members.rm1 && members.rm2 && members.rm3
  // after (proposed, draft §7.1)
    rmState: Total[ResourceManager, ResourceManagerState],
    tmPrepared: Set[ResourceManager],
  def allPrepared(s: Transaction): Boolean = ResourceManager.values.forall(s.tmPrepared.contains)
```

**8. A fact-less `witness(...).ends(p)`, and `verify` of a monitor.** Cost S. Fixes findings 16 and
19, in TwoPhaseCommit, ReliableBroadcast and TwoLayeredCache. Not in the draft.
```scala
    // before: ReliableBroadcast 254-256
    query(
      "refutesNaiveAgreement"
    ) verify naiveSafety in anyNaiveRun limits broadcastLimits total 4074840
    // after (proposed)
    query("refutesNaiveAgreement") verify Naive.monitors.agreement in anyNaiveRun limits broadcastLimits
```

**9. A partial match in an effect as a precondition the guard must imply.** Cost M. Fixes finding
9 (ReliableBroadcast). Not in the draft.
```scala
      // before: ReliableBroadcast 134-139
      def deliverAt(state: Broadcast, p: Proc): List[Step[Broadcast, Outcome, Nothing]] =
        localOf(state, p) match
          case Local.relayed(payload) => enter(withLocal(state, p, Local.delivered(payload)))
          case Local.idle             => stay(state)
          case Local.relaying(_)      => stay(state)
          case Local.delivered(_)     => stay(state)
      // after (proposed): the missing cases are a precondition, checked against `relayed`
      def deliverAt(state: Broadcast, p: Proc): List[Step[Broadcast, Outcome, Nothing]] =
        localOf(state, p) match
          case Local.relayed(payload) => enter(withLocal(state, p, Local.delivered(payload)))
```

**10. Defaults.** Cost S. Covers `Machine[S, O]`, shared outcomes, `action` without `this`,
`Limits.depth`, and inputs bound by name. Fixes findings 11, 12, 13 and 24 in all samples. Not in
the draft.
```scala
  // before: Kettle 32-38 and 96
  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)
  object user extends Actor:
    val pressButton = action(this)
  val toBoiling = Limits(steps = 144, actions = 144, search = 65536)
  // after (proposed)
  import framework.dsl.outcomes.Outcome
  object user extends Actor:
    val pressButton = action
  val toBoiling = Limits.depth(144)
```

**Just outside the top 10:**
- several starts (finding 7, M);
- inline effect lambdas (finding 10, S-M);
- a lint for names that differ only in case (finding 15, S);
- `property.atEnds` (finding 16, S);
- field forwarding on step records (finding 23, M);
- an inductive-invariant mode in Go (findings 21 and 22, L).
