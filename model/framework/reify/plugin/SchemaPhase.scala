package framework.reify

import scala.collection.mutable
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Flags.{Case, Enum, Opaque, Param, Synthetic}
import dotty.tools.dotc.core.Names.termName
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.core.Types.Type
import dotty.tools.dotc.plugins.PluginPhase
import dotty.tools.dotc.report
import io.temporal.server.api.umpire.v1 as ir

final private class SchemaPhase(store: DefinitionStore) extends PluginPhase:
  val phaseName = "reifySchema"
  override val runsAfter = Set("reifyNames")
  override val runsBefore = Set("sbt-api", "pickler")

  private def finiteArgument(tpe: Type)(using Context): Option[Type] =
    val finite = getClassIfDefined("framework.dsl.Finite")
    if !finite.exists then None
    else tpe.widen.baseType(finite.asClass).argInfos.headOption

  private def attach(rhs: tpd.Tree, declared: Type)(using
      ctx: Context
  ): tpd.Tree =
    val finite = getClassIfDefined("framework.dsl.Finite")
    val companion = if finite.exists then finite.companionModule else NoSymbol
    val own = ctx.owner.enclosingClass
    if rhs.isEmpty || own == finite || own == companion.moduleClass then rhs
    else
      finiteArgument(declared) match
        case Some(arg) if concrete(arg) =>
          metadata(arg, rhs) { (e, root) =>
            val model =
              ir.Model(types = e.ctx.types.values.map(_.clearPosition).toVector)
            val method =
              finite.asClass.typeRef.member(termName("retainSchema")).symbol
            rhs
              .select(method)
              .appliedToArgs(
                List(
                  Literal(
                    Constant(
                      e.identity(
                        arg.asInstanceOf[e.ctx.quotes.reflect.TypeRepr],
                        root,
                        rhs.asInstanceOf[e.ctx.quotes.reflect.Tree]
                      )
                    )
                  ),
                  Carrier.literals(Carrier.chunks(root.toByteArray)),
                  Carrier.literals(Carrier.chunks(model)),
                  Carrier.literals(
                    Carrier.chunks(
                      ir.Record(fields =
                        e.pending.toVector.map((id, ref) => ir.Field(name = id, `type` = Some(ref)))
                      ).toByteArray
                    )
                  )
                )
              )
              .withSpan(rhs.span)
          }
        case _ => rhs

  private def metadata(arg: Type, at: tpd.Tree)(
      build: (SchemaEngine, ir.TypeRef) => tpd.Tree
  )(using
      ctx: Context
  ): tpd.Tree =
    val e = new SchemaEngine(store.engine(ctx.compilationUnit.tpdTree).ctx)
    try
      e.readFinites()
      val root = e.root(
        arg.asInstanceOf[e.ctx.quotes.reflect.TypeRepr],
        at.asInstanceOf[e.ctx.quotes.reflect.Tree]
      )
      build(e, root)
    catch
      case err: LiftError =>
        report.error(err.message, e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos)
        at
      case err: CompilerApiError =>
        report.error(err.getMessage, at.srcPos)
        at

  private def schemaRead(tree: tpd.Select)(using Context): tpd.Tree =
    val finite = getClassIfDefined("framework.dsl.Finite")
    if finite.exists && tree.symbol == finite.asClass.typeRef
        .member(termName("schema"))
        .symbol
    then
      finiteArgument(tree.qualifier.tpe).filter(concrete) match
        case Some(arg) =>
          metadata(arg, tree) { (e, root) =>
            tree.qualifier
              .select(
                finite.asClass.typeRef.member(termName("schemaFor")).symbol
              )
              .appliedTo(
                Literal(
                  Constant(
                    e.identity(
                      arg.asInstanceOf[e.ctx.quotes.reflect.TypeRepr],
                      root,
                      tree.asInstanceOf[e.ctx.quotes.reflect.Tree]
                    )
                  )
                )
              )
              .withSpan(tree.span)
          }
        case _ => tree
    else tree

  override def transformSelect(tree: tpd.Select)(using Context): tpd.Tree =
    schemaRead(tree)

  private def concrete(tpe: Type)(using Context): Boolean =
    def open(tpe: Type): Boolean =
      tpe.typeSymbol.is(Param) || tpe.argInfos.exists(open)
    val sym = tpe.typeSymbol
    !open(tpe) && !sym.is(Synthetic) && !(sym.is(Enum) && sym.is(Case)) &&
    sym != defn.IntClass

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    if tree.symbol.is(Param) || tree.symbol.is(Synthetic) then tree
    else cpy.ValDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    if tree.symbol.is(Synthetic) then tree
    else cpy.DefDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
    val reads = new tpd.TreeMap:
      override def transform(tree: tpd.Tree)(using Context): tpd.Tree =
        tree match
          case selected: tpd.Select => super.transform(schemaRead(selected))
          case _                    => super.transform(tree)
    val args = tree.args.map {
      case arg: tpd.NamedArg =>
        cpy.NamedArg(arg)(arg.name, reads.transform(arg.arg))
      case arg => arg
    }
    attach(cpy.Apply(tree)(tree.fun, args), tree.tpe)

  override def transformTypeDef(tree: tpd.TypeDef)(using
      ctx: Context
  ): tpd.Tree =
    if tree.symbol.is(Opaque) then
      val e = store.engine(ctx.compilationUnit.tpdTree)
      e.readFinites()
      e.ctx
        .opaqueRange(tree.symbol.asInstanceOf[e.ctx.quotes.reflect.Symbol])
        .foreach { (low, high) =>
          tree.symbol.addAnnotation(
            Annotation(
              requiredClass("framework.dsl.finiteRange"),
              List(Literal(Constant(low)), Literal(Constant(high))),
              tree.span
            )
          )
        }
    tree

