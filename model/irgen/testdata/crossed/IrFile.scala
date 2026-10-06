// IR files whose roots the compiler refuses before anything is lifted: the lifter's tests build
// this and expect an error at each marked line.
package fixture.crossed

import umpire.*

// A root that names nothing.
val unknownRoot = irFile("unknown")(noSuchDeclaration)

// A channel is no root either, which the lifter refuses as it lifts the root (lifts/Rejects.scala):
// an object may be a machine's section, so the root's type admits any object.
