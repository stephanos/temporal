package framework

import scala.collection.mutable

// A facet: one behaviour area of one feature's machine at one level, named as a noun for what
// happens (`Pausing`, `Cancellation`), distinct from a capability, a cross-Temporal contract named
// as an adjective for what an entity can be (`Pausable`), which a facet declares. A facet owns the
// actions it binds: their rules, the effects those rules use and, beside it in its file, the claims
// that check them. The machine stays one machine, assembled from its facets.
//
// A facet is a trait the machine object mixes in, in its own file named after it, holding one object
// named after it whose members are its sections, `effects` and `rules` (model/README.md, "Facets"):
//
// {{{
// trait Pausing extends Facet[State, Outcome, Fact], Phased[State, Phase]:
//   this: Dispatch =>
//   object pausing:
//     object effects: ...
//     object rules extends Rules: ...
// }}}
//
// A facet that uses another's step names it in its self-type, `this: Dispatch =>`, so a machine
// that does not mix that facet in does not compile. The machine object mixes in its facets and lists
// their rules, in facet order: `object ActivityProtocol extends Machine[...], Phased[...](...),
// Dispatch, Pausing` with `object rules extends Facets(dispatch.rules, pausing.rules)`. What stays
// machine-wide stays in the machine object: its state and phase types, `init`, `end`, the projection,
// the refinement, monitors, properties and queries over the whole feature. A facet declares the
// capabilities of the actions it owns in its object's `object capabilities extends Capabilities`,
// and the machine lists them as it lists their rules, `object capabilities extends
// FacetCapabilities(pausing.capabilities, ...)`. Core form: the machine whose `rules` holds every
// facet's blocks, in facet order, whose `effects` holds every facet's effects, and whose
// `capabilities` holds every facet's capabilities and waivers.
trait Facet[S, O, F] extends Machine[S, O, F]:
  // The owner its sections read the machine's types from: the machine that mixes it in. A facet's
  // sections cannot read the machine's own given, a val of the class `Machine`, from a trait, and
  // this one, of a subclass of `Machine`, is preferred to it in the machine object as well.
  protected given facetOwner: Owner[S, O, F] = Owner(this)

  // The machine a facet's `capabilities` section declares the capabilities of, read as `facetOwner`
  // is, from a val of the facet rather than the machine's own.
  protected given facetDeclaring: Declaring[S, O, F] = Declaring(this)

// A machine's rules assembled from its facets' rules, in the order listed: each action's cases are
// the cases of the one facet that binds it. An action bound by two facets is refused as the rules
// construct, naming the action and both facets. Core form: one `object rules extends Rules` holding
// every facet's blocks in facet order.
abstract class Facets[S, O, F](using @scala.annotation.unused owner: Owner[S, O, F])(
    facets: RuleBook[S, O, F]*
) extends RuleBook[S, O, F]:
  private[framework] lazy val table: Vector[(ActionDecl, Bound[S, O, F])] =
    val owners = mutable.LinkedHashMap.empty[ActionDecl, String]
    facets.toVector.flatMap { rules =>
      val facet = Facets.nameOf(rules)
      rules.table.map { (decl, bound) =>
        for other <- owners.get(decl) do
          val action = bound match
            case Bound.Ruled(rs) if rs.nonEmpty => rs.head.action
            case _                              => "an action"
          throw IllegalArgumentException(
            s"$action is bound by the facets $other and $facet: an action belongs to one facet, " +
              "which holds all its rules"
          )
        owners(decl) = facet
        decl -> bound
      }
    }

object Facets:
  // The facet a facet's `rules` belongs to: the object named after the facet that holds it.
  private[framework] def nameOf(rules: AnyRef): String =
    rules.getClass.getName.split('$').filter(_.nonEmpty).reverse.drop(1).headOption.getOrElse("?")

// A machine's capabilities assembled from its facets' `capabilities` sections, in the order listed:
// each facet's vals and waivers are the section's, so the machine's Queries bound them as one
// section, `capabilities.bound(three)`. Core form: one `object capabilities extends Capabilities`
// holding every facet's capabilities and waivers in facet order.
abstract class FacetCapabilities[S, O, F](using Declaring[S, O, F])(
    val facets: Capabilities[S, O, F]*
) extends Capabilities[S, O, F]