final private class SchemaEngine(context: PluginContext) extends PluginEngine(context):
  import ctx.quotes.reflect.*
  val pending = mutable.LinkedHashMap.empty[String, ir.TypeRef]

  def identity(tpe: TypeRepr, ref: ir.TypeRef, at: Tree): String = ref.ref match
    case ir.TypeRef.Ref.Named(name) => name
    case ir.TypeRef.Ref.Bool(_)     => "scala.Boolean"
    case ir.TypeRef.Ref.IntRange(_) =>
      if tpe.widen.typeSymbol.fullName == upToType then s"$upToType[${upToBound(tpe, at)}]"
      else ctx.irTypeName(tpe.widen.typeSymbol)
    case ir.TypeRef.Ref.Channel(message) => s"framework.dsl.Inbox[$message]"
    case _ => ctx.fail(at, s"${tpe.show} has no finite schema identity")

  def root(tpe: TypeRepr, at: Tree): ir.TypeRef =
    val result = typeRef(tpe, at)
    val complete = mutable.Set.empty[String]
    def visit(ref: ir.TypeRef, path: List[String]): Unit = ref.ref match
      case ir.TypeRef.Ref.Named(name) if name != "scala.Nothing" =>
        if path.contains(name) then
          ctx.fail(
            at,
            s"recursive finite schema: ${(path :+ name).mkString(" -> ")}"
          )
        if !complete(name) then
          val shape = ctx.types.getOrElse(
            name,
            ctx.fail(at, s"finite schema $name is missing")
          )
          val fields =
            shape.getRecord.fields ++ shape.getEnum.cases.flatMap(_.fields)
          fields.flatMap(_.`type`).foreach(visit(_, path :+ name))
          complete += name: Unit
      case ir.TypeRef.Ref.Channel(id) => pending.get(id).foreach(visit(_, path))
      case _                          => ()
    visit(result, Nil)
    result

  override def inboxRef(tpe: TypeRepr, at: Tree, owner: String): ir.TypeRef =
    val message = tpe.typeArgs.head
    val ref = typeRef(message, at)
    val id = identity(message, ref, at)
    pending(id) = ref
    ir.TypeRef(ir.TypeRef.Ref.Channel(id))
