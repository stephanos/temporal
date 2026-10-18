package framework.endpointtiming

import framework.dsl.*
import framework.endpointsupport.{Inherited, Reads}

object Counts:
  var catalogs = 0
  var names = 0
  var rules = 0
  var phases = 0
  var refinements = 0
// A counting witness; the reify plugin does not compile an anonymous `new Finite[T]:`.
final class Unread[T](using Data.Structural[T]) extends Finite[T]:
  def values: IndexedSeq[T] =
    Counts.catalogs += 1
    throw AssertionError("catalog read")
given witness: Finite[Boolean] = Unread[Boolean]()
object Inert extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  override def name: String =
    Counts.names += 1
    throw AssertionError("name read")
  def rules: RuleBook[Boolean, Boolean, Nothing] =
    Counts.rules += 1
    throw AssertionError("rules read")
  override private[framework] def phaseEvidence: Option[Boolean => Any] =
    Counts.phases += 1
    throw AssertionError("phase read")
  override private[framework] def refinementEvidence: Option[Refinement[Boolean, ?]] =
    Counts.refinements += 1
    throw AssertionError("refinement read")
abstract class Early extends Machine[Boolean, Boolean, Nothing]:
  def init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
  val early: String = try
    val unexpected = MachineEndpoints.collect(this)
    throw AssertionError(s"premature contribution: $unexpected")
  catch case error: IllegalArgumentException => error.getMessage
object Registered extends Early
class Plain extends Machine[Boolean, Boolean, Nothing]:
  def init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
object Timing:
  private def refused(value: Machine[?, ?, ?], expected: String): Unit =
    val error = try
      val unexpected = MachineEndpoints.collect(value)
      throw AssertionError(s"unexpected contribution: $unexpected")
    catch case error: IllegalArgumentException => error.getMessage
    assert(error.contains(expected), error)
  def main(args: Array[String]): Unit =
    val before = MachineEndpoints.collect(Inert)
    assert(MachineEndpoints.collect(Inert).equals(before))
    assert(Inert.fs eq witness)
    assert(Inert.fo eq witness)
    val chain = Inert.unmonitored.unmonitored
    assert(MachineEndpoints.collect(chain).equals(before))
    assert(chain.fs eq witness)
    assert(!chain.init && chain.end(true))
    assert(
      Vector(Counts.catalogs, Counts.names, Counts.rules, Counts.phases, Counts.refinements).forall(
        _ == 0
      )
    )
    assert(Registered.early.contains("registration"))
    refused(Registered, "metadata")
    refused(new Plain, "registration")
    object Local extends Plain
    refused(Local, "registration")
    refused(new Plain {}, "registration")
    val inherited = Inherited
    assert(Reads.init == 1 && Reads.end == 0)
    refused(inherited, "metadata")
    refused(inherited.unmonitored.unmonitored, "metadata")
    assert(Reads.init == 1 && Reads.end == 0)
    assert(!inherited.init && inherited.end(true))
    assert(Reads.init == 1 && Reads.end == 1)
    println("endpoint-timing:passed; catalog-and-getter-increment=0; inherited-val-init=1,end=1")
