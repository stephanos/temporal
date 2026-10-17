package framework.dsl

import scala.deriving.Mirror
import scala.quoted.*

private[framework] object InboxAssociation:
  def capture[S: Type, C: Type, M: Type](
      owner: Expr[RecordMachine[S, ?, ?]],
      selector: Expr[C => Inbox[M]],
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
    def fields(term: Term, roots: Map[Symbol, List[String]]): Option[List[String]] = unwrap(
      term
    ) match
      case parameter: Ident => roots.get(parameter.symbol)
      case Select(receiver, "get") if receiver.tpe.widen.dealias <:< TypeRepr.of[Option[Any]] =>
        fields(receiver, roots)
      case Select(receiver, name)
          if receiver.tpe.widen.dealias.typeSymbol.flags.is(Flags.Case) &&
            receiver.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        fields(receiver, roots).map(_ :+ name)
      case Match(receiver, cases) =>
        val returning = cases.filterNot(_.rhs.tpe <:< TypeRepr.of[Nothing])
        if returning.size != 1 || cases.exists(_.guard.nonEmpty) then None
        else
          returning.head.pattern match
            case binding @ Bind(_, Typed(_, branch))
                if exactCase(receiver.tpe.widen.dealias, branch.tpe) =>
              fields(receiver, roots).flatMap: path =>
                val coordinate = path :+ s"<${branch.tpe.typeSymbol.name}>"
                fields(returning.head.rhs, roots.updated(binding.symbol, coordinate))
            case _ => None
      case _ => None
    val selected = unwrap(selector.asTerm) match
      case Lambda(List(parameter), body) =>
        fields(body, Map(parameter.symbol -> Nil)).filter(_.nonEmpty)
      case _ => None
    val path = selected.getOrElse:
      report.errorAndAbort("an Inbox association requires a rooted Inbox field selector", selector)
    val root = TypeRepr.of[S].dealias
    val branch = TypeRepr.of[C].dealias
    val prefix =
      if branch =:= root then Nil
      else if exactCase(root, branch) then List(s"<${branch.typeSymbol.name}>")
      else
        report.errorAndAbort("an Inbox branch association requires an exact state case", selector)
    val at = Position.ofMacroExpansion
    val location = s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"
    '{
      val declaring = $owner
      declaring.retainNativeInbox(
        new InboxOwnership.Native(
          declaring,
          ${ Expr(root.show) },
          ${ Expr.ofList((prefix ++ path).map(Expr(_))) },
          ${ Expr(TypeRepr.of[M].dealias.show) },
          $channel,
          ${ Expr(location) }
        )
      )
    }
