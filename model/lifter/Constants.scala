package umpire.lift

private[lift] trait Constants:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Values folded at lift time: names, strings and bounds the declarations reference

  val constOps = Map[String, (Long, Long) => Long](
    "+" -> (_ + _),
    "-" -> (_ - _),
    "*" -> (_ * _),
    "<<" -> (_ << _)
  )

  def resolve(t: Term): Term = t match
    case Typed(e, _)        => resolve(e)
    case Inlined(_, Nil, e) => resolve(e)
    case NamedArg(_, e)     => resolve(e)
    case r: Ref
        if !r.symbol.flags.is(Flags.Param) && r.symbol.isValDef && defs.contains(r.symbol) =>
      defs(r.symbol) match
        case ValDef(_, _, Some(rhs)) if !isEnumCase(r.symbol) => resolve(rhs)
        case _                                                => t
    // `Entity(key = …, refer = Map(…))`: a call whose arguments after a default the compiler first
    // binds to synthetic vals, read as the call with each argument in its place.
    case Block(stats, call @ Apply(fn, args))
        if stats.nonEmpty && stats.forall(syntheticArgument) =>
      val bound = stats.collect { case v @ ValDef(_, _, Some(rhs)) => v.symbol -> rhs }.toMap
      def inPlace(a: Term): Term = a match
        case NamedArg(n, e)                       => NamedArg.copy(a)(n, inPlace(e))
        case r: Ident if bound.contains(r.symbol) => bound(r.symbol)
        case _                                    => a
      Apply.copy(call)(fn, args.map(inPlace))
    case _ => t

  /** A val the compiler binds a call's argument to, such as `refer$1`, in the order written. */
  private def syntheticArgument(s: Statement): Boolean = s match
    case v: ValDef =>
      v.rhs.nonEmpty && (v.symbol.flags.is(Flags.Synthetic) || v.name.matches(".+\\$\\d+"))
    case _ => false

  def constString(t: Term): String = resolve(t) match
    case Literal(StringConstant(s)) => s
    // `x.name` of a case class value declared elsewhere: the argument its constructor got, or for a
    // party, entity or observation declared with no name, the name of its val.
    case Select(qual, field) =>
      resolve(qual) match
        case Apply(Select(companion, "apply"), args) =>
          val cls = companion.tpe.typeSymbol.companionClass
          val i = cls.caseFields.indexWhere(_.name == field)
          if i < 0 then fail(t, s"cannot read .$field of this value")
          else if field == "name" && namedByVal(cls) && isDefault(args(i)) then constString(qual)
          else constString(args(i))
        case other => fail(t, s"cannot fold ${other.show} to a string")
    // `Party()`, `Entity(key = …)` and `Observation(on = …, read = …)`, named after their val.
    case Apply(Select(companion, "apply"), name :: _)
        if namedByVal(companion.tpe.typeSymbol.companionClass) && isDefault(name) =>
      val kind = companion.tpe.typeSymbol.companionClass.name
      val article = if kind == "Party" then "a" else "an"
      captured(resolvedVal(t), s"$article $kind", s"`$kind(\"...\")`", t)
    // A family's root, which every Definition ID it names hangs off, written out.
    case Apply(Select(companion, "apply"), List(root))
        if companion.tpe.typeSymbol.companionClass.fullName == "umpire.Family" =>
      resolve(root) match
        case Literal(StringConstant(s)) => s
        case other => fail(root, s"a Family names its root as a string literal, not ${other.show}")
    // Party(...), Entity(...), Observation(...): the name is the first argument.
    case Apply(Select(_, "apply"), first :: _) => constString(first)
    case other                                 => fail(t, s"expected a string, got ${other.show}")

  /** Whether a declaration of this class takes its name from its val where it states none. */
  def namedByVal(cls: Symbol): Boolean =
    Set("umpire.Party", "umpire.Entity", "umpire.Observation")(cls.fullName)

  /** The val whose right-hand side `resolve` reaches: the last of the vals it goes through. */
  private def resolvedVal(t: Term): Option[Symbol] = t match
    case Typed(e, _)        => resolvedVal(e)
    case Inlined(_, Nil, e) => resolvedVal(e)
    case NamedArg(_, e)     => resolvedVal(e)
    case r: Ref
        if !r.symbol.flags.is(Flags.Param) && r.symbol.isValDef && defs.contains(r.symbol) =>
      defs(r.symbol) match
        case ValDef(_, _, Some(rhs)) => resolvedVal(rhs).orElse(Some(r.symbol))
        case _                       => None
    case _ => None

  def constInt(t: Term): Long = resolve(t) match
    case Literal(IntConstant(i)) => i.toLong
    // Operators on constants the compiler leaves unfolded, such as `1 << 20`.
    case Apply(Select(a, op), List(b)) if constOps.contains(op) =>
      constOps(op)(constInt(a), constInt(b))
    case other => fail(t, s"expected an integer constant, got ${other.show}")

  /** An argument the default of its parameter supplies. */
  def isDefault(t: Term): Boolean = t match
    case TypeApply(Select(_, n), _) => n.contains("$default$")
    case Select(_, n)               => n.contains("$default$")
    case _                          => false
