// Calls whose values cross their inputs' types, named or positional, which the framework's types
// refuse before anything is lifted. The lifter's tests build this and expect a type error at each
// marked line.
package fixture.crossed.namedinput

import umpire.*

enum Timeout derives Finite:
  case unset, expires

val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val heartbeat = input[Timeout]

val start = action(Party("fixture"))
  .input(scheduleToStart)
  .input(startToClose)
  .input(heartbeat)

// A named supply whose value has another type than its token's.
val wrongValue: Class = start(scheduleToStart := true)

// A named supply whose left side is no slot.
val noSlot: Class = start("scheduleToStart" := Timeout.expires)

// A positional call with a value of another type than its input's.
val wrongPosition: Class = start(Timeout.unset, false, Timeout.unset)

// A positional call with fewer values than the action has inputs.
val wrongArity: Class = start(Timeout.expires)
