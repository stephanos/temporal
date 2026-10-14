package framework.`export`

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.Path
import framework.check.Ran

// The lifter run in the tests' own JVM, with the arguments of `lift`'s command line, rather than a
// JVM of its own per lift: what it prints and its exit status are those a JVM would have left, so
// the tests read a run as they read one of a tool. `rules` turns switched lifter rules on, as
// `UMPIRE_LIFT_RULES` does for a JVM (Switches).
// Lifts run side by side share the JVM's heap, so at most `slots` lift at once, whatever the suites'
// own limits on builds and lifts.
object InProcess:
  private val slots = java.util.concurrent.Semaphore(3, true)

  def lift(directory: Path, arguments: Seq[String], rules: Set[String] = Set.empty): Ran =
    slots.acquire()
    try lifted(directory, arguments, rules)
    finally slots.release()

  private def lifted(directory: Path, arguments: Seq[String], rules: Set[String]): Ran =
    val printed = ByteArrayOutputStream()
    val err = PrintStream(printed, true, UTF_8)
    val exit = LiftRun.run(arguments, err, rules)
    err.flush()
    Ran("lift" +: arguments, directory, exit, printed.toString(UTF_8))
