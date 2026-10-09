//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../build/model-scala.jar
//> using jar ../../../build/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
// fn-114 R3: the string-named and string-keyed declaration forms are retired, so a declaration
// takes its name from its val and a composition names its members by field. Each form is written
// once as it is declared now, then as it was; each old spelling fails to compile on its line.
package fixture.retiredNames

import framework.*

enum Note derives Finite:
  case ping

final case class LampState(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class PairState(left: LampState, right: LampState)

val press = action(Actor("fixture"))
object Lamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(false)
  def end(lampState: State) = true
  object rules extends Bindings(press ~> (s => List(Step(Outcome.accepted, LampState(!s.on)))))

val expire = timer
val expireSpelled = timer("expire")

val flush = internal
val flushSpelled = internal("flush")

val crash = hole
val crashSpelled = hole("crash")

val wire = channel[Note](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val wireSpelled = channel[Note]("wire", capacity = 1, order = Order.fifo, loss = Loss.reliable)

object PressOnly extends Derived(Lamp.restrict(press))
val pressOnlySpelled = Lamp.restrict("pressOnly")(press)

object Pair extends Composition[PairState](_.left -> Lamp, _.right -> Lamp):
  def end(pairState: State) = true
  object syncs extends Syncs:
    sync(_.left -> press, _.right -> press)
    replaces(_.left, Lamp)
object PairSpelled extends Composition[PairState]("pair")
object PairKeyed extends Composition[PairState]("left" -> Lamp, "right" -> Lamp)
object PairSynced extends Composition[PairState](_.left -> Lamp, _.right -> Lamp):
  def end(pairState: State) = true
  object syncs extends Syncs:
    sync("pressBoth", "left" -> press, "right" -> press)
    replaces("left", Lamp)

val pressed = Lamp.scenario.actions(press)
val pressedKeyed = Lamp.scenario.actionKeys("press")
