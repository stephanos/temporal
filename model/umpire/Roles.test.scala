package umpire

import scala.reflect.TypeTest

// Roles are metadata: a phase enum whose cases take roles has the values, in the same order, of the
// enum without them, and its cases test as the roles they declare and imply.
class RolesSuite extends munit.FunSuite:
  enum Roled derives Finite:
    case unstarted
    case backingOff extends Roled, Retrying
    case running extends Roled, Held
    case done extends Roled, Succeeded

  enum Bare derives Finite:
    case unstarted, backingOff, running, done

  test("a phase enum's roles leave its Finite values and their order as they are"):
    assertEquals(Finite[Roled].values.map(_.toString), Finite[Bare].values.map(_.toString))
    assertEquals(Finite[Roled].values, Roled.values.toIndexedSeq)

  test("a case is the roles it declares and those they extend"):
    def takes[R](p: Roled)(using test: TypeTest[Roled, R]) = test.unapply(p).nonEmpty
    def roles(p: Roled) = Seq(
      takes[Live](p),
      takes[Waiting](p),
      takes[Retrying](p),
      takes[Held](p),
      takes[Closed](p),
      takes[Succeeded](p)
    )
    assertEquals(
      Roled.values.toSeq.map(roles),
      Seq(
        Seq(false, false, false, false, false, false),
        Seq(true, true, true, false, false, false),
        Seq(true, false, false, true, false, false),
        Seq(false, false, false, false, true, true)
      )
    )
