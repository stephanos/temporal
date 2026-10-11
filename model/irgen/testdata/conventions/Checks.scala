//> using scala 3.9.0
//> using jvm 27
//> using jar ../../../build/model-scala.jar
package fixture.conventions

import framework.*
import framework.Finite.given CanEqual[?, ?]
import temporal.features.nexus.workflow.protocol.{
  Fact as WorkflowFact,
  Phase as WorkflowPhase,
  State as WorkflowState
}

enum Kind:
  case a, b, c
final case class State(kind: Kind)
enum Outcome:
  case accepted

val _ = () => summon[CanEqual[Kind, Kind]]
val kindFinite: Finite[Kind] = summon[Finite[Kind]]
val aliased = List(classOf[WorkflowPhase], classOf[WorkflowFact], classOf[WorkflowState])
val qualified = classOf[temporal.features.nexus.standalone.protocol.State]

object Checked extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a)
  object rules extends Bindings()
  def end(s: State) = s.kind == Kind.a
  def plain(k: Kind) = k match
    case Kind.a => true
    case _      => false
  def bound(k: Kind) = k match
    case rest @ _ => rest == Kind.a
  def typedWhole(k: Kind) = k match
    case _: Kind => true
  def stateWhole(s: State) = s match
    case _ => true
  def infix(k: Kind) = k `in` (Kind.a, Kind.b)
  def chainedInfix(k: Kind) = k.in(Kind.a, Kind.b) `in` (true, false)
  def dotted(k: Kind) = k.in(Kind.a, Kind.b)
  def unrelated(s: String) = s match
    case "a" => true
    case _   => false

enum Unrelated:
  case a, b

def unrelatedEnum(s: Unrelated) = s match
  case _ => true
