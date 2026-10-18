package framework.dsl
package compose

import Joined.*
import Probe.*

// What run.sh prints: each composition's members and syncs, its reachable graph with every composed
// step, the verdicts of its sections, its witness, the refuted control and the runtime's refusals.
object report:
  private def graph[F](composition: Composition[F]): Unit =
    val runtime = ComposeRuntime(composition)
    val members = runtime.members.map: (m, actual) =>
      val vars = Vars.of(actual).schema.vars.mkString(", ")
      val stands = if actual eq m.declared then "" else s" (in place of ${m.declared.name})"
      s"${m.name} = ${actual.name}$stands { $vars }"
    println(s"members: ${members.mkString("; ")}")
    for (s, (i, a), (j, b)) <- runtime.syncs do
      println(
        s"sync ${s.name}: ${runtime.name(i)}.${NativeCheck.actionName(a)} with " +
          s"${runtime.name(j)}.${NativeCheck.actionName(b)}"
      )
    println(
      s"own steps: ${runtime.own.map((i, d) => s"${runtime.name(i)}.${NativeCheck.actionName(d)}").mkString(", ")}"
    )
    val search = runtime.search()
    println(s"reachable: ${search.states.size} states, ${search.edges.size} steps")
    for (state, at) <- search.states.zipWithIndex do
      val end = if runtime.end(state) then " (may end)" else ""
      println(s"  $at ${ComposeRuntime.shown(runtime, state)}$end")
      for (from, taken, step, to) <- search.edges if from == at do
        println(s"     ${runtime.shown(taken)} -> $to  facts ${step.facts.mkString("[", ", ", "]")}")

  private def results(rs: Vector[NativeCheck.Result]): Unit =
    rs.foreach(r => println(r.toString.linesIterator.map("  " + _).mkString("\n")))

  private def checked[F](composition: Composition[F], sections: Option[AnyRef] = None): Unit =
    val (_, rs, warnings) = checkAll(composition, sections)
    results(rs)
    warnings.foreach(w => println(s"  warning: $w"))

  def main(args: Array[String]): Unit =
    println("== Together (Witness.test.scala): left and right both bind start, synced")
    graph(Together)
    checked(Together)
    println("== Reordered (Expect.test.scala): the same members, right first")
    graph(Reordered)
    checked(Reordered)
    println("== Apart: no sync; Together's sections checked on it")
    graph(Apart)
    checked(Apart, Some(Together))
    println("== LateTogether: Together.withMember(right = LateOther)")
    graph(LateTogether)
    checked(LateTogether)
    println("   Together's sections checked on LateTogether:")
    checked(LateTogether, Some(Together))
    println("== Queue (SEMANTICS section 17's example, cut down): each action bound by one member")
    graph(Scheduling.Queue)
    checked(Scheduling.Queue)
    println("== negative control: invariant(check(left.phase == right.phase)) on Together")
    val sameSides = invariant(check(Together.left.phase == Together.right.phase))
    results(Vector(ComposeRuntime.verify(ComposeRuntime(Together).search(), sameSides)))
    println("== refused when read")
    for (n, why) <- Vector(
        "Doubled" -> refusal(ComposeRuntime(Refused.Doubled)),
        "Ambiguous" -> refusal(ComposeRuntime(Refused.Ambiguous)),
        "Unbound" -> refusal(ComposeRuntime(Refused.Unbound)),
        "withMember(rigth = ...)" -> refusal(Together.withMember(rigth = LateOther)),
        "withMember(right = Subject)" -> refusal(Together.withMember(right = Subject))
      )
    do println(s"  $n: $why")
