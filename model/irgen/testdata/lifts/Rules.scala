// Machine objects with effects, rules and sections (fn-126 R15, R16): the machine object is the
// machine, its rules say when each action fires and its effects what it does. `Switch` and its twin
// `Core.coreSwitch`, written in the core with hand-written step functions, lift to the same tables
// (tools/umpire/model's TestRulesLowerToTheCoreTables). Then a bare binding that keeps the rules'
// guards, rules a derivation binds in their source's place, rules a derivation adds, and a
// composition object with its derived twin. The lifter's tests lift the roots `rules` lists and
// compare the IR with expected/rules.json.
package fixture.rules

import umpire.*

given Family = Family("fixture.rules")

enum Light derives Finite:
  case off, on, broken

final case class Lamp(light: Light, presses: UpTo[2]) derives Finite

enum Outcome derives Finite:
  case accepted, refused

enum Fact derives Finite:
  case lit, darkened, worn

enum Knob derives Finite:
  case up, down

given Ok[Outcome] = Ok(Outcome.accepted)

val knob = input[Knob]

val two = Limits(steps = 2, actions = 2, search = 4096)

object hand extends Actor:
  val press = action(this)
  val turn = action(this).input(knob)

object clock extends Section:
  val tick = timer
  val rest = timer
  val dim = timer

object Switch extends Machine[Lamp, Outcome, Fact]:
  val init = Lamp(Light.off, UpTo(0))
  def end(s: State) = states.broken(s)
  val evidence: PartialFunction[Fact, String] = { case Fact.worn => "wear" }

  object states extends Section:
    /** A broken lamp, its state named through the machine, `State`, and as `Lamp` (below). */
    def broken(s: State) = s.light == Light.broken
    def brokenLamp(s: Lamp) = s.light == Light.broken

    /** The presses counted so far, one more, up to two. */
    def worn(p: UpTo[2]): UpTo[2] = UpTo((p + 1).min(2))

  object effects extends Section:
    def light(s: Lamp) = enter(s.copy(light = Light.on, presses = states.worn(s.presses)), Fact.lit)
    def dark(s: Lamp) = enter(s.copy(light = Light.off), Fact.darkened)
    def refuse(s: Lamp, k: Knob) = List(Step[Lamp, Outcome, Fact](Outcome.refused, s))
    def turned(s: Lamp, k: Knob) =
      enter[Lamp, Outcome, Fact](s.copy(light = if k == Knob.up then Light.on else Light.off))
    def wear(s: Lamp) = enter(s.copy(light = Light.broken), Fact.worn)
    def keep(s: Lamp) = stay[Lamp, Outcome, Fact](s)

  object monitors extends Section:
    val neverWorn = sticky[Lamp, Outcome, Fact](after => !after.records(Fact.worn))

  object rules extends Rules(_.light):
    in(Light.off)(hand.press ~> effects.light)
    in(Light.on) {
      hand.press ~> effects.dark
      hand.turn(Knob.down) ~> effects.dark
      hand.turn(Knob.up) ~> effects.light
    }
    in(Light.off)(hand.turn ~> effects.turned)
    in(Light.broken)(hand.turn ~> effects.refuse)
    when(s => s.presses == 2 && s.light.in(Light.off, Light.on))(clock.tick ~> effects.wear)
    disabled(clock.rest)

  object properties extends Section:
    val pressLights = property when hand.press holds (after => after.state.light != Light.broken)

  object queries extends Section:
    val pressedTwice = scenario.actions(hand.press, hand.press)
    val pressing = query verify properties.pressLights in pressedTwice limits two total 18
    val wornOut =
      query find properties.pressLights in scenario("wornOut").actions(
        hand.press,
        clock.tick
      ) limits
        two total 18

/** `Switch` in the core: one hand-written step function per action, as its rules lower. */
object Core:
  import Switch.effects.*

  def pressStep(s: Lamp) =
    if s.light == Light.off then light(s) else if s.light == Light.on then dark(s) else Nil

  def turnStep(s: Lamp, k: Knob) = k match
    case Knob.down =>
      if s.light == Light.on then dark(s)
      else if s.light == Light.off then turned(s, k)
      else if s.light == Light.broken then refuse(s, k)
      else Nil
    case Knob.up =>
      if s.light == Light.on then light(s)
      else if s.light == Light.off then turned(s, k)
      else if s.light == Light.broken then refuse(s, k)
      else Nil

  def tickStep(s: Lamp) =
    if s.presses == 2 && s.light.in(Light.off, Light.on) then wear(s) else Nil

  def restStep(s: Lamp): List[Step[Lamp, Outcome, Fact]] = Nil

  val coreSwitch = machine[Lamp, Outcome, Fact] {
    starts(Lamp(Light.off, UpTo(0)))
    ends(s => s.light == Light.broken)
    evidence { case Fact.worn => "wear" }
    monitors(Switch.monitors.neverWorn)
    steps(
      hand.press ~> pressStep,
      hand.turn ~> turnStep,
      clock.tick ~> tickStep,
      clock.rest ~> restStep
    )
  }

/** A lamp that reads as the switch it refines, state for state, and whose rest is unobservable. */
object Mirror extends Machine[Lamp, Outcome, Fact]:
  val init = Lamp(Light.off, UpTo(0))
  def end(s: Lamp) = Switch.states.brokenLamp(s)
  object refinement extends Refinement(Switch):
    def toProduct(s: Lamp) = s
    val unobservable = List(clock.rest)
  object rules extends Rules(_.light):
    in(Light.off)(hand.press ~> Switch.effects.light)
    in(Light.on)(hand.press ~> Switch.effects.dark)
    disabled(clock.rest)

/** A tick that keeps the lamp where a tick fires: the rules' guards, another effect. */
object Steady extends Derived(Switch.rebind(clock.tick ~> Switch.effects.keep))

/** A press that keeps the lamp in every state: rules in place of the press's rules. */
object Loose extends Derived(Switch.rebind(when(_ => true)(hand.press ~> Switch.effects.keep)))

/** A lamp that also dims while it is off: rules a derivation adds. */
object Dimming
    extends Derived(
      Switch.extend(when(s => s.light == Light.off)(clock.dim ~> Switch.effects.keep))
    )

object Plain extends Derived(Switch.unmonitored)

final case class Pair(left: Lamp, right: Lamp)

/** Two lamps whose presses step together. */
object Twins extends Composition[Pair](_.left -> Plain, _.right -> Plain):
  def end(s: Pair) = Switch.end(s.left)
  object syncs extends Syncs:
    sync(_.left -> hand.press, _.right -> hand.press)

object Unbending extends Derived(Steady.unmonitored)

/** The twins, the right one steady. */
object Unequal extends Composition(Twins.withMember(_.right -> Unbending))
