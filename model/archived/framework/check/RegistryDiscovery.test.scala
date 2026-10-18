package framework.check

import java.nio.file.{Files, Path}

class RegistryDiscoveryTest extends munit.FunSuite:
  private def write(root: Path, file: String, source: String): Unit =
    val path = root.resolve(file)
    Files.createDirectories(path.getParent): Unit
    Files.writeString(path, source): Unit

  private def names(root: Path): Vector[String] =
    val registry = new Registry:
      val source = "fixture"
      val current: Set[String] = Set.empty
      val baselines: Map[String, Baseline] = Map.empty
      val committedIrHeld = true
      val rules: Seq[(String, Area)] = Nil
      val selectsEveryFeature: Set[Area] = Set.empty
      val modules = Seq(
        Module(
          "fixture",
          Seq("tests"),
          Seq("tests", "helpers"),
          Seq("tests/held")
        )
      )
      val suites: Map[String, Covers] = Map.empty
    registry.declared(root).map(_.name)

  test(
    "source ancestry resolves multiline aliases and helper traits across explicit inputs"
  ):
    val root = Scratch.directory("registry-ancestry")
    write(
      root,
      "helpers/Base.scala",
      """package fixture.base
      |import framework.dsl.{ModelSuite as BoundSuite}
      |abstract class Base extends BoundSuite(instance)
      |trait Shared extends Base
      |type Alias = Shared
      |""".stripMargin
    )
    write(
      root,
      "tests/Leaf.test.scala",
      """package fixture.client
      |import fixture.base.{Alias => Parent}
      |final class Leaf
      |  extends Parent
      |  with Serializable:
      |  val helper = 1
      |class Direct extends framework.dsl.ModelSuite(instance)
      |abstract class Abstract extends framework.dsl.ModelSuite(instance)
      |trait Helper extends framework.dsl.ModelSuite(instance)
      |class Plain
      |""".stripMargin
    )
    assertEquals(
      names(root),
      Vector("fixture.client.Leaf", "fixture.client.Direct")
    )

  test(
    "comma-separated and grouped imports preserve exact suite budget identities"
  ):
    for (imports, parent) <- Vector(
        "import scala.collection.mutable, framework.dsl.ModelSuite" -> "ModelSuite",
        "import scala.collection.{mutable, immutable}, framework.dsl.{ModelSuite as BoundSuite}" -> "BoundSuite",
        "import framework.dsl.{ModelSuite => BoundSuite}, scala.collection.{mutable, immutable}" -> "BoundSuite",
        "import scala.collection.mutable,\n  framework.dsl.{ModelSuite as BoundSuite}" -> "BoundSuite"
      )
    do
      val root = Scratch.directory("registry-comma-imports")
      write(
        root,
        "tests/Leaf.test.scala",
        s"package fixture\n$imports\nclass Leaf extends $parent(instance)\n"
      )
      val discovered = names(root)
      assertEquals(discovered, Vector("fixture.Leaf"))
      Tools(root, Map("UMPIRE_CHECKER_BUDGETS" -> "fixture.Leaf=7"))
        .withCheckerBudgets(discovered.toSet)

  test(
    "method and block local types cannot impersonate addressable helper ancestors"
  ):
    for (declaration, opening, closing) <- Vector(
        ("def make", "{", "}"),
        ("val make", "{", "}"),
        ("val make", "locally:", ""),
        ("val make", "List(1).map: value =>", "")
      )
    do
      val root = Scratch.directory("registry-local-ancestor")
      write(
        root,
        "helpers/Helpers.scala",
        s"""package fixture
        |object Helpers {
        |  $declaration = $opening
        |    class Base extends munit.FunSuite
        |    new Base
        |  $closing
        |  class Base
        |  abstract class Actual extends framework.dsl.ModelSuite(instance)
        |}
        |""".stripMargin
      )
      write(
        root,
        "tests/Leaf.test.scala",
        "package fixture\nclass NotSuite extends Helpers.Base\nclass Valid extends Helpers.Actual\n"
      )
      assertEquals(names(root), Vector("fixture.Valid"))

  test("wildcard exclusions apply only to their own import selector group"):
    val root = Scratch.directory("registry-import-exclusion")
    write(
      root,
      "helpers/Suite.scala",
      "package suites\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
    )
    write(root, "helpers/Plain.scala", "package plain\nclass Base\n")
    write(
      root,
      "tests/Leaf.test.scala",
      "package fixture\nimport suites.{Base as _, *}\nimport plain.*\nclass Leaf extends Base\n"
    )
    assertEquals(names(root), Vector.empty)
    write(
      root,
      "tests/Leaf.test.scala",
      "package fixture\nimport suites.{Base as _, *}\nimport suites.*\nclass Restored extends Base\n"
    )
    assertEquals(names(root), Vector("fixture.Restored"))

  test("renamed imports remove the original wildcard name while retaining the alias"):
    for rename <- Vector("as", "=>") do
      val root = Scratch.directory("registry-import-rename")
      write(
        root,
        "helpers/Suite.scala",
        "package suites\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
      )
      write(root, "helpers/Plain.scala", "package plain\nclass Base\n")
      write(
        root,
        "tests/Leaf.test.scala",
        s"package fixture\nimport suites.{Base $rename Bound, *}\nimport plain.*\nclass Leaf extends Base\nclass Renamed extends Bound\n"
      )
      val discovered = names(root)
      assertEquals(discovered, Vector("fixture.Renamed"))
      Tools(root, Map("UMPIRE_CHECKER_BUDGETS" -> "fixture.Renamed=7"))
        .withCheckerBudgets(discovered.toSet)
      write(
        root,
        "tests/Leaf.test.scala",
        s"package fixture\nimport suites.{Base $rename Bound, *}\nimport suites.*\nclass Restored extends Base\nclass Renamed extends Bound\n"
      )
      assertEquals(names(root), Vector("fixture.Restored", "fixture.Renamed"))

  test("typed helper expressions retain accessible sibling ancestor declarations"):
    for
      declaration <- Vector(
        "val n: Int = 0",
        "var n: Int = 0",
        "def n: Int = 0",
        "val n:\n    Int = 0",
        "var n:\n    Int = 0",
        "def n:\n    Int = 0"
      )
      separator <- Vector("; ", "\n  ")
    do
      val root = Scratch.directory("registry-typed-helper")
      write(
        root,
        "helpers/Helpers.scala",
        s"package fixture\nobject Helpers {\n  $declaration${separator}abstract class Base extends framework.dsl.ModelSuite(instance)\n}\n"
      )
      write(root, "tests/Leaf.test.scala", "package fixture\nclass Leaf extends Helpers.Base\n")
      assertEquals(names(root), Vector("fixture.Leaf"))

  test(
    "qualified and wildcard parent resolution does not confuse equal short names"
  ):
    val root = Scratch.directory("registry-bindings")
    write(
      root,
      "helpers/Suite.scala",
      "package suite\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
    )
    write(root, "helpers/Plain.scala", "package plain\nclass Base\n")
    write(
      root,
      "tests/A.test.scala",
      "package client\nimport suite.*\nclass First extends Base\nclass Second extends suite.Base\n"
    )
    write(
      root,
      "tests/B.test.scala",
      "package other\nimport plain.Base\nclass NotSuite extends Base\n"
    )
    write(
      root,
      "helpers/Shadow.scala",
      "package client.framework.dsl\nclass ModelSuite\n"
    )
    write(
      root,
      "tests/C.test.scala",
      "package client\nclass Rooted extends _root_.framework.dsl.ModelSuite(instance)\n"
    )
    assertEquals(
      names(root),
      Vector("client.First", "client.Second", "client.Rooted")
    )

  test("direct FunSuite identity and exclusions survive code-only scanning"):
    val root = Scratch.directory("registry-exclusions")
    write(
      root,
      "tests/A.test.scala",
      "package outer\npackage `type`\nimport munit.FunSuite\nclass Original extends FunSuite\n"
    )
    write(root, "tests/B.test.scala", "class Default extends munit.FunSuite\n")
    write(
      root,
      "tests/C.test.scala",
      "\"\"\"class Fake extends munit.FunSuite\"\"\"\n// class Comment extends munit.FunSuite\n"
    )
    write(
      root,
      "tests/D.test.scala",
      "val delimiter = '{'\nclass CharacterSafe extends munit.FunSuite\n"
    )
    for directory <- Vector("testdata", "build", ".scala-build", ".bsp", "held")
    do
      write(
        root,
        s"tests/$directory/Hidden.test.scala",
        "class Hidden extends munit.FunSuite\n"
      )
    write(
      root,
      "helpers/NotSelected.test.scala",
      "class NotSelected extends munit.FunSuite\n"
    )
    assertEquals(
      names(root),
      Vector("outer.type.Original", "Default", "CharacterSafe")
    )

  test(
    "lexical helper ancestry follows object imports without registering helper declarations"
  ):
    val root = Scratch.directory("registry-owned")
    write(
      root,
      "helpers/Base.scala",
      """package fixture
      |object Helpers:
      |  import framework.dsl.ModelSuite
      |  abstract class Base extends ModelSuite(instance)
      |""".stripMargin
    )
    write(
      root,
      "tests/Leaf.test.scala",
      "package fixture\nimport Helpers.Base\nclass Leaf extends Base\n"
    )
    assertEquals(names(root), Vector("fixture.Leaf"))

  test(
    "excluded suite locations may supply explicitly compiled helper ancestry"
  ):
    val root = Scratch.directory("registry-helper-exclusion")
    write(
      root,
      "tests/held/Base.scala",
      "package fixture\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
    )
    write(
      root,
      "tests/Leaf.test.scala",
      "package fixture\nclass Leaf extends Base\n"
    )
    assertEquals(names(root), Vector("fixture.Leaf"))

  test(
    "package blocks retain separate exact identities and cyclic unrelated types terminate"
  ):
    val root = Scratch.directory("registry-package-blocks")
    write(
      root,
      "tests/Blocks.test.scala",
      """package first {
      |  class Leaf extends framework.dsl.ModelSuite(instance)
      |}
      |package second:
      |  class Leaf extends framework.dsl.ModelSuite(instance)
      |  trait A extends B
      |  trait B extends A
      |  class Unrelated extends A
      |""".stripMargin
    )
    assertEquals(names(root), Vector("first.Leaf", "second.Leaf"))

  test(
    "object-owned and encoded suite names equal the pinned runner's binary identities"
  ):
    val nested = classOf[RegistryDiscoveryFixtures.Nested]
    assertEquals(
      nested.getName,
      "framework.check.RegistryDiscoveryFixtures$Nested"
    )
    val result = org.junit.runner.JUnitCore.runClasses(nested)
    assert(result.wasSuccessful(), result.getFailures.toString)
    assertEquals(result.getRunCount, 1)
    val root = Scratch.directory("registry-binary-names")
    write(
      root,
      "tests/Nested.test.scala",
      """package framework.check
      |object RegistryDiscoveryFixtures:
      |  class Nested extends munit.FunSuite
      |class `Suite-name` extends munit.FunSuite
      |""".stripMargin
    )
    assertEquals(
      names(root),
      Vector(
        "framework.check.RegistryDiscoveryFixtures$Nested",
        "framework.check.Suite$minusname"
      )
    )

  test(
    "local parent identity wins over wildcard names and ambiguous imports refuse"
  ):
    val root = Scratch.directory("registry-parent-shadow")
    write(
      root,
      "helpers/Base.scala",
      "package suites\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
    )
    write(
      root,
      "tests/Local.test.scala",
      "package local\nimport suites.*\nclass Base\nclass NotSuite extends Base\n"
    )
    assertEquals(names(root), Vector.empty)
    write(
      root,
      "helpers/Other.scala",
      "package other\nabstract class Base extends framework.dsl.ModelSuite(instance)\n"
    )
    write(
      root,
      "tests/Ambiguous.test.scala",
      "package client\nimport suites.*\nimport other.*\nclass Leaf extends Base\n"
    )
    val error = intercept[GateError](names(root))
    assert(error.getMessage.contains("Base"))
    assert(error.getMessage.contains("Ambiguous.test.scala"))

  test("constructor-bound and local helpers are not runnable suite targets"):
    val result =
      org.junit.runner.JUnitCore
        .runClasses(classOf[RegistryDiscoveryFixtures.Parameterized])
    assert(!result.wasSuccessful())
    assert(
      result.getFailures.toString.contains("constructor"),
      result.getFailures.toString
    )
    val root = Scratch.directory("registry-runnable")
    write(
      root,
      "tests/Helpers.test.scala",
      """package fixture
      |class Parameterized(value: Int) extends framework.dsl.ModelSuite(instance)
      |class Outer:
      |  class Inner extends framework.dsl.ModelSuite(instance)
      |object Helpers:
      |  def make =
      |    class Local extends framework.dsl.ModelSuite(instance)
      |    ()
      |  val initialize = {
      |    class InBlock extends framework.dsl.ModelSuite(instance)
      |    ()
      |  }
      |class Runnable() extends framework.dsl.ModelSuite(instance)
      |abstract
      |class Deferred extends framework.dsl.ModelSuite(instance)
      |abstract class Hidden extends framework.dsl.ModelSuite(instance); class AlsoRunnable extends framework.dsl.ModelSuite(instance)
      |""".stripMargin
    )
    assertEquals(
      names(root),
      Vector("fixture.Runnable", "fixture.AlsoRunnable")
    )

  test(
    "erased inline section members refuse before compilation with their member location"
  ):
    for
      section <- Vector("examples", "explorations", "conformance", "unit")
      member <- Vector(
        "inline val forbidden = 42",
        "inline def forbidden = 42",
        "transparent inline def forbidden = 42"
      )
      inherited <- Vector(false, true)
    do
      val root = Scratch.directory("registry-inline-section")
      val file =
        if inherited then "helpers/Base.scala" else "tests/Leaf.test.scala"
      val declaring = if inherited then "abstract class Base" else "class Leaf"
      val identity = if inherited then "fixture.Base" else "fixture.Leaf"
      write(
        root,
        file,
        s"package fixture\n$declaring extends framework.dsl.ModelSuite(instance):\n  object $section:\n    $member\n"
      )
      if inherited then
        write(
          root,
          "tests/Leaf.test.scala",
          "package fixture\nclass Leaf extends Base\n"
        )
      val error = intercept[GateError](names(root))
      assert(error.getMessage.contains(s"$file:4:"), error.getMessage)
      assert(
        error.getMessage.contains(s"$identity.$section.forbidden"),
        error.getMessage
      )
      assert(error.getMessage.contains("inline"), error.getMessage)

  test(
    "inline shared helpers and ordinary suites retain their source ownership"
  ):
    val root = Scratch.directory("registry-inline-helpers")
    write(
      root,
      "helpers/Unused.scala",
      """package fixture
      |abstract class Unused extends framework.dsl.ModelSuite(instance):
      |  object unit:
      |    inline val unused = 42
      |""".stripMargin
    )
    write(
      root,
      "tests/Allowed.test.scala",
      """package fixture
      |class Allowed extends framework.dsl.ModelSuite(instance):
      |  inline val shared = 42
      |  transparent inline def helper = 42
      |  def localHelper =
      |    object unit:
      |      inline val value = 42
      |    ()
      |  def localSuite =
      |    class Local extends framework.dsl.ModelSuite(instance):
      |      object unit:
      |        inline val local = 42
      |    ()
      |  object Shared:
      |    object unit:
      |      inline val value = 42
      |  object unit:
      |    // inline val forbidden = 42
      |    val description = "inline def forbidden = 42"
      |    def body =
      |      inline val local = 42
      |      local
      |class Ordinary extends munit.FunSuite:
      |  object unit:
      |    inline val value = 42
      |""".stripMargin
    )
    assertEquals(names(root), Vector("fixture.Allowed", "fixture.Ordinary"))

object RegistryDiscoveryFixtures:
  class Nested extends munit.FunSuite:
    test("inherited runner accepts object-owned suite"):
      assert(true)

  class Parameterized(value: Int) extends munit.FunSuite:
    test("requires constructor input"):
      assert(value > 0)
