// Machines derived by `rebind`, `extend`, `refining`, `assuming` and `unmonitored`, each beside the
// machine it stands for spelled out. The lifter's tests lift both and require one IR of each pair,
// but for names, positions and the names of the functions the declarations refer to.
package fixture.derived

import umpire.*

given Family = Family("fixture.derived")

enum Light derives Finite:
  case off, on, broken

final case class Lamp(light: Light) derives Finite

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

final case class View(shown: Shown) derives Finite

def pressView(v: View): List[Step[View, Outcome, Fact]] =
  if v.shown == Shown.dark then List(Step(Outcome.accepted, View(Shown.bright), List(Fact.lit)))
  else List(Step(Outcome.accepted, View(Shown.dark), List(Fact.darkened)))

def burnOutView(v: View): List[Step[View, Outcome, Fact]] =
  List(Step(Outcome.refused, View(Shown.dark), List(Fact.burnedOut)))

/** The opaque lamp the detailed ones refine. */
val view = machine[View, Outcome, Fact] {
  starts(View(Shown.dark))
  ends(_ => true)
  steps(press ~> pressView)
}

/** The opaque lamp under faults: the same types as `view`, with a burn-out. */
val viewUnderFaults = machine[View, Outcome, Fact] {
  starts(View(Shown.dark))
  ends(_ => true)
  steps(press ~> pressView, burnOut ~> burnOutView)
}

def seen(l: Lamp): View = if l.light == Light.on then View(Shown.bright) else View(Shown.dark)

def pressLamp(l: Lamp): List[Step[Lamp, Outcome, Fact]] = l.light match
  case Light.off    => List(Step(Outcome.accepted, Lamp(Light.on), List(Fact.lit)))
  case Light.on     => List(Step(Outcome.accepted, Lamp(Light.off), List(Fact.darkened)))
  case Light.broken => Nil

def pressStiff(l: Lamp): List[Step[Lamp, Outcome, Fact]] = l.light match
  case Light.off => List(Step(Outcome.refused, l))
  case _         => pressLamp(l)

def wearLamp(l: Lamp): List[Step[Lamp, Outcome, Fact]] =
  if l.light == Light.on then List(Step(Outcome.accepted, Lamp(Light.off), List(Fact.darkened)))
  else Nil

def burnOutLamp(l: Lamp): List[Step[Lamp, Outcome, Fact]] =
  if l.light == Light.broken then Nil
  else List(Step(Outcome.refused, Lamp(Light.broken), List(Fact.burnedOut)))

enum Lit derives Finite:
  case never, once, again

def countLit(seen: Lit, before: Lamp, after: Step[Lamp, Outcome, Fact]): Lit =
  if !after.facts.contains(Fact.lit) then seen
  else if seen == Lit.never then Lit.once
  else Lit.again

val litAgain = monitor[Lamp, Outcome, Fact, Lit](Lit.never)(countLit)(seen => seen == Lit.again)

/** The detailed lamp: every part a derivation keeps or replaces. */
val lamp = machine[Lamp, Outcome, Fact] {
  refines(view)(seen)
  visible(f => f != Fact.burnedOut)
  visibleOutcomes(o => o == Outcome.accepted)
  monitors(litAgain)
  assumes(lampOpaque)
  starts(Lamp(Light.off))
  ends(l => l.light != Light.broken)
  evidence { case Fact.burnedOut => "lampBurnedOut" }
  unobservable(wear)
  steps(press ~> pressLamp, wear ~> wearLamp)
}

/** One step function replaced, in its place. */
val stiffLamp = lamp.rebind(press ~> pressStiff)

val stiffLampSpelled = machine[Lamp, Outcome, Fact] {
  refines(view)(seen)
  visible(f => f != Fact.burnedOut)
  visibleOutcomes(o => o == Outcome.accepted)
  monitors(litAgain)
  assumes(lampOpaque)
  starts(Lamp(Light.off))
  ends(l => l.light != Light.broken)
  evidence { case Fact.burnedOut => "lampBurnedOut" }
  unobservable(wear)
  steps(press ~> pressStiff, wear ~> wearLamp)
}

/** An action added, the refined machine replaced and an assumption appended, in one chain. */
val faultyLamp = lamp
  .extend(burnOut ~> burnOutLamp)
  .refining(viewUnderFaults)(seen)
  .assuming(burnOutAssumed)

val faultyLampSpelled = machine[Lamp, Outcome, Fact] {
  refines(viewUnderFaults)(seen)
  visible(f => f != Fact.burnedOut)
  visibleOutcomes(o => o == Outcome.accepted)
  monitors(litAgain)
  assumes(lampOpaque, burnOutAssumed)
  starts(Lamp(Light.off))
  ends(l => l.light != Light.broken)
  evidence { case Fact.burnedOut => "lampBurnedOut" }
  unobservable(wear)
  steps(press ~> pressLamp, wear ~> wearLamp, burnOut ~> burnOutLamp)
}

/** The monitors and the refinement dropped, with its visibility. */
val plainLamp = lamp.unmonitored

val plainLampSpelled = machine[Lamp, Outcome, Fact] {
  assumes(lampOpaque)
  starts(Lamp(Light.off))
  ends(l => l.light != Light.broken)
  evidence { case Fact.burnedOut => "lampBurnedOut" }
  unobservable(wear)
  steps(press ~> pressLamp, wear ~> wearLamp)
}

/** A derivation of a derivation: a lambda bound by it is named after the machine it declares. */
val plainStiffLamp =
  stiffLamp.unmonitored.rebind(wear ~> (l => if l.light == Light.broken then Nil else wearLamp(l)))

val plainStiffLampSpelled = machine[Lamp, Outcome, Fact] {
  assumes(lampOpaque)
  starts(Lamp(Light.off))
  ends(l => l.light != Light.broken)
  evidence { case Fact.burnedOut => "lampBurnedOut" }
  unobservable(wear)
  steps(press ~> pressStiff, wear ~> (l => if l.light == Light.broken then Nil else wearLamp(l)))
}
