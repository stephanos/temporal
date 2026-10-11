package temporal.capabilities

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import org.json4s.*
import org.json4s.jackson.JsonMethods.parse
import temporal.capabilities.product.{Closable, Describable, Pausable, Pollable, Terminable}
import temporal.capabilities.system.{Expirable, Retryable}
// Every shared capability Property has two machines with their own state types.

import framework.{
  Capabilities,
  CapabilityKind,
  CapabilityOf,
  ExpirableFixture,
  Machine,
  Property,
  RetryFixture
}
import framework.outcomes.{Outcome, Rejection}
import temporal.features.activity.standalone.{product, protocol}
import temporal.features.nexus.standalone.protocol.NexusProtocol
import temporal.features.nexus.workflow.protocol.NexusProtocol as WorkflowNexusProtocol

class CapabilityPropertiesTest extends munit.FunSuite:
  private def present(value: Any): Boolean = value match
    case option: Option[?] => option.nonEmpty
    case other             => fail(s"expected optional capability binding, got $other")
  private def enabled(value: Any): Boolean = value match
    case flag: Boolean => flag
    case other         => fail(s"expected Boolean capability binding, got $other")
  final case class Brought(name: String, by: Set[CapabilityKind])
  private def only(by: Set[CapabilityKind], kind: CapabilityKind): Boolean =
    by.size == 1 && by.contains(kind)
  final case class Declared(
      machine: String,
      state: String,
      capabilities: Set[String],
      applicable: Set[String] = Set.empty
  )

  val companions =
    Seq(Closable, Terminable, Pausable, Pollable, Describable, Retryable, Expirable)
  val properties = Seq(
    Brought("Closable.terminalStatesAreFinal", Set(Closable)),
    Brought("Closable.closedIsRejectedUniformly", Set(Closable)),
    Brought("Terminable.terminateSettles", Set(Terminable)),
    Brought("Pausable.pausedIsNotDispatched", Set(Pausable, Pollable)),
    Brought("Retryable.failureReturnsToWaiting", Set(Retryable)),
    Brought("Retryable.failureEndsFailed", Set(Retryable)),
    Brought("Retryable.failurePauses", Set(Retryable)),
    Brought("Retryable.failureCancels", Set(Retryable)),
    Brought("Retryable.attemptCountIsWithinPolicy", Set(Retryable)),
    Brought("Expirable.firesInWindow", Set(Expirable)),
    Brought("Expirable.deadlineTimesOut", Set(Expirable)),
    Brought("Expirable.deadlineReturnsToWaiting", Set(Expirable)),
    Brought("Expirable.deadlinePauses", Set(Expirable))
  )

  private val expectedLawNames = properties.map(_.name).toSet

  private def requireComplete(native: Set[String], lifted: Set[String]): Unit =
    assertEquals(native, expectedLawNames, "native capability companion scan")
    assertEquals(lifted, expectedLawNames, "lifted capability-origin scan")

  private def nativeLawNames: Set[String] = companions.flatMap { companion =>
    companion.getClass.getDeclaredMethods
      .filter(_.getReturnType eq classOf[Property[?]])
      .map(method => s"${companion.name}.${method.getName}")
  }.toSet

  private def liftedLawNames: Set[String] =
    val stream = Files.list(Path.of("model/ir"))
    val origins = try
      stream.iterator.asScala
        .filter(path => path.getFileName.toString.endsWith(".json"))
        .flatMap(path => (parse(Files.readString(path)) \ "properties").children)
        .flatMap(property =>
          (property \ "origin" \ "name") match
            case JString(name) if name.startsWith("temporal.capabilities.") => Some(name)
            case _                                                          => None
        )
        .toSet
    finally stream.close()
    origins.map(
      _.stripPrefix("temporal.capabilities.")
        .replace("Retries.", "Retryable.")
        .replace("Deadline.", "Expirable.")
    )

  def declaring[S, O, F](m: Machine[S, O, F], section: Capabilities[S, O, F]): Declared =
    // A machine of facets declares its facets' capabilities (fn-161).
    def sections(c: Capabilities[S, O, F]): Seq[Capabilities[S, O, F]] = c match
      case f: framework.FacetCapabilities[S, O, F] @unchecked => f.facets.flatMap(sections)
      case other                                              => Seq(other)
    val bindings = sections(section).flatMap(c =>
      c.getClass.getMethods
        .filter(method => classOf[CapabilityOf[?, ?, ?]].isAssignableFrom(method.getReturnType))
        .map(_.invoke(c))
    )
    val applicable = bindings.flatMap {
      case binding: Product =>
        val fields = binding.productElementNames.zip(binding.productIterator).toMap
        binding.productPrefix match
          case "Retryable" =>
            Seq("failureReturnsToWaiting", "failureEndsFailed", "attemptCountIsWithinPolicy")
              .concat(Option.when(present(fields("pendingPause")))("failurePauses"))
              .concat(Option.when(present(fields("pendingCancel")))("failureCancels"))
              .map("Retryable." + _)
          case "Expirable" =>
            Seq("firesInWindow", "deadlineTimesOut")
              .concat(Option.when(enabled(fields("retryable")))("deadlineReturnsToWaiting"))
              .concat(Option.when(present(fields("pendingPause")))("deadlinePauses"))
              .map("Expirable." + _)
          case _ => Seq.empty
      case _ => Seq.empty
    }.toSet
    Declared(
      m.name,
      m.init.getClass.getName,
      bindings.map(_.getClass.getSimpleName).toSet,
      applicable
    )

  val declared = Seq(
    declaring(product.ActivityProduct, product.ActivityProduct.capabilities),
    declaring(protocol.ActivityRecord, protocol.ActivityRecord.capabilities),
    declaring(protocol.TrustingActivityRecord, protocol.TrustingActivityRecord.capabilities),
    declaring(protocol.ActivityProtocol, protocol.ActivityProtocol.capabilities),
    declaring(NexusProtocol, NexusProtocol.capabilities),
    declaring(WorkflowNexusProtocol, WorkflowNexusProtocol.capabilities)
  )

  val conditionalFixtures = Seq(
    Declared(
      "retryFixture",
      classOf[RetryFixture.Snapshot].getName,
      Set("Retryable"),
      Set(
        "Retryable.failurePauses",
        "Retryable.failureCancels",
        "Retryable.attemptCountIsWithinPolicy"
      )
    ),
    Declared(
      "deadlineFixture",
      classOf[ExpirableFixture.Snapshot].getName,
      Set("Expirable"),
      Set("Expirable.deadlineReturnsToWaiting", "Expirable.deadlinePauses")
    )
  )

  test("Activity and workflow Nexus both declare Retryable and Expirable") {
    for machine <- declared.filter(d =>
        Set("activityProtocol", "nexusProtocol").contains(d.machine) &&
          Set(classOf[protocol.State].getName, WorkflowNexusProtocol.init.getClass.getName)
            .contains(d.state)
      )
    do assert(Set("Retryable", "Expirable").subsetOf(machine.capabilities), s"$machine")
  }

  def instantiating(p: Brought, machines: Seq[Declared]): Seq[String] =
    machines
      .filter(d =>
        p.by.map(_.name).subsetOf(d.capabilities) &&
          (!(only(p.by, Retryable) || only(p.by, Expirable)) || d.applicable.contains(p.name))
      )
      .map(_.state)
      .distinct

  def underInstantiated(claims: Seq[Brought], machines: Seq[Declared]): Seq[String] =
    claims.flatMap { p =>
      val states = instantiating(p, machines)
      Option.when(states.size < 2)(
        s"${p.name} is brought to ${states.size} machine(s) with their own state type " +
          s"(${states.mkString(", ")}): a capability Property needs two"
      )
    }

  test("capability companions define the Properties they bring") {
    val defined = nativeLawNames
    assertEquals(defined, expectedLawNames)
    assertEquals(defined.size, 13)
  }

  test("native and lifted scans match the fixed thirteen-law universe") {
    requireComplete(nativeLawNames, liftedLawNames)
  }

  test("the fixed universe rejects the same law omitted by both candidate scans") {
    val omitted = expectedLawNames - "Closable.terminalStatesAreFinal"
    val failure = intercept[munit.ComparisonFailException](requireComplete(omitted, omitted))
    assert(failure.getMessage.contains("Closable.terminalStatesAreFinal"), failure.getMessage)
  }

  test("every capability Property has two declared machines with their own state types") {
    assertEquals(underInstantiated(properties, declared ++ conditionalFixtures), Seq.empty)
    val finitePolicy = properties
      .find(_.name == "Retryable.attemptCountIsWithinPolicy")
      .get
    val finitePolicySites = declared.filterNot(
      _.state == WorkflowNexusProtocol.init.getClass.getName
    )
    assertEquals(
      instantiating(finitePolicy, finitePolicySites ++ conditionalFixtures),
      Seq(classOf[protocol.State].getName, classOf[RetryFixture.Snapshot].getName)
    )
  }

  test("workflow Nexus brings only retry and deadline branches its bindings supply") {
    val nexus = declared
      .find(_.state == WorkflowNexusProtocol.init.getClass.getName)
      .get
    assertEquals(
      nexus.applicable,
      Set(
        "Retryable.failureReturnsToWaiting",
        "Retryable.failureEndsFailed",
        "Retryable.attemptCountIsWithinPolicy",
        "Expirable.firesInWindow",
        "Expirable.deadlineTimesOut"
      )
    )
    val activity =
      declared.find(_.state == classOf[protocol.State].getName).get
    assertEquals(
      activity.applicable,
      properties.filter(p => only(p.by, Retryable) || only(p.by, Expirable)).map(_.name).toSet
    )
  }

  test("every Closable site binds only its rejection, including all four derived owners") {
    val closables = Seq(
      product.ActivityProduct.lifecycle.capabilities.closable,
      protocol.ActivityRecord.admission.capabilities.closable,
      protocol.TrustingActivityRecord.capabilities.closable,
      protocol.RecordOverQueue.capabilities.closable,
      protocol.RecordOverMatching.capabilities.closable,
      protocol.TrustingRecordOverQueue.capabilities.closable,
      protocol.TrustingRecordOverMatching.capabilities.closable,
      protocol.RecordOverLossyMatching.capabilities.closable,
      NexusProtocol.operation.capabilities.closable
    )
    assertEquals(closables.size, 9)
    for closable <- closables do
      closable match
        case product: Product => assertEquals(product.productElementNames.toSeq, Seq("rejected"))
        case _                => fail(s"$closable is no rejection binding")
  }

  test("a composed shared rejection derives the capability outcome key") {
    val rejected = protocol.RecordOverQueue.composedOutcome(
      protocol.RecordMember,
      Outcome.rejected(Rejection.notFound)
    )
    assertEquals(rejected, "activity_rejected-notFound")
    assertEquals(protocol.RecordOverQueue.capabilities.closable.rejected, rejected)
  }

  test("every Pausable and Pollable site binds only actions, including all four derived owners") {
    val pausableAndPollable = Seq(
      product.ActivityProduct.pausing.capabilities.pausable ->
        product.ActivityProduct.dispatching.capabilities.pollable,
      protocol.ActivityRecord.admission.capabilities.pausable -> protocol.ActivityRecord.admission.capabilities.pollable,
      protocol.TrustingActivityRecord.capabilities.pausable ->
        protocol.TrustingActivityRecord.capabilities.pollable,
      protocol.RecordOverQueue.capabilities.pausable -> protocol.RecordOverQueue.capabilities.pollable,
      protocol.TrustingRecordOverQueue.capabilities.pausable ->
        protocol.TrustingRecordOverQueue.capabilities.pollable,
      protocol.RecordOverMatching.capabilities.pausable ->
        protocol.RecordOverMatching.capabilities.pollable,
      protocol.TrustingRecordOverMatching.capabilities.pausable ->
        protocol.TrustingRecordOverMatching.capabilities.pollable,
      protocol.RecordOverLossyMatching.capabilities.pausable ->
        protocol.RecordOverLossyMatching.capabilities.pollable
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
    val record = declared.filter(
      _.state == classOf[protocol.AdmissionState].getName
    )
    assertEquals(record.map(_.machine), Seq("activityRecord", "trustingActivityRecord"))
    assertEquals(
      underInstantiated(properties.filter(p => only(p.by, Closable)), record),
      Seq(
        "Closable.terminalStatesAreFinal is brought to 1 machine(s) with their own state type " +
          "(temporal.features.activity.standalone.protocol.AdmissionState): a capability Property needs two",
        "Closable.closedIsRejectedUniformly is brought to 1 machine(s) with their own state type " +
          "(temporal.features.activity.standalone.protocol.AdmissionState): a capability Property needs two"
      )
    )
  }

  test("Pausable's dispatch Property needs Pollable as well") {
    val dispatch = properties
      .find(_.name == "Pausable.pausedIsNotDispatched")
      .get
    assertEquals(
      instantiating(dispatch, Seq(Declared("pausedOnly", "Paused", Set("Pausable")))),
      Seq.empty
    )
    assertEquals(
      instantiating(dispatch, declared),
      Seq(classOf[product.State].getName, classOf[protocol.AdmissionState].getName)
    )
  }
