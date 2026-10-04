// A choose of one alternative, which names no choice: its second alternative is a required
// parameter, so the framework's types refuse it before anything is lifted. The lifter's tests build
// this and expect a type error at the marked line.
package fixture.crossed.onechoice

import umpire.*

enum Outcome derives Finite:
  case accepted

given Accepted[Outcome] = Accepted(Outcome.accepted)

final case class Lamp(lit: Boolean) derives Finite

val committed = choice

// One alternative.
def lightStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = choose(committed -> accept(Lamp(true)))
