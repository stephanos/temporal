package framework.lint

import framework.`export`.*

import java.nio.file.Files
import scala.jdk.CollectionConverters.*
import scala.concurrent.duration.*
import framework.check.Tools
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class EligibleWitnessFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/framework/export/testdata/eligibleWitness/"
  private val scratch =
    retained(build.resolve("history"), "eligible-witness.")
  private def fixture(positive: Boolean) =
    val from = tools.directory.resolve(stored)
    val directory = Files.createDirectories(
      scratch.resolve(if positive then "positive" else "negative")
    )
    val files = Files.walk(from)
    try
      for
        file <- files.iterator.asScala
        if Files.isRegularFile(file) && file.toString.endsWith(".scala")
      do
        val target = directory.resolve(from.relativize(file))
        Files.createDirectories(target.getParent): Unit
        var text = Files
          .readString(file)
          .replace(
            "../../../../build/model-scala.jar",
            build.resolve("model-scala.jar").toString
          )
        if positive then
          for (old, current) <- Seq(
              "query find properties.safety.privateClaim in privatePath limits two" -> "witness(worker.finish(Bit.first)).records(Fact.done).ends(_.lit).limits(two)",
              "query find properties.safety.distinctInputClaim in distinctInputPath limits two" -> "witness(worker.finish(Bit.first), worker.finish(Bit.second)).records(Fact.done).limits(two)",
              "query find properties.safety.factoryClaim in factoryPath limits two" -> "witness(worker.factory).records(Fact.done).limits(two)",
              "query find properties.safety.coreClaim in corePath limits two" -> "witness(worker.core).records(Fact.done).limits(two)",
              "query find properties.safety.negativeClaim in negativePath limits two" -> "witness(worker.negativeFact).excludes(Fact.done).limits(two)"
            )
          do text = text.replace(old, current)
        Files.writeString(target, text): Unit
    finally files.close()
    val out = directory.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          directory.toString,
          "-f",
          "-o",
          out.toString
        )
      )
      .orFail(): Unit
    out
  private lazy val jar = fixture(false)

  test(
    "typed private find lint distinguishes exact input classes and independent retained boundaries"
  ):
    val output = scratch.resolve("refused.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        "fixture.features.witnesses.Witnesses$.queries"
      )
    )
    assert(result.failed, result.output)
    val findings = result.output.linesIterator
      .filter(_.contains("R5.eligible-witness"))
      .toVector
    assertEquals(findings.size, 5, result.output)
    val authored = Files
      .readAllLines(
        tools.directory.resolve(stored + "witnesses/Witnesses.scala")
      )
      .asScala
    for name <- Vector(
        "eligible",
        "distinctInputsEligible",
        "factoryEligible",
        "coreEligible",
        "negativeRecords"
      )
    do
      val line = authored.indexWhere(_.trim.startsWith(s"val $name =")) + 1
      assert(
        findings.exists(s => s.contains(s"Witnesses.scala:$line:") && s.contains(s"`$name`")),
        result.output
      )
    for retained <- Vector(
        "shared",
        "invariant",
        "monitorOnly",
        "sharedScenario",
        "sharedScenarioOther",
        "repeated",
        "customStart",
        "nonRecorded",
        "fakeRecords",
        "unrelatedContains",
        "statementReader",
        "parentReader"
      )
    do assert(!findings.exists(_.contains(s"`$retained`")), result.output)
    assert(!Files.exists(output), result.output)

  test(
    "migrated private witnesses admit independently retained typed boundaries"
  ):
    val positive = fixture(true)
    val output = scratch.resolve("positive.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$positive=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        "fixture.features.witnesses.Witnesses$.queries"
      )
    )
    assertEquals(result.exit, 0, result.output)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(output))
    assertEquals(model.path("queries").size(), 17)
    val scenarios = model.path("scenarios").elements().asScala.toVector
    val repeated = scenarios
      .find(_.path("name").asText() == "repeatedPath")
      .get
      .path("actions")
    assertEquals(repeated.size(), 2)
    assertEquals(repeated.get(0), repeated.get(1))
    val distinct = scenarios
      .find(_.path("name").asText() == "distinctInputsEligible")
      .get
      .path("actions")
    assertNotEquals(distinct.get(0), distinct.get(1))
    assert(
      model
        .path("queries")
        .elements()
        .asScala
        .exists(q =>
          q.path("name").asText() == "sharedScenarioOther" && q
            .path("form")
            .asText() == "FORM_VERIFY"
        )
    )
