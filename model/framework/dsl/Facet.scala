package framework.dsl

import scala.collection.mutable

// A facet: one behaviour area of one feature's Machine at one level, named as a noun for what
// happens (`Pausing`, `Cancellation`), distinct from a capability, a reusable contract named
// as an adjective for what an entity can be (`Holdable`), which a facet declares. A facet owns the
// actions it binds: their transitions, the effects those transitions call and the claims inside its
// namespace that check them. The Machine stays one Machine, assembled from its facets.
//
// A facet is a trait the Machine object mixes in, in its own file named after it, holding one object
// named after it whose members are its sections (model/README.md, "Facets"). It declares the vars it
// reads and assigns, the phase and any state it shares with other facets, abstractly, and the
// Machine defines them:
//
// {{{
// trait Pausing extends Facet[Outcome, Fact]:
//   this: Dispatch =>
//   var phase: Phase
//   object pausing:
//     object effects: ...
//     object transitions extends Transitions: ...
// }}}
//
// A facet that uses another's step names it in its self-type, `this: Dispatch =>`, so a Machine
// that does not mix that facet in does not compile, and a var a facet declares that the Machine does
// not define is Scala's abstract-member error. The Machine object mixes in its facets, defines their
// vars, in the order its state lists them, and lists their transitions, in facet order: `object
// ActivityProtocol extends Machine[Outcome, Fact], Dispatch, Pausing` with `var phase =
// Phase.unstarted` and `object transitions extends Facets(dispatch.transitions,
// pausing.transitions)`. What stays Machine-wide stays in the Machine object: its vars, `end`, the
// refinement, monitors, properties and queries over the whole feature. A facet declares the
// capabilities of the actions it owns in its object's `object capabilities extends Capabilities`,
// and the Machine lists them as it lists their transitions, `object capabilities extends
// FacetCapabilities(pausing.capabilities, ...)`. Core form: the Machine whose `transitions` holds
// every facet's blocks, in facet order, whose `effects` holds every facet's effects, and whose
// `capabilities` holds every facet's capabilities and waivers.
trait Facet[O, F] extends Machine[O, F], RecordFacet[State, O, F]

// A facet of a machine over a state record, the hidden form the framework's own fixtures keep
// until the record machine is retired: the sections of a facet read the owner of the machine that
// mixes it in. Core form: the machine with the facet's sections as its own.
private[framework] trait RecordFacet[S, O, F] extends RecordMachine[S, O, F]:
  // The owner its sections read the machine's types from: the machine that mixes it in. A facet's
  // sections cannot read the machine's own given, a val of the class `RecordMachine`, from a trait,
  // and this one, of a subclass of it, is preferred to it in the machine object as well.
  protected given facetOwner: Owner[S, O, F] = Owner(this)

  // The machine a facet's `capabilities` section declares the capabilities of, read as `facetOwner`
  // is, from a val of the facet rather than the machine's own.
  protected given facetDeclaring: Declaring[S, O, F] = Declaring(this)

// A Machine's transitions assembled from its facets' transitions, in the order listed: each
// action's blocks are the blocks of the one facet that binds it. An action bound by two facets is
// refused as the transitions construct, naming the action and both facets. Core form: one `object
// transitions extends Transitions` holding every facet's blocks in facet order.
abstract class Facets[S, O, F](using
    owner: Owner[S, O, F]
)(
    facets: RuleBook[S, O, F]*
) extends RuleBook[S, O, F]:
  private[framework] val assembly: RecordMachine[S, O, F] = owner.machine
  private[framework] val contributors: Vector[RuleBook[S, O, F]] = facets.toVector
  private[framework] lazy val table: Vector[(ActionDecl, Bound[S, O, F])] =
    val owners = mutable.LinkedHashMap.empty[ActionDecl, String]
    facets.toVector.flatMap { rules =>
      val facet = rules.facetSource.getOrElse(
        throw IllegalArgumentException(
          s"${rules.sectionName} is not a facet's transitions: Facets lists the transitions declared in its facets"
            + s"; ${rules.table.map((decl, _) => rules.actionName(decl)).mkString(", ")} has no owning facet"
        )
      )
      rules.table.map { (decl, bound) =>
        for other <- owners.get(decl) do
          val action = rules.actionName(decl)
          throw IllegalArgumentException(
            s"$action is bound by the facets $other and $facet: an action belongs to one facet, " +
              "which holds all its transitions"
          )
        owners(decl) = facet
        decl -> bound
      }
    }

// A machine's capabilities assembled from its facets' `capabilities` sections, in the order listed:
// each facet's vals and waivers are the section's, so the machine's Queries bound them as one
// section, `capabilities.bound(three)`. Core form: one `object capabilities extends Capabilities`
// holding every facet's capabilities and waivers in facet order.
abstract class FacetCapabilities[S, O, F](using Declaring[S, O, F])(
    val facets: Capabilities[S, O, F]*
) extends Capabilities[S, O, F]
