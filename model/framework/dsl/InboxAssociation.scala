package framework.dsl

import scala.deriving.Mirror
import scala.quoted.*

private[framework] object InboxAssociation:
  def captureVar[M: Type](
      owner: Expr[Machine[?, ?]],
      variable: Expr[Inbox[M]],
      channel: Expr[Channel[M]]
  )(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    def exactCase(root: TypeRepr, branch: TypeRepr): Boolean = root.asType match
      case '[state] =>
        Expr
          .summon[Mirror.SumOf[state]]
          .flatMap(StructuralValue.sumCases(_))
          .exists(_.exists(_ =:= branch))
    def unwrap(term: Term): Term = term match
      case Inlined(_, _, body) => unwrap(body)
      case Typed(body, _)      => unwrap(body)
      case Block(Nil, body)    => unwrap(body)
      case other               => other
    type Coordinate = (Ref, List[String])
    def fields(term: Term, roots: Map[Symbol, Coordinate]): Option[Coordinate] = unwrap(term) match
      case parameter: Ident if roots.contains(parameter.symbol) => roots.get(parameter.symbol)
      case reference: Ref
          if reference.symbol.flags.is(Flags.Mutable) && reference.symbol.owner.isClassDef &&
            reference.symbol.owner.typeRef.baseClasses.contains(
              TypeRepr.of[Machine[?, ?]].typeSymbol
            ) =>
        Some(reference -> List(reference.symbol.name))
      case Select(receiver, "get") if receiver.tpe.widen.dealias <:< TypeRepr.of[Option[Any]] =>
        fields(receiver, roots)
      case Select(receiver, name)
          if receiver.tpe.widen.dealias.typeSymbol.flags.is(Flags.Case) &&
            receiver.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        fields(receiver, roots).map((root, path) => root -> (path :+ name))
      case Match(receiver, cases) =>
        val returning = cases.filterNot(_.rhs.tpe <:< TypeRepr.of[Nothing])
        if returning.size != 1 || cases.exists(_.guard.nonEmpty) then None
        else
          returning.head.pattern match
            case binding @ Bind(_, Typed(_, branch))
                if exactCase(receiver.tpe.widen.dealias, branch.tpe) =>
              fields(receiver, roots).flatMap: (root, path) =>
                fields(
                  returning.head.rhs,
                  roots.updated(
                    binding.symbol,
                    root -> (path :+ s"<${branch.tpe.typeSymbol.name}>")
                  )
                )
            case _ => None
      case _ => None
    val (root, path) = fields(variable.asTerm, Map.empty).getOrElse:
      report.errorAndAbort("an Inbox association requires a rooted state var Inbox field", variable)
    val receiver = root match
      case Select(receiver, _) => receiver
      case _                   => This(root.symbol.owner)
    if !(receiver.tpe.widen <:< owner.asTerm.tpe.widen) then
      report.errorAndAbort(
        "an Inbox association requires this Machine's owning state var",
        variable
      )
    val heldBy = receiver.asExprOf[Model]
    val at = Position.ofMacroExpansion
    val location = s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"
    '{
      val declaring = $owner
      require(
        Vars.of($heldBy).eq(Vars.of(declaring)),
        "an Inbox association requires its exact owning Machine"
      )
      declaring.retainNativeInbox(
        new InboxOwnership.Native(
          declaring,
          ${ Expr(TypeRepr.of[State].show) },
          ${ Expr.ofList(path.map(Expr(_))) },
          ${ Expr(TypeRepr.of[M].dealias.show) },
          $channel,
          ${ Expr(location) }
        )
      )
    }
