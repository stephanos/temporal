package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task5CentralBounds extends SyntacticRule("Fn159Task5CentralBounds"):
  override def fix(using doc: SyntacticDocument): Patch =
    val budgets = Map(
      (2, 2, 512) -> "two",
      (3, 3, 4096) -> "three",
      (4, 4, 32768) -> "four",
      (5, 5, 65536) -> "five",
      (6, 6, 262144) -> "six",
      (7, 7, 262144) -> "seven",
      (8, 8, 262144) -> "eight",
      (12, 12, 262144) -> "twelve",
      (4, 4, 1048576) -> "fourWide",
      (5, 5, 1048576) -> "fiveWide",
      (12, 12, 4194304) -> "twelveWide"
    )
    def integer(t: Term): Option[Int] = t match
      case Lit.Int(n)                                                      => Some(n)
      case a: Term.ApplyInfix if a.lhs.syntax == "1" && a.op.value == "<<" =>
        a.argClause.values match
          case List(Lit.Int(n)) if n >= 0 && n < 31 => Some(1 << n)
          case _                                    => None
      case _ => None
    def central(t: Tree): Boolean = t.parent.exists {
      case o: Defn.Object if o.name.value == "Bounds" =>
        doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".") == "temporal"
      case parent => central(parent)
    }
    val calls = doc.tree
      .collect {
        case a: Term.Apply if Set("Limits", "framework.Limits")(a.fun.syntax) => a
      }
      .filterNot(central)
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    val edits = scala.collection.mutable.ArrayBuffer.empty[Patch]
    def refuse(at: Tree): Unit =
      errors += Diagnostic(
        "unsafe-central-bound",
        "cannot centralize this bound safely: manually inventory every external user, choose the matching temporal.Bounds budget, qualify its uses, then remove the local declaration; unknown budgets need an explicit Bounds decision",
        at.pos
      )
    val blocks = calls.flatMap(
      _.parent
        .collect { case v: Defn.Val => v }
        .flatMap(
          _.parent.collect { case b: Term.Block => b }
        )
    )
    if blocks.distinct.size != blocks.size || blocks.distinct.exists(b =>
        blocks.exists(other => b.pos.start < other.pos.start && other.pos.end < b.pos.end)
      )
    then calls.foreach(refuse)
    for call <- calls do
      call.parent match
        case Some(v: Defn.Val)
            if v.pats.size == 1 && v.pats.head.isInstanceOf[Pat.Var] &&
              v.parent.exists(_.isInstanceOf[Term.Block]) =>
          val block = v.parent.get.asInstanceOf[Term.Block]
          val old = v.pats.head.asInstanceOf[Pat.Var].name.value
          val values = call.argClause.values.collect { case Term.Assign(Term.Name(key), value) =>
            key -> integer(value)
          }
          val budget =
            if values.map(_._1).toSet == Set("steps", "actions", "search") && values.size == 3 then
              val fields = values.toMap
              for s <- fields("steps"); a <- fields("actions"); n <- fields("search")
              yield (s, a, n)
            else None
          budget.flatMap(budgets.get) match
            case None       => refuse(call)
            case Some(name) =>
              val bindings = block.collect {
                case p: Pat.Var if p.name.value == old     => p
                case p: Term.Param if p.name.value == old  => p
                case d: Defn.Def if d.name.value == old    => d
                case d: Defn.Object if d.name.value == old => d
              }
              val imports = doc.tree.collect { case i: Import => i }
              val visible = imports.filter(i =>
                i.pos.start < v.pos.start && i.parent.exists(p =>
                  p.pos.start <= block.pos.start && block.pos.end <= p.pos.end
                )
              )
              val competing = visible.exists(
                _.importers.exists(i =>
                  i.ref.syntax != "framework" && !Set("Bounds", "temporal.Bounds")(i.ref.syntax) &&
                    i.importees.exists {
                      case n: Importee.Name =>
                        n.name.value == old || n.name.value == "temporal" || n.name.value == "Limits"
                      case n: Importee.Rename   => Set(old, "temporal", "Limits")(n.rename.value)
                      case _: Importee.Wildcard => true
                      case _                    => false
                    }
                )
              ) || block.collect {
                case n: Name
                    if Set("temporal", "Bounds", "Limits")(n.value) &&
                      n.pos.start != call.fun.pos.start && !n.parent.exists(
                        _.isInstanceOf[Term.Select]
                      ) =>
                  n
                case _: Import => block
              }.nonEmpty
              val targetShadowed = doc.tree.collect {
                case p: Pat.Var if p.name.value == name && p.pos.start != v.pats.head.pos.start => p
                case p: Term.Param if p.name.value == name                                      => p
                case d: Defn.Def if d.name.value == name                                        => d
                case d: Defn.Object if d.name.value == name                                     => d
              }.nonEmpty
              val qualifierShadowed = doc.tree.collect {
                case p: Pat.Var if Set("temporal", "Bounds", "framework", "Limits")(p.name.value) =>
                  p
                case p: Term.Param
                    if Set("temporal", "Bounds", "framework", "Limits")(p.name.value) =>
                  p
                case d: Defn.Def
                    if Set("temporal", "Bounds", "framework", "Limits")(d.name.value) =>
                  d
                case d: Defn.Object
                    if Set("temporal", "Bounds", "framework", "Limits")(d.name.value) =>
                  d
              }.nonEmpty
              val uses = block.collect {
                case n: Term.Name if n.value == old && n.pos.start != v.pats.head.pos.start => n
              }
              val unsafe = uses.exists(n =>
                n.pos.start < v.pos.end || n.parent.exists {
                  case s: Term.Select => s.name.pos.start == n.pos.start
                  case _              => false
                }
              )
              val removedComments = block.tokens.exists {
                case c: Token.Comment =>
                  !block.stats
                    .filterNot(_ == v)
                    .exists(t => t.pos.start <= c.pos.start && c.pos.end <= t.pos.end)
                case _ => false
              }
              if bindings.size != 1 || competing || targetShadowed || qualifierShadowed ||
                removedComments || unsafe || uses.isEmpty
              then refuse(v)
              else
                val imported = visible.exists(
                  _.importers.exists(i =>
                    Set("Bounds", "temporal.Bounds")(i.ref.syntax) && i.importees.exists {
                      case n: Importee.Name => n.name.value == name
                      case _                => false
                    }
                  )
                )
                val replacement = if imported then name else "temporal.Bounds." + name
                val changes = uses.map(n => (n.pos.start, n.pos.end, replacement)) :+
                  (v.pos.start, v.pos.end, "")
                val remaining = block.stats.filterNot(_ == v)
                if remaining.size == 1 && remaining.head.isInstanceOf[Term] then
                  val term = remaining.head
                  val rewritten = changes
                    .filter((s, e, _) => term.pos.start <= s && e <= term.pos.end)
                    .sortBy(-_._1)
                    .foldLeft(term.pos.text) { case (text, (s, e, value)) =>
                      text.take(s - term.pos.start) + value + text.drop(e - term.pos.start)
                    }
                  edits += Patch.replaceTree(block, rewritten)
                else
                  edits += Patch.replaceTree(v, "")
                  edits ++= uses.map(n => Patch.replaceTree(n, replacement))
        case _ => refuse(call)
    if errors.nonEmpty then errors.map(Patch.lint).asPatch
    else edits.toSeq.asPatch.atomic
