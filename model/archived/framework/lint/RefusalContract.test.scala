package framework.lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import framework.`export`.SweepHeld

private[framework] object RefusalContract extends munit.Assertions:
  def authored(testdata: Path, expected: Path): Vector[String] =
    val historicalBytes = Files.readAllBytes(expected.resolve("rejects.txt"))
    val digest =
      java.security.MessageDigest.getInstance("SHA-256").digest(historicalBytes)
    assertEquals(
      digest.map(b => f"${b & 0xff}%02x").mkString,
      // fn-159.10 reworded the refusal of an input without a default (Rejects.scala:573); fn-169.1
      // moved the fixtures to model/framework/export/testdata and the DSL to model/framework/dsl,
      // package framework.dsl, whose type names the refusals spell.
      "1508edff707b185b595a8bb221c94243f01359a65cb608386c19a9ed3497d66c"
    )
    val historical =
      Files.readAllLines(expected.resolve("rejects.txt")).asScala.toVector
    assertEquals(historical.size, 226)
    val prefix = "lift: model/framework/export/testdata/lifts/"
    val adaptations = Map(
      (prefix + "HintRejects.scala:44: 9223372036854775807 milliseconds is outside the range of a protobuf Duration") ->
        (prefix + "HintRejects.scala:44: an elapsed bound is outside the protobuf Duration range"),
      (prefix + "ScriptRejects.scala:203: inspect is no action activitySystem binds: a realization binds its own machine's actions") ->
        (prefix + "ScriptRejects.scala:203: inspect is no action activityProtocol binds: a realization binds its own machine's actions"),
      (prefix + "ScriptRejects.scala:105: expires is a case of Timeout, and activitySystem records facts of temporal.features.activity.standalone.system.Fact: a realization names the facts its machine records") ->
        (prefix + "ScriptRejects.scala:105: expires is a case of Timeout, and activityProtocol records facts of temporal.features.activity.standalone.protocol.Fact: a realization names the facts its machine records"),
      (prefix + "Rejects.scala:28: fixture.rejects.Rejects$package$.closed is a framework.dsl.Channel[fixture.rejects.Note]; a root is a machine, a composition, a Query, a list of Queries, a progress claim, a realization, or a machine's `capabilities` or `queries` section") ->
        (prefix + "Rejects.scala:28: fixture.rejects.Rejects$package$.closed is a framework.dsl.Channel[fixture.rejects.Note]; a root is a machine, a composition, a Property, a Query, a list of Queries, a progress claim, a claim bundle, a realization, or a machine's `properties`, `capabilities` or `queries` section")
    )
    assert(adaptations.keys.forall(old => historical.count(_ == old) == 1))
    val markerSource =
      Files.readAllLines(testdata.resolve("lifts/MarkerRejects.scala")).asScala
    val markerDeclarations = Map(
      35 -> "FaultUnmarked",
      41 -> "FaultlessFailure",
      47 -> "HopelessFailure",
      70 -> "UnrefutedControl",
      83 -> "RefinedControl",
      103 -> "SystemicControl",
      123 -> "TornMarkers",
      144 -> "UnmarkedPair",
      150 -> "FaultlessPair",
      158 -> "DerivedControl"
    )
    val markerCoordinates = markerDeclarations.map { (old, name) =>
      val matches = markerSource.zipWithIndex.filter((line, _) =>
        line.startsWith(s"object $name ") || line == s"object $name"
      )
      assertEquals(matches.size, 1, name)
      (prefix + s"MarkerRejects.scala:$old:") -> (prefix + s"MarkerRejects.scala:${matches.head._2 + 1}:")
    }
    def sourceLine(file: String, text: String): Int =
      val source = Files.readAllLines(testdata.resolve("lifts/" + file)).asScala
      val matches = source.zipWithIndex.filter((line, _) => line.trim == text)
      assertEquals(matches.size, 1, file + ": " + text)
      matches.head._2 + 1
    val formattedCoordinates = Map(
      (prefix + "HintRejects.scala:44:") -> (prefix + s"HintRejects.scala:${sourceLine("HintRejects.scala", "atMost = framework.realize.Elapsed.milliseconds(9223372036854775807L)")}:"),
      (prefix + "ScriptRejects.scala:219:") -> (prefix + s"ScriptRejects.scala:${sourceLine("ScriptRejects.scala", "ServerStep(")}:"),
      (prefix + "ScriptRejects.scala:226:") -> (prefix + s"ScriptRejects.scala:${sourceLine("ScriptRejects.scala", "object Replaced extends DerivesFrom(Restated, activityProtocol):") + 1}:")
    )
    val coordinates = markerCoordinates ++ formattedCoordinates
    val authored = historical.map { old =>
      val vocabulary = adaptations.getOrElse(old, old)
      coordinates.find((from, _) => vocabulary.startsWith(from)) match
        case Some((from, to)) => to + vocabulary.drop(from.length)
        case None             => vocabulary
    }
    // The refusals of a source held for the sweep are held with it (SweepHeld.sources).
    authored.filterNot(line =>
      SweepHeld.sources.exists(held =>
        line.startsWith(s"lift: model/framework/export/testdata/$held:")
      )
    )
