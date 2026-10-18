package framework.endpointsupport

import framework.dsl.*

object Reads:
  var init = 0
  var end = 0
abstract class Instrumented extends Machine[Boolean, Boolean, Nothing]:
  val init: Boolean =
    Reads.init += 1
    false
  def end(s: Boolean): Boolean =
    Reads.end += 1
    s
  object rules extends Bindings()
object Inherited extends Instrumented
object Support:
  def main(args: Array[String]): Unit =
    val value = Inherited
    assert(Reads.init == 1 && Reads.end == 0)
    assert(MachineHeaders.collect(value).header.name == "inherited")
    assert(Reads.init == 1 && Reads.end == 0)
    assert(!value.init && value.end(true))
    assert(Reads.init == 1 && Reads.end == 1)
    println("inherited-native-only:init=1,end=1; header-increment=0")
