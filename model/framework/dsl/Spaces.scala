package framework.dsl

import scala.quoted.*

@annotation.implicitNotFound("any is only available in an exploration space")
final class SpaceContext @scala.annotation.publicInBinary private[framework] ()

def any(using SpaceContext): Nothing =
  throw IllegalStateException("any requires direct exploration capture")

final class SpaceFind[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    val space: Explore[M],
    val steps: Int,
    val target: RunAssertions,
    val position: RunPosition
)

final class SpaceSample[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    val space: Explore[M],
    val count: Int,
    val seed: Long,
    val position: RunPosition
)

final class SpaceRefuted[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    private[framework] val owner: ModelSuite[M],
    private[framework] val machine: MachineDeclaration[State, ?, ?],
    private[framework] val claim: ClaimQuery,
    val position: RunPosition
)

final class Explore[M <: Machine[?, ?]] @scala.annotation.publicInBinary private[framework] (
    val model: M,
    val steps: Int,
    val position: RunPosition,
    targets: Vector[Option[RunTarget]],
    private[framework] val owner: ModelSuite[M]
) extends ExplorationSpace:
  require(steps >= 1, s"$position: explore requires a positive step bound")
  require(targets.size <= steps, s"$position: exploration selectors exceed the step bound")

  private lazy val retained: scala.util.Try[Vector[Option[Vector[Class]]]] =
    scala.util.Try(targets.map(_.map: target =>
      target.owners.foreach: (at, owner) =>
        require(owner().eq(model), s"$at: a selector belongs to another Model instance")
      val selected = try target.evaluate()
      catch
        case scala.util.control.NonFatal(error) =>
          throw IllegalArgumentException(s"${target.position}: ${error.getMessage}", error)
      val declaration = selected match
        case action: Action[?] => action.decl
        case cls: Class        => cls.decl
      require(
        model.nativeEffects.exists(_._1.eq(declaration)),
        s"${target.position}: a selector belongs to another Model instance"
      )
      val catalog = NativeRuntime.installed.classesOf(declaration).toVector
      selected match
        case _: Action[?] => catalog
        case cls: Class   =>
          cls.supplied.map(_.value) match
            case Some(pattern: SpaceInputs) => pattern.select(catalog)
            case _                          =>
              require(
                catalog.contains(cls),
                s"${target.position}: a class is outside its declared domain"
              )
              Vector(cls)))

  private[framework] def acquire(owner: M, from: Option[State]): SpaceDescriptor[M] =
    require(owner.eq(model), s"$position: a space belongs to another Model instance")
    val starts = from.fold(model.initialStates.toVector)(Vector(_))
    require(starts.size == 1, s"$position: an exploration with several starts requires from")
    starts.foreach(state => model.stateVars.foreach(_.owned(state): Unit))
    new SpaceDescriptor(model, steps, starts, retained.get)

  transparent inline def find(steps: Int)(inline body: RunContext[M] ?=> Any): SpaceFind[M] =
    ${ SpacesMacros.find[M]('this, 'steps, 'body) }

  inline def sample(count: Int, seed: Long): SpaceSample[M] =
    ${ SpacesMacros.sample[M]('this, 'count, 'seed) }

final private[framework] class SpaceDescriptor[M <: Machine[?, ?]](
    val model: M,
    val steps: Int,
    val starts: Vector[State],
    private val positions: Vector[Option[Vector[Class]]]
):
  def selectors: Vector[Vector[Class]] = positions.map(_.getOrElse(Vector.empty))
  def select(position: Int, cls: Class): Boolean =
    position >= 0 && position < steps &&
      positions.lift(position).flatten.forall(_.contains(cls))

final private[framework] class SpaceInputs(
    val constraints: Vector[(Int, RunPosition, Any => Boolean)]
):
  def select(catalog: Vector[Class]): Vector[Class] =
    constraints.foreach: (index, position, matches) =>
      require(
        catalog.exists(cls => matches(cls.values(index))),
        s"$position: an input is outside its declared domain"
      )
    catalog.filter(cls => constraints.forall((index, _, matches) => matches(cls.values(index))))

@scala.annotation.publicInBinary
private[framework] object SpacesMacros:
  def refuted[M <: Machine[?, ?]: Type](
      suite: Expr[ModelSuite[M]],
      machine: Expr[Model],
      claim: Expr[Any]
  )(using q: Quotes): Expr[SpaceRefuted[M]] =
    import q.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case Typed(inner, _)        => unwrap(inner)
      case other                  => other
    val selected = unwrap(claim.asTerm)
    val member = selected.symbol
    val section = if member.exists then member.owner else Symbol.noSymbol
    if !member.exists || !member.isValDef || !(selected.tpe.widen =:= TypeRepr.of[Unit]) ||
      !section.isClassDef || !section.flags.is(Flags.Module) ||
      !Set("invariants$", "postconditions$").contains(section.name)
    then
      report.errorAndAbort(
        "refuted takes an exact named invariant or postcondition",
        claim.asTerm.pos
      )
    val subject = unwrap(machine.asTerm)
    if !(subject.tpe <:< TypeRepr.of[MachineDeclaration[State, ?, ?]]) then
      report.errorAndAbort("refuted takes a native Machine or its derivation", machine.asTerm.pos)
    val native = subject.asExprOf[MachineDeclaration[State, ?, ?]]
    val captured = QueryMacros.verifying(claim)
    val at = RunBodyMacros.position(claim.asTerm)
    '{ new SpaceRefuted($suite, $native, $captured, $at) }

  def pattern(decl: ActionDecl, constraints: Vector[(Int, RunPosition, Any => Boolean)]): Class =
    val cls = Class(decl, Nil)
    cls.supplied = Some(SuppliedInput(new SpaceInputs(constraints)))
    cls

  def sample[M <: Machine[?, ?]: Type](space: Expr[Explore[M]], count: Expr[Int], seed: Expr[Long])(
      using q: Quotes
  ): Expr[SpaceSample[M]] =
    import q.reflect.*
    val at = RunBodyMacros.position(count.asTerm)
    '{ new SpaceSample($space, $count, $seed, $at) }

  def wildcard(using q: Quotes)(term: q.reflect.Term): Boolean =
    import q.reflect.*
    term match
      case Inlined(_, Nil, inner) => wildcard(inner)
      case Typed(inner, _)        => wildcard(inner)
      case Apply(function, _)     =>
        function.symbol.owner.fullName.stripSuffix("$") == "framework.dsl.Spaces$package" &&
        function.symbol.name == "any"
      case _ => false

  def capture[M <: Machine[?, ?]: Type](
      suite: Expr[ModelSuite[M]],
      steps: Expr[Int],
      body: Expr[SpaceContext ?=> Any]
  )(using q: Quotes): Expr[Explore[M]] =
    import q.reflect.*
    val section = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(symbol => symbol.isClassDef && symbol.name.stripSuffix("$") == "explorations")
      .getOrElse(report.errorAndAbort("explore belongs in explorations", body.asTerm.pos))
    val declaring = Iterator
      .iterate(section.owner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(symbol =>
        symbol.isClassDef && symbol.typeRef.baseClasses.contains(
          TypeRepr.of[ModelSuite[?]].typeSymbol
        )
      )
      .getOrElse(report.errorAndAbort("explore requires its ModelSuite receiver", body.asTerm.pos))
    val receiver = This(declaring).asExprOf[AnyRef]
    def unwrap(term: Term): Term = term match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case other                  => other
    val original = unwrap(body.asTerm) match
      case Lambda(List(_), rhs) => unwrap(rhs)
      case other => report.errorAndAbort("explore requires a literal selector body", other.pos)
    def statements(term: Term): List[Term] = term match
      case Block(stats, last) =>
        (stats :+ last).map:
          case value: Term => value
          case other       =>
            report.errorAndAbort("explore contains only action selectors or any", other.pos)
      case other => List(other)
    val declared = statements(original)
    if steps.value.exists(_ <= 0) then
      report.errorAndAbort("explore requires a positive step bound", steps.asTerm.pos)
    if steps.value.exists(_ < declared.size) then
      report.errorAndAbort("exploration selectors exceed the step bound", declared.last.pos)
    def retain(scope: Expr[SpaceContext]): Expr[Vector[Option[RunTarget]]] =
      val reduced = unwrap(Expr.betaReduce('{ $body(using $scope) }).asTerm)
      val captured = statements(reduced).map: term =>
        if wildcard(term) then '{ None: Option[RunTarget] }
        else
          if !(term.tpe <:< TypeRepr.of[Action[?] | Class]) then
            report.errorAndAbort("explore contains only action selectors or any", term.pos)
          if term.tpe <:< TypeRepr.of[Class] then
            val target = RunBodyMacros.target(term)
            '{ Some($target) }
          else
            unwrap(term) match
              case ref: Ref
                  if !ref.symbol.flags.is(Flags.Mutable) &&
                    (!ref.symbol.isDefDef || ref.symbol.flags.is(
                      Flags.FieldAccessor
                    ) || ref.symbol.flags.is(Flags.Module)) =>
                ()
              case _ =>
                report.errorAndAbort("an exploration requires a stable whole action", term.pos)
            val at = RunBodyMacros.position(term)
            val owners = RunBodyMacros.roots(term)
            val value = term.asExprOf[Action[?] | Class]
            '{ Some(new RunTarget($at, $owners, () => $value)) }
      val values = '{ ${ Expr.ofSeq(captured) }.toVector }
      original match
        case block: Block if declared.dropRight(1).exists(term => !wildcard(term)) =>
          RunBodyMacros
            .annotated(
              block,
              values.asTerm,
              TypeTree.of[Vector[Option[RunTarget]]],
              "msg=(unused value of type .*framework.dsl.(Class|.*Action).*|A pure expression does nothing in statement position)"
            )
            .asExprOf[Vector[Option[RunTarget]]]
        case _ => values
    val at = RunBodyMacros.position(steps.asTerm)
    '{
      require(
        $receiver.eq($suite),
        s"${$at}: a space receiver belongs to another ModelSuite instance"
      )
      val scope = new SpaceContext
      val _ = scope
      new Explore($suite.model, $steps, $at, ${ retain('scope) }, $suite)
    }

  def find[M <: Machine[?, ?]: Type](
      space: Expr[Explore[M]],
      steps: Expr[Int],
      body: Expr[RunContext[M] ?=> Any]
  )(using q: Quotes): Expr[SpaceFind[M]] =
    import q.reflect.*
    val captured = RunBodyMacros.capture[M, M]('{ $space.model }, '{ None }, body)
    val at = RunBodyMacros.position(steps.asTerm)
    '{
      val owner = $space
      val run = $captured
      val target = run.operations match
        case Vector(assertions: RunAssertions) => assertions
        case _ => throw IllegalArgumentException(s"${$at}: find requires an expect target")
      new SpaceFind(owner, $steps, target, $at)
    }
