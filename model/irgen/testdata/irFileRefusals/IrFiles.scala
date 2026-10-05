// IR files the lifter refuses to read, at their lines; a refusal here stops every file of the run.
package fixture.irfilerefusals

import umpire.*

val twice = irFile("twice")(temporal.shared.worker.Polling)

// A second declaration of one file.
val twiceAgain = irFile("twice")(temporal.shared.worker.Polling)

private val computed = "comp" + "uted"

// A name that is no literal.
val computedName = irFile(computed)(temporal.shared.worker.Polling)

// A path, where a name is asked for.
val pathName = irFile("nested/file")(temporal.shared.worker.Polling)

private val roots = Seq(temporal.shared.worker.Polling)

// Roots passed as one list rather than named one by one.
val splatted = irFile("splatted")(roots*)
