package framework.check

import java.nio.file.Path
import java.util.concurrent.ConcurrentLinkedQueue
import scala.jdk.CollectionConverters.*

// A suite whose scratch directories outlive its JVM (model/build/history, .flow/tmp): they are
// deleted when every test of the suite passed, and kept, marked failed, when one failed, with only
// the newest failed runs' kept (Retained, Tools.scala). The lifter's and the migration rules'
// suites take it too (their project.scala).
trait RetainedScratch extends munit.FunSuite:
  private val made = ConcurrentLinkedQueue[Path]()
  private val failed = ConcurrentLinkedQueue[String]()

  // A new scratch directory under `parent`, kept only if a test of this suite fails.
  protected def retained(parent: Path, prefix: String): Path =
    val directory = Retained.directory(parent, prefix)
    made.add(directory): Unit
    directory

  override def munitTestTransforms: List[TestTransform] =
    super.munitTestTransforms :+ TestTransform(
      "retained scratch",
      test =>
        test.withBodyMap(
          _.transform: result =>
            // A test that assumes what does not hold (`assume`) is skipped, not failed.
            if result.failed.toOption.exists(e =>
                !e.getClass.getName.endsWith("AssumptionViolatedException")
              )
            then failed.add(test.name): Unit
            result
          (using munitExecutionContext)
        )
    )

  override def afterAll(): Unit =
    super.afterAll()
    Retained.finish(failed.isEmpty, made.asScala.toVector, failed.asScala.mkString("", "\n", "\n"))
