package fixture.bindingdiscovery

import framework.dsl.*
import scala.annotation.unused

enum State:
  case idle, done
enum Outcome:
  case accepted

val named = action(Actor("user"))
val second = action(Actor("user"))
val other = action(Actor("user"))
val explicit = action("authoredName", Actor("user"))
val alias = explicit

@lifted def namedStep(s: State): List[Step[State, Outcome, Nothing]] =
  List(Step(Outcome.accepted, s))
@lifted def changedStep(@unused s: State): List[Step[State, Outcome, Nothing]] =
  List(Step(Outcome.accepted, State.done))
val functionAlias: State => List[Step[State, Outcome, Nothing]] = namedStep

object Base extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules
      extends Bindings(
        named ~> ((s: State) => List(Step(Outcome.accepted, s))),
        second ~> namedStep
      )
object Named extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(named ~> namedStep)
object Eta extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(named ~> (namedStep: State => List[Step[State, Outcome, Nothing]]))
object Forwarding extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(named ~> ((s: State) => namedStep(s)))
object ValAlias extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(named ~> functionAlias)
object Explicit extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(explicit ~> ((s: State) => List(Step(Outcome.accepted, s))))
object ActionAlias extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(alias ~> ((s: State) => List(Step(Outcome.accepted, s))))
object Replaced
    extends Derived(Base.rebind(named ~> ((_: State) => List(Step(Outcome.accepted, State.done)))))
object Added extends Derived(Base.extend(other ~> ((s: State) => List(Step(Outcome.accepted, s)))))
object Restricted extends Derived(Base.restrict(named))
object Unmonitored extends Derived(Base.unmonitored)
object Chained
    extends Derived(
      Base
        .rebind(named ~> ((_: State) => List(Step(Outcome.accepted, State.done))))
        .extend(other ~> ((s: State) => List(Step(Outcome.accepted, s))))
        .rebind(second ~> ((_: State) => List(Step(Outcome.accepted, State.idle))))
        .restrict(named, second)
    )
object DuplicateInitial extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings(named ~> namedStep, named ~> changedStep)
object DuplicateAnonymous extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules
      extends Bindings(
        named ~> ((_: State) => List(Step(Outcome.accepted, State.idle))),
        named ~> ((_: State) => List(Step(Outcome.accepted, State.done)))
      )
object DuplicateReplace extends Derived(Base.rebind(named ~> namedStep, named ~> changedStep))
object DuplicateAdd extends Derived(Base.extend(other ~> namedStep, other ~> changedStep))
object UnknownReplace extends Derived(Base.rebind(other ~> namedStep))
object AlreadyBound extends Derived(Base.extend(named ~> namedStep))
object InvalidThenRestrict extends Derived(Base.rebind(other ~> namedStep).restrict(named))
object UnchangedCopy extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings((named ~> namedStep).copy())
object ChangedCallableCopy extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings((named ~> namedStep).copy(function = changedStep))
object ChangedDeclarationCopy extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State) = true
  object rules extends Bindings((named ~> namedStep).copy(decl = other.decl))
