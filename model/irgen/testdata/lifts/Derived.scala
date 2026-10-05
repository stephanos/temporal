// Machines derived by `rebind`, `extend`, `refining`, `assuming`, `unmonitored` and `restrict`, each
// beside the machine it stands for spelled out. The lifter's tests lift both and require one IR of
// each pair, but for names, positions and the names of the functions the declarations refer to.
package fixture.derived

import umpire.*

given Family = Family("fixture.derived")

enum Light derives Finite:
  case off, on, broken

/** A lamp's state, named apart from the machine object `Lamp`. */
final case class LampState(light: Light) derives Finite

enum Outcome derives Finite:
  case accepted, refused

enum Fact derives Finite:
  case lit, darkened, burnedOut

val press = action(Party("user"))
val wear = timer
val burnOut = action(Party("fault"))

val lampOpaque = assume
val burnOutAssumed = assume

enum Shown derives Finite:
  case dark, bright

/** The view's state, named apart from the machine object `View`. */
final case class ViewState(shown: Shown) derives Finite

def pressView(v: ViewState): List[Step[ViewState, Outcome, Fact]] =
  if v.shown == Shown.dark then
    List(Step(Outcome.accepted, ViewState(Shown.bright), List(Fact.lit)))
  else List(Step(Outcome.accepted, ViewState(Shown.dark), List(Fact.darkened)))

def burnOutView(v: ViewState): List[Step[ViewState, Outcome, Fact]] =
  List(Step(Outcome.refused, ViewState(Shown.dark), List(Fact.burnedOut)))

/** The opaque lamp the detailed ones refine. */
object View extends Machine[ViewState, Outcome, Fact]:
  val init = ViewState(Shown.dark)
  def end(view: State) = true

  object rules extends Bindings(press ~> pressView)

/** The opaque lamp under faults: the same types as `View`, with a burn-out. */
object ViewUnderFaults extends Machine[ViewState, Outcome, Fact], FailureModel:
  val init = ViewState(Shown.dark)
  def end(view: State) = true

  object rules extends Bindings(press ~> pressView, burnOut ~> burnOutView)

/** What a lamp shows: the map `refining` names, and each refinement below writes out. */
def seen(l: LampState): ViewState =
  if l.light == Light.on then ViewState(Shown.bright) else ViewState(Shown.dark)

def pressLamp(l: LampState): List[Step[LampState, Outcome, Fact]] = l.light match
  case Light.off    => List(Step(Outcome.accepted, LampState(Light.on), List(Fact.lit)))
  case Light.on     => List(Step(Outcome.accepted, LampState(Light.off), List(Fact.darkened)))
  case Light.broken => Nil

def pressStiff(l: LampState): List[Step[LampState, Outcome, Fact]] = l.light match
  case Light.off => List(Step(Outcome.refused, l))
  case _         => pressLamp(l)

def wearLamp(l: LampState): List[Step[LampState, Outcome, Fact]] =
  if l.light == Light.on then
    List(Step(Outcome.accepted, LampState(Light.off), List(Fact.darkened)))
  else Nil

def burnOutLamp(l: LampState): List[Step[LampState, Outcome, Fact]] =
  if l.light == Light.broken then Nil
  else List(Step(Outcome.refused, LampState(Light.broken), List(Fact.burnedOut)))

enum Lit derives Finite:
  case never, once, again

def countLit(seen: Lit, before: LampState, after: Step[LampState, Outcome, Fact]): Lit =
  if !after.facts.contains(Fact.lit) then seen
  else if seen == Lit.never then Lit.once
  else Lit.again

val litAgain =
  monitor[LampState, Outcome, Fact, Lit](Lit.never)(countLit)(seen => seen == Lit.again)

/** The detailed lamp: every part a derivation keeps or replaces. */
object Lamp extends Machine[LampState, Outcome, Fact]:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }

  object refinement extends Refinement(View):
    def toProduct(l: LampState): ViewState =
      if l.light == Light.on then ViewState(Shown.bright) else ViewState(Shown.dark)
    val visible = (f: Fact) => f != Fact.burnedOut
    val visibleOutcomes = (o: Outcome) => o == Outcome.accepted
    val unobservable = List(wear)

  object monitors extends Section:
    val lit = litAgain
    val opaque = lampOpaque

  object rules extends Bindings(press ~> pressLamp, wear ~> wearLamp)

/** One step function replaced, in its place. */
object StiffLamp extends Derived(Lamp.rebind(press ~> pressStiff))

object StiffLampSpelled extends Machine[LampState, Outcome, Fact]:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }

  object refinement extends Refinement(View):
    def toProduct(l: LampState): ViewState =
      if l.light == Light.on then ViewState(Shown.bright) else ViewState(Shown.dark)
    val visible = (f: Fact) => f != Fact.burnedOut
    val visibleOutcomes = (o: Outcome) => o == Outcome.accepted
    val unobservable = List(wear)

  object monitors extends Section:
    val lit = litAgain
    val opaque = lampOpaque

  object rules extends Bindings(press ~> pressStiff, wear ~> wearLamp)

/** An action added, the refined machine replaced and an assumption appended, in one chain. */
object FaultyLamp
    extends Derived(
      Lamp
        .extend(burnOut ~> burnOutLamp)
        .refining(ViewUnderFaults)(seen)
        .assuming(burnOutAssumed)
    ),
      FailureModel

object FaultyLampSpelled extends Machine[LampState, Outcome, Fact], FailureModel:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }

  object refinement extends Refinement(ViewUnderFaults):
    def toProduct(l: LampState): ViewState =
      if l.light == Light.on then ViewState(Shown.bright) else ViewState(Shown.dark)
    val visible = (f: Fact) => f != Fact.burnedOut
    val visibleOutcomes = (o: Outcome) => o == Outcome.accepted
    val unobservable = List(wear)

  object monitors extends Section:
    val lit = litAgain
    val opaque = lampOpaque
    val faults = burnOutAssumed

  object rules extends Bindings(press ~> pressLamp, wear ~> wearLamp, burnOut ~> burnOutLamp)

/** The monitors and the refinement dropped, with its visibility. */
object PlainLamp extends Derived(Lamp.unmonitored)

object PlainLampSpelled extends Machine[LampState, Outcome, Fact]:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }
  val unobservable = List(wear)

  object monitors extends Section:
    val opaque = lampOpaque

  object rules extends Bindings(press ~> pressLamp, wear ~> wearLamp)

/** A derivation of a derivation: a lambda bound by it is named after the machine it declares. */
object PlainStiffLamp
    extends Derived(
      StiffLamp.unmonitored.rebind(
        wear ~> (l => if l.light == Light.broken then Nil else wearLamp(l))
      )
    )

object PlainStiffLampSpelled extends Machine[LampState, Outcome, Fact]:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }
  val unobservable = List(wear)

  object monitors extends Section:
    val opaque = lampOpaque

  object rules
      extends Bindings(
        press ~> pressStiff,
        wear ~> (l => if l.light == Light.broken then Nil else wearLamp(l))
      )

/** A restriction chained after a derivation: it keeps the monitors and assumptions, not the rest. */
object StiffPressOnly extends Derived(Lamp.rebind(press ~> pressStiff).restrict(press))

object StiffPressOnlySpelled extends Machine[LampState, Outcome, Fact]:
  val init = LampState(Light.off)
  def end(l: State) = l.light != Light.broken
  val evidence: PartialFunction[Fact, String] = { case Fact.burnedOut => "lampBurnedOut" }

  object monitors extends Section:
    val lit = litAgain
    val opaque = lampOpaque

  object rules extends Bindings(press ~> pressStiff)
