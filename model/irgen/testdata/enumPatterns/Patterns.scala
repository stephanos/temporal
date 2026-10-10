//> using scala 3.9.0
//> using jvm 27
//> using jar ../../../build/model-scala.jar
package fixture.enumpatterns

import framework.*
import framework.Finite.given CanEqual[?, ?]

enum Kind derives Finite:
  case a, b, c
enum Handler derives Finite:
  case done(ok: Boolean)
  case running, cancelReceived
final case class State(kind: Kind, handler: Handler) derives Finite
enum Outcome derives Finite:
  case accepted

val _ = () => summon[CanEqual[Kind, Kind]]

object Guarded extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  @scala.annotation.nowarn("msg=match may not be exhaustive")
  def end(s: State) = s.kind match
    case Kind.a                                              => false
    case _: Kind.b.type | _: Kind.c.type if s.kind == Kind.b => true

object Incomplete extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  @scala.annotation.nowarn("msg=match may not be exhaustive")
  def end(s: State) = s.kind match
    case Kind.a         => false
    case _: Kind.b.type => true

object Overlapping extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.kind match
    case Kind.a                                           => false
    case _: Kind.a.type | _: Kind.b.type | _: Kind.c.type => true

object Binding extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.kind match
    case Kind.a                                => false
    case x @ (_: Kind.b.type | _: Kind.c.type) => x == Kind.b

object EarlierBinding extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.handler match
    case Handler.done(x @ false)                                                    => x
    case _: Handler.done | _: Handler.running.type | _: Handler.cancelReceived.type => true

object BoundPrefix extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.kind match
    case x @ Kind.a => x == Kind.a // scalafix:ok DisableSyntax.==; typed equality
    case _: Kind.b.type | _: Kind.c.type => true

enum Lifecycle derives Finite:
  case held extends Lifecycle, Held
  case waiting, closed
final case class RoleState(phase: Lifecycle) derives Finite

object RolePrefix extends Machine[RoleState, Outcome, Nothing]:
  val init = RoleState(Lifecycle.held)
  object rules extends Bindings()
  def end(s: RoleState) = s.phase match
    case p: Held => p != Lifecycle.closed // scalafix:ok DisableSyntax.!=; typed equality
    case _: Lifecycle.waiting.type | _: Lifecycle.closed.type => false

object Explicit extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = (s.kind match
    case Kind.a          => false
    case Kind.b | Kind.c => true
  ) && (s.handler match
    case Handler.done(_)        => true
    case Handler.running        => false
    case Handler.cancelReceived => false)

object Remainder extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.handler match
    case Handler.done(_)                                          => true
    case _: Handler.running.type | _: Handler.cancelReceived.type => false

val storedOnce =
  Remainder.property.once(_.kind == Kind.a) // scalafix:ok DisableSyntax.==; typed equality

object Keeps extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.kind == Kind.c // scalafix:ok DisableSyntax.==; typed equality
  object properties:
    val kept = property
      .once(_.kind == Kind.a)
      .keeps[Kind](_.kind) // scalafix:ok DisableSyntax.==; typed equality
  object queries:
    val kept: Query =
      query verify properties.kept in scenario("keeps").actions() limits Limits("keeps", 1, 1, 1)

object StoredKeeps extends Machine[State, Outcome, Nothing]:
  val init = State(Kind.a, Handler.running)
  object rules extends Bindings()
  def end(s: State) = s.kind == Kind.c // scalafix:ok DisableSyntax.==; typed equality
  object properties:
    val kept = storedOnce.keeps[Kind](_.kind)
  object queries:
    val kept: Query =
      query verify properties.kept in scenario("keeps").actions() limits Limits("keeps", 1, 1, 1)
