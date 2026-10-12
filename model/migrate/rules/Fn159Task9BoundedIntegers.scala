package umpire.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task9BoundedIntegers extends SyntacticRule("Fn159Task9BoundedIntegers"):
  override def fix(using doc: SyntacticDocument): Patch =
    val patches = scala.collection.mutable.ArrayBuffer.empty[Patch]
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    val records = doc.tree.collect { case c: Defn.Class if c.mods.exists(_.isInstanceOf[Mod.Case]) => c }
    val givens = doc.tree.collect { case g: Defn.GivenAlias => g }
    val fields = scala.collection.mutable.Map.empty[(String, String), Int]
    val inboxes = scala.collection.mutable.Map.empty[(String, String), Int]
    val channels = scala.collection.mutable.Map.empty[String, Int]
    def refuse(at: Tree): Unit =
      errors += Diagnostic("unsafe-bounded-integer", "manually resolve the field/channel's literal nonnegative bound and every constructor, copy and send writer; use UpTo[N] and UpTo[N](value), then remove only its retired local Finite[Int] given", at.pos)
    def applied(t: Type, name: String): Boolean = t match
      case a: Type.Apply => a.tpe.syntax == name
      case _ => false
    def params(c: Defn.Class) = c.ctor.paramClauses.flatMap(_.values)
    def intFields(c: Defn.Class) = params(c).filter(_.decltpe.exists(_.syntax == "Int"))
    def bound(t: Term): Option[Int] = t match
      case a: Term.Apply if a.fun.syntax == "Finite.upTo" => a.argClause.values match
        case List(Lit.Int(n)) if n >= 0 => Some(n)
        case _ => None
      case _ => None
    def removeLine(t: Tree): Unit =
      val start = doc.input.text.lastIndexOf('\n', t.pos.start - 1) + 1
      val end = doc.input.text.indexOf('\n', t.pos.end) match
        case -1 => t.pos.end
        case n => n + 1
      if doc.input.text.substring(start, t.pos.start).trim.isEmpty && doc.input.text.substring(t.pos.end, end).trim.isEmpty then
        patches += Patch.removeTokens(doc.tokens.filter(t => start <= t.pos.start && t.pos.end <= end))
      else refuse(t)
    for g <- givens if applied(g.decltpe, "Finite") do
      val owner = g.decltpe.asInstanceOf[Type.Apply].argClause.values.head.syntax
      records.find(_.name.value == owner).foreach: c =>
        if intFields(c).nonEmpty then
          g.body match
            case b: Term.Block =>
              val ranges = b.stats.collect { case r: Defn.GivenAlias if r.decltpe.syntax == "Finite[Int]" => r }
              if ranges.size != 1 || bound(ranges.head.body).isEmpty || b.stats.last.syntax != "Finite.derived" then refuse(g)
              else
                val n = bound(ranges.head.body).get
                for p <- intFields(c) do
                  fields((owner, p.name.value)) = n
                  patches += Patch.replaceTree(p.decltpe.get, s"UpTo[$n]")
                removeLine(ranges.head)
            case _ => refuse(g)
    for v <- doc.tree.collect { case v: Defn.Val if v.decltpe.exists(applied(_, "Channel")) => v } do
      if v.decltpe.get.syntax == "Channel[Int]" then
        v.rhs match
          case a: Term.ApplyUsing if a.fun.isInstanceOf[Term.Apply] && a.argClause.values.size == 1 && bound(a.argClause.values.head).nonEmpty =>
            val call = a.fun.asInstanceOf[Term.Apply]
            if call.fun.syntax != "channel[Int]" || v.pats.size != 1 || !v.pats.head.isInstanceOf[Pat.Var] then refuse(v)
            else
              val name = v.pats.head.syntax
              val n = bound(a.argClause.values.head).get
              channels(name) = n
              patches += Patch.replaceTree(v.decltpe.get, s"Channel[UpTo[$n]]")
              patches += Patch.replaceTree(a, s"channel[UpTo[$n]]${call.argClause.syntax}")
          case _ => refuse(v)
    for g <- givens if applied(g.decltpe, "Finite") && g.body.isInstanceOf[Term.Block] do
      val owner = g.decltpe.asInstanceOf[Type.Apply].argClause.values.head.syntax
      val associations = g.body.asInstanceOf[Term.Block].stats.collect {
        case r: Defn.GivenAlias if r.decltpe.syntax == "Finite[Inbox[Int]]" => r
      }
      if associations.nonEmpty && channels.nonEmpty then
        val matches = associations.flatMap(r => channels.get(r.body.syntax.stripSuffix(".contents")).map(n => (r, n)))
        val cls = records.filter(_.name.value == owner)
        val held = cls.flatMap(params).filter(_.decltpe.exists(_.syntax == "Inbox[Int]"))
        if associations.size != 1 || matches.size != 1 || cls.size != 1 || held.size != 1 then refuse(g)
        else
          val (givenInbox, n) = matches.head
          patches += Patch.replaceTree(givenInbox.decltpe, s"Finite[Inbox[UpTo[$n]]]")
          patches += Patch.replaceTree(held.head.decltpe.get, s"Inbox[UpTo[$n]]")
          inboxes((owner, held.head.name.value)) = n
    val changed = fields.nonEmpty || channels.nonEmpty
    if changed then
      val imports = doc.tree.collect { case i: Importer => i }
      val owned = imports.exists(i => i.ref.syntax == "framework" && i.importees.exists(_.isInstanceOf[Importee.Wildcard]))
      val shadows = doc.tree.collect {
        case n: Name if Set("UpTo", "Finite", "channel", "framework")(n.value) && n.parent.exists {
          case _: Pat.Var | _: Term.Param | _: Defn.Class | _: Defn.Object | _: Defn.Def => true
          case _ => false
        } => n
      }
      if !owned || imports.exists(i => i.importees.exists(_.isInstanceOf[Importee.Wildcard]) && !Set("framework", "framework.Finite")(i.ref.syntax)) || shadows.nonEmpty || records.map(_.name.value).distinct.size != records.size then refuse(doc.tree)
      def numeric(t: Term, parameter: Option[(String, String)]): Boolean = t match
        case _: Lit.Int => true
        case s: Term.Select => parameter.exists((p, c) => s.qual.syntax == p && fields.contains((c, s.name.value)))
        case a: Term.ApplyInfix => Set("+", "-")(a.op.value) && numeric(a.lhs, parameter) && a.argClause.values.forall(numeric(_, parameter))
        case a: Term.Apply => a.fun match
          case s: Term.Select => Set("min", "max")(s.name.value) && numeric(s.qual, parameter) && a.argClause.values.forall(numeric(_, parameter))
          case _ => false
        case _ => false
      def wrap(t: Term, n: Int, parameter: Option[(String, String)]): Unit =
        if numeric(t, parameter) then patches += Patch.replaceTree(t, s"UpTo[$n](${t.syntax})")
        else refuse(t)
      def parameter(at: Tree): Option[(String, String)] =
        Iterator.iterate(at.parent)(_.flatMap(_.parent)).takeWhile(_.nonEmpty).flatten.collectFirst { case d: Defn.Def => d }.flatMap(d =>
          val ps = d.paramClauseGroups.flatMap(_.paramClauses).flatMap(_.values).filter(p => p.decltpe.exists(t => records.exists(_.name.value == t.syntax)))
          Option.when(ps.size == 1)((ps.head.name.value, ps.head.decltpe.get.syntax))
        )
      for a <- doc.tree.collect { case a: Term.Apply => a } do
        a.fun match
          case n: Term.Name if fields.keys.exists(_._1 == n.value) =>
            val c = records.find(_.name.value == n.value).get
            val ps = params(c)
            val args = a.argClause.values
            if args.size != ps.size then refuse(a)
            else for (p, arg) <- ps.zip(args) if fields.contains((c.name.value, p.name.value)) do
              arg match
                case named: Term.Assign if named.lhs.syntax == p.name.value => wrap(named.rhs, fields((c.name.value, p.name.value)), parameter(a))
                case _: Term.Assign => refuse(arg)
                case t => wrap(t, fields((c.name.value, p.name.value)), parameter(a))
          case s: Term.Select if s.name.value == "copy" =>
            parameter(a).foreach: (p, c) =>
              if s.qual.syntax == p then
                for case arg: Term.Assign <- a.argClause.values if fields.contains((c, arg.lhs.syntax)) do
                  wrap(arg.rhs, fields((c, arg.lhs.syntax)), Some((p, c)))
          case s: Term.Select if s.name.value == "send" =>
            parameter(a) match
              case Some((p, c)) => s.qual match
                case receiver: Term.Select if receiver.qual.syntax == p && inboxes.contains((c, receiver.name.value)) && a.argClause.values.size == 1 =>
                  wrap(a.argClause.values.head, inboxes((c, receiver.name.value)), Some((p, c)))
                case _ => refuse(a)
              case _ => refuse(a)
          case _ => ()
      for n <- doc.tree.collect { case n: Term.Name if fields.keys.exists(_._1 == n.value) => n } do
        if !n.parent.exists {
          case a: Term.Apply => a.fun eq n
          case _ => false
        } then refuse(n)
    if errors.nonEmpty then errors.map(Patch.lint).asPatch else patches.asPatch.atomic
