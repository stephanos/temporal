package framework.dsl

import scala.quoted.*

final case class RunPosition(file: String, line: Int, text: String):
  override def toString: String = s"$file:$line: $text"

sealed trait RunOperation

final class RunRecords[F] @scala.annotation.publicInBinary private[framework] ():
  def apply[G](fact: G)(using G <:< F): Boolean = NativeClaims.facts.contains(fact)

final class RunAlternatives @scala.annotation.publicInBinary private[framework] ():
  inline def apply(inline alternatives: (Action[?] | Class)*): RunStep =
    ${ RunBodyMacros.alternatives('alternatives) }

final class RunAssertions @scala.annotation.publicInBinary private[framework] (
    val assertions: Vector[RunAssertion]
) extends RunOperation

final class RunStep @scala.annotation.publicInBinary private[framework] (
    val targets: Vector[RunTarget],
    private[framework] val count: () => Int,
    val position: RunPosition
) extends RunOperation

final class RunTarget @scala.annotation.publicInBinary private[framework] (
    val position: RunPosition,
    private[framework] val owners: Vector[(RunPosition, () => Model)],
    private[framework] val evaluate: () => Action[?] | Class
)

final class RunContext[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    val model: M
):
  @scala.annotation.publicInBinary
  private[framework] def previous: State =
    NativeClaims.start(throw IllegalStateException("before requires a preceding replay step"))
  @scala.annotation.publicInBinary
  private[framework] def answer: model.Outcome =
    NativeClaims
      .answer(throw IllegalStateException("outcome requires a preceding replay step"))
      .asInstanceOf[model.Outcome]
  def records[G](fact: G)(using G <:< model.Fact): Boolean = NativeClaims.facts.contains(fact)

final class RunBody[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    val model: M,
    private[framework] val from: Option[() => State],
    val operations: Vector[RunOperation],
    val position: RunPosition
)

object RunBody:
  transparent inline def capture[M <: Machine[?, ?]](model: M)(
      inline body: RunContext[model.type] ?=> Any
  ): RunBody[M] = ${ RunBodyMacros.capture[M, model.type]('model, 'None, 'body) }

  transparent inline def capture[M <: Machine[?, ?]](model: M, inline from: State)(
      inline body: RunContext[model.type] ?=> Any
  ): RunBody[M] = ${ RunBodyMacros.capture[M, model.type]('model, '{ Some(() => from) }, 'body) }

@scala.annotation.publicInBinary
private[framework] object RunBodyMacros:
  private def stable(using q: Quotes)(value: q.reflect.Term): Boolean =
    import q.reflect.*
    value match
      case Inlined(_, Nil, inner) => stable(inner)
      case Typed(inner, _)        => stable(inner)
      case _: This                => true
      case reference: Ref         =>
        !reference.symbol.flags.is(Flags.Mutable) &&
        (!reference.symbol.isDefDef || reference.symbol.flags.is(Flags.FieldAccessor) ||
          reference.symbol.flags.is(Flags.Module)) && (reference match
          case Select(receiver, _) => stable(receiver)
          case _                   => true)
      case _ => false

  def target(using q: Quotes)(term: q.reflect.Term): Expr[RunTarget] =
    import q.reflect.*
    def callee(tree: Tree): Tree = tree match
      case Apply(function, _)     => callee(function)
      case TypeApply(function, _) => callee(function)
      case other                  => other
    def origin(tree: Term): Tree = tree match
      case Inlined(Some(call), _, _) => call
      case Inlined(None, Nil, inner) => origin(inner)
      case Typed(inner, _)           => origin(inner)
      case other                     => other
    val source = origin(term)
    val admitted = source match
      case call: Apply =>
        callee(call) match
          case selection @ Select(receiver, name) =>
            selection.symbol.owner.fullName == "framework.dsl.Action" &&
            Set("applyDynamicNamed", "applyDynamic")(name) && stable(receiver)
          case _ => false
      case reference: Term => stable(reference)
      case _               => false
    if !admitted then
      report.errorAndAbort(
        "a run step requires a named action call or a stable whole action",
        term.pos
      )
    term.tpe.widen.baseType(TypeRepr.of[Action[?]].typeSymbol) match
      case AppliedType(_, List(input)) if !(input =:= TypeRepr.of[EmptyTuple]) =>
        report.errorAndAbort(
          "a whole run action must be fieldless; supply its named inputs",
          term.pos
        )
      case _ => ()
    val value = term.asExprOf[Action[?] | Class]
    val located = position(term)
    val owners = roots(term)
    '{ new RunTarget($located, $owners, () => $value) }

  def roots(using q: Quotes)(term: q.reflect.Term): Expr[Vector[(RunPosition, () => Model)]] =
    import q.reflect.*
    val selected = scala.collection.mutable.ArrayBuffer.empty[(Term, Term)]
    new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case selection @ Select(receiver, name) if receiver.tpe <:< TypeRepr.of[Model] =>
          val state = VarSchemaMacro.stateVars(receiver.tpe.widen.typeSymbol).exists(_.name == name)
          val fact = receiver.tpe.widen.baseType(TypeRepr.of[Machine[?, ?]].typeSymbol) match
            case AppliedType(_, List(_, fact)) => fact
            case _                             => TypeRepr.of[Nothing]
          val shared = selection.symbol.isValDef && !selection.symbol.flags.is(Flags.Mutable) &&
            selection.tpe.widen <:< fact
          if state || !shared then
            if !stable(receiver) then
              report.errorAndAbort("a captured Model root needs a stable val holder", receiver.pos)
            selected += ((selection, receiver)): Unit
          super.traverseTree(selection)(owner)
        case _ => super.traverseTree(tree)(owner)
    .traverseTree(term)(Symbol.spliceOwner)
    val entries = selected.toVector
      .distinctBy(_._2.show)
      .map: (selection, receiver) =>
        val located = position(selection)
        val model = receiver.asExprOf[Model]
        '{ ($located, () => $model) }
    '{ ${ Expr.ofSeq(entries) }.toVector }

  private def combinator(using q: Quotes)(term: q.reflect.Term): Boolean =
    import q.reflect.*
    def method(tree: Tree): Symbol = tree match
      case Apply(function, _)     => method(function)
      case TypeApply(function, _) => method(function)
      case other                  => other.symbol
    def admitted(call: Tree): Boolean =
      val symbol = method(call)
      val owner = symbol.owner.fullName.stripSuffix("$")
      owner == "framework.dsl.Syntax$package" && Set("expect", "oneOf", "repeat")(symbol.name) ||
      owner == "framework.dsl.RunAlternatives" && symbol.name == "apply"
    term match
      case Inlined(Some(call), _, _) => admitted(call)
      case call: Apply               => admitted(call)
      case Inlined(None, Nil, inner) => combinator(inner)
      case Typed(inner, _)           => combinator(inner)
      case _                         => false

  def position(using q: Quotes)(term: q.reflect.Term): Expr[RunPosition] =
    val pos = term.pos
    '{
      RunPosition(
        ${ Expr(pos.sourceFile.name) },
        ${ Expr(pos.startLine + 1) },
        ${ Expr(pos.sourceCode.getOrElse(term.show)) }
      )
    }

  def annotated(using
      q: Quotes
  )(
      original: q.reflect.Block,
      result: q.reflect.Term,
      tpe: q.reflect.TypeTree,
      filter: String
  ): q.reflect.Term =
    import q.reflect.*
    val positioned = if result.tpe <:< TypeRepr.of[Unit] then
      Block.copy(original)(List(result), Literal(UnitConstant()))
    else
      val saved =
        Symbol.newVal(Symbol.spliceOwner, "captured", result.tpe, Flags.EmptyFlags, Symbol.noSymbol)
      Block.copy(original)(List(ValDef(saved, Some(result))), Ref(saved))
    val annotation = Apply(
      Select(
        New(TypeTree.of[scala.annotation.nowarn]),
        Symbol.requiredClass("scala.annotation.nowarn").primaryConstructor
      ),
      List(Literal(StringConstant(filter)))
    )
    Typed(positioned, Annotated(tpe, annotation))

  def alternatives(alternatives: Expr[Seq[Action[?] | Class]])(using q: Quotes): Expr[RunStep] =
    import q.reflect.*
    val values = alternatives match
      case Varargs(values) if values.size >= 2 => values
      case _                                   =>
        report.errorAndAbort("oneOf requires at least two alternatives", alternatives.asTerm.pos)
    val targets = Expr.ofSeq(values.map(value => target(value.asTerm)))
    val located = position(alternatives.asTerm)
    '{ new RunStep($targets.toVector, () => 1, $located) }

  def repeat(count: Expr[Int], step: Expr[Action[?] | Class])(using q: Quotes): Expr[RunStep] =
    import q.reflect.*
    if count.value.exists(_ < 1) then
      report.errorAndAbort("repeat requires a positive count", count.asTerm.pos)
    val located = position(step.asTerm)
    val selected = target(step.asTerm)
    '{ new RunStep(Vector($selected), () => $count, $located) }

  def capture[M <: Machine[?, ?]: Type, T <: M: Type](
      model: Expr[T],
      from: Expr[Option[() => State]],
      body: Expr[RunContext[T] ?=> Any]
  )(using q: Quotes): Expr[RunBody[M]] =
    import q.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(None, Nil, inner) => unwrap(inner)
      case other                     => other
    val original = unwrap(body.asTerm) match
      case Lambda(List(_), rhs) => unwrap(rhs)
      case other => report.errorAndAbort("an authored run requires a literal body", other.pos)
    val declared = original match
      case Block(stats, last) => stats :+ last
      case other              => List(other)
    declared.foreach:
      case term: Term if term.tpe <:< TypeRepr.of[RunOperation] =>
        if !combinator(term) then
          report.errorAndAbort("an authored run requires a direct declaration combinator", term.pos)
      case term: Term if term.tpe <:< TypeRepr.of[Action[?] | Class] =>
        val _ = target(term)
      case other =>
        report.errorAndAbort(
          "an authored run contains only action calls, whole actions, oneOf, repeat and expect",
          other.pos
        )
    val located = position(original)
    '{
      val owner = $model
      val scope = new RunContext[T](owner)
      val _ = scope
      new RunBody[M](
        owner,
        $from,
        ${ operations(Expr.betaReduce('{ $body(using scope) })) },
        $located
      )
    }

  private def operations(body: Expr[Any])(using q: Quotes): Expr[Vector[RunOperation]] =
    import q.reflect.*
    def unwrap(term: Term): Term = term match
      case value if combinator(value) => value
      case Inlined(_, Nil, inner)     => unwrap(inner)
      case other                      => other
    val original = unwrap(body.asTerm)
    val statements = original match
      case Block(stats, last) => stats :+ last
      case other              => List(other)
    val operations = statements.map:
      case term: Term if term.tpe <:< TypeRepr.of[RunOperation] =>
        term.asExprOf[RunOperation]
      case term: Term if term.tpe <:< TypeRepr.of[Action[?] | Class] =>
        val selected = target(term)
        val located = position(term)
        '{ new RunStep(Vector($selected), () => 1, $located) }
      case other =>
        report.errorAndAbort(
          "an authored run contains only action calls, whole actions, oneOf, repeat and expect",
          other.pos
        )
    val values = '{ ${ Expr.ofSeq(operations) }.toVector }.asTerm
    val vector = original match
      case block: Block if statements.size > 1 =>
        annotated(
          block,
          values,
          TypeTree.of[Vector[RunOperation]],
          "msg=unused value of type .*framework.dsl.(RunStep|RunAssertions|Class|.*Action).*"
        )
      case _ => values
    vector.asExprOf[Vector[RunOperation]]
