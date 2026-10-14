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
import dotty.tools.dotc.core.Types.{Type, TypeBounds}
import dotty.tools.dotc.plugins.PluginPhase
import dotty.tools.dotc.report
import io.temporal.server.api.umpire.v1 as ir

final private class SchemaPhase(store: DefinitionStore, signatures: Boolean = false)
    extends PluginPhase:
  val phaseName = if signatures then "reifySignature" else "reifySchema"
  override val runsAfter = if signatures then Set("pickler") else Set("reifyNames")
  override val runsBefore = if signatures then Set("inlining") else Set("sbt-api", "pickler")

  private def open(tpe: Type)(using Context): Boolean = tpe match
    case _: TypeBounds => true
    case _             =>
      tpe.typeSymbol.is(Param) || tpe.argInfos.exists(open) || tpe.dealias.argInfos.exists(open)

  private def appendedInput(tree: tpd.Tree)(using Context): Option[Type] = tree match
    case Apply(fun, _) => appendedInput(fun)
    case TypeApply(fun, List(arg))
        if fun.symbol.owner.fullName.toString == "framework.dsl.Action" && fun.symbol.name.toString == "input" =>
      Some(arg.tpe)
    case _ => None

  private def signatureAttach(rhs: tpd.Tree, declared: Type, appended: Option[Type] = None)(using
      ctx: Context
  ): tpd.Tree =
    if rhs.isEmpty then return rhs
    def argument(name: String): Option[Type] =
      val cls = getClassIfDefined(name)
      if cls.exists then declared.widen.baseType(cls.asClass).argInfos.headOption else None
    def tuple(tpe: Type): List[Type] =
      val t = tpe.dealias.simplified
      val name = t.typeSymbol.fullName.toString
      if name == "scala.*:" then t.argInfos.head :: tuple(t.argInfos(1))
      else if name.matches("scala.Tuple[0-9]+") then t.argInfos
      else if name == "scala.Tuple$.Append" && t.argInfos.size == 2 then
        tuple(t.argInfos.head) :+ t.argInfos(1)
      else Nil
    val action = argument("framework.dsl.Action")
    val inputs: List[(Type, Option[Int])] = action match
      case Some(_) if appended.nonEmpty => appended.toList.map(_ -> None)
      case Some(args) if !open(args)    =>
        tuple(args).zipWithIndex.map((arg, index) => arg -> Some(index))
      case Some(_) => Nil
      case None    =>
        argument("framework.dsl.Input")
          .orElse(argument("framework.dsl.Channel"))
          .toList
          .map(_ -> None)
    inputs.foldLeft(rhs) { case (value, (arg, index)) =>
      if open(arg) then value
      else
        metadata(arg, rhs) { (e, root) =>
          val key = e.identity(
            arg.asInstanceOf[e.ctx.quotes.reflect.TypeRepr],
            root,
            rhs.asInstanceOf[e.ctx.quotes.reflect.Tree]
          )
          val payload = List(
            Literal(Constant(key)),
            Carrier.literals(Carrier.chunks(root.toByteArray)),
            Carrier.literals(
              Carrier.chunks(ir.Model(types = e.ctx.types.values.map(_.clearPosition).toVector))
            ),
            Carrier.literals(
              Carrier.chunks(
                ir.Record(fields = e.pending.toVector.map((id, ref) => ir.Field(id, Some(ref))))
                  .toByteArray
              )
            )
          )
          val method = if action.nonEmpty && index.isEmpty then "retainLastInputSchema"
          else "retainInputSchema"
          value
            .select(value.tpe.widen.member(termName(method)).symbol)
            .appliedToArgs(index.map(i => Literal(Constant(i))).toList ++ payload)
            .withSpan(rhs.span)
        }
    }

  private def declarationName(rhs: tpd.Tree, tree: tpd.ValDef)(using Context): tpd.Tree =
    val names = List("framework.dsl.Action", "framework.dsl.Channel")
    if tree.symbol.is(Param) || tree.symbol.is(Synthetic) || !names.exists { name =>
        val cls = getClassIfDefined(name)
        cls.exists && tree.tpt.tpe.widen.derivesFrom(cls.asClass)
      }
    then rhs
    else
      val id = tree.symbol.fullName.toString
        .split('.')
        .filterNot(_.endsWith("$package$"))
        .map(_.stripSuffix("$"))
        .mkString(".")
      rhs
        .select(rhs.tpe.widen.member(termName("retainName")).symbol)
        .appliedToArgs(List(Literal(Constant(id)), Literal(Constant(tree.name.toString))))
        .withSpan(rhs.span)

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
    if signatures then tree else schemaRead(tree)

  private def concrete(tpe: Type)(using Context): Boolean =
    val sym = tpe.typeSymbol
    !open(tpe) && !sym.is(Synthetic) && !(sym.is(Enum) && sym.is(Case)) &&
    sym != defn.IntClass

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    if signatures then
      cpy.ValDef(tree)(rhs = declarationName(signatureAttach(tree.rhs, tree.tpt.tpe), tree))
    else if tree.symbol.is(Param) || tree.symbol.is(Synthetic) then tree
    else cpy.ValDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    if signatures then cpy.DefDef(tree)(rhs = signatureAttach(tree.rhs, tree.tpt.tpe))
    else if tree.symbol.is(Synthetic) then tree
    else cpy.DefDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
    if signatures then
      val named = tree.fun match
        case selected: tpd.Select if selected.hasAttachment(ChannelNames.generated) =>
          selected.qualifier
            .select(selected.qualifier.tpe.widen.member(termName("sourceAt")).symbol)
            .appliedToArgs(tree.args)
            .withSpan(tree.span)
        case _ => tree
      return signatureAttach(named, tree.tpe, appendedInput(tree))
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
    if !signatures && tree.symbol.is(Opaque) then
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

  private def schemaType(tpe: TypeRepr): TypeRepr =
    if tpe.widen.typeSymbol.flags.is(Flags.Opaque) then tpe.widen else tpe.widen.dealias

  override def typeRef(tpe: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
    super.typeRef(schemaType(tpe), at, owner)

  override def optionType(arg: TypeRepr, at: Tree): String =
    super.optionType(schemaType(arg), at)

  def identity(declared: TypeRepr, ref: ir.TypeRef, at: Tree): String =
    val tpe = schemaType(declared)
    ref.ref match
      case ir.TypeRef.Ref.Named(name) => name
      case ir.TypeRef.Ref.Bool(_)     => "scala.Boolean"
      case ir.TypeRef.Ref.Int(_)      => "scala.Int"
      case ir.TypeRef.Ref.IntRange(_) =>
        if tpe.widen.typeSymbol.fullName == upToType then s"$upToType[${upToBound(tpe, at)}]"
        else ctx.irTypeName(tpe.widen.typeSymbol)
      case ir.TypeRef.Ref.Channel(message) => s"framework.dsl.Inbox[$message]"
      case _ => ctx.fail(at, s"${tpe.show} has no finite schema identity")

  def root(tpe: TypeRepr, at: Tree): ir.TypeRef =
    val result = typeRef(schemaType(tpe), at)
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
