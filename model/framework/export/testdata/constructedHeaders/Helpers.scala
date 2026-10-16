package fixture.headerlibrary

import framework.dsl.*
import fixture.constructedheader.{Inner, Mode, Outcome, State}

// Native state needs its admission evidence: a generic S is refused without StateData.
abstract class Generic[S, O, F](using Finite[S], Finite[O], Finite[F], StateData[S])
    extends Machine[S, O, F]
object Counts:
  var init = 0
  var name = 0
abstract class LibrarySupport extends Generic[State, Outcome, Nothing]:
  override def name: String =
    Counts.name += 1
    "native-library-name"
  val init =
    Counts.init += 1
    State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
object LibraryMachine extends LibrarySupport
