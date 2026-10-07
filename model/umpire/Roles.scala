package umpire

import scala.reflect.{ClassTag, TypeTest}

// A role's phase cases, checked when a reader first needs them so an explicit override needs no
// case of that role. Readers share the finite catalog, witness and missing-role diagnostic.
private[umpire] def roleCases[P, R](owner: String, role: String)(using
    finite: Finite[P],
    witness: TypeTest[P, R],
    phaseType: ClassTag[P]
): IndexedSeq[P] =
  val cases = finite.values.filter(p => witness.unapply(p).nonEmpty)
  require(
    cases.nonEmpty,
    s"$owner's phase type ${phaseType.runtimeClass.getName} has no $role case"
  )
  cases

// The roles of a phase: what a case of a phase enum says about the entity it is the phase of, mixed
// into the case, `case backingOff extends Phase, Retrying`. A role is metadata and not another state
// dimension: it adds no field and no state, and a phase's `Finite` values are those of the enum
// without roles. A phase with no role is one where the entity does not exist yet.
//
// A narrower role extends the broader one it implies, so a `Retrying` phase is `Waiting` and `Live`.
// A Model declares a role of its own by extending one of these, `trait Expired extends TimedOut`; a
// trait that extends none is not a role. A phase is never both `Live` and `Closed`, never more than
// one of `Waiting`, `Held` and `Suspended`, and never more than one closure role, which the IR
// generator (model/irgen/Roles.scala) checks of every enum whose cases take roles.
//
// A test of a phase against a role, `p.isInstanceOf[Closed]` or `case _: Closed`, is the membership
// of the phase in the cases that have the role. Core form: `p.in(<those cases, in declaration
// order>)`, and the pattern `case Phase.done | Phase.failed`.

// The entity exists and is not over.
trait Live

// It exists, and no attempt is held.
trait Waiting extends Live

// It backs off between attempts.
trait Retrying extends Waiting

// Its attempt is held by whoever runs it.
trait Held extends Live

// It is paused.
trait Suspended extends Live

// The entity is over.
trait Closed

// It ended as it was asked to.
trait Succeeded extends Closed

// It ended in a failure.
trait Failed extends Closed

// It was canceled.
trait Canceled extends Closed

// It was terminated.
trait Terminated extends Closed

// It ran out of time.
trait TimedOut extends Closed
