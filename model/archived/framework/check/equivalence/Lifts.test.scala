package framework.check
package equivalence

import java.nio.file.{Files, Path}

class LiftsSuite extends munit.FunSuite:
  private def lift(files: (String, String)*): Path =
    val directory = Scratch.directory("umpire-lift")
    for (name, text) <- files do Files.writeString(directory.resolve(name), text): Unit
    directory

  private val before =
    """{
      |  "types": [ {
      |    "name": "pkg.Kind",
      |    "position": { "file": "model/a/Kind.scala", "line": 12 }
      |  }, {
      |    "name": "pkg.Other",
      |    "position": { "file": "model/a/Kind.scala", "line": 4 }
      |  } ]
      |}
      |""".stripMargin

  test("lifts the same byte for byte compare equal, and any difference needs a delta"):
    val waivers = "{ \"waivers\": [ ] }\n"
    assertEquals(
      Lifts.compare(
        lift("m.json" -> before, "m.waivers.json" -> waivers),
        lift("m.json" -> before, "m.waivers.json" -> waivers),
        LiftDelta.empty
      ),
      Vector.empty
    )
    assertEquals(
      Lifts.compare(
        lift("m.json" -> before),
        lift("m.json" -> before.replace("12", "14")),
        LiftDelta.empty
      ),
      Vector("m.json: the lifts differ, and no delta is declared")
    )
    assertEquals(
      Lifts.compare(lift("m.json" -> before), lift(), LiftDelta.empty),
      Vector("m.json: written by one lift only")
    )

  test("moved lines shift the positions below them, renames rename, and each entry must explain"):
    val after = before.replace("\"line\": 12", "\"line\": 14").replace("pkg.Kind", "pkg.Sort")
    val delta = LiftDelta.parse("moved model/a/Kind.scala 10 2\n# a rename\npkg.Kind pkg.Sort\n")
    assertEquals(
      Lifts.compare(lift("m.json" -> before), lift("m.json" -> after), delta),
      Vector.empty
    )
    val idle = LiftDelta.parse(
      "moved model/a/Kind.scala 10 2\npkg.Kind pkg.Sort\nmoved model/b/B.scala 1 1\n"
    )
    assertEquals(
      Lifts.compare(lift("m.json" -> before), lift("m.json" -> after), idle),
      Vector("delta: Moved(model/b/B.scala,1,1) moves no position")
    )
    val short = LiftDelta.parse("moved model/a/Kind.scala 13 2\npkg.Kind pkg.Sort\n")
    assertEquals(
      Lifts.compare(lift("m.json" -> before), lift("m.json" -> after), short),
      Vector(
        "m.json: /types/0/position/line: 12 before, 14 after",
        "delta: Moved(model/a/Kind.scala,13,2) moves no position"
      )
    )

  test("a delta line that is no move and no rename is refused"):
    intercept[GateError](LiftDelta.parse("moved model/a/Kind.scala zero 2\n"))
    intercept[GateError](LiftDelta.parse("pkg.Kind -\n"))
