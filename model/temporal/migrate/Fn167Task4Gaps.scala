package temporal.migrate

import scala.meta.*

// fn-167.4: the pure transformation from a feature's accepted lint findings to `@gap` stacks on the
// declarations that own them. It reads what the caller resolved: where each finding key is declared
// and with which typed subject, which keys are provably their base's (fn-167.3 provenance), and the
// capability waivers the `except`/`overriding` declarations already state. It writes one `@gap` per
// declaration and exact reason, in the order the acceptances first name them, inserted above the
// declaration and changing no other byte. Anything it cannot place, reduce safely or reconcile is a
// diagnostic, and then no source is returned at all. Reading the acceptance and waiver files is the
// input adapter's (Fn167Task4GapInputs); the runnable IR-backed adapter is fn-167.7's.
object Fn167Task4Gaps:
  type Key = (String, String, String)

  // One acceptance of the existing `<file>.lint.json`: a kind, an owner, its subjects and why.
  final case class Acceptance(kind: String, owner: String, subjects: Seq[String], because: String)

  // Where a finding key is declared: the declaration's source identity (`<package>.<objects>.<name>`)
  // and the typed subject constructor that names it there, as source text.
  final case class Placement(declaration: String, subject: String)

  // A waiver an `except` or `overriding` states: its machine, `<machine>.<property>` and its reason.
  final case class Waiver(machine: String, subject: String, because: String)

  final case class Inputs(
      acceptances: Seq[Acceptance],
      placements: Map[Key, Placement],
      inherited: Map[Key, Key],
      waivers: Seq[Waiver],
      sources: Map[String, String]
  )

  // What was written, and what was dropped with why: a waiver that lowers on its own, or a finding
  // its base's acceptance covers.
  final case class Migrated(
      sources: Map[String, String],
      inserted: Seq[String],
      dropped: Seq[String]
  )

  private val waiverKind = "capability-waiver"

  def migrate(in: Inputs): Either[Seq[String], Migrated] =
    val problems = Seq.newBuilder[String]
    val dropped = Seq.newBuilder[String]
    val keys = in.acceptances.flatMap(a => a.subjects.map(s => ((a.kind, a.owner, s), a.because)))
    for (key, n) <- keys.groupBy(_._1).view.mapValues(_.size) if n > 1 do
      problems += s"${show(key)} is accepted $n times"
    val accepted = keys.map(_._1).toSet
    // Each placed finding: its declaration, reason and typed subject, in the acceptances' order.
    val placed = keys.flatMap { (key, because) =>
      val (kind, owner, subject) = key
      if kind == waiverKind then
        in.waivers.filter(w => w.machine == owner && w.subject == subject) match
          case Seq(w) if w.because == because =>
            dropped += s"${show(key)}: lowered from its except/overriding declaration"
          case Seq(w) =>
            problems += s"${show(key)}: the acceptance's reason \"$because\" is not the " +
              s"declaration's \"${w.because}\"; reconcile them by hand"
          case Seq() => problems += s"${show(key)}: no except/overriding declaration states it"
          case _     => problems += s"${show(key)}: stated by several waivers"
        None
      else
        in.inherited.get(key) match
          case Some(base) if accepted(base) =>
            dropped += s"${show(key)}: inherited from ${show(base)}"
            None
          case Some(base) =>
            problems += s"${show(key)}: inherited from ${show(base)}, which no acceptance covers, " +
              "so dropping it would leave the finding unaccepted"
            None
          case None =>
            in.placements.get(key) match
              case Some(p) => Some((p.declaration, because, p.subject))
              case None    =>
                problems += s"${show(key)}: no declaration is resolved for it"
                None
    }
    val groups = placed
      .groupBy(p => (p._1, p._2))
      .toSeq
      .sortBy((group, _) => placed.indexWhere(p => (p._1, p._2) == group))
      .map { case ((declaration, because), ps) => (declaration, because, ps.map(_._3).distinct) }
    val located = groups.map(_._1).distinct.map(d => d -> locate(in.sources, d)).toMap
    for (d, found) <- located do
      found match
        case Left(problem) => problems += problem
        case Right(_)      => ()
    val reported = problems.result()
    if reported.nonEmpty then Left(reported)
    else
      // Each insertion: its file, offset and text; one existing gap of a reason converges.
      val insertions = Seq.newBuilder[(String, Int, String)]
      val inserted = Seq.newBuilder[String]
      val conflicts = Seq.newBuilder[String]
      for (declaration, because, subjects) <- groups do
        val d = located(declaration).toOption.get
        val same = d.gaps.filter(_.because == because)
        val elsewhere = d.gaps.filter(_.because != because).flatMap(g => g.subjects.map(_ -> g))
        for s <- subjects; (_, g) <- elsewhere.find(_._1 == normal(s)) do
          conflicts += s"$declaration already declares $s a gap because \"${g.because}\""
        if same.exists(g => subjects.map(normal).toSet == g.subjects.toSet) then ()
        else if same.nonEmpty then
          conflicts += s"$declaration already has a gap because \"$because\" naming other subjects"
        else
          insertions += ((d.file, d.lineStart, annotation(d.indent, subjects, because)))
          inserted += s"$declaration: @gap(${subjects.mkString(", ")}) because \"$because\""
      val conflicted = conflicts.result()
      if conflicted.nonEmpty then Left(conflicted)
      else
        val edits = insertions.result().groupBy(_._1)
        val sources = in.sources.map { (file, text) =>
          // Insertions at one offset keep the acceptances' order; later offsets go first.
          val at = edits.getOrElse(file, Nil).groupBy(_._2).toSeq.sortBy(-_._1)
          file -> at.foldLeft(text) { case (t, (offset, texts)) =>
            t.substring(0, offset) + texts.map(_._3).mkString + t.substring(offset)
          }
        }
        Right(Migrated(sources, inserted.result(), dropped.result()))

  private def show(key: Key) = s"${key._1} ${key._2} \"${key._3}\""

  private def normal(subject: String) = subject.filterNot(_.isWhitespace)

  // `@gap(<subjects>)(because = "<reason>")`, on one line where it fits, else with its reason apart.
  private def annotation(indent: String, subjects: Seq[String], because: String): String =
    val reason = literal(because)
    val one = s"$indent@gap(${subjects.mkString(", ")})(because = $reason)"
    if one.length <= 100 then one + "\n"
    else s"$indent@gap(${subjects.mkString(", ")})(\n$indent  because = $reason\n$indent)\n"

  private def literal(s: String): String =
    s.flatMap {
      case '"'              => "\\\""
      case '\\'             => "\\\\"
      case '\n'             => "\\n"
      case '\t'             => "\\t"
      case c if c.isControl => f"\\u${c.toInt}%04x"
      case c                => c.toString
    }.mkString("\"", "", "\"")

  final private case class Existing(subjects: Seq[String], because: String)
  final private case class Declared(
      file: String,
      lineStart: Int,
      indent: String,
      gaps: Seq[Existing]
  )

  // The one declaration of the sources with this identity, with where its first line starts and the
  // gaps it already declares.
  private def locate(sources: Map[String, String], identity: String): Either[String, Declared] =
    val found = sources.toSeq.sortBy(_._1).flatMap { (file, text) =>
      dialects.Scala3(Input.VirtualFile(file, text)).parse[Source].toOption.toSeq.flatMap { tree =>
        declarations(tree).collect {
          case (name, d, mods) if name == identity => (file, text, d, mods)
        }
      }
    }
    found match
      case Seq((file, text, d, mods)) =>
        val start = d.pos.start
        val lineStart = text.lastIndexOf('\n', start - 1) + 1
        val indent = text.substring(lineStart, start).takeWhile(_.isWhitespace)
        if text.substring(lineStart, start).exists(!_.isWhitespace) then
          Left(s"$identity does not start its line in $file, so a gap cannot be placed above it")
        else Right(Declared(file, lineStart, indent, mods.flatMap(existing)))
      case Seq() => Left(s"$identity is declared by none of the sources")
      case many  =>
        Left(s"$identity is declared ${many.size} times: ${many.map(_._1).mkString(", ")}")

  private def existing(m: Mod): Option[Existing] = m match
    case Mod.Annot(init) if Set("gap", "framework.gap")(init.tpe.syntax) =>
      init.argClauses.toList match
        case List(subjects, reasons) =>
          val because = reasons.values.collectFirst {
            case Term.Assign(Term.Name("because"), Lit.String(v)) => v
            case Lit.String(v)                                    => v
          }
          because.map(b => Existing(subjects.values.map(v => normal(v.syntax)), b))
        case _ => None
    case _ => None

  // Every object, trait and single-name val of a source, by `<package>.<objects>.<name>`.
  private def declarations(tree: Tree): Seq[(String, Tree, Seq[Mod])] =
    def walk(t: Tree, path: Seq[String]): Seq[(String, Tree, Seq[Mod])] = t match
      case p: Pkg         => p.body.stats.flatMap(walk(_, path ++ p.ref.syntax.split('.')))
      case o: Defn.Object =>
        val here = path :+ o.name.value
        (here.mkString("."), o, o.mods) +: o.templ.body.stats.flatMap(walk(_, here))
      case c: Defn.Trait => Seq(((path :+ c.name.value).mkString("."), c, c.mods))
      case v: Defn.Val   =>
        v.pats match
          case List(Pat.Var(n)) => Seq(((path :+ n.value).mkString("."), v, v.mods))
          case _                => Nil
      case s: Source => s.stats.flatMap(walk(_, path))
      case _         => Nil
    walk(tree, Nil)
