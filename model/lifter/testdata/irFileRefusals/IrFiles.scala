// IR files the lifter refuses to read, at their lines; a refusal here stops every file of the run.
package fixture.irfilerefusals

import umpire.*

val twice = irFile("twice")(temporal.worker.polling)

// A second declaration of one file.
val twiceAgain = irFile("twice")(temporal.worker.polling)

private val computed = "comp" + "uted"

// A name that is no literal.
val computedName = irFile(computed)(temporal.worker.polling)

// A path, where a name is asked for.
val pathName = irFile("nested/file")(temporal.worker.polling)

private val roots = Seq(temporal.worker.polling)

// Roots passed as one list rather than named one by one.
val splatted = irFile("splatted")(roots*)
