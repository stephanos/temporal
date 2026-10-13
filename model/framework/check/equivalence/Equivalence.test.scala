package framework.check
package equivalence

import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.GZIPOutputStream

// The equivalence check's comparisons, over tables and IR written here as the step tables
// (model/framework/dsl/StepTables.scala) and the IR generator write them. The negative controls are
// the table pins' (model/testdata/table-pins/mutants.py): a changed row, result, alternative
// order, fact order, binding order, explanation or pair, and a missing or extra machine.
class EquivalenceSuite extends munit.FunSuite:
  private val product = Vector(
    "init\tscheduled",
    "bindings\tclient.start worker.poll system.timeout",
    "state\tscheduled\tend=false",
    "state\tstarted\tend=false",
    "state\tcompleted\tend=true",
    "step\tclient.start[unset]\tscheduled\t{ accepted | scheduled | [statusScheduled,timerArmed] | scheduled once }",
    "step\tclient.start[unset]\tstarted\t{  }",
    "step\tclient.start[unset]\tcompleted\t{  }",
    "step\tworker.poll[]\tscheduled\t{ ok | started | [statusStarted] |  ; ok | scheduled | [] | lost }",
    "step\tworker.poll[]\tstarted\t{  }",
    "step\tworker.poll[]\tcompleted\t{  }",
    "step\tsystem.timeout[]\tscheduled\t{  }",
    "step\tsystem.timeout[]\tstarted\t{ timedOut | completed | [statusTimedOut] |  }",
    "step\tsystem.timeout[]\tcompleted\t{  }"
  )
  private val queue = Vector(
    "init\tempty",
    "bindings\tworker.poll",
    "state\tempty\tend=true",
    "step\tworker.poll[]\tempty\t{  }"
  )
  private val baseline = Map("f.activityProduct" -> product, "f.taskQueue" -> queue)

  private def hex(bytes: Array[Byte]) =
    MessageDigest.getInstance("SHA-256").digest(bytes).map(b => f"${b & 0xff}%02x").mkString

  // A directory of tables as StepTables.write writes it.
  private def tables(models: Map[String, Vector[String]]): Path =
    val dir = Scratch.directory("umpire-tables")
    val index = models.toVector
      .sortBy(_._1)
      .map: (key, lines) =>
        val bytes = lines.map(_ + "\n").mkString.getBytes(UTF_8)
        val digest = hex(bytes)
        val out = GZIPOutputStream(Files.newOutputStream(dir.resolve(digest + ".steps.gz")))
        try out.write(bytes)
        finally out.close()
        s"$key\t$digest\t${lines.size}"
    Files.writeString(dir.resolve("index.tsv"), index.mkString("", "\n", "\n"))
    dir

  private def compare(now: Map[String, Vector[String]], map: String = ""): Vector[String] =
    val renames = RenameMap.parse(map)
    val found = Equivalence.compareTables(tables(baseline), tables(now), renames)
    val (unused, _) =
      Equivalence.unused(renames, found.removalsUsed, true, Scratch.directory("b"))
    found.differences ++ unused

  private def changed(f: Vector[String] => Vector[String]) =
    baseline.updated("f.activityProduct", f(product))

  private def replaced(from: String, to: String) =
    changed(_.map(l => if l.equals(from) then to else l))

  test("an unchanged tree is equivalent under the empty map"):
    assertEquals(compare(baseline), Vector.empty)

  test("a one-fact model mutation is rejected"):
    val found = compare(changed(_.map(_.replace("[statusStarted]", "[statusScheduled]"))))
    assert(found.exists(_.contains("table f.activityProduct: line 9 differs")), found)

  test("every named mutant of the table pins is rejected"):
    val row = product(8)
    val mutants = Map(
      "row" -> replaced(row, row.replace("\tscheduled\t", "\tcompleted\t")),
      "result" -> replaced(row, row.replace("ok | started", "ok | completed")),
      "result-order" -> replaced(
        row,
        "step\tworker.poll[]\tscheduled\t{ ok | scheduled | [] | lost ; ok | started | [statusStarted] |  }"
      ),
      "fact-order" -> replaced(
        product(5),
        product(5).replace("statusScheduled,timerArmed", "timerArmed,statusScheduled")
      ),
      "binding-order" -> replaced(product(1), "bindings\tworker.poll client.start system.timeout"),
      "explanation" -> replaced(
        product(5),
        product(5).replace("scheduled once", "scheduled twice")
      ),
      "pair" -> replaced(
        product(6),
        "step\tclient.start[unset]\tstarted\t{ accepted | started | [] |  }"
      ),
      "missing-pair" -> changed(_.filterNot(_.equals(product(6)))),
      "missing-machine" -> (baseline - "f.taskQueue"),
      "extra-machine" -> baseline.updated("f.seededExtraMachine", queue)
    )
    for (name, now) <- mutants do assert(compare(now).nonEmpty, s"mutant escaped: $name")
    assert(
      compare(mutants("missing-machine")).exists(_.contains("f.taskQueue (as f.taskQueue) is gone"))
    )
    assert(compare(mutants("extra-machine")).exists(_.contains("f.seededExtraMachine is new")))
    assert(compare(mutants("missing-pair")).exists(_.contains("the baseline has 1 more lines")))

  test("renames explain renamed machines, actions, states and facts, and only those"):
    val renamed = Map(
      "g.activityProduct" -> product.map(
        _.replace("worker.poll", "activityWorker.poll").replace("started", "running")
      ),
      "g.taskQueue" -> queue.map(_.replace("worker.poll", "activityWorker.poll"))
    )
    val map = "f g  # the family moved\nworker.poll activityWorker.poll\nstarted running\n"
    assertEquals(compare(renamed, map), Vector.empty)
    // A word is renamed whole: `statusStarted` keeps its name.
    assert(product.exists(_.contains("statusStarted")))
    val missed = compare(renamed, "f g\nworker.poll activityWorker.poll\n")
    assert(missed.exists(_.contains("table g.activityProduct")), missed)

  test("an entry of the map that matches nothing fails, and so does an unused removal"):
    val found = compare(baseline, "noSuchName other\nf.gone -\nf.activityProduct/client.stop -\n")
    assert(found.exists(_.contains("the rename noSuchName matches nothing")), found)
    assert(found.exists(_.contains("the removal f.gone names no baseline table")), found)
    assert(
      found.exists(_.contains("the removal f.activityProduct/client.stop removes nothing")),
      found
    )

  test("a removal explains a removed table and a removed action's rows"):
    val withoutStart = changed(
      _.filterNot(_.startsWith("step\tclient.start")).map(
        _.replace("bindings\tclient.start ", "bindings\t")
      )
    ) - "f.taskQueue"
    assertEquals(
      compare(withoutStart, "f.taskQueue -\nf.activityProduct/client.start -\n"),
      Vector.empty
    )

  test("the map's lines are checked"):
    intercept[GateError](RenameMap.parse("a b c"))
    intercept[GateError](RenameMap.parse("a b\na c"))
    val map = RenameMap.parse("a b\nb a\nactivity-standalone-race activity-standalone\n")
    assertEquals(map("a.b-c a_b"), "b.a-c a_b")
    assertEquals(
      map("activity-standalone-race-held.stale-case.json activity-standalone-x"),
      "activity-standalone-held.stale-case.json activity-standalone-x"
    )

  // ---- The projection.

  private def tree(files: (String, String)*): Path =
    val dir = Scratch.directory("umpire-projection")
    for (name, text) <- files do Files.writeString(dir.resolve(name), text)
    dir

  private def machine(name: String, line: Int, fact: String = "statusStarted") =
    s"""{"family":"f","name":"$name","position":{"file":"model/F.scala","line":$line},"steps":[{"facts":["$fact"]}]}"""

  private val baseIr = tree(
    "a.json" -> s"""{"source":"model: f.A","machines":[${machine(
        "activityProduct",
        3
      )}],"queries":[{"name":"q","position":{"file":"x","line":1}}]}""",
    "a-race.json" -> s"""{"source":"model: f.B","machines":[${machine(
        "activityProduct",
        3
      )},${machine("held", 9)}]}""",
    "a.lint.json" -> """{"accepted":[{"kind":"k","owner":"activityProduct","subjects":["q"],"because":"b"}]}"""
  )
  private val baseCases = tree(
    "a-race-held.stale-case.json" -> """{"caseId":"temporal.case.scala.a-race.held.stale","provenance":{"definitions":[{"behaviorFingerprint":"sha256:1"}],"sources":[{"path":"model/F.scala","line":9,"column":1,"provenance":"scala-model"}]}}""",
    "manifest.json" -> """{"version":1,"queries":[{"model":"a-race.json","query":{"Name":"held.stale"},"file":"a-race-held.stale-case.json","unsupported":[{"Position":"model/F.scala:9"}]}]}"""
  )

  private def project(ir: Path, cases: Path, map: String): Vector[String] =
    val renames = RenameMap.parse(map)
    val found = Equivalence.compareProjection(baseIr, baseCases, ir, cases, renames)
    found ++ Equivalence.unused(renames, Set.empty, true, Scratch.directory("b"))._1

  test("the projection holds an unchanged tree equal"):
    assertEquals(project(baseIr, baseCases, ""), Vector.empty)

  test("merged IR files, moved positions and a renamed file's Cases are equivalent under the map"):
    val ir = tree(
      "a.json" -> s"""{"source":"model: f","machines":[${machine("activityProduct", 30)},${machine(
          "held",
          90
        )}],"queries":[{"name":"q","position":{"file":"y","line":2}}]}""",
      "a.lint.json" -> """{"accepted":[{"kind":"k","owner":"activityProduct","subjects":["q"],"because":"b"}]}"""
    )
    val cases = tree(
      "a-held.stale-case.json" -> """{"caseId":"temporal.case.scala.a.held.stale","provenance":{"definitions":[{"behaviorFingerprint":"sha256:2"}],"sources":[{"path":"model/G.scala","line":90,"column":1,"provenance":"scala-model"}]}}""",
      "manifest.json" -> """{"version":1,"queries":[{"model":"a.json","query":{"Name":"held.stale"},"file":"a-held.stale-case.json","unsupported":[{"Position":"model/G.scala:90"}]}]}"""
    )
    assertEquals(project(ir, cases, "a-race a\n"), Vector.empty)
    val unmapped = project(ir, cases, "")
    assert(unmapped.exists(_.contains("Case case a-race-held.stale-case.json is gone")), unmapped)

  test("the projection rejects a changed fact, a changed acceptance and an unused entry"):
    val ir = tree(
      "a.json" -> s"""{"machines":[${machine(
          "activityProduct",
          3,
          "statusScheduled"
        )}],"queries":[{"name":"q"}]}""",
      "a-race.json" -> s"""{"machines":[${machine("held", 9)}]}""",
      "a.lint.json" -> """{"accepted":[{"kind":"k","owner":"activityProduct","subjects":["q"],"because":"c"}]}"""
    )
    val found = project(ir, baseCases, "unusedName other\n")
    assert(
      found.exists(d => d.startsWith("IR machines") && d.contains("name=activityProduct differs")),
      found
    )
    assert(found.exists(_.startsWith("accepted lint finding")), found)
    assert(found.exists(_.contains("the rename unusedName matches nothing")), found)

  test("declarations of one name in two families are two, each qualified by its machine's family"):
    def file(family: String, total: Int) =
      s"""{"machines":[{"family":"$family","name":"nexusSystem"}],"functions":[{"name":"nexusSystem.rules.reply","body":"$family"}],"queries":[{"name":"q","scenario":{"machine":"nexusSystem","name":"s"},"total":"$total"}]}"""
    val two = tree("standalone.json" -> file("a", 42), "workflow.json" -> file("b", 576))
    val (declared, conflicts) = Equivalence.declarations(two, RenameMap.empty)
    assertEquals(conflicts, Vector.empty)
    assertEquals(declared.size, 6)
    assertEquals(
      Equivalence.compareProjection(two, baseCases, two, baseCases, RenameMap.empty),
      Vector.empty
    )
