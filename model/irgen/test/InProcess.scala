package umpire.irgen

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.Path
import umpire.check.Ran

// The lifter run in the tests' own JVM, with the arguments of `lift`'s command line, rather than a
// JVM of its own per lift: what it prints and its exit status are those a JVM would have left, so
// the tests read a run as they read one of a tool. `rules` turns switched lifter rules on, as
// `UMPIRE_LIFT_RULES` does for a JVM (Switches).
object InProcess:
  def lift(directory: Path, arguments: Seq[String], rules: Set[String] = Set.empty): Ran =
    val printed = ByteArrayOutputStream()
    val err = PrintStream(printed, true, UTF_8)
    val exit = LiftRun.run(arguments, err, rules)
    err.flush()
    Ran("lift" +: arguments, directory, exit, printed.toString(UTF_8))
