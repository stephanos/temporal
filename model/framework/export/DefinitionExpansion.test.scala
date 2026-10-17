package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

class DefinitionExpansionTest extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "definition-expansion.")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  private def liftSource(
      name: String,
      sugar: String,
      expression: String,
      extra: String = "",
      roots: Seq[String] = Seq("fixture.expansion.MachineUnderTest")
  ) =
    val dir = Files.createDirectory(scratch.resolve(name))
    Files.writeString(
      dir.resolve("Source.scala"),
      s"""package fixture.expansion
import framework.dsl.*
import framework.dsl.Finite.given
enum State:
  case idle, done
enum Outcome:
  case accepted
enum Fact:
  case first, second
given Ok[Outcome] = Ok(Outcome.accepted)
object kit:
$sugar
val press = action(Actor("user"))
object MachineUnderTest extends Machine[State, Outcome, Fact]:
  val init = State.idle
  def end(s: State) = $expression
  object rules extends Bindings(press ~> ((s: State) => List(Step(Outcome.accepted, s))))
$extra
"""
    ): Unit
    Files.writeString(
      dir.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = dir.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          dir.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val out = dir.resolve("fixture.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/,${ModelBuild.jarArgument}",
        ModelBuild.classpath.toString,
        out.toString
      ) ++ roots
    )
    (result, out)

  test("a marked kit definition expands without registering its name"):
    val (result, out) = liftSource(
      "kit",
      "  inline def selected(value: State): Boolean = value == State.done",
      "kit.selected(s)"
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val model = mapper.readTree(Files.readString(out))
    val functions = model.path("functions").elements().asScala.toList
    assert(
      !functions.exists(_.path("name").asText().contains("selected")),
      functions.mkString("\n")
    )
    assertEquals(model.at("/machines/0/ends/lambda/body/binary/op").asText(), "OP_EQ")

  test("declaration definitions preserve callback names without name registration"):
    val (result, out) = liftSource(
      "declaration",
      """  inline def invariant(b: PropertyBuilder[State, Outcome, Fact], predicate: Step[State, Outcome, Fact] => Boolean): Property[State] =
    b.expect(check(!predicate(after)))
  inline def latched(predicate: Step[State, Outcome, Fact] => Boolean): Monitor[State, Outcome, Fact, Boolean] =
    monitor[State, Outcome, Fact, Boolean](false)((broken, before, after) => broken || !predicate(after))(broken => broken)
""",
      "false",
      """val checked = kit.invariant(MachineUnderTest.property, _.state == State.done)
val namedBefore = MachineUnderTest.property.never(_.state == State.done).from(before = _ == State.idle)
def completed(after: Step[State, Outcome, Fact]) = after.state == State.done
val restricted = kit.invariant(MachineUnderTest.property.when(press), completed)
object Watched extends Machine[State, Outcome, Fact]:
  val init = State.idle
  def end(s: State) = false
  object monitors:
    val watched = kit.latched(_.state == State.done)
  object rules extends Bindings(press ~> ((s: State) => List(Step(Outcome.accepted, s))))
val restrictedCore = Watched.property.when(press).expect(check(!completed(after)))
""",
      Seq(
        "fixture.expansion.Source$package$.checked",
        "fixture.expansion.Source$package$.namedBefore",
        "fixture.expansion.Source$package$.restricted",
        "fixture.expansion.Source$package$.restrictedCore",
        "fixture.expansion.Watched"
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val model = mapper.readTree(Files.readString(out))
    val (_, _, property) = FixtureDeclarations(model, "declaration")
    assertEquals(property("restricted"), property("restrictedCore"))
    val functions =
      model.path("functions").elements().asScala.map(f => f.path("name").asText() -> f).toMap
    val holds = model.at("/properties/0/holds").asText()
    assertEquals(
      functions(holds).at("/body/unary/operand/call/function").asText(),
      s"$holds.invariant"
    )
    val monitor = model.at("/monitors/0/id").asText()
    assertEquals(
      functions(s"$monitor.next").at("/body/binary/right/unary/operand/call/function").asText(),
      s"$monitor.latched"
    )
    assert(functions.keys.forall(!_.startsWith("framework.")))

  test(
    "expansion binds types, function arguments, contextual values, defaults and repeated arguments"
  ):
    val (result, out) = liftSource(
      "binding",
      """  final case class Wanted[A](value: A)
  given Wanted[State] = Wanted(State.done)
  inline def selected[A](value: A, predicate: A => Boolean, enabled: Boolean = true)(using wanted: Wanted[A]): Boolean =
    enabled && predicate(value) && wanted.value == value
  inline def among[A](value: A, rest: A*): Boolean = rest.toList.contains(value)
  inline def outer(value: State): Boolean = selected(value, (s: State) => among(s, State.idle, State.done))
""",
      "kit.outer(s)"
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val body = mapper.readTree(Files.readString(out)).at("/machines/0/ends/lambda/body")
    assertEquals(body.at("/binary/op").asText(), "OP_AND")
    assert(body.toString.contains("OP_CONTAINS"), body.toPrettyString)
    assert(!body.toString.contains("\"call\""), body.toPrettyString)

  test("generated nodes use the call line and explicit arguments retain their lines"):
    val (result, out) = liftSource(
      "positions",
      "  inline def selected(value: State): Boolean = value == State.done",
      "kit.selected(\n    s\n  )"
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val body = mapper.readTree(Files.readString(out)).at("/machines/0/ends/lambda/body")
    val source = Files.readString(out.resolveSibling("Source.scala")).linesIterator.toList
    val callLine = source.indexWhere(_.contains("def end")) + 1
    assertEquals(body.at("/position/line").asInt(), callLine)
    assertEquals(body.at("/binary/right/position/line").asInt(), callLine)
    assertEquals(body.at("/binary/left/position/line").asInt(), callLine + 1)

  test("a kit definition composes the captured member and exact fact key"):
    val (result, out) = liftSource(
      "member",
      """  final case class Pair(left: State, right: State)
  inline def selectedMember(after: Step[Pair, String, String]): Boolean = after.records(_.left, Fact.first)
""",
      "s == State.done",
      """object PairUnderTest extends Composition[kit.Pair](_.left -> MachineUnderTest, _.right -> MachineUnderTest):
  def end(s: State) = false
  object syncs extends Syncs
val checked = PairUnderTest.property expect(check(kit.selectedMember(after)))
""",
      Seq("fixture.expansion.Source$package$.checked")
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val model = mapper.readTree(Files.readString(out))
    val functions = model.path("functions").elements().asScala.toList
    assert(!functions.exists(_.path("name").asText().contains("selectedMember")))
    val holds = model.at("/properties/0/holds").asText()
    val body = functions.find(_.path("name").asText() == holds).get.path("body")
    assertEquals(body.at("/binary/op").asText(), "OP_CONTAINS")
    assertEquals(body.at("/binary/left/literal/text").asText(), "left_first")
    assertEquals(body.at("/binary/right/field/field").asText(), "facts")

  for (name, definitions, cycle) <- Seq(
      (
        "direct",
        "  inline def selected(value: State): Boolean = selected(value)",
        "selected -> fixture.expansion.kit$.selected"
      ),
      (
        "indirect",
        "  inline def selected(value: State): Boolean = other(value)\n  inline def other(value: State): Boolean = selected(value)",
        "selected -> fixture.expansion.kit$.other -> fixture.expansion.kit$.selected"
      )
    )
  do
    test(s"$name sugar recursion names its cycle"):
      val (result, out) = liftSource(name, definitions, "kit.selected(s)")
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains("sugar cycle:"), result.diagnostics)
      assert(result.diagnostics.contains(cycle), result.diagnostics)
      assert(!Files.exists(out))

  test("a kit helper cannot forward an opaque selector that native capture cannot read"):
    val (result, out) = liftSource(
      "opaque-member",
      """  final case class Pair(left: State, right: State)
  inline def selectedMember(after: Step[Pair, String, String], member: Pair => State): Boolean =
    after.records(member, Fact.first)
""",
      "s == State.done",
      """object PairUnderTest extends Composition[kit.Pair](_.left -> MachineUnderTest, _.right -> MachineUnderTest):
  def end(s: State) = false
  object syncs extends Syncs
val checked = PairUnderTest.property expect(check(kit.selectedMember(after, _.left)))
""",
      Seq("fixture.expansion.Source$package$.checked")
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(
      result.diagnostics.contains("selector capture cannot cross a plain function parameter"),
      result.diagnostics
    )
    assert(!Files.exists(out))

  for (name, definitions) <- Seq(
      "local" -> """  final case class Pair(left: State, right: State)
  inline def selectedMember(after: Step[Pair, String, String]): Boolean =
    val member: Pair => State = _.left
    after.records(member, Fact.first)
""",
      "object" -> """  final case class Pair(left: State, right: State)
  val member: Pair => State = _.left
  inline def selectedMember(after: Step[Pair, String, String]): Boolean = after.records(member, Fact.first)
"""
    )
  do
    test(s"a $name selector reference retains its direct-selector admission contract"):
      val (result, out) = liftSource(
        s"$name-member",
        definitions,
        "s == State.done",
        """object PairUnderTest extends Composition[kit.Pair](_.left -> MachineUnderTest, _.right -> MachineUnderTest):
  def end(s: State) = false
  object syncs extends Syncs
val checked = PairUnderTest.property expect(check(kit.selectedMember(after)))
""",
        Seq("fixture.expansion.Source$package$.checked")
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(
        result.diagnostics.contains("records reads the facts of the member one field"),
        result.diagnostics
      )
      assert(!Files.exists(out))

  test("an unliftable definition names the sugar and definition position"):
    val (result, out) = liftSource(
      "unliftable",
      "  inline def selected(value: State): Boolean = System.nanoTime() > 0",
      "kit.selected(s)"
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(
      result.diagnostics.contains("sugar fixture.expansion.kit$.selected defined at"),
      result.diagnostics
    )
    assert(result.diagnostics.contains("Source.scala:12"), result.diagnostics)
    assert(!Files.exists(out))

  for (name, definition) <- Seq(
      "unused" -> "  inline def selected(value: Boolean): Boolean = true",
      "conditional" -> "  inline def selected(value: Boolean): Boolean = false && value",
      "repeated" -> "  inline def selected(value: Boolean): Boolean = value && value"
    )
  do
    test(s"a by-value $name argument is evaluated once before the expanded body"):
      val (result, out) = liftSource(
        s"strict-$name",
        definition,
        "kit.selected(missing.reached)",
        "val missing = hole"
      )
      assertEquals(result.exit, 0, result.diagnostics)
      val body = mapper.readTree(Files.readString(out)).at("/machines/0/ends/lambda/body")
      assertEquals(
        body.at("/let/value/hole").asText(),
        "fixture.expansion.missing",
        body.toPrettyString
      )
      assert(!body.at("/let/body").toString.contains("\"hole\""), body.toPrettyString)

  test("strict argument calls are staged in caller order while by-name arguments stay in the body"):
    val (result, out) = liftSource(
      "strict-order",
      """  def reached(): Boolean = missing.reached
  inline def selected(first: Boolean, second: Boolean): Boolean = second && first
""",
      "kit.selected(kit.reached(), other.reached)",
      "val missing = hole\nval other = hole"
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val body = mapper.readTree(Files.readString(out)).at("/machines/0/ends/lambda/body")
    assertEquals(
      body.at("/let/value/call/function").asText(),
      "fixture.expansion.kit$.reached",
      body.toPrettyString
    )
    assertEquals(
      body.at("/let/body/let/value/hole").asText(),
      "fixture.expansion.other",
      body.toPrettyString
    )
    val (lazyResult, lazyOut) = liftSource(
      "lazy",
      "  inline def selected(value: => Boolean): Boolean = false && value",
      "kit.selected(missing.reached)",
      "val missing = hole"
    )
    assertEquals(lazyResult.exit, 0, lazyResult.diagnostics)
    val lazyBody = mapper.readTree(Files.readString(lazyOut)).at("/machines/0/ends/lambda/body")
    assertEquals(lazyBody.at("/binary/op").asText(), "OP_AND")
    assertEquals(lazyBody.at("/binary/right/hole").asText(), "fixture.expansion.missing")

  test("a nested caller invocation of the same marker is not a definition cycle"):
    val (result, out) = liftSource(
      "nested-call",
      "  inline def negated(value: Boolean): Boolean = !value",
      "kit.negated(kit.negated(s == State.done))"
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val body = mapper.readTree(Files.readString(out)).at("/machines/0/ends/lambda/body")
    assertEquals(body.at("/unary/operand/unary/op").asText(), "OP_NOT", body.toPrettyString)

  test("an opaque function producer cannot silently discard its eager Hole read"):
    val (result, out) = liftSource(
      "function-producer",
      """  def makeFn(): Boolean => Boolean =
    val forced: Boolean = missing.reached
    value => value && forced
  inline def selected(predicate: Boolean => Boolean): Boolean = true
""",
      "kit.selected(kit.makeFn())",
      "val missing = hole"
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("function-producing argument"), result.diagnostics)
    assert(!Files.exists(out))

  for (name, expression) <- Seq(
      "named" -> "kit.selected(s, kit.predicate)",
      "default" -> "kit.selected(s)"
    )
  do
    test(s"a $name callback retains structural function binding"):
      val (result, out) = liftSource(
        s"callback-$name",
        """  def predicate(value: State): Boolean = value == State.done
  inline def selected(value: State, predicate: State => Boolean = kit.predicate): Boolean = predicate(value)
""",
        expression
      )
      assertEquals(result.exit, 0, result.diagnostics)
      val model = mapper.readTree(Files.readString(out))
      val body = model.at("/machines/0/ends/lambda/body")
      assertEquals(
        body.at("/call/function").asText(),
        "fixture.expansion.kit$.predicate",
        body.toPrettyString
      )
      assertEquals(body.at("/call/args/0/var").asText(), "s", body.toPrettyString)
      val callback = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == "fixture.expansion.kit$.predicate")
        .get
      assertEquals(callback.path("params").size(), 1)
      assertEquals(callback.at("/body/binary/op").asText(), "OP_EQ")
