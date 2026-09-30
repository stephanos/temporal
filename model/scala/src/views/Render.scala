package views

import java.nio.file.{Files, Paths}
import umpire.*
import views.Views.Declarations

object Render:
  /** Every checked-in view, by file name. */
  def all: Checked[Map[String, String]] = checked {
    val tables = Seq(
      "nexusProduct" -> nexuscaller.nexusProduct, "nexusProtocol" -> nexuscaller.nexusProtocol,
      "activityProduct" -> standaloneactivity.activityProduct, "activityProtocol" -> standaloneactivity.activityProtocol)
    val machineViews = tables.flatMap { (name, m) =>
      val t = m.table.get
      Seq(s"$name-table.md" -> Views.table(t), s"$name-diagram.md" -> Views.diagram(t))
    }
    val nexus = Views.summary(Declarations("Nexus caller",
      Seq(nexuscaller.nexusProduct, nexuscaller.nexusProtocol, nexuscaller.nexusCaller),
      nexuscaller.functionalQueries :+ nexuscaller.terminalHolds :+ nexuscaller.stoppedWorkerRepliesNothing,
      Seq(nexuscaller.nexusCallerTests, nexuscaller.nexusCallerCanary, nexuscaller.nexusCallerExploration))).get
    val activity = Views.summary(Declarations("Standalone activity",
      Seq(standaloneactivity.activityProduct, standaloneactivity.activityProtocol, standaloneactivity.standaloneActivity),
      standaloneactivity.functionalQueries :+ standaloneactivity.terminalHolds :+ standaloneactivity.pauseHolds :+
        standaloneactivity.stoppedWorkerStartsNothing,
      Seq(standaloneactivity.standaloneActivityTests, standaloneactivity.standaloneActivityCanary,
        standaloneactivity.standaloneActivityExploration))).get
    val diff = Views.diff("activityProduct: adding the caller's controls",
      standaloneactivity.productWithoutControls.table.get, standaloneactivity.activityProduct.table.get)
    (machineViews ++ Seq("nexusCaller-summary.md" -> nexus, "standaloneActivity-summary.md" -> activity,
      "activityProduct-controls-diff.md" -> diff)).toMap
  }

/** Writes every view into a directory: `scala-cli run src --main-class views.renderViews -- <dir>`. */
@main def renderViews(out: String): Unit =
  val dir = Paths.get(out)
  Files.createDirectories(dir)
  Render.all match
    case Left(e) =>
      System.err.println(s"render: $e")
      sys.exit(1)
    case Right(views) => views.foreach((name, content) => Files.writeString(dir.resolve(name), content))
