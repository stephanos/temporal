package temporal
// Every IR file's roots, constructed as Scala: the model gate runs this, so a machine whose rules
// overlap, which its `rules` object refuses as it is constructed, fails the gate (fn-126 R16). Every
// machine an IR file lifts must be among those constructed, so none escapes that check.

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import umpire.IrFile

class IrFilesTest extends munit.FunSuite:
  // Each feature's `object exports`, which declares its IR files as it initializes.
  val declaring: Seq[AnyRef] = Seq(
    features.activity.standalone.exports,
    features.nexus.workflow.exports,
    features.nexus.standalone.exports
  )

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
  def liftedMachines(file: String): Set[String] =
    import org.json4s.*
    val ir = org.json4s.jackson.JsonMethods.parse(Files.readString(Path.of(s"model/ir/$file.json")))
    (ir \ "machines" \ "name").children.collect { case JString(n) => n }.toSet

  test("every IR file is declared by a feature's exports, and every root of each constructs") {
    assertEquals(declaring.size, 3)
    val declared = IrFile.declared
    assertEquals(declared.map(_.name).toSet, checkedIn)
    for file <- declared do
      val unconstructed = liftedMachines(file.name) -- file.construct()
      assert(
        unconstructed.isEmpty,
        s"${file.name}.json lifts ${unconstructed.toSeq.sorted.mkString(", ")}, which constructing " +
          "its roots never reaches, so no rule overlap of theirs is checked"
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
    import features.nexus.product as nexusProduct
    import features.nexus.standalone.system as nexusStandalone
    import features.nexus.workflow.system as nexusWorkflow

    val projected: Seq[AnyRef] = Seq(
      activityProduct.ActivityProduct,
      activitySystem.ActivitySystem,
      activitySystem.ActivityRecord,
      activitySystem.HeldDispatch,
      activitySystem.StandaloneActivity,
      activitySystem.RecordOverQueue,
      activitySystem.RecordOverMatching,
      nexusProduct.NexusProduct,
      nexusStandalone.NexusSystem,
      nexusWorkflow.NexusSystem,
      nexusWorkflow.TrustingCaller,
      nexusWorkflow.RejectAfterClose,
      nexusWorkflow.NexusCaller,
      shared.worker.Polling,
      shared.taskqueue.product.TaskQueueProduct,
      shared.taskqueue.system.TaskQueueSystem
    )
    for model <- projected do
      model match
        case _: umpire.Phased[?, ?] => ()
        case _                      => fail(model.getClass.getName)
  }
