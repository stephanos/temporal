package framework.reify

import scala.collection.mutable
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Flags.{
  Case,
  Enum,
  Given,
  Method,
  Module,
  Opaque,
  Package,
  Param,
  Synthetic
}
import dotty.tools.dotc.core.Names.termName
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.core.Types.{MethodType as CompilerMethodType, Type, TypeBounds}
import dotty.tools.dotc.plugins.PluginPhase
import dotty.tools.dotc.report
import io.temporal.server.api.umpire.v1 as ir

final private class SchemaPhase(store: DefinitionStore, signatures: Boolean = false)
    extends PluginPhase:
  val phaseName = if signatures then "reifySignature" else "reifySchema"
  override val runsAfter = if signatures then Set("reifyRuntimeCapture") else Set("reifyNames")
  override val runsBefore = if signatures then Set("inlining") else Set("sbt-api", "pickler")
  private var selectorRun = Option.empty[dotty.tools.dotc.Run]
  private val selectorUnits = mutable.Set.empty[dotty.tools.dotc.CompilationUnit]

  override def transformTemplate(tree: tpd.Template)(using ctx: Context): tpd.Tree =
    if !signatures || !ctx.owner.isClass || !ctx.owner.is(Module) then return tree
    val receiver = This(ctx.owner.asClass).withSpan(tree.span)
    val name = ctx.owner.name.toString.stripSuffix("$")
    val spelling = name.take(1).toLowerCase + name.drop(1)
    val actor = getClassIfDefined("framework.dsl.Actor")
    val actorRegistration =
      if actor.exists && ctx.owner.asClass.typeRef.derivesFrom(actor.asClass) then
        List(
          receiver
            .select(receiver.tpe.member(termName("retainActorName")).symbol)
            .appliedTo(Literal(Constant(spelling)))
        )
      else Nil
    val machine = getClassIfDefined("framework.dsl.Machine")
    val declared = Iterator
      .iterate(ctx.owner)(_.owner)
      .takeWhile(s => s.exists && !s.is(Package))
      .forall(s => s.isClass && s.is(Module))
    val machineRegistration =
      if machine.exists && declared && ctx.owner.asClass.typeRef.derivesFrom(machine.asClass) then
        val e = store.engine(ctx.compilationUnit.tpdTree)
        val family = e.ctx.familyOf(ctx.owner.asInstanceOf[e.ctx.quotes.reflect.Symbol])
        val identity = receiver
          .select(receiver.tpe.member(termName("retainHeaderIdentity")).symbol)
          .appliedToArgs(List(Literal(Constant(family)), Literal(Constant(spelling))))
        val arguments = ctx.owner.asClass.typeRef.baseType(machine.asClass).argInfos
        val schemas = arguments.zipWithIndex.collect {
          case (arg, index) if !open(arg) =>
            metadata(arg, receiver) { (e, root) =>
              val key = e.identity(
                arg.asInstanceOf[e.ctx.quotes.reflect.TypeRepr],
                root,
                receiver.asInstanceOf[e.ctx.quotes.reflect.Tree]
              )
              receiver
                .select(receiver.tpe.member(termName("retainHeaderSchema")).symbol)
                .appliedToArgs(
                  List(
                    Literal(Constant(index)),
                    Literal(Constant(key)),
                    Carrier.literals(Carrier.chunks(root.toByteArray)),
                    Carrier.literals(
                      Carrier
                        .chunks(ir.Model(types = e.ctx.types.values.map(_.clearPosition).toVector))
                    ),
                    Carrier.literals(
                      Carrier.chunks(
                        ir.Record(fields =
                          e.pending.toVector
                            .map((id, ref) => ir.Field(id, Some(ref)))
                        ).toByteArray
                      )
                    )
                  )
                )
            }
        }
        identity :: schemas
      else Nil
    cpy.Template(tree)(body = actorRegistration ++ machineRegistration ++ tree.body)

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
    val names = List(
      "framework.dsl.Action",
      "framework.dsl.Channel",
      "framework.dsl.Entity",
      "framework.dsl.Observation",
      "framework.dsl.Assumption",
      "framework.dsl.Hole",
      "framework.dsl.Limits"
    )
    if tree.symbol.is(Param) || tree.symbol.is(Synthetic) || !names.exists { name =>
        val cls = getClassIfDefined(name)
        cls.exists && tree.tpt.tpe.widen.derivesFrom(cls.asClass)
      }
    then rhs
    else
      val reference =
        List("framework.dsl.Entity", "framework.dsl.Observation", "framework.dsl.Limits")
          .exists { name =>
            val cls = getClassIfDefined(name)
            cls.exists && tree.tpt.tpe.widen.derivesFrom(cls.asClass)
          }
      if reference && tree.symbol.owner.is(Method) then return rhs
      val identityOwner = List("framework.dsl.Assumption", "framework.dsl.Hole").exists { name =>
        val cls = getClassIfDefined(name)
        cls.exists && tree.tpt.tpe.widen.derivesFrom(cls.asClass)
      }
      if identityOwner && !Iterator
          .iterate(tree.symbol.owner)(_.owner)
          .takeWhile(s => s.exists && !s.is(Package))
          .forall(s => s.isClass && s.is(Module))
      then return rhs
      val id = tree.symbol.fullName.toString
        .split('.')
        .filterNot(_.endsWith("$package$"))
        .map(_.stripSuffix("$"))
        .mkString(".")
      val retained = rhs
        .select(rhs.tpe.widen.member(termName("retainName")).symbol)
        .appliedToArgs(List(Literal(Constant(id)), Literal(Constant(tree.name.toString))))
        .withSpan(rhs.span)
      val anonymous = retained.tpe.widen.member(termName("retainAnonymousName")).symbol
      if anonymous.exists && tree.symbol.is(Given) && tree.name.toString.startsWith("given_") then
        retained.select(anonymous).appliedTo(Literal(Constant(id))).withSpan(rhs.span)
      else retained

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
    if signatures then
      val observation = getClassIfDefined("framework.dsl.Observation")
      if observation.exists && tree.symbol == observation.asClass.typeRef
          .member(termName("name"))
          .symbol
      then
        tree.qualifier
          .select(observation.asClass.typeRef.member(termName("sourceName")).symbol)
          .withSpan(tree.span)
      else tree
    else schemaRead(tree)

  private def concrete(tpe: Type)(using Context): Boolean =
    val sym = tpe.typeSymbol
    val singleton = tpe.dealias.termSymbol
    !open(tpe) && !sym.is(Synthetic) && !(sym.is(Enum) && sym.is(Case)) &&
    !(singleton.is(Enum) && singleton.is(Case)) &&
    sym != defn.IntClass

  override def transformValDef(tree: tpd.ValDef)(using ctx: Context): tpd.Tree =
    if signatures then
      val rhs = if tree.symbol.exists then
        inboxOwnership(tree.rhs, tree.tpt.tpe)(using ctx.fresh.setOwner(tree.symbol))
      else tree.rhs
      cpy.ValDef(tree)(rhs = declarationName(signatureAttach(rhs, tree.tpt.tpe), tree))
    else if tree.symbol.is(Param) || tree.symbol.is(Synthetic) then tree
    else cpy.ValDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  private def inboxOwnership(rhs: tpd.Tree, declared: Type)(using ctx: Context): tpd.Tree =
    val argument = finiteArgument(declared).filter(concrete)
    if argument.isEmpty || rhs.isEmpty then return rhs
    def unwrapped(value: tpd.Tree): tpd.Tree = value match
      case Apply(fun: Select, _) if fun.symbol.name.toString == "retainSchema" =>
        unwrapped(fun.qualifier)
      case Typed(value, _) => unwrapped(value)
      case _               => value
    unwrapped(rhs) match
      case block: tpd.Block =>
        metadata(argument.get, rhs) { (e, root) =>
          import e.ctx.quotes.reflect.*
          val key = e.identity(argument.get.asInstanceOf[TypeRepr], root, rhs.asInstanceOf[Tree])
          val shape = e.ctx.types.get(root.getNamed)
          val coordinates = shape.toList.flatMap(t =>
            t.getRecord.fields.map("" -> _) ++ t.getEnum.cases.flatMap(c =>
              c.fields.map(c.name -> _)
            )
          )
          val bindings = block.stats
            .collect { case givenValue: tpd.ValDef if givenValue.symbol.is(Given) => givenValue }
            .flatMap { givenValue =>
              finiteArgument(givenValue.tpt.tpe).toList.flatMap { inbox =>
                val applied = inbox.widen.dealias
                if applied.typeSymbol.fullName.toString != "framework.dsl.Inbox" then Nil
                else
                  unwrapped(givenValue.rhs) match
                    case contents: tpd.Select if contents.name.toString == "contents" =>
                      val receiver = contents.qualifier.asInstanceOf[Term]
                      receiver match
                        case _: Ref =>
                          val message = applied.argInfos.head.asInstanceOf[TypeRepr]
                          val messageRef = e.typeRef(message, receiver)
                          val messageKey = e.identity(message, messageRef, receiver)
                          val id = e.ctx.definitionId(e.ctx.resolveSymbol(receiver), receiver)
                          coordinates.collect {
                            case (caseName, field)
                                if field.getType.ref.channel.contains(messageKey) =>
                              (caseName, field.name, messageKey, id, receiver)
                          }
                        case _ => Nil
                    case _ => Nil
              }
            }
          val result = bindings.foldLeft(block.expr) {
            case (value, (caseName, field, message, id, receiver)) =>
              val supplier = Lambda(
                Symbol.spliceOwner,
                MethodType(Nil)(_ => Nil, _ => receiver.tpe.widen),
                (owner, _) => receiver.changeOwner(owner)
              ).asInstanceOf[tpd.Tree]
              value
                .select(value.tpe.widen.member(termName("retainInbox")).symbol)
                .appliedToArgs(
                  List(key, root.getNamed, caseName, field, message, id)
                    .map(s => tpd.Literal(dotty.tools.dotc.core.Constants.Constant(s))) :+ supplier
                )
                .withSpan(block.expr.span)
          }
          def replace(value: tpd.Tree): tpd.Tree = value match
            case current: tpd.Block if current eq block => cpy.Block(current)(current.stats, result)
            case call @ tpd.Apply(fun: tpd.Select, args)
                if fun.symbol.name.toString == "retainSchema" =>
              cpy.Apply(call)(cpy.Select(fun)(replace(fun.qualifier), fun.name), args)
            case typed: tpd.Typed => cpy.Typed(typed)(replace(typed.expr), typed.tpt)
            case _                => value
          replace(rhs)
        }
      case _ => rhs

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    if signatures then cpy.DefDef(tree)(rhs = signatureAttach(tree.rhs, tree.tpt.tpe))
    else if tree.symbol.is(Synthetic) then tree
    else
      retainForwarding()
      cpy.DefDef(tree)(rhs = attach(tree.rhs, tree.tpt.tpe))

  private def selectorParameters(method: Symbol)(using Context): Set[String] =
    if !method.exists then Set.empty
    else
      val owner = method.owner.fullName.toString
      val name = method.name.toString
      if owner == "framework.dsl.Composition" && Set("own", "synced", "withMember", "<init>")(name)
      then Set("member", "move", "members")
      else if owner == "framework.dsl.Syncs" && name == "sync" then Set("first", "second")
      else
        val annotation = getClassIfDefined("framework.dsl.memberSelectors")
        if !annotation.exists then Set.empty
        else
          method
            .getAnnotation(annotation)
            .toList
            .flatMap(_.arguments)
            .collect:
              case Literal(c) if c.value.isInstanceOf[String] => c.stringValue
            .flatMap(_.split(',').filter(_.nonEmpty))
            .toSet

  private def selectorArguments(call: tpd.Apply, parameters: Set[String])(using Context): Set[Int] =
    call.fun.tpe.widen match
      case method: CompilerMethodType =>
        method.paramNames.zipWithIndex.collect {
          case (name, index) if parameters(name.toString) => index
        }.toSet
      case _ => Set.empty

  private def retainForwarding()(using ctx: Context): Unit =
    val annotation = getClassIfDefined("framework.dsl.memberSelectors")
    if !annotation.exists then return
    if !selectorRun.exists(_ eq ctx.run) then
      selectorRun = Some(ctx.run)
      selectorUnits.clear()
    if !selectorUnits.add(ctx.compilationUnit) then return
    val definitions = mutable.LinkedHashMap.empty[Symbol, tpd.DefDef]
    new tpd.TreeTraverser:
      def traverse(value: tpd.Tree)(using Context): Unit =
        value match
          case d: tpd.DefDef if !d.symbol.is(Synthetic) => definitions(d.symbol) = d
          case _                                        => ()
        traverseChildren(value)
    .traverse(ctx.compilationUnit.tpdTree)
    val contracts = mutable.Map.empty[Symbol, Set[String]]
    var changed = true
    while changed do
      changed = false
      definitions.foreach { (symbol, tree) =>
        val parameters = tree.paramss.flatten.collect { case parameter: tpd.ValDef =>
          parameter.symbol
        }.toSet
        val forwarded = mutable.Set.empty[String]
        def parameter(argument: tpd.Tree): Unit = argument match
          case NamedArg(_, value)                       => parameter(value)
          case Typed(value, _)                          => parameter(value)
          case SeqLiteral(values, _)                    => values.foreach(parameter)
          case value: Ident if parameters(value.symbol) =>
            forwarded += value.symbol.name.toString: Unit
          case _ => ()
        new tpd.TreeTraverser:
          def traverse(value: tpd.Tree)(using Context): Unit =
            value match
              case call: tpd.Apply =>
                val selected = selectorParameters(call.fun.symbol) ++ contracts.getOrElse(
                  call.fun.symbol,
                  Set.empty
                )
                selectorArguments(call, selected).foreach(index =>
                  call.args.lift(index).foreach(parameter)
                )
              case _ => ()
            traverseChildren(value)
        .traverse(tree.rhs)
        val next = contracts.getOrElse(symbol, Set.empty) ++ forwarded
        if contracts.get(symbol).forall(_ != next) then
          contracts(symbol) = next
          changed = true
      }
    definitions.foreach { (symbol, tree) =>
      val contract = contracts.getOrElse(symbol, Set.empty)
      if contract.nonEmpty then
        symbol.addAnnotation(
          Annotation(
            annotation.asClass,
            List(Literal(Constant(contract.toVector.sorted.mkString(",")))),
            tree.span
          )
        )
    }

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
    if signatures then
      val named = tree.fun match
        case selected: tpd.Select if selected.hasAttachment(ChannelNames.generated) =>
          selected.qualifier
            .select(selected.qualifier.tpe.widen.member(termName("sourceAt")).symbol)
            .appliedToArgs(tree.args)
            .withSpan(tree.span)
        case _ => tree
      val indexes = selectorArguments(tree, selectorParameters(tree.fun.symbol))
      val args = named.args.zipWithIndex.map((argument, index) =>
        if indexes(index) then retainSelector(argument) else argument
      )
      val retained = cpy.Apply(named)(named.fun, args)
      return signatureAttach(nameProvenance(retained), tree.tpe, appendedInput(tree))
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

  private def nameProvenance(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val owner = tree.fun.symbol.owner.fullName.toString.stripSuffix("$")
    val operation = tree.fun.symbol.name.toString
    if !Set("framework.dsl.Entity", "framework.dsl.Observation", "framework.dsl.Limits")(owner) ||
      !Set("apply", "<init>", "copy")(operation)
    then return tree
    val index = tree.fun.tpe.widen match
      case method: CompilerMethodType => method.paramNames.indexWhere(_.toString == "name")
      case _                          => -1
    if index < 0 || index >= tree.args.size then return tree
    def default(value: tpd.Tree, seen: Set[Symbol] = Set.empty): Boolean = value match
      case NamedArg(_, arg)  => default(arg, seen)
      case Typed(arg, _)     => default(arg, seen)
      case Apply(fun, _)     => default(fun, seen)
      case TypeApply(fun, _) => default(fun, seen)
      case _ if value.symbol.exists && value.symbol.name.toString.contains("$default$") => true
      case ref: Ident if !seen(ref.symbol)                                              =>
        var rhs = Option.empty[tpd.Tree]
        new tpd.TreeTraverser:
          def traverse(value: tpd.Tree)(using Context): Unit =
            value match
              case v: ValDef if v.symbol == ref.symbol => rhs = Some(v.rhs)
              case _                                   => traverseChildren(value)
        .traverse(ctx.compilationUnit.tpdTree)
        rhs.exists(default(_, seen + ref.symbol))
      case _ => false
    val explicit = !default(tree.args(index))
    val method = if operation == "copy" then "retainCopyName" else "retainExplicitName"
    tree
      .select(tree.tpe.widen.member(termName(method)).symbol)
      .appliedTo(Literal(Constant(explicit)))
      .withSpan(tree.span)

  private def retainSelector(argument: tpd.Tree)(using ctx: Context): tpd.Tree = argument match
    case arg: tpd.NamedArg   => cpy.NamedArg(arg)(arg.name, retainSelector(arg.arg))
    case arg: tpd.Typed      => cpy.Typed(arg)(retainSelector(arg.expr), arg.tpt)
    case arg: tpd.SeqLiteral => cpy.SeqLiteral(arg)(arg.elems.map(retainSelector), arg.elemtpt)
    case _                   =>
      val function = argument.tpe.widen.baseType(requiredClass("scala.Function1").asClass).argInfos
      val module = getModuleIfDefined("framework.dsl.MemberPath")
      if function.size != 2 || !module.exists then return argument
      val e = store.engine(ctx.compilationUnit.tpdTree)
      import e.ctx.quotes.reflect.*
      val term = argument.asInstanceOf[Term]
      def result(body: Term): Term = body match
        case Block(_, value)        => result(value)
        case Typed(value, _)        => result(value)
        case Inlined(_, Nil, value) => result(value)
        case _                      => body
      def arrow(body: Term): Option[(Term, Term)] = result(body) match
        case Apply(
              TypeApply(Select(Apply(TypeApply(Ident("ArrowAssoc"), _), List(key)), "->"), _),
              List(value)
            ) =>
          Some(key -> value)
        case _ => None
      def action(value: Term): Boolean =
        def is(tpe: TypeRepr, name: String): Boolean = tpe.widen match
          case OrType(left, right) => is(left, name) || is(right, name)
          case _                   =>
            val cls = Symbol.classSymbol(name)
            cls != Symbol.noSymbol && tpe.widen.baseClasses.contains(cls)
        is(value.tpe, "framework.dsl.Action") || is(value.tpe, "framework.dsl.Class")
      e.lambda(term) match
        case Some((List(parameter), body)) =>
          val pair = arrow(body)
          def independent(value: Term): Boolean =
            var reads = false
            new TreeTraverser:
              override def traverseTree(tree: Tree)(owner: Symbol): Unit =
                if tree.symbol == parameter.symbol ||
                  (tree.symbol != Symbol.noSymbol && tree.symbol.owner == parameter.symbol.owner)
                then reads = true
                super.traverseTree(tree)(owner)
            .traverseTree(value)(Symbol.spliceOwner)
            !reads
          if pair.exists((_, value) => !independent(value)) then return argument
          val key = pair.fold(result(body))(_._1)
          e.fieldPath(parameter, key) match
            case Some(List(field)) =>
              val selected = pair.filter((_, value) => action(value))
              val method = if selected.nonEmpty then "retainMove" else "retain"
              val extra = selected.toList.map { (_, value) =>
                Lambda(
                  Symbol.spliceOwner,
                  MethodType(Nil)(_ => Nil, _ => value.tpe.widen),
                  (owner, _) => value.changeOwner(owner)
                ).asInstanceOf[tpd.Tree]
              }
              tpd
                .ref(module)
                .select(termName(method))
                .appliedToTypes(function)
                .appliedToArgs(
                  List(
                    argument,
                    tpd.Literal(dotty.tools.dotc.core.Constants.Constant(field))
                  ) ++ extra
                )
                .withSpan(argument.span)
            case _ => argument
        case _ => argument

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
