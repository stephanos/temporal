package views

// Every view renders the same bytes twice and equals its golden in model/scala/goldens/views. The
// goldens are also compared with the Go goldens: the views are the same code over the same tables,
// so every line matches except the ones listed below, each with its reason.

import java.nio.file.Files
import testsupport.Lean

class CheckedViews extends munit.FunSuite:
  val rendered: Map[String, String] = Render.all.fold(e => fail(e.toString), identity)

  test("the views are deterministic and match the goldens") {
    assertEquals(Render.all, Right(rendered))
    for (name, content) <- rendered do
      val golden = Lean.root.resolve(s"model/scala/goldens/views/$name")
      assert(Files.exists(golden), s"missing golden $name")
      assertEquals(content, Files.readString(golden), name)
  }

  /** Lines that differ from the Go golden, by file: (Go line, Scala line). */
  val expectedDifferences: Map[String, Set[(String, String)]] = Map(
    // Scala's evidence is a total function over the facts, so the product machine names evidence for
    // the scheduled fact too, which no product step records; Go and Lean declare no line for it.
    "nexusCaller-summary.md" -> Set(
      "- evidence: nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut" ->
        "- evidence: nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut"),
    // The evidence lines come out in the facts' catalog order, where Go keeps declaration order and
    // declares the attempt count first.
    "standaloneActivity-summary.md" -> Set(
      "- evidence: attemptCount, statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut" ->
        "- evidence: statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut, attemptCount"),
  )

  test("the views equal the Go views except the listed lines") {
    for (name, content) <- rendered do
      val goLines = Files.readString(Lean.root.resolve(s"model/go/views/testdata/$name")).linesIterator.toVector
      val scalaLines = content.linesIterator.toVector
      assertEquals(scalaLines.size, goLines.size, name)
      val differing = goLines.zip(scalaLines).filter(_ != _).toSet
      assertEquals(differing, expectedDifferences.getOrElse(name, Set.empty), name)
  }
