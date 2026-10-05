// IR files whose roots the compiler refuses before anything is lifted: the lifter's tests build
// this and expect an error at each marked line.
package fixture.crossed

import umpire.*

// A root that names nothing.
val unknownRoot = irFile("unknown")(noSuchDeclaration)

// A channel, which is no root: an IR file holds what a machine or a claim reaches.
val channelRoot = irFile("channel")(wire)
