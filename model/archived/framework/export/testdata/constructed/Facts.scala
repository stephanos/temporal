package fixture.factcases

import framework.dsl.*
import framework.realize.*

enum Fact:
  case recorded(flag: Boolean)
enum ForeignFact:
  case recorded(flag: Boolean)

object Subject extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean): Boolean = s
  val evidence: PartialFunction[Fact, String] = { case Fact.recorded(_) => "recorded" }
  object rules extends Rules

val caseAlias = Fact.recorded
val recordedSource = Recorded.runEvent[com.google.protobuf.duration.Duration](
  EventKind.diagnostic,
  "controller",
  "hold",
  Operand.runKey()
)
def evidence(fact: framework.realize.Fact): TypedEvidence[com.google.protobuf.duration.Duration] =
  Evidence.runEvent("record", fact, "fixture", recordedSource, Commitment.durable)
def forwarded(fact: framework.realize.Fact): TypedEvidence[com.google.protobuf.duration.Duration] =
  evidence(fact)
val hold = Instruction.Hold("gate")
val root = Realization(
  machine = Subject,
  producer = "fixture.factcases",
  producerVersion = "1",
  roles = Vector.empty,
  correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
  scripts = Vector(script("controller", Activation.Controller)(everyCase(hold))),
  evidence = Vector(
    Evidence.runEvent("record", Fact.recorded, "fixture", recordedSource, Commitment.durable)
  )
)
val forwardedRoot = root.copy(evidence = Vector(forwarded(caseAlias)))
val foreign = root.copy(evidence = Vector(forwarded(ForeignFact.recorded)))

def metadata(machine: String): io.temporal.server.api.umpire.v1.Model =
  import io.temporal.server.api.umpire.v1 as ir
  ir.Model(
    machines = Vector(ir.Machine(name = machine, factType = "fixture.factcases.Fact")),
    types = Vector(
      ir.Type(
        name = "fixture.factcases.Fact",
        shape = ir.Type.Shape.Enum(ir.Enum(cases = Vector(ir.Case(name = "recorded"))))
      )
    )
  )

object DuringConstruction extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
  val converted = root.copy(machine = this).toIR("fixture.factcases.during", metadata(name))

object NativeProbe:
  def main(args: Array[String]): Unit =
    assert(caseAlias eq Fact.recorded)
    assert(caseAlias(true).flag)
    assert(
      root
        .toIR("fixture.factcases.root", metadata(Subject.name))
        .equals(forwardedRoot.toIR("fixture.factcases.root", metadata(Subject.name)))
    )
    val derived = Subject.restrict()
    assert(
      root
        .copy(machine = derived)
        .toIR("fixture.factcases.root", metadata(derived.name))
        .evidence
        .head
        .records == "recorded"
    )
    assert(DuringConstruction.converted.evidence.head.records == "recorded")
