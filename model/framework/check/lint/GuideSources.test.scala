package framework.check
package lint

import java.nio.file.{Files, Path}

class GuideSourcesTest extends munit.FunSuite:
  test("every SEMANTICS Scala excerpt matches its cited source and line"):
    val text = Files.readString(Path.of("model/framework/SEMANTICS.md"))
    val cited = "(?s)Source: \\[.*?\\]\\(([^)#]+)#L([0-9]+)\\)\\.\\n\\n```scala\\n(.*?)\\n```".r
    val blocks = cited.findAllMatchIn(text).toVector
    assertEquals(blocks.size, "```scala".r.findAllIn(text).size)
    assertEquals(blocks.size, 31)
    def normalized(lines: Seq[String]) = lines.map(_.dropWhile(_ == ' ')).mkString("\n")
    for block <- blocks do
      val file = Path.of("model/framework").resolve(block.group(1))
      val line = block.group(2).toInt
      val excerpt = block.group(3).linesIterator.toVector
      val actual = Files.readString(file).linesIterator.drop(line - 1).take(excerpt.size).toVector
      assertEquals(normalized(excerpt), normalized(actual), s"$file:$line")
