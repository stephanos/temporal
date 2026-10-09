// A choose of one alternative, which names no choice: its second alternative is a required
// parameter, so the framework's types refuse it before anything is lifted. The lifter's tests build
// this and expect a type error at the marked line.
package fixture.crossed.onechoice

import framework.*

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

final case class Lamp(lit: Boolean) derives Finite

val committed = choice

// One alternative.
def lightStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = choose(committed -> enter(Lamp(true)))
