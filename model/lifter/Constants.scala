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
    case _ => t

  def constString(t: Term): String = resolve(t) match
    case Literal(StringConstant(s)) => s
    // `x.name` of a case class value declared elsewhere: the argument its constructor got.
    case Select(qual, field) =>
      resolve(qual) match
        case Apply(Select(companion, "apply"), args) =>
          val cls = companion.tpe.typeSymbol.companionClass
          val i = cls.caseFields.indexWhere(_.name == field)
          if i < 0 then fail(t, s"cannot read .$field of this value") else constString(args(i))
        case other => fail(t, s"cannot fold ${other.show} to a string")
    // Party(...), Entity(...), Family(...), Observation(...): the name is the first argument.
    case Apply(Select(_, "apply"), first :: _) => constString(first)
    case other                                 => fail(t, s"expected a string, got ${other.show}")

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
