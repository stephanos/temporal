package framework.dsl

// The words a Machine's claims and effects are written in where they read its vars by name, which
// a Machine and a derivation of one inherit, so that inside the object they are these. Each reads
// where it is written (ClaimMacros): in an authored run it reads that run's exact Model context;
// in the Machine's sections it asserts natively (NativeClaims), and outside the contexts it belongs
// in it is refused.
//
//   - `after(target):` a block of a postcondition about the steps of one action, or class of it;
//   - `before`: the State record the step started from, in a postcondition or a monitor;
//   - `records(fact)`: in a claim, that the step recorded the fact; in an effect, a fact it records;
//   - `oneOf:` with `branch:` blocks: alternatives, of which one holds.
trait ClaimWords:
  protected transparent inline def after: Any = ${ ClaimMacros.after }
  protected transparent inline def before: Any = ${ ClaimMacros.before }
  protected transparent inline def records: Any = ${ ClaimMacros.records }
  protected transparent inline def oneOf: Any = ${ ClaimMacros.oneOf }
  protected transparent inline def branch: Any = ${ ClaimMacros.branch }

private[framework] trait NativeClaimWords[O, F] extends ClaimWords:
  final def monitor[M](initial: M)(
      next: M => M
  )(violated: M => Boolean)(using Data[M])(using StateData[M]): Monitor[State, O, F, M] =
    framework.dsl.monitor[State, O, F, M](initial)((m, before, step) =>
      NativeRuntime.installed.monitorNext(next, m, before, step)
    )(violated)

  final def monitor[S, MO, MF, M](initial: M)(next: (M, S, Step[S, MO, MF]) => M)(
      violated: M => Boolean
  )(using Data[M])(using StateData[M]): Monitor[S, MO, MF, M] =
    framework.dsl.monitor[S, MO, MF, M](initial)(next)(violated)

  final def monitor[S, MO, MF, M](name: String, initial: M)(next: (M, S, Step[S, MO, MF]) => M)(
      violated: M => Boolean
  )(using Data[M])(using StateData[M]): Monitor[S, MO, MF, M] =
    framework.dsl.monitor[S, MO, MF, M](name, initial)(next)(violated)

  final def sticky(promise: => Boolean): Monitor[State, O, F, Boolean] =
    monitor(false)(Programs.StickyVar(() => promise))(identity)

  final def facts: List[F] = NativeClaims.facts.asInstanceOf[List[F]]
  final def recorded(fact: F): Boolean = NativeClaims.facts.contains(fact)
