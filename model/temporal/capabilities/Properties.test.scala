package temporal.capabilities
// Every shared capability Property has two machines with their own state types.

import umpire.{Capabilities, CapabilityKind, CapabilityOf, Machine, Property}
import umpire.outcomes.{Outcome, Rejection}
import temporal.features.activity.standalone.{product, system}
import temporal.features.nexus.standalone.system.NexusSystem

class CapabilityPropertiesTest extends munit.FunSuite:
  final case class Brought(name: String, by: Set[CapabilityKind])
  final case class Declared(machine: String, state: String, capabilities: Set[String])

  val companions = Seq(Closable, Terminable, Cancelable, Pausable, Pollable, Describable)
  val properties = Seq(
    Brought("Closable.terminalStatesAreFinal", Set(Closable)),
    Brought("Closable.closedIsRejectedUniformly", Set(Closable)),
    Brought("Terminable.terminateSettles", Set(Terminable)),
    Brought("Cancelable.cancelIsRequested", Set(Cancelable)),
    Brought("Pausable.pausedIsNotDispatched", Set(Pausable, Pollable))
  )

  def declaring[S, O, F](m: Machine[S, O, F], section: Capabilities[S, O, F]): Declared =
    val kinds = section.getClass.getMethods
      .filter(method => classOf[CapabilityOf[?, ?, ?]].isAssignableFrom(method.getReturnType))
      .map(_.invoke(section).getClass.getSimpleName)
      .toSet
    Declared(m.name, m.init.getClass.getName, kinds)

  val declared = Seq(
    declaring(product.ActivityProduct, product.ActivityProduct.capabilities),
    declaring(system.ActivityRecord, system.ActivityRecord.capabilities),
    declaring(system.TrustingActivityRecord, system.TrustingActivityRecord.capabilities),
    declaring(system.ActivitySystem, system.ActivitySystem.capabilities),
    declaring(NexusSystem, NexusSystem.capabilities)
  )

  def instantiating(p: Brought, machines: Seq[Declared]): Seq[String] =
    machines.filter(d => p.by.map(_.name).subsetOf(d.capabilities)).map(_.state).distinct

  def underInstantiated(claims: Seq[Brought], machines: Seq[Declared]): Seq[String] =
    claims.flatMap { p =>
      val states = instantiating(p, machines)
      Option.when(states.size < 2)(
        s"${p.name} is brought to ${states.size} machine(s) with their own state type " +
          s"(${states.mkString(", ")}): a capability Property needs two"
      )
    }

  test("capability companions define the Properties they bring") {
    val defined = companions.flatMap { companion =>
      companion.getClass.getDeclaredMethods
        .filter(_.getReturnType == classOf[Property[?]])
        .map(method => s"${companion.name}.${method.getName}")
    }
    assertEquals(defined.sorted, properties.map(_.name).sorted)
    assertEquals(defined.distinct.size, defined.size)
  }

  test("every capability Property has two declared machines with their own state types") {
    assertEquals(underInstantiated(properties, declared), Seq.empty)
  }

  test("every Closable site binds only its rejection, including all four derived owners") {
    val closables = Seq(
      product.ActivityProduct.capabilities.closable,
      system.ActivityRecord.capabilities.closable,
      system.TrustingActivityRecord.capabilities.closable,
      system.RecordOverQueue.capabilities.closable,
      system.RecordOverMatching.capabilities.closable,
      system.TrustingRecordOverQueue.capabilities.closable,
      system.TrustingRecordOverMatching.capabilities.closable,
      system.RecordOverLossyMatching.capabilities.closable,
      NexusSystem.capabilities.closable
    )
    assertEquals(closables.size, 9)
    for closable <- closables do
      closable match
        case product: Product => assertEquals(product.productElementNames.toSeq, Seq("rejected"))
        case _                => fail(s"$closable is no rejection binding")
  }

  test("a composed shared rejection derives the capability outcome key") {
    val rejected = system.RecordOverQueue.composedOutcome(
      system.RecordMember,
      Outcome.rejected(Rejection.notFound)
    )
    assertEquals(rejected, "activity_rejected-notFound")
    assertEquals(system.RecordOverQueue.capabilities.closable.rejected, rejected)
  }

  test("every Pausable and Pollable site binds only actions, including all four derived owners") {
    val pausableAndPollable = Seq(
      product.ActivityProduct.capabilities.pausable -> product.ActivityProduct.capabilities.pollable,
      system.ActivityRecord.capabilities.pausable -> system.ActivityRecord.capabilities.pollable,
      system.TrustingActivityRecord.capabilities.pausable ->
        system.TrustingActivityRecord.capabilities.pollable,
      system.RecordOverQueue.capabilities.pausable -> system.RecordOverQueue.capabilities.pollable,
      system.TrustingRecordOverQueue.capabilities.pausable ->
        system.TrustingRecordOverQueue.capabilities.pollable,
      system.RecordOverMatching.capabilities.pausable ->
        system.RecordOverMatching.capabilities.pollable,
      system.TrustingRecordOverMatching.capabilities.pausable ->
        system.TrustingRecordOverMatching.capabilities.pollable,
      system.RecordOverLossyMatching.capabilities.pausable ->
        system.RecordOverLossyMatching.capabilities.pollable
    )
    assertEquals(pausableAndPollable.size, 8)
    for (pausable, pollable) <- pausableAndPollable do
      pausable match
        case binding: Product =>
          assertEquals(binding.productElementNames.toSeq, Seq("pause", "unpause"))
        case _ => fail(s"$pausable is no action-only Pausable binding")
      pollable match
        case binding: Product => assertEquals(binding.productElementNames.toSeq, Seq("dispatch"))
        case _                => fail(s"$pollable is no action-only Pollable binding")
  }

  test("a Property brought to one state type fails by name; derived machines count once") {
    val record = declared.filter(_.state == classOf[system.AdmissionState].getName)
    assertEquals(record.map(_.machine), Seq("activityRecord", "trustingActivityRecord"))
    assertEquals(
      underInstantiated(properties.filter(_.by == Set(Closable)), record),
      Seq(
        "Closable.terminalStatesAreFinal is brought to 1 machine(s) with their own state type " +
          "(temporal.features.activity.standalone.system.AdmissionState): a capability Property needs two",
        "Closable.closedIsRejectedUniformly is brought to 1 machine(s) with their own state type " +
          "(temporal.features.activity.standalone.system.AdmissionState): a capability Property needs two"
      )
    )
  }

  test("Pausable's dispatch Property needs Pollable as well") {
    val dispatch = properties.find(_.name == "Pausable.pausedIsNotDispatched").get
    assertEquals(
      instantiating(dispatch, Seq(Declared("pausedOnly", "Paused", Set("Pausable")))),
      Seq.empty
    )
    assertEquals(
      instantiating(dispatch, declared),
      Seq(classOf[product.State].getName, classOf[system.AdmissionState].getName)
    )
  }
