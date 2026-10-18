package framework.dsl

import scala.annotation.StaticAnnotation
import framework.realize.{Command, Instruction}
import scalapb.{GeneratedEnum, GeneratedOneof}

// Source metadata: the lifter reads the typed arguments without evaluating them.
final class gap(val subjects: Subject*)(val because: String, val until: String = "")
    extends StaticAnnotation

sealed abstract class Subject(val kind: String)

final case class silent[P](action: ClassRef | Action[?] | CompositionTarget, in: P*)(using
    val domain: Finite[P]
) extends Subject("silent-rejection")

final case class byDefault[P](action: ClassRef | Action[?] | CompositionTarget, in: P*)(using
    val domain: Finite[P]
) extends Subject("disabled-by-default")

final case class neverEnabled(actions: (ClassRef | Action[?] | CompositionTarget)*)
    extends Subject("never-enabled")

final case class unconstrained(classes: (ClassRef | Action[?] | CompositionTarget)*)
    extends Subject("unconstrained-result")

final case class unproduced[A](values: A*)(using val domain: Finite[A])
    extends Subject("unproduced")

final case class unperformed(actions: Action[?]*) extends Subject("unperformed-action")

final case class unevidenced[F](facts: F*)(using val domain: Finite[F])
    extends Subject("unevidenced-fact")

final case class uncovered(classes: (ClassRef | Action[?] | CompositionTarget)*)
    extends Subject("uncovered-class")

final case class unmapped(values: (GeneratedEnum | GeneratedOneof)*)
    extends Subject("unmodeled-api-value")

case object unrealized extends Subject("unrealized-find")

case object unread extends Subject("unread-refinement")

case object witnessOnly extends Subject("witness-only")
final case class untaken(choices: Choice*) extends Subject("untaken-choice")

case object unasked extends Subject("unasked-property")

case object unfired extends Subject("unfired-verify")

final case class unreachable[A](values: A*)(using val domain: Finite[A])
    extends Subject("unreachable-value")

final case class stuck[S](states: S*)(using val domain: Data[S])(using StateData[S])
    extends Subject("stuck-state")

final case class unreadObservation(observations: Observation*) extends Subject("unread-observation")
final case class explicitWait(polls: (Command | Instruction)*) extends Subject("explicit-wait")

final case class unreachableBinding(classes: (ClassRef | Action[?] | CompositionTarget)*)
    extends Subject("unreachable-binding")

final case class mustNotPinned[P](action: ClassRef | Action[?] | CompositionTarget, in: P*)(using
    val domain: Finite[P]
) extends Subject("must-not-pinned")
