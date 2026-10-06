package umpire.irgen

// The role closure of a phase enum (Roles.scala), by class names alone: the enums of the lifter's
// role fixtures (testdata/lifts/Roles.scala, RoleRejects.scala), written out as the classes each
// case derives from, so no Model is built or read.
class RolesSuite extends munit.FunSuite:
  // Every class a role derives from, itself included, as model/umpire/Roles.scala declares them,
  // and the fixture's own role `Expired` and stand-alone trait `Audited`.
  private val parents = Map(
    "Live" -> Nil,
    "Waiting" -> List("Live"),
    "Retrying" -> List("Waiting"),
    "Held" -> List("Live"),
    "Suspended" -> List("Live"),
    "Closed" -> Nil,
    "Succeeded" -> List("Closed"),
    "Failed" -> List("Closed"),
    "Canceled" -> List("Closed"),
    "Terminated" -> List("Closed"),
    "TimedOut" -> List("Closed")
  ).map((r, ps) => s"umpire.$r" -> ps.map("umpire." + _)) ++ Map(
    "fixture.Expired" -> List("umpire.TimedOut"),
    "fixture.Audited" -> Nil
  )
  private def bases(c: String): Seq[String] =
    c +: parents.getOrElse(c, Nil).flatMap(bases).distinct

  // A case declaring `roles`, with every class it derives from, as the compiler lists them.
  private def phase(name: String, roles: String*): (String, Seq[String]) =
    name -> (Seq("fixture.Phase") ++ roles.flatMap(r => bases(qualified(r))).distinct ++
      Seq("scala.reflect.Enum", "java.lang.Object", "scala.Any"))
  private def qualified(r: String) =
    if parents.contains(s"umpire.$r") then s"umpire.$r" else s"fixture.$r"

  private val phases = Seq(
    phase("unstarted"),
    phase("queued", "Waiting"),
    phase("backingOff", "Retrying"),
    phase("running", "Held"),
    phase("paused", "Suspended"),
    phase("done", "Succeeded", "Audited"),
    phase("failed", "Failed"),
    phase("expired", "Expired")
  )

  test("a role holds in the cases that have it, inherited roles included, in declaration order"):
    val roles = Roles.closure(phases, bases)
    assertEquals(
      Seq("Live", "Waiting", "Retrying", "Closed", "TimedOut").map(r => roles.cases(s"umpire.$r")),
      Seq(
        Seq("queued", "backingOff", "running", "paused"),
        Seq("queued", "backingOff"),
        Seq("backingOff"),
        Seq("done", "failed", "expired"),
        Seq("expired")
      )
    )
    assertEquals(roles.cases("fixture.Expired"), Seq("expired"), "a Model's own role")
    assertEquals(roles.cases("fixture.Audited"), Nil, "a stand-alone trait is no role")
    assertEquals(roles.cases("umpire.Canceled"), Nil, "a role no case has")
    assertEquals(roles.conflicts, Nil, "Retrying with Waiting is no conflict")

  test("a stand-alone trait is no role, and a trait extending a role is one"):
    assertEquals(
      Seq("fixture.Audited", "fixture.Expired", "umpire.Closed").map(c => Roles.isRole(bases(c))),
      Seq(false, true, true)
    )

  test("a case that is Live and Closed, two of Waiting, Held and Suspended, or two closure roles"):
    val conflicting = Seq(
      phase("ok", "Retrying"),
      phase("both", "Retrying", "Failed"),
      phase("twoLive", "Held", "Suspended"),
      phase("twoClosed", "Expired", "Canceled")
    )
    assertEquals(
      Roles.closure(conflicting, bases).conflicts.map(c => (c.phase, c.declared, c.roles)),
      Seq(
        ("both", Seq("umpire.Retrying", "umpire.Failed"), Seq("umpire.Live", "umpire.Closed")),
        ("twoLive", Seq("umpire.Held", "umpire.Suspended"), Seq("umpire.Held", "umpire.Suspended")),
        (
          "twoClosed",
          Seq("fixture.Expired", "umpire.Canceled"),
          Seq("umpire.Canceled", "umpire.TimedOut")
        )
      )
    )
