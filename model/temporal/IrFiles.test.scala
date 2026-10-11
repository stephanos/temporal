package temporal
// Every IR file's roots, constructed as Scala: the model gate runs this, so a machine whose rules
// overlap, which its `rules` object refuses as it is constructed, fails the gate (fn-126 R16). Every
// machine an IR file lifts must be among those constructed, so none escapes that check.

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import temporal.actors.client.Client
import framework.IrFile

class IrFilesTest extends munit.FunSuite:
  val declared: Vector[IrFile] = IrFile.discover("temporal")

  // The checked-in IR files, by name: model/ir/<name>.json, beside no sidecar.
  def checkedIn: Set[String] =
    val ir = Path.of("model/ir")
    assert(
      Files.isDirectory(ir),
      "model/ir is missing: the Models' tests run at the repository's root"
    )
    val stream = Files.list(ir)
    try
      stream.iterator.asScala
        .map(_.getFileName.toString)
        .filter(n => n.endsWith(".json") && !n.endsWith(".lint.json"))
        .map(_.stripSuffix(".json"))
        .toSet
    finally stream.close()

  // The names of the machines an IR file lifts: its `machines`, each by its `name`.
  def liftedMachines(file: String): Vector[String] =
    import org.json4s.*
    val ir = org.json4s.jackson.JsonMethods.parse(Files.readString(Path.of(s"model/ir/$file.json")))
    val rows = (ir \ "machines").children ++ (ir \ "compositions").children
    rows
      .map(row =>
        (row \ "family", row \ "name") match
          case (JString(family), JString(name)) => s"$family.$name"
          case other => fail(s"$file has an unnamed lifted subject: $other")
      )
      .toVector

  def declarationKey(model: framework.Model): String =
    IrFile.identity(model).split('.').dropRight(1).mkString(".") + "." + model.name

  def assertComplete(actual: Seq[String], expected: Seq[String], context: String): Unit =
    assertEquals(actual.distinct.size, actual.size, s"$context repeats a declaration identity")
    assertEquals(expected.distinct.size, expected.size, s"$context repeats a lifted identity")
    assertEquals(actual.toSet, expected.toSet, s"$context dropped or added a subject")

  test("every IR file is declared by a feature's exports, and every root of each constructs") {
    assertComplete(declared.map(_.name), checkedIn.toVector, "IR exports")
    for file <- declared do
      assertComplete(file.subjects().map(declarationKey), liftedMachines(file.name), file.name)
      file.construct(): Unit
  }

  test("completeness rejects a dropped whole export, a dropped machine and duplicate identities") {
    val names = declared.map(_.name)
    val _ =
      intercept[munit.ComparisonFailException](assertComplete(names.tail, names, "dropped export"))
    val subjects = declared.head.subjects().map(declarationKey)
    val _ = intercept[munit.ComparisonFailException](
      assertComplete(subjects.tail, subjects, "dropped machine")
    )
    val _ = intercept[munit.ComparisonFailException](
      assertComplete(subjects :+ subjects.head, subjects, "duplicate identity")
    )
  }

  test("standalone public API callers share the Temporal Client actor") {
    val standaloneClient: Client = features.activity.standalone.client
    val operationClient: Client = features.nexus.standalone.client
    assertEquals(standaloneClient.name, "client")
    assertEquals(operationClient.name, "client")
  }

  test("every model that names phases declares Phased") {
    import features.activity.standalone.{product as activityProduct, system as activitySystem}
    import features.nexus.shared.product as nexusProduct
    import features.nexus.standalone.protocol as nexusStandalone
    import features.nexus.workflow.protocol as nexusWorkflow

    val projected: Seq[AnyRef] = Seq(
      activityProduct.ActivityProduct,
      activitySystem.ActivitySystem,
      activitySystem.ActivityRecord,
      activitySystem.HeldDispatch,
      activitySystem.StandaloneActivity,
      activitySystem.RecordOverQueue,
      activitySystem.RecordOverMatching,
      nexusProduct.NexusProduct,
      nexusStandalone.NexusProtocol,
      nexusWorkflow.NexusProtocol,
      nexusWorkflow.TrustingCaller,
      nexusWorkflow.RejectAfterClose,
      nexusWorkflow.NexusCaller,
      actors.worker.Polling,
      foundations.taskqueue.product.TaskQueueProduct,
      foundations.taskqueue.system.TaskQueueSystem
    )
    for model <- projected do
      model match
        case _: framework.Phased[?, ?] => ()
        case _                         => fail(model.getClass.getName)
  }
