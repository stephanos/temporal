package temporal.laws
// The two-entity rule of the catalog, run as Scala.

import umpire.laws.{Capability, Catalog, Law}
import umpire.laws.Capability.*

/**
 * Every law of the catalog has at least two instantiating entities: machines, each with its own
 * state type, that declare the capabilities bringing it. A composition reading a member's capability
 * through its projection, and a machine derived from another, share a counted machine's state type
 * and so do not count again.
 *
 * The declarations counted are the activity's, which its Models declare (model/ir/<file>.laws.json lists
 * them as each law's instantiating machines), and the Nexus operation's that fn-122 task 4 declares:
 * until it does, Terminable and Cancelable have one checked-in machine, so the set stays written here.
 */
class CatalogTest extends munit.FunSuite:
  /** A machine that declares capabilities: its name, the state type it owns, what it declares. */
  final case class Declaring(machine: String, state: String, capabilities: Set[Capability])

  val planned: Seq[Declaring] = Seq(
    Declaring("activityProduct", "ProductState", Set(Closable, Pausable, Pollable)),
    Declaring("currentAdmission", "AdmissionState", Set(Closable, Pausable, Pollable)),
    // Derived from currentAdmission by rebinding its dispatch: one entity with it.
    Declaring("staleAdmission", "AdmissionState", Set(Closable, Pausable, Pollable)),
    // Reads the record through its `activity` member: the record's entity again.
    Declaring("currentOverQueue", "AdmissionState", Set(Closable, Pausable, Pollable)),
    Declaring("activityProtocol", "ProtocolState", Set(Terminable, Cancelable, Describable)),
    // fn-122 task 4's Model, grounded in chasm/lib/nexusoperation.
    Declaring(
      "nexusOperation",
      "OperationState",
      Set(Closable, Terminable, Cancelable, Describable)
    )
  )

  /** The state types of the machines that instantiate a law brought by `by`. */
  def instantiating(by: Set[Capability], declared: Seq[Declaring]): Seq[String] =
    declared.filter(d => by.subsetOf(d.capabilities)).map(_.state).distinct

  /** Each law of `catalog` with fewer than two instantiating entities, by name. */
  def underInstantiated(catalog: Catalog, declared: Seq[Declaring]): Vector[String] =
    catalog.entries.flatMap { b =>
      val entities = instantiating(b.by, declared)
      Option.when(entities.size < 2)(
        s"${b.law.name} is instantiated by ${entities.size} machine(s) with their own state " +
          s"type (${entities.mkString(", ")}): a law joins the catalog with two"
      )
    }

  test("every law of the Temporal catalog has two instantiating machines among the planned ones") {
    assertEquals(underInstantiated(catalog, planned), Vector.empty)
  }

  test("a law with one instantiating machine fails by its name; derived machines count once") {
    object describedWhilePaused extends Law(Nil, "", "")
    val lonely = describedWhilePaused
    val failing = Catalog.pair(Pausable, Describable)(lonely)
    assertEquals(
      instantiating(Set(Closable), planned),
      Seq("ProductState", "AdmissionState", "OperationState")
    )
    assertEquals(
      underInstantiated(failing, planned),
      Vector(
        "describedWhilePaused is instantiated by 0 machine(s) with their own state type (): a law " +
          "joins the catalog with two"
      )
    )
    val record = planned.filter(_.state == "AdmissionState")
    assertEquals(underInstantiated(Catalog.single(Closable)(lonely), record).size, 1)
  }

  test("a machine receives the law of a pair it declares both of, without naming the pair") {
    val product = planned.head.capabilities
    assert(catalog.laws(product).map(_.name).contains("pausedIsNotDispatched"))
    assert(!catalog.laws(Set(Pausable)).map(_.name).contains("pausedIsNotDispatched"))
  }

  test("no two laws share a name, which generated claims take as `<machine>.<law>`") {
    val names = catalog.entries.map(_.law.name)
    assertEquals(names.distinct, names)
  }

  test("a law is named after its object, which generated claims are named after") {
    assertEquals(
      catalog.entries.map(_.law.name).sorted,
      Vector(
        "cancelIsRequested",
        "closedIsRejectedUniformly",
        "pausedIsNotDispatched",
        "terminalStatesAreFinal",
        "terminateSettles"
      )
    )
  }
