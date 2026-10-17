package framework.inspect

import scala.quoted.Quotes

private[framework] object LiftableSubset:
  val binary: Set[String] = Set("==", "!=", "&&", "||", "<", "<=", ">", ">=", "+", "-", "++")
  val body: Set[String] = binary ++ Set("unary_!", "contains", "copy", "min", "max", "+:", "toList")
  val declaration: Set[String] = Set("map", "toVector", "find")
  val subsetTable = "model/framework/SEMANTICS.md#liftable-subset"

  def unsupported(using quotes: Quotes)(tree: quotes.reflect.Tree): Option[String] =
    import quotes.reflect.*
    tree match
      case v: ValDef if v.symbol.flags.is(Flags.Mutable) => Some(s"var ${v.name}")
      case _: Assign                                     => Some("assignment")
      case _: While                                      => Some("while")
      case _: Return                                     => Some("return")
      case _: Try                                        => Some("try")
      case _: New                                        => Some("new")
      case _: ClassDef                                   => Some("class or anonymous class")
      case d: DefDef if !d.name.startsWith("$anonfun")   => Some("local def")
      case Literal(_: BooleanConstant | _: IntConstant | _: StringConstant) => None
      case _: Literal => Some(s"literal ${tree.show}")
      case _: Ref | _: TypeTree | _: Apply | _: TypeApply | _: TypedOrTest | _: Inlined | _: Block |
          _: If | _: Match | _: CaseDef | _: Bind | _: Unapply | _: Alternatives | _: Repeated |
          _: NamedArg | _: ValDef | _: DefDef | _: Closure | _: This | _: Wildcard =>
        None
      case _ => Some(tree.show.takeWhile(c => !c.isWhitespace))

  def library(owner: String): Boolean = owner.startsWith("scala.") || owner.startsWith("java.")

  def permitted(using
      quotes: Quotes
  )(call: quotes.reflect.Term, factFamily: Boolean = false): Boolean =
    import quotes.reflect.*
    def on(t: Term): Option[Term] = t match
      case Apply(fn, _)     => on(fn)
      case TypeApply(fn, _) => on(fn)
      case Select(value, _) => Some(value)
      case _                => None
    val name = call.symbol.name
    val owner = call.symbol.maybeOwner.fullName.replace("$", "")
    val receiver = on(call)
    val tpe = receiver.map(_.tpe.widen.dealias)
    val integer = tpe.exists(_ <:< defn.IntClass.typeRef)
    val boolean = tpe.exists(_ <:< defn.BooleanClass.typeRef)
    val sequence = tpe.exists(
      _.baseClasses.exists(s =>
        Set(
          "scala.collection.immutable.List",
          "scala.collection.immutable.Seq",
          "scala.collection.Seq"
        )(s.fullName)
      )
    )
    val module = tpe.fold("")(_.typeSymbol.fullName.stripSuffix("$"))
    if binary(name) then
      name match
        case "==" | "!=" => true
        case "&&" | "||" => boolean
        case "++"        => sequence
        case _           => integer
    else if name == "forall" then sequence && factFamily
    else if Set("contains", "+:", "toList")(name) then sequence
    else if name == "unary_!" then boolean
    else if name == "min" || name == "max" then owner == "scala.runtime.RichInt"
    else if name == "copy" || name.startsWith("copy$default$") then
      tpe.exists(_.typeSymbol.flags.is(Flags.Case))
    else if name == "apply" then
      Set("scala.collection.immutable.List", "scala.Some", "scala.StringContext")(module) || tpe
        .exists(_.isFunctionType)
    else if owner == "scala.Predef" then
      Set("intWrapper", "wrapRefArray", "require", "Ensuring")(name)
    else if owner == "scala.LowPriorityImplicits" then name == "intWrapper"
    else if owner == "scala.runtime.ScalaRunTime" then name == "wrapRefArray"
    else if owner.startsWith("scala.Predef.Ensuring") then name == "ensuring"
    else owner == "scala.StringContext" && name == "s"
