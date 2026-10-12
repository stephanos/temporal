package umpire.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class ObservationReadTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task6ObservationRead))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  test("literal retired labels are removed while name, entity and typed selectors survive"):
    for (legacy, accepted) <- List(
        ("Observation(\"count\", order, \"info.attempt\")", "Observation(\"count\", order)"),
        ("Observation(on = order, read = \"info.attempt\")", "Observation(on = order)"),
        (
          "Observation(name = \"count\", on = order, read = \"info.attempt\")",
          "Observation(name = \"count\", on = order)"
        ),
        ("Observation(read = \"info.attempt\", on = order)", "Observation(on = order)"),
        (
          "framework.Observation(\"count\", order, \"info.attempt\")",
          "framework.Observation(\"count\", order)"
        ),
        (
          "_root_.framework.Observation(\"count\", order, \"info.attempt\")",
          "_root_.framework.Observation(\"count\", order)"
        )
      )
    do
      val suffix =
        "\n// info.attempt\nval selector = Field[Response, Info](_.getInfo)\nval text = \"read = info.attempt\"\nval other = Other(read = \"keep\")"
      val source = "import framework.*\nval count = " + legacy + suffix
      val result = rewrite(source)
      assertEquals(result.diagnostics, Nil)
      assertEquals(result.fixed, "import framework.*\nval count = " + accepted + suffix)
      assertEquals(rewrite(result.fixed).fixed, result.fixed)
      assertEquals(rewrite(result.fixed).diagnostics, Nil)

  test("unsupported and potentially effectful constructors refuse the entire document"):
    for bad <- List(
        "Observation(on = order, read = label())",
        "Observation(\"count\", order, label())",
        "Observation(\"count\", order, \"label\", extra)",
        "Observation(read = \"label\")",
        "Observation(on = order, read = \"label\", read = \"twice\")",
        "Observation(on = order, mystery = \"label\")",
        "new _root_.framework.Observation(\"count\", order, \"label\")",
        "Observation.apply(\"count\", order, \"label\")"
      )
    do
      val source =
        "import framework.*\nval good = Observation(on = order, read = \"safe\")\nval bad = " + bad
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
      assert(result.diagnostics.forall(_.message.contains("manually")))

  test("shadowed or unresolved names refuse atomically and unrelated constructors stay intact"):
    for prefix <- List(
        "",
        "import other.*\n",
        "import framework.*\nimport other.*\n",
        "import framework.*\nobject Observation {}\n",
        "import framework.*\ndef f(Observation: Any) = ()\n",
        "import framework.*\nval Observation = other\n",
        "import framework.{Entity as Observation}\n",
        "object framework {}\n"
      )
    do
      val source = prefix + "val count = Observation(on = order, read = \"label\")"
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
    val unrelated = "val count = other.Observation(on = order, read = \"keep\")"
    assertEquals(rewrite(unrelated).fixed, unrelated)
    assertEquals(rewrite(unrelated).diagnostics, Nil)
    val hiddenPackage =
      "import other.framework\nval count = framework.Observation(on = order, read = \"label\")"
    assertEquals(rewrite(hiddenPackage).fixed, hiddenPackage)
    assert(rewrite(hiddenPackage).diagnostics.nonEmpty)
    val rooted =
      "import other.*\nval count = _root_.framework.Observation(on = order, read = \"label\")"
    assertEquals(rewrite(rooted).diagnostics, Nil)
    assertEquals(
      rewrite(rooted).fixed,
      "import other.*\nval count = _root_.framework.Observation(on = order)"
    )
    for source <- List(
        "import framework.{Observation as Obs}\nval count = Obs(on = order, read = \"label\")",
        "import framework.*\nval count = new Observation(\"count\", order, \"label\")"
      )
    do
      assertEquals(rewrite(source).fixed, source)
      assert(rewrite(source).diagnostics.nonEmpty)

  test("already accepted constructors converge and comments in the removed argument survive"):
    val accepted = "import framework.Observation\nval count = Observation(on = order)"
    assertEquals(rewrite(accepted).fixed, accepted)
    assertEquals(rewrite(accepted).diagnostics, Nil)
    val legacy =
      "import framework.Observation\nval count = Observation(on = order, /* field hint */ read = \"label\")"
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assert(result.fixed.contains("/* field hint */"))
    assert(!result.fixed.contains("read ="))
