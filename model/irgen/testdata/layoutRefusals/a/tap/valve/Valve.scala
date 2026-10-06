// A feature of one level that hides a machine in a subfolder of its own: refused at its file's
// first declaration, as a level folder is.
package fixture.features.tap
package valve

import umpire.*

object Valve extends Machine[Tap, Outcome, Nothing]:
  val init = Tap(open = false)
  def end(s: State) = true

  object effects:
    def shut(s: State): List[Step[Tap, Outcome, Nothing]] = enter(s.copy(open = false))

  object rules extends Rules:
    on(plumber.turn)(where(_.open) ~> effects.shut)
