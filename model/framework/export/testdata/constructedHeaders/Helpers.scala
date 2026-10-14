package fixture.headerlibrary

import framework.dsl.*
import fixture.constructedheader.{Inner, Mode, Outcome, State}

abstract class Generic[S, O, F](using Finite[S], Finite[O], Finite[F]) extends Machine[S, O, F]
object Counts:
  var init = 0
  var name = 0
object LibraryMachine extends Generic[State, Outcome, Nothing]:
  override def name: String =
    Counts.name += 1
    "native-library-name"
  val init =
    Counts.init += 1
    State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
