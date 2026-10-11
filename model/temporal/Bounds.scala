// Every bound the temporal Models run their Queries under, declared here once and imported where a
// Query names it (`import temporal.Bounds.{five, six}`); no feature or foundation file declares one.
// A Query's receipt names its Limits by name, so each name means one budget and each budget has one
// name. A name says the depth and the actions; a `Wide` bound searches further than the bound of the
// same depth.
package temporal

import framework.*

object Bounds:
  // Nine actions are enabled before a Nexus operation is scheduled and eleven once it is, so an
  // exact sequence of two is found among ninety-nine candidates.
  val two = Limits(steps = 2, actions = 2, search = 512)
  val three = Limits(steps = 3, actions = 3, search = 4096)
  val four = Limits(steps = 4, actions = 4, search = 32768)
  val five = Limits(steps = 5, actions = 5, search = 65536)
  val six = Limits(steps = 6, actions = 6, search = 262144)
  val seven = Limits(steps = 7, actions = 7, search = 262144)
  val eight = Limits(steps = 8, actions = 8, search = 262144)

  // Past the depth of the detailed task queue's table and of a design composed with it, which is ten.
  val twelve = Limits(steps = 12, actions = 12, search = 262144)

  // The Nexus close policy's designs search further than the bounds of the same depth.
  val fourWide = Limits(steps = 4, actions = 4, search = 1 << 20)
  val fiveWide = Limits(steps = 5, actions = 5, search = 1 << 20)

  // Past the depth of every close-policy design's table, so a free search that verifies reads every
  // step.
  val twelveWide = Limits(steps = 12, actions = 12, search = 1 << 22)
