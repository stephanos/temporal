package framework.dsl

import framework.dsl.Finite.given

object BankControls:
  import samples.Bank.*

  val _ = () => summon[CanEqual[Outcome, Outcome]]

  object effects:
    // A send that credits the receiver without debiting the sender: the coins it pays come from
    // nowhere, and the total supply grows.
    def sendFromNowhere(in: Send) =
      val toBalance = balanceOf(Model.balances, in.toAddr, in.denom)
      if in.fromAddr == in.toAddr then ()
      else if toBalance + in.amount > supply then Model.stateVars.get.refuse(Outcome.overflow)
      else Model.balances = withBalance(Model.balances, in.toAddr, in.denom, toBalance + in.amount)

  // A send that pays from nowhere: the negative control of the total supply, which its test asserts
  // refuted.
  object Leaky
      extends Derived(
        Model.rebind(
          on(client.send): s =>
            effects.sendFromNowhere(s)
        )
      )

object TwoPhaseCommitControls:
  import samples.TwoPhaseCommit.*

  object states:
    // the transaction manager has heard from any one resource manager
    def eagerlyCommits: Boolean =
      Model.tmState == TransactionManagerState.init && any(Model.tmPrepared)

  // A transaction manager that commits once any one resource manager has prepared: the negative
  // control of consistency, which its test asserts refuted.
  object EagerCommit
      extends Derived(
        Model.rebind(
          on(tm.decideOnCommit):
            if states.eagerlyCommits then Model.effects.decideOnCommit()
        )
      )

object ReliableBroadcastControls:
  import samples.ReliableBroadcast.*

  object effects:
    // The naive algorithm: deliver m on receiving it, forwarding nothing.
    def receiveNaive(msg: Msg) =
      if localOf(Model.current, msg.receiver) == Local.idle then
        msg.receiver match
          case Proc.p1 => Model.p1 = Local.delivered(msg.payload)
          case Proc.p2 => Model.p2 = Local.delivered(msg.payload)

  // The naive algorithm, which delivers a message on receiving it and forwards nothing: the negative
  // control of agreement, which the sample tests refute.
  object Naive
      extends Derived(
        Model.rebind(
          on(Model.monitors.network.deliver): msg =>
            effects.receiveNaive(msg)
        )
      ),
        NegativeControl:

    object invariants:
      // The naive algorithm delivers only what was broadcast too: what refutes it is the agreement
      // monitor, which a run that stops with one process delivered and the other not breaks.
      val naiveSafety = check(onlyBroadcast(Model.current))

object VotingControls:
  import samples.Voting.*

  object effects:
    // A vote that skips the quorum's showing that v is safe at b.
    def voteUnsafe(in: VoteFor) =
      if maxBalAtMost(Model.current, in.a, in.b) && didNotVoteAt(Model.current, in.a, in.b) &&
        othersAgree(Model.current, in.b, in.v)
      then
        val next = cast(Model.current, in.a, in.b, in.v)
        Model.votes = next.votes
        Model.maxBal = next.maxBal
      else Model.stateVars.get.refuse(Outcome.refused)

  // A vote that skips the quorum's showing: the negative control of consistency, which the sample
  // tests refute.
  object UnsafeVoting
      extends Derived(
        Model.unmonitored.rebind(
          on(voter.voteFor): in =>
            effects.voteUnsafe(in)
        )
      ),
        NegativeControl:
    import Model.current

    object invariants:
      val unsafeConsistency = check(!(isChosen(current, Value.v1) && isChosen(current, Value.v2)))

object TwoLayeredCacheControls:
  import samples.TwoLayeredCache.*

  object effects:
    // A buggy write that stores its value in layer 1 only.
    def writeL1Only() =
      Model.l1 = Some(Model.num)
      Model.num = Model.num + 1
      Model.stateVars.get.add(Seq(Event.wrote))

  // A negative control: writes that store their value in layer 1 only. The readAfterWrite monitor
  // it keeps from Model is refuted, in the tests.
  object WriteThroughL1
      extends Derived(
        Model.rebind(
          on(client.write):
            if Model.num < 3 then effects.writeL1Only()
        )
      ):

    object postconditions:
      // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
      // monitor, which a read of a value older than the last write breaks.
      val buggyWritesRecorded = after(client.write):
        records(Event.wrote)

object KettleControls:
  import samples.Kettle.*

  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Model.init.copy(temperature = -40)

  // the temperature in the kettle in Celsius, whether it is heating and whether it is beeping: room
  // temperature (20C), the element off and quiet. The temperature is an unbounded host Int; a signed
  // bounded reading would need a domain the DSL does not have yet.
  val room = Model.init.copy(temperature = 20)

  // the kettle from room temperature, and from the coldest start
  val fromRoomTemperature = starts(temperature = 20)
  val fromAnyTemperature = starts(temperature = -40)
