// Capabilities: what a machine declares it can do, binding each protocol's parameters to its own
// vocabulary, and the laws it receives for them.
//
// A machine or composition object declares its own in its `implements` section,
// `object implements extends Implements(limits = three)(…)`, which is the declaration;
// `capabilities(m, limits)(…)`, the form a function declaring them for several designs writes, names
// the machine. Either names the Limits its generated `verify` Queries run under, and its
// capabilities; the given `Catalog` brings the laws of each capability and of each unordered
// pair it declares both of. The IR generator (model/irgen/Capabilities.scala) expands the
// declaration into a Property, a Scenario and a Query per law, each named `<machine>.<law>`, and
// writes what it expanded into the law sidecar beside the IR file. A function-valued field names a
// def of the lifted sources, never a lambda, as a shared def's function-valued argument does.
//
// A machine's `capabilities` section, `object capabilities extends Capabilities`, is the form that
// replaces both: its capabilities bring the Properties their kinds' companions define, and its
// `queries` section bounds their Queries (`Capabilities` below). A machine declares one form.
//
// The framework names no capability: a feature kit declares its kinds and laws (model/umpire/Catalog.scala).
// Everything here is core: it declares what the IR and the sidecar need.
package umpire

import scala.annotation.unused

// A capability of a machine whose steps are `Step[S, O, F]`: a predicate of another state type, an
// outcome of another type or a fact of another type does not compile. A feature kit declares each
// capability as a case class whose companion is its `CapabilityKind`. The lifter reads a field of
// an action class as an action the machine must bind, a list of them as the path to a live state a
// functional law's find starts from, and a `RunExpectation` as the Run that find expects.
trait CapabilityOf[S, +O, +F]

// A law a declaration waives, with the reason: lifted not at all, or as the entity's own def.
enum Waiver:
  case Except(law: Law, because: String)
  case Overriding(law: Law, by: AnyRef, because: String)

// What a machine declares: its capabilities, the catalog that brings their laws, the Limits the
// generated `verify` Queries run under, and the laws it waives.
final class LawDeclaration[S] private[umpire] (
    val model: Declares[S],
    val limits: Limits,
    val declared: Seq[CapabilityOf[S, ?, ?]],
    val catalog: Catalog,
    val waived: Seq[Waiver]
):
  // The Property this declaration generates for `law`, `<machine>.<law>`, for a Query of its own
  // to read: a law the declaration waives with `except` has none.
  def claim(law: Law): Property[S] =
    Property(PropertyDecl(s"${model.name}.${law.name}", model, None, None, None))

  // Lifts no Property and no Query for `law`, for the reason `because` gives.
  def except(law: Law, because: String): LawDeclaration[S] =
    LawDeclaration(model, limits, declared, catalog, waived :+ Waiver.Except(law, because))

  // Lifts `replaced`'s def, which takes the law's parameters, under the law's name, for the reason
  // `because` gives: `overriding(closedIsRejectedUniformly -> ownDef, because = "…")`.
  def overriding(replaced: (Law, AnyRef), because: String): LawDeclaration[S] =
    LawDeclaration(
      model,
      limits,
      declared,
      catalog,
      waived :+ Waiver.Overriding(replaced._1, replaced._2, because)
    )

// A capability field's value, backed by the server code that answers it so, e.g.
// `rejected = cited(Answer.gone, "server/jobs.go")`: each citation is a file path from the
// repository's root, as a law's `cites` are. The IR generator binds the value alone and records
// the citations in the law sidecar beside each claim that binds the field, so Scala reads no
// citation.
def cited[A](value: A, @unused cites: String*): A = value

// Declares the capabilities of `m`, whose generated `verify` Queries run under `limits`, and receives
// the laws the given catalog brings for them.
def capabilities[S](m: Declares[S], limits: Limits)(
    declared: CapabilityOf[S, m.Outcome, m.Fact]*
)(using catalog: Catalog): LawDeclaration[S] =
  LawDeclaration(m, limits, declared, catalog, Nil)

// A machine or composition object's capabilities, declared by its `implements` section itself:
// `object implements extends Implements(limits = three)(capability, ...)`, whose generated `verify`
// Queries run under `limits`. The section is the declaration, so an IR file names it,
// `irFile("orders")(OrderProduct.implements, ...)`, and a Query reads a law's claim off it,
// `OrderProduct.implements.claim(law)`. A law it waives is a statement of its body,
// `except(law, because = ...)` or `overriding(law -> ownDef, because = ...)`.
abstract class Implements[S, O, F](using declaring: Declaring[S, O, F])(limits: Limits)(
    declared: CapabilityOf[S, O, F]*
)(using catalog: Catalog):
  private var waivers = Vector.empty[Waiver] // scalafix:ok DisableSyntax.var

  // What the section declares, with the laws its body waives.
  def capabilities: LawDeclaration[S] =
    LawDeclaration(declaring.model, limits, declared, catalog, waivers)

  // The Property this declaration generates for `law`, `<machine>.<law>`, for a Query of its own
  // to read: a law the declaration waives with `except` has none.
  def claim(law: Law): Property[S] = capabilities.claim(law)

  // Lifts no Property and no Query for `law`, for the reason `because` gives.
  protected def except(law: Law, because: String): Unit = waivers :+= Waiver.Except(law, because)

  // Lifts `replaced`'s def, which takes the law's parameters, under the law's name, for the reason
  // `because` gives: `overriding(closedIsRejectedUniformly -> ownDef, because = "…")`.
  protected def overriding(replaced: (Law, AnyRef), because: String): Unit =
    waivers :+= Waiver.Overriding(replaced._1, replaced._2, because)

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
