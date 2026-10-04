//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../gen/model-scala.jar
//> using jar ../../../gen/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
// fn-114 R3: the string-named and string-keyed declaration forms are retired, so a declaration
// takes its name from its val and a composition names its members by field. Each form is written
// once as it is declared now, then as it was; each old spelling fails to compile on its line.
package fixture.retiredNames

import umpire.*

given Family = Family("fixture.retiredNames")

enum Note derives Finite:
  case ping

final case class Lamp(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class Pair(left: Lamp, right: Lamp)

val press = action(Party("fixture"))
val lamp = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(press ~> (s => List(Step(Outcome.accepted, Lamp(!s.on)))))
}

val expire = timer
val expireSpelled = timer("expire")

val flush = internal
val flushSpelled = internal("flush")

val crash = hole
val crashSpelled = hole("crash")

val wire = channel[Note](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val wireSpelled = channel[Note]("wire", capacity = 1, order = Order.fifo, loss = Loss.reliable)

val pressOnly = lamp.restrict(press)
val pressOnlySpelled = lamp.restrict(summon[Family], "pressOnly")(press)

val pair = compose[Pair](_.left -> lamp, _.right -> lamp).sync(_.left -> press, _.right -> press)
val pairSpelled = compose[Pair](summon[Family], "pair")("left" -> lamp, "right" -> lamp)
val pairKeyed = compose[Pair]("left" -> lamp, "right" -> lamp)
val pairSynced = pair.sync("pressBoth", "left" -> press, "right" -> press)
val pairReplacing = pair.replaces(_.left, lamp)
val pairReplacingSpelled = pair.replaces("left", lamp)

val pressed = lamp.scenario.actions(press)
val pressedKeyed = lamp.scenario.actionKeys("press")
