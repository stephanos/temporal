package umpire.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*

class BoundsInventoryTest extends munit.FunSuite:
  private val expected = Map(
    "two" -> (2, 2, 512),
    "three" -> (3, 3, 4096),
    "four" -> (4, 4, 32768),
    "five" -> (5, 5, 65536),
    "six" -> (6, 6, 262144),
    "seven" -> (7, 7, 262144),
    "eight" -> (8, 8, 262144),
    "twelve" -> (12, 12, 262144),
    "fourWide" -> (4, 4, 1048576),
    "fiveWide" -> (5, 5, 1048576),
    "twelveWide" -> (12, 12, 4194304)
  )
  private def parse(text: String) = dialects.Scala3(Input.String(text)).parse[Source].get
  private def constructors(tree: Tree) = tree.collect {
    case a: Term.Apply if Set("Limits", "framework.Limits")(a.fun.syntax) => a
  }
  private def inventory(text: String): Unit =
    val tree = parse(text)
    val bounds = tree.collect { case o: Defn.Object if o.name.value == "Bounds" => o }
    assertEquals(bounds.size, 1)
    val entries = bounds.head.templ.body.stats.collect {
      case v: Defn.Val if constructors(v.rhs).nonEmpty =>
        val name = v.pats.head.asInstanceOf[Pat.Var].name.value
        val call = constructors(v.rhs).head
        def integer(t: Term): Int = t match
          case Lit.Int(n)                                                      => n
          case a: Term.ApplyInfix if a.lhs.syntax == "1" && a.op.value == "<<" =>
            1 << a.argClause.values.head.asInstanceOf[Lit.Int].value
          case _ => fail("bound fields must be literal integers or powers of two")
        val fields = call.argClause.values.map {
          case Term.Assign(Term.Name(key), value) => key -> integer(value)
          case _                                  => fail("central bounds use named fields")
        }.toMap
        name -> (fields("steps"), fields("actions"), fields("search"))
    }
    assertEquals(constructors(tree).size, 11)
    assertEquals(entries.size, 11)
    assertEquals(entries.map(_._1).distinct.size, 11)
    assertEquals(entries.map(_._2).distinct.size, 11)
    assertEquals(entries.toMap, expected)
    val progress = bounds.head.templ.body.stats.collect {
      case v: Defn.Val if v.pats.head.syntax == "dispatchSteps" => v.rhs.syntax
    }
    assertEquals(progress, List("1"))

  test(
    "complete central inventory has one name per independent budget and a plain progress integer"
  ):
    val source = Files.readString(Path.of("model/temporal/Bounds.scala"))
    inventory(source)
    intercept[munit.ComparisonFailException](
      inventory(source.replace("search = 512", "search = 513"))
    )
    intercept[munit.ComparisonFailException](
      inventory(source + "\nval extra = Limits(steps = 2, actions = 2, search = 512)\n")
    )
    intercept[munit.ComparisonFailException](
      inventory(source.replace("val twelveWide", "val fourWide"))
    )

  test("selected Temporal author roots declare no Limits outside Bounds"):
    val roots = List("foundations", "actors", "capabilities", "realize", "features/activity_mini")
      .map(root => Path.of("model/temporal").resolve(root))
    val files = roots.flatMap { root =>
      val walked = Files.walk(root)
      try
        walked.iterator.asScala
          .filter(p =>
            p.toString.endsWith(".scala") &&
              !p.toString.endsWith(".test.scala")
          )
          .toList
      finally walked.close()
    } :+ Path.of("model/temporal/features/activity/Activity.scala")
    assert(files.size > 30)
    for file <- files do
      assertEquals(constructors(parse(Files.readString(file))), Nil, file.toString)
