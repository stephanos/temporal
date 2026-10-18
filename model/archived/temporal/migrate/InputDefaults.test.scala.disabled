package temporal.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class InputDefaultsTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task10InputDefaults))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
  private val source = """package temporal
package features.activity
package mini
import framework.*
import temporal.features.activity.Timeout
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val maxAttempts = input[MaxAttempts]
"""

  test("the documented omission inventory supplies mini defaults and converges"):
    val expected = source
      .replace("input[Timeout]", "input[Timeout](default = Timeout.unset)")
      .replace("input[MaxAttempts]", "input[MaxAttempts](default = MaxAttempts.one)")
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, expected)
    assertEquals(rewrite(expected).fixed, expected)
    assertEquals(rewrite(expected).diagnostics, Nil)

  test("declared defaults and the inventoried shared required failure remain untouched"):
    val explicit = source
      .replace("input[Timeout]", "input[Timeout](default = Timeout.expires)")
      .replace("input[MaxAttempts]", "input[MaxAttempts](default = MaxAttempts.two)")
    assertEquals(rewrite(explicit).fixed, explicit)
    assertEquals(rewrite(explicit).diagnostics, Nil)
    val required =
      "package temporal.features.activity\nimport framework.*\nval failure = input[Failure]\n"
    assertEquals(rewrite(required).fixed, required)
    assertEquals(rewrite(required).diagnostics, Nil)

  test("unknown values, declarations, imports and ownership refuse the whole document"):
    for unsafe <- List(
        source + "val unknown = input[Boolean]\n",
        source.replace("input[MaxAttempts]", "input[Boolean]"),
        source + "import other.input\n",
        source + "def input[A] = other.input[A]\n",
        source.replace("val maxAttempts", "object Nested { val maxAttempts") + "}\n",
        source.replace("package mini", "package another"),
        source + "val escape = input[Timeout]\n",
        source + "val unknown = framework.input[Timeout]\n",
        source.replace(
          "input[MaxAttempts]",
          "declare[MaxAttempts]"
        ) + "import framework.{input as declare}\n",
        source.replace("input[MaxAttempts]", "elsewhere.maxAttempts"),
        "package temporal.features.activity.mini\nimport framework.*\nval maxAttempts = elsewhere.maxAttempts\n",
        source + "val Timeout = other.Timeout\n"
      )
    do
      val result = rewrite(unsafe)
      assertEquals(result.fixed, unsafe)
      assertEquals(result.diagnostics.size, 1, unsafe)
      assert(result.diagnostics.head.message.contains("manual inventory"))
