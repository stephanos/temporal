// Capabilities: what a machine declares it can do, binding each protocol's parameters to its own
// vocabulary. A capability kind's companion defines the Properties it brings. The IR generator
// expands them for each machine's `capabilities` section, and its `queries` section bounds their
// generated Queries.
package umpire

import scala.annotation.unused

// A capability of a machine whose steps are `Step[S, O, F]`: a predicate of another state type, an
// outcome of another type or a fact of another type does not compile. A feature kit declares each
// capability as a case class whose companion is its `CapabilityKind`. The lifter reads a field of
// an action class as an action the machine must bind, a list of them as the path to a live state a
// functional Property's find starts from, and a `RunExpectation` as the Run that find expects.
trait CapabilityOf[S, +O, +F]

// A capability kind: the companion object of a capability binding. The IR generator uses its
// qualified name to find the Properties the companion defines and to match their parameters to
// the fields declared by a machine.
trait CapabilityKind:
  def name: String = getClass.getSimpleName.stripSuffix("$")

// A machine or composition object's capabilities, declared by its `capabilities` section, one named
// val per capability, typed `Capability`, and the capability Properties it waives as statements:
//
// {{{
// object capabilities extends Capabilities:
//   val lockable: Capability = Lockable(status = states.phase, locked = states.locked, ...)
//   val holdable: Capability = Holdable(hold = ..., release = ..., held = states.held)
//   except(Lockable.lockedIsRefused, because = "...")
// }}}
//
// A capability kind's companion defines the Properties it brings, each a def that takes the model,
// then fields of declared capabilities by name; declaring the capabilities whose fields a def reads
// brings it. The IR generator (model/irgen/Capabilities.scala) expands each into a Property, a
// Scenario and a Query named `<machine>.<property>`, the Property with the def as its origin, and
// bounds the Queries where a `queries` section says, `capabilities.bound(three)`. `Capability` is
// the machine's own capability type, so a capability of another machine's states, outcomes or
// facts does not compile, and one whose state type no field names still takes the machine's.
//
// A capability set several designs share, waivers included, is written once as a class of its own
// that extends this one, `abstract class Shared(m: M)(using Declaring[S, O, F])`, which each
// design's section extends, `object capabilities extends Shared(this)`: its vals and waivers are
// the section's.
abstract class Capabilities[S, O, F](using declaring: Declaring[S, O, F]):
  // A capability of this machine or composition.
  protected type Capability = CapabilityOf[S, O, F]

  // The machine or composition the section declares the capabilities of.
  private[umpire] def model: Declares[S] = declaring.model

  // Lifts no Property and no Query for the capability Property `property`, for the reason
  // `because` gives: `except(<Capability>.<property>, because = "…")`.
  protected def except(@unused property: AnyRef, @unused because: String): Unit = ()

  // Lifts `replaced`'s def, which takes the capability Property's parameters, in its place, for the
  // reason `because` gives: `overriding(<Capability>.<property> -> ownDef, because = "…")`.
  protected def overriding(@unused replaced: (AnyRef, AnyRef), @unused because: String): Unit = ()

  // The Property this section generates for the capability Property `property`,
  // `<machine>.<property>`, for a Query of its own to read; the IR generator names it, and refuses
  // one the section does not bring or waives with `except`.
  def claim(@unused property: AnyRef): Property[S] =
    Property(PropertyDecl("", model, None, None, None))

  // Bounds every Query generated from the Properties these capabilities bring under `limits`, and
  // each capability Property `overrides` names under its own: a statement of a `queries` section,
  // `capabilities.bound(three, <Capability>.<property> -> five)`.
  def bound(@unused limits: Limits, @unused overrides: (AnyRef, Limits)*): Unit = ()
