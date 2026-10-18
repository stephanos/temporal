package framework.dsl

import scala.quoted.*

private[framework] trait RunAssertionEvaluation:
  def test(position: RunPosition, condition: () => Boolean): Boolean

final class RunAssertion @scala.annotation.publicInBinary private[framework] (
    val position: RunPosition,
    private[framework] val owners: Vector[(RunPosition, () => Model)],
    private[framework] val evaluate: RunAssertionEvaluation => Boolean
)

@scala.annotation.publicInBinary
private[framework] object ExpectMacros:
  def previous[M <: Machine[?, ?]: Type](scope: Expr[RunContext[M]])(using q: Quotes): Expr[Any] =
    snapshot(scope, q.reflect.TypeRepr.of[M])

  def modelType(scope: Expr[RunContext[?]])(using q: Quotes): q.reflect.TypeRepr =
    import q.reflect.*
    scope.asTerm.tpe.widen.baseType(TypeRepr.of[RunContext[?]].typeSymbol) match
      case AppliedType(_, List(model)) => model
      case _ => report.errorAndAbort("an authored run requires its Model context")

  def snapshot(using q: Quotes)(scope: Expr[RunContext[?]], model: q.reflect.TypeRepr): Expr[Any] =
    import q.reflect.*
    val receiver = Select.unique(scope.asTerm, "model")
    val cls = model.widen.typeSymbol
    val record = VarSchemaMacro
      .stateVars(cls)
      .foldLeft(TypeRepr.of[State]): (record, variable) =>
        Refinement(record, variable.name, receiver.tpe.memberType(variable).widenByName)
    record.asType match
      case '[r] => '{ $scope.previous.asInstanceOf[r] }

  def capture(body: Expr[Any], scope: Expr[RunContext[?]])(using q: Quotes): Expr[RunAssertions] =
    import q.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case other                  => other
    def statements(term: Term): List[Term] = unwrap(term) match
      case Block(stats, last) =>
        (stats :+ last).flatMap:
          case term: Term => statements(term)
          case other => report.errorAndAbort("expect requires bare Boolean statements", other.pos)
      case term if term.tpe <:< TypeRepr.of[Boolean] => List(term)
      case other => report.errorAndAbort("expect requires bare Boolean statements", other.pos)
    val original = unwrap(body.asTerm)
    val terms = statements(original)
    val warningFilter =
      "msg=(unused value of type .*Boolean|A pure expression does nothing in statement position|discarded non-Unit value of type .*Boolean)"
    def instrument(term: Term, evaluation: Expr[RunAssertionEvaluation]): Expr[Boolean] =
      val rewrite = new TreeMap:
        override def transformTerm(tree: Term)(owner: Symbol): Term = tree match
          case block @ Block(stats, last) if block.tpe <:< TypeRepr.of[Boolean] =>
            val conditions = stats.map:
              case condition: Term if condition.tpe <:< TypeRepr.of[Boolean] => condition
              case other                                                     =>
                report.errorAndAbort("expect requires bare Boolean statements", other.pos)
            val sequence = conditions.foldRight(transformTerm(last)(owner).asExprOf[Boolean]):
              (condition, rest) =>
                val located = RunBodyMacros.position(condition)
                val value = transformTerm(condition)(owner).asExprOf[Boolean]
                '{ $evaluation.test($located, () => $value) && $rest }
            if conditions.isEmpty then sequence.asTerm
            else
              RunBodyMacros.annotated(block, sequence.asTerm, TypeTree.of[Boolean], warningFilter)
          case block @ Block(stats, last) =>
            val retained = stats.map:
              case condition: Term if condition.tpe <:< TypeRepr.of[Boolean] =>
                val located = RunBodyMacros.position(condition)
                val value = transformTerm(condition)(owner).asExprOf[Boolean]
                '{ val _ = $evaluation.test($located, () => $value) }.asTerm
              case other => transformStatement(other)(owner)
            val rewritten = Block.copy(block)(retained, transformTerm(last)(owner))
            if stats.exists {
                case term: Term => term.tpe <:< TypeRepr.of[Boolean]
                case _          => false
              }
            then RunBodyMacros.annotated(block, rewritten, Inferred(block.tpe), warningFilter)
            else rewritten
          case other => super.transformTerm(other)(owner)
      rewrite.transformTerm(term)(Symbol.spliceOwner).asExprOf[Boolean]
    val assertions = terms.map: term =>
      val located = RunBodyMacros.position(term)
      val owners = RunBodyMacros.roots(term)
      '{
        new RunAssertion(
          $located,
          $owners,
          (evaluation: RunAssertionEvaluation) =>
            val _ = evaluation
            ${ instrument(term, 'evaluation) }
        )
      }
    val result = '{
      val _ = $scope
      new RunAssertions(${ Expr.ofSeq(assertions) }.toVector)
    }
    if terms.size == 1 then result
    else
      original match
        case block: Block =>
          RunBodyMacros
            .annotated(block, result.asTerm, TypeTree.of[RunAssertions], warningFilter)
            .asExprOf[RunAssertions]
        case _ => report.errorAndAbort("expect requires a Boolean statement block", original.pos)
