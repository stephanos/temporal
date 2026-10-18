package fixture.gaps

import framework.dsl.*

object Helpers:
  @gap(unasked)(because = "not a lifted helper")
  def helper = true
  @gap(unasked)(because = "not a property")
  val ordinary = true

@gap(unasked)(because = "helper object")
object Utility

final class Parameters(@gap(unasked)(because = "constructor parameter") val parameter: Boolean)

object Foreign:
  def unasked: Subject = framework.dsl.unasked

@gap(Foreign.unasked)(because = "foreign constructor")
object ForeignSubject extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object Lookalike:
  final class gap extends scala.annotation.StaticAnnotation
  final class accepted extends scala.annotation.StaticAnnotation

@Lookalike.gap
object Impostor
@Lookalike.accepted
object OldAcceptance

@gap()(because = "missing subject")
object Empty extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

@gap(unasked)(because = " ")
object Blank extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
