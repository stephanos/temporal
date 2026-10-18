package framework.lint

import java.nio.file.Files
import java.nio.file.Path
import framework.`export`.ModelBuild

class MarkersFixtures extends munit.FunSuite with LintFixtures:
  // fn-126 decision 20: what a machine is for, as its markers say (lifts/MarkerRejects.scala).
  private val markerRejects: Seq[String] =
    Seq(
      "FaultUnmarked",
      "FaultlessFailure",
      "HopelessFailure$.queries$.brokenHopelessFailure",
      "UnrefutedControl$.queries$.foundUnrefutedControl",
      "ControlRefiner",
      "RefinedControl$.queries$.askedRefinedControl",
      "SystemicControl$.queries$.askedSystemicControl",
      "TornMarkers$.queries$.askedTornMarkers",
      "UnmarkedPair",
      "FaultlessPair",
      "DerivedControl$.queries$.foundDerivedControl"
    ).map("fixture.markers." + _)

  test("every marker refusal retains its text and source position, and writes no IR"):
    val sources = Files.createDirectories(scratch.resolve("markers"))
    val copied = ModelBuild.copied(testdata.resolve("lifts"), scratch.resolve("stored"))
    for file <- Seq("project.scala", "MarkerRejects.scala") do
      Files.copy(copied.resolve(file), sources.resolve(file)): Unit
    val jar = packaged("markers", sources)
    val out = scratch.resolve("markers.json")
    val result = lift(
      (Seq(
        s"$jar=${stored("lifts")},$modelJar=model/",
        modelClasspath.toString,
        out.toString
      ) ++ markerRejects)*
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(!Files.exists(out), result.diagnostics)
    val expected = RefusalContract
      .authored(testdata, testdata.resolve("lifts/expected"))
      .filter(_.contains("/MarkerRejects.scala:"))
    assertEquals(refused(result).toVector, expected)
    val checkedOut = scratch.resolve("checked-markers.json")
    val checked = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.checkAndLift",
        s"$jar=${stored("lifts")},$modelJar=model/",
        modelClasspath.toString,
        checkedOut.toString
      ) ++ markerRejects
    )
    assertEquals(checked.exit, 1, checked.diagnostics)
    assert(!Files.exists(checkedOut), checked.diagnostics)
    assertEquals(refused(checked).toVector, expected)
    assertEquals(
      checked.output.linesIterator.filter(_.startsWith("== ")).toList,
      List("== lint admitted sources", "== lift IR", "== check lifted markers")
    )
