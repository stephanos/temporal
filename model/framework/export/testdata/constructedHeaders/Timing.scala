package framework.headertiming

import framework.dsl.*
import framework.headerselection.{Counting, Counts, Throwing}

object Reads:
  var catalogs = 0
  var init = 0
  var end = 0
  var rules = 0
  var name = 0
  var phase = 0
  var refinement = 0
final case class State(flag: Boolean)
// Counting witnesses; the reify plugin does not compile an anonymous `new Finite[T]:`.
final class Counted[T](read: () => IndexedSeq[T])(using Data.Structural[T]) extends Finite[T]:
  def values: IndexedSeq[T] =
    Reads.catalogs += 1
    read()
given state: Finite[State] = Counted[State](() => throw AssertionError("state catalog read"))
given outcome: Finite[Boolean] = Counted[Boolean](() => IndexedSeq.empty)
given facts: Finite[Nothing] = Counted[Nothing](() => throw AssertionError("fact catalog read"))
abstract class InertSupport extends Machine[State, Boolean, Nothing]:
  override def name: String =
    Reads.name += 1
    throw AssertionError("name read")
  def init: State =
    Reads.init += 1
    throw AssertionError("init read")
  def end(s: State): Boolean =
    Reads.end += 1
    throw AssertionError("end read")
  def rules: RuleBook[State, Boolean, Nothing] =
    Reads.rules += 1
    throw AssertionError("rules read")
  override private[framework] def phaseEvidence: Option[State => Any] =
    Reads.phase += 1
    throw AssertionError("phase read")
  override private[framework] def refinementEvidence: Option[Refinement[State, ?]] =
    Reads.refinement += 1
    throw AssertionError("refinement read")
object Inert extends InertSupport
abstract class Early extends Machine[State, Boolean, Nothing]:
  def init: State = throw AssertionError("init read")
  def end(s: State): Boolean = throw AssertionError("end read")
  val early: String = try
    val unexpected = MachineHeaders.collect(this)
    throw AssertionError(s"premature contribution: $unexpected")
  catch case error: IllegalArgumentException => error.getMessage
object Registered extends Early:
  def rules: RuleBook[State, Boolean, Nothing] = throw AssertionError("rules read")
class Plain extends Machine[State, Boolean, Nothing]:
  def init: State = throw AssertionError("init read")
  def end(s: State): Boolean = throw AssertionError("end read")
  def rules: RuleBook[State, Boolean, Nothing] = throw AssertionError("rules read")
object DerivedCounting extends Derived(Counting.assuming())
object Timing:
  private def refused(value: Machine[?, ?, ?]): Unit =
    val failure = try
      val unexpected = MachineHeaders.collect(value)
      throw AssertionError(s"unowned contribution: $unexpected")
    catch case error: IllegalArgumentException => error.getMessage
    assert(failure.contains("declaration identity"), failure)
  def main(args: Array[String]): Unit =
    val value = Inert
    val before = MachineHeaders.collect(value)
    assert(before.header.name == "inert")
    assert(before.header.stateType == "framework.headertiming.State")
    assert(MachineHeaders.collect(value) == before)
    assert(value.fs eq state)
    assert(value.fo eq outcome)
    assert(value.ff eq facts)
    assert(
      Vector(
        Reads.catalogs,
        Reads.init,
        Reads.end,
        Reads.rules,
        Reads.name,
        Reads.phase,
        Reads.refinement
      ).forall(_ == 0)
    )
    assert(Registered.early.contains("declaration identity"))
    assert(MachineHeaders.collect(Registered).header.name == "registered")
    refused(new Plain)
    object Local extends Plain
    refused(Local)
    val anonymous = new Plain {}
    refused(anonymous)
    val direct = MachineHeaders.collect(Counting)
    assert(direct.header.name == "counting")
    assert(Counts.init == 0 && Counts.refinement == 0 && Counts.rules == 0 && Counts.end == 0)
    // Native Built construction reads the refinement and defers its start (fn-123.10): the start
    // is read once, when the built machine's init is first read.
    val built = Counting.assuming()
    assert(Counts.init == 0 && Counts.refinement == 1)
    assert(!built.init && !built.init)
    assert(Counts.init == 1 && Counts.refinement == 1)
    assert(built.headerSchemaKeys == Counting.headerSchemaKeys)
    refused(built)
    val derived = DerivedCounting
    assert(Counts.init == 1 && Counts.refinement == 2)
    assert(!derived.init)
    assert(Counts.init == 2 && Counts.refinement == 2)
    assert(MachineHeaders.collect(derived).header.name == "derivedCounting")
    assert(derived.fs eq Counting.fs)
    assert(Counts.init == 2 && Counts.refinement == 2 && Counts.rules == 0 && Counts.end == 0)
    assert(MachineHeaders.collect(Throwing).header.name == "throwing")
    // The native-controls opaque checks: native state refuses opaque types, so the shared opaque
    // witness is the outcome and fact evidence of Throwing.
    assert(Throwing.fo eq Throwing.ff)
    assert(
      Throwing.ff
        .schemaFor("fixture.constructedheader.Domains.Left")
        .get
        .root
        .getIntRange
        .high == 2
    )
    assert(
      Throwing.fo
        .schemaFor("fixture.constructedheader.Domains.Right")
        .get
        .root
        .getIntRange
        .high == 2
    )
    println("timing:catalogs=0,getters=0,native-init=2,native-refinement=2")
