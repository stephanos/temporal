# Scala front end for a Go-interpreted Umpire IR (model/scalav2)

Spike of the SCALA.md architecture: Scala is the authoring language, a protobuf IR is the
specification, Go interprets it. model/scala's Models are lifted from their TASTy by a post-compile
lifter into ir/nexus-caller.json; goir/ validates and interprets the IR and derives tables through
model/go's umpire.Table. Done when the Go-derived tables, IDs, refinement rows and target
fingerprint of the Nexus caller equal the Lean dumps, and the lifter and loader report errors at
Scala source lines. Out of scope: Properties, Queries, Cases, composition, the activity.
