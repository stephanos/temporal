package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

class BlockFormRuleSuite extends munit.FunSuite:
  private def corpus(files: (String, String)*)(check: Path => Unit): Unit =
    val root = Files.createTempDirectory("block-form-")
    try
      files.foreach: (name, source) =>
        val path = root.resolve(name)
        Files.createDirectories(path.getParent)
        Files.writeString(path, source)
      check(root)
    finally
      val stream = Files.walk(root)
      try stream.iterator.asScala.toVector.reverse.foreach(Files.deleteIfExists(_))
      finally stream.close()

  test("an on whose cases are in parentheses is found at its line, and a block is not"):
    val source =
      """object rules extends Rules:
        |  on(client.pause)(when(started) ~> effects.pause)
        |  on(client.unpause) {
        |    when(paused) ~> effects.unpause
        |  }
        |  on(worker.poll, worker.respond)(
        |    when(started) ~> effects.poll
        |  )
        |  on(client.control(Control.pause)) { when(started) ~> effects.pause }
        |""".stripMargin
    assertEquals(BlockFormRule.parenthesized(source), Vector(2, 6))

  test("an on in a string or a comment, a selection's on and a derivation's block are no finding"):
    val source =
      """// on(x)(case) in a comment.
        |val text = "on(x)(case)"
        |val start = action(this).on(activity)(using given)
        |object Stuck extends Derived(Switch.rebind(on(hand.press) {
        |  always ~> Switch.effects.wear
        |}))
        |""".stripMargin
    assertEquals(BlockFormRule.parenthesized(source), Vector.empty)

  test("native DSL brace blocks report their file and opening call line"):
    val source =
      """object rules extends Rules {
        |  from(owner) {
        |    on(
        |      action(input(nested(1)), { val x = 2; x })
        |    ) { always ~> effects.go }
        |  }
        |}
        |val step = effect[State, Answer, Fact] { record(done) }
        |val predicate = is { phase == ready }
        |val promise = property after(action()) expect {
        |  oneOf {
        |    branch { check(true) }
        |    branch { check(false) }
        |  }
        |}
        |val call = rpc(service, method) { field(_.child) { field(_.name) := value } }
        |val poll = readUntil(evidence, role, condition, None) { field(_.id) := value }
        |val message = proto[Message] { field(_.name) := value }
        |val amended = call.withFields { field(_.id) := value }
        |val another = call.extended { field(_.id) := value }
        |on(action(nested(1)))(always ~> effects.go)
        |""".stripMargin
    corpus("model/framework/samples/Native.scala" -> source): root =>
      val findings = BlockFormRule.findings(root)
      val lines = findings.map(_.split(':')(1).toInt)
      assertEquals(lines, Vector(1, 2, 3, 8, 9, 10, 11, 12, 13, 16, 16, 17, 18, 19, 20, 21))
      assert(findings.forall(_.startsWith("model/framework/samples/Native.scala:")))
      assert(findings.dropRight(1).forall(_.contains("colon")), findings.mkString("\n"))
      assert(findings.last.contains("on(...):"), findings.last)

  test("colon blocks, declarations, required braces and unrelated calls pass"):
    val source =
      """import framework.dsl.{is, effect}
        |object rules extends Rules:
        |  from(owner):
        |    on(action(nested(1))):
        |      always ~> effects.go
        |val predicate = is:
        |  phase == ready
        |val promise = property expect:
        |  oneOf:
        |    branch:
        |      check(true)
        |    branch:
        |      check(false)
        |val message = proto[Message]:
        |  field(_.child):
        |    field(_.name) := value
        |def effect[A](body: A): A = { body }
        |class branch { val value = 1 }
        |object rules { val value = 1 }
        |val selected = unrelated.effect { value }
        |val other = unrelated.on(action)(value)
        |val ordinary = collection.map { value => value }
        |// expect { oneOf { branch { check(false) } } }
        |/* on(action)(value) /* effect { nested } */ */
        |val text = "on(action)(value); effect { value }"
        |val multi = s\"\"\"expect { branch { value } }\"\"\"
        |""".stripMargin.replace("\\\"", "\"")
    corpus("model/framework/dsl/Native.test.scala" -> source): root =>
      assertEquals(BlockFormRule.findings(root), Vector.empty)

  test("discovery visits native files and prunes generated, cache and tool trees"):
    val native = Vector("dsl/Example.test.scala", "realize/Example.scala", "samples/Example.scala")
    val ignored = Vector(
      "dsl/build/Hidden.scala",
      "dsl/generated/Hidden.scala",
      "samples/.scala-build/Hidden.scala",
      "realize/.bsp/Hidden.scala",
      "check/Hidden.scala",
      "export/testdata/Hidden.scala"
    )
    val files = (native ++ ignored).map(path => s"model/framework/$path" -> "effect { value }")
    corpus(files*): root =>
      var visited = Vector.empty[String]
      val findings = BlockFormRule.findings(
        root,
        path =>
          visited :+= root.relativize(path).toString
          true
      )
      assertEquals(visited.sorted, native.map("model/framework/" + _).sorted)
      assertEquals(findings.size, 3)
      assertEquals(BlockFormRule.findings(root, _ => false), Vector.empty)

  test("the native repository scan visits authored framework files"):
    var visited = 0
    val findings = BlockFormRule.findings(
      Tools.here.directory,
      _ =>
        visited += 1
        true
    )
    assert(visited > 0, "native source discovery must not pass an empty scan")
    println(s"Native block-form scan: $visited files, ${findings.size} findings")
    findings.foreach(println)

  test("selected expectation builders and qualified sugar require colon blocks"):
    val braces =
      """import framework.dsl as Syntax
        |val same = property.expect { check(true) }
        |val transition = property.across.expect { check(true) }
        |val predicate = expect.across { check(true) }
        |val state = Syntax.is { true }
        |val step = Syntax.effect { record(done) }
        |val checked = Syntax.expect { check(true) }
        |val qualified = framework.dsl.effect { record(done) }
        |val full = framework.dsl.is { true }
        |""".stripMargin
    corpus("model/framework/samples/Selected.scala" -> braces): root =>
      assertEquals(BlockFormRule.findings(root).map(_.split(':')(1).toInt), (2 to 9).toVector)
    val colons = braces.replace(" {", ":\n  ").replace(" }", "")
    corpus("model/framework/samples/Selected.scala" -> colons): root =>
      assertEquals(BlockFormRule.findings(root), Vector.empty)
