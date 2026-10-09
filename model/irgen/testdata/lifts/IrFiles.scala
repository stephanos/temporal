// IR files declared beside the fixtures they hold, which `lift --ir` lifts in one run: one
// declaration a root of two files, and a file whose root the lifter refuses.
package fixture.irfiles

import framework.*
import fixture.presence.Presence
import fixture.specimens.admission.{currentQueries, staleQueries}

// Lifted first, so a file lifted after it shows nothing of it carried over.
val sharedAdmission = irFile("shared-admission")(Presence, currentQueries, staleQueries)

// The presence fixture's one root, so its file is the presence fixture's expected IR.
val sharedPresence = irFile("shared-presence")(Presence)

// A root the lifter refuses, as it refuses it among the rejected declarations.
val refusedFile = irFile("refused")(fixture.rejects.Hoarding)
