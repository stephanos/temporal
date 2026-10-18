package framework.dsl

// A sample's claim judged on one state or one step, as the native checker judges it on each state
// and step of a graph: the claim named by `verify(Model.postconditions.x)`, its section evaluated
// with the state restored into the Machine's vars and, for a postcondition, the step under
// evaluation.
import framework.checker.*

object SampleClaims:
  // Whether the claim holds of the step of `cls` from `before` to `after`; None where it is about
  // the steps of another target, which this step is not.
  def judged[S, O, F](claim: Query, before: S, cls: Class, after: Step[S, O, F]): Option[Boolean] =
    val ref = claim.property.claim.get
    val context =
      if ref.section == "invariants" then ClaimEvaluation.Context()
      else ClaimEvaluation.Context(Some(state(before)), Some(cls), Some(after.outcome), after.facts)
    val evaluated = ClaimEvaluation.evaluate(vars(ref), section(ref), state(after.state), context)
    Option.when(evaluated.applies(ref.member))(evaluated.failure(ref.member).isEmpty)

  // Whether `member` of the section `section` of `owner` holds of `step`: None where it is about the
  // steps of a target `taken` is not one of.
  def judged(
      machine: RecordMachine[?, ?, ?],
      owner: AnyRef,
      section: String,
      member: String,
      step: Step[State, ?, ?],
      from: Option[State] = None,
      taken: Option[Class] = None
  ): Option[Boolean] =
    val cls = ClaimEvaluation.sectionClass(owner, section).get
    val context = ClaimEvaluation.evaluate(
      machine.stateVars.get,
      cls,
      step.state,
      ClaimEvaluation.Context(from, taken, Some(step.outcome), step.facts)
    )
    Option.when(context.applies(member))(context.failure(member).isEmpty)

  // Whether an invariant holds in `now`.
  def holds(claim: Query, now: Any): Boolean =
    val ref = claim.property.claim.get
    require(ref.section == "invariants", s"$ref is no invariant")
    val evaluated =
      ClaimEvaluation.evaluate(vars(ref), section(ref), state(now), ClaimEvaluation.Context())
    evaluated.failure(ref.member).isEmpty

  // Whether a claim written in a test, `invariant(check(!beeping))`, holds in `now` of `machine`.
  def holds(machine: RecordMachine[?, ?, ?], claim: NativeClaims.Adhoc, now: Any): Boolean =
    ClaimEvaluation
      .evaluate(machine.stateVars.get, claim, state(now), ClaimEvaluation.Context())
      .checks
      .forall(_.holds)

  private def section(ref: NativeClaims.Ref): java.lang.Class[?] =
    ClaimEvaluation.sectionClass(ref.machine, ref.section).get

  private def vars(ref: NativeClaims.Ref): Vars = ref.machine match
    case m: RecordMachine[?, ?, ?] => m.stateVars.get
    case other                     => throw IllegalArgumentException(s"${other.name} has no vars")

  private def state(value: Any): State = value match
    case s: State => s
    case other    => throw IllegalArgumentException(s"$other is no state of a Machine's vars")
