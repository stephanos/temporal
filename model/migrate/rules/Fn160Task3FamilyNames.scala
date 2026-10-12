package umpire.migrate

import scala.meta.*
import scalafix.v1.*

// fn-160.3 (R7): machines renamed into their family prefix, so a level folder's files sort by
// family. The rule renames every reference to an inventoried object; moving each file to the new
// name is the manual step (Scalafix cannot move files; model/migrate/README.md, "Flat layout").
// The shared task queue's providers are renamed wherever they are named; the mini's own objects
// only inside the mini's packages, where they are declared.
final class Fn160Task3FamilyNames extends SyntacticRule("Fn160Task3FamilyNames"):
  override def fix(using doc: SyntacticDocument): Patch =
    val owner = doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".")
    val mini = owner == "temporal.features.activity.mini" ||
      owner.startsWith("temporal.features.activity.mini.")
    val renames = Fn160Task3FamilyNames.shared ++ (if mini then Fn160Task3FamilyNames.mini else Map.empty)
    val names = doc.tree.collect {
      case n: Term.Name if renames.contains(n.value) => n
      case n: Type.Name if renames.contains(n.value) => n
      case n: Name.Indeterminate if renames.contains(n.value) => n
    }
    val targets = renames.filter((old, _) => names.exists(_.value == old)).values.toSet
    val collisions = doc.tree.collect { case n: Name if targets(n.value) => n }
    if names.isEmpty then Patch.empty
    else if collisions.nonEmpty then
      Patch.lint(
        Diagnostic(
          "family-name-collision",
          s"${collisions.head.value} is already named here: rename by hand, resolving the collision",
          collisions.head.pos
        )
      )
    else names.map(n => Patch.replaceTree(n, renames(n.value))).asPatch.atomic

object Fn160Task3FamilyNames:
  // The task queue foundation's derived providers (family queue).
  val shared: Map[String, String] = Map(
    "ForgetfulQueue" -> "QueueForgetful",
    "LossyMatchingQueue" -> "QueueLossyMatching",
    "VolatileQueue" -> "QueueVolatile"
  )
  // activity_mini's designs (families answer and queue, facet dispatching).
  val mini: Map[String, String] = Map(
    "TrustingAnswers" -> "AnswerTrusting",
    "ScheduledQueue" -> "QueueScheduled",
    "DispatchProgress" -> "DispatchingProgress"
  )
  // The file moves that go with the renames, and the one move of a file with no machine.
  val moves: Map[String, String] = (shared ++ mini).map((o, n) => s"$o.scala" -> s"$n.scala") +
    ("Faults.scala" -> "AnswerFaults.scala")
