package framework.reify

import scala.collection.mutable
import scala.util.control.NonFatal
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Flags.{Accessor, Module, Package, Param, ParamAccessor}
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.PluginPhase
import dotty.tools.dotc.report
import dotty.tools.dotc.transform.{Pickler, PostTyper}

final private class FamilyPhase extends PluginPhase:
  val phaseName = "reifyFamily"
  override val runsAfter = Set(PostTyper.name)
  override val runsBefore = Set("reifyStore", Pickler.name)

  private case class Flow(result: Set[Int], refused: Map[Int, String])
  private val sites = mutable.Map.empty[Symbol, tpd.MemberDef]
  private val templates = mutable.Map.empty[Symbol, tpd.Template]
  private val bindings = mutable.Set.empty[Symbol]
  private val closureMethods = mutable.Set.empty[Symbol]
  private val flows = mutable.Map.empty[Symbol, Flow]
  private val active = mutable.Set.empty[Symbol]
  private var trustedArtifacts = Set.empty[String]
  private var run = Option.empty[dotty.tools.dotc.Run]
  private val contextual = "\u0000umpire-family-context:v1\u0000"
  private val reserved = "\u0000umpire-family-"

  private def isFamily(sym: Symbol)(using Context): Boolean =
    sym.exists && sym.name.toString == "family" &&
      sym.owner.fullName.toString.stripSuffix("$") == "framework.realize.Realize$package"

  private def index()(using ctx: Context): Unit =
    if !run.exists(_ eq ctx.run) then
      sites.clear()
      templates.clear()
      bindings.clear()
      closureMethods.clear()
      flows.clear()
      active.clear()
      run = Some(ctx.run)
      trustedArtifacts = List(
        "framework.dsl.familyFlow",
        "io.temporal.server.api.umpire.v1.Model",
        "scala.Option",
        "scala.deriving.Mirror",
        "scalapb.GeneratedMessage",
        "scalapb.grpc.Marshaller",
        "scalapb.lenses.Lens",
        "com.google.protobuf.Message",
        "io.grpc.Channel",
        "io.grpc.stub.AbstractStub",
        "io.grpc.protobuf.ProtoUtils",
        "io.grpc.protobuf.lite.ProtoLiteUtils"
      ).flatMap(name => artifact(getClassIfDefined(name))).toSet
      ctx.run.units.foreach { unit =>
        new tpd.TreeTraverser:
          def traverse(tree: tpd.Tree)(using Context): Unit =
            tree match
              case definition: tpd.TypeDef if definition.rhs.isInstanceOf[tpd.Template] =>
                templates(definition.symbol) = definition.rhs.asInstanceOf[tpd.Template]
              case binding: tpd.Bind     => bindings += binding.symbol
              case closure: tpd.Closure  => closureMethods += closure.meth.symbol
              case member: tpd.MemberDef => sites(member.symbol) = member
              case _                     => ()
            traverseChildren(tree)
        .traverse(unit.tpdTree)
      }

  private def params(sym: Symbol)(using Context): List[Symbol] =
    sym.paramSymss.flatten.filter(_.isTerm)

  private def packageOf(sym: Symbol)(using Context): String =
    Iterator.iterate(sym)(_.owner).find(_.is(Package)).get.fullName.toString

  private def vocabulary(pkg: String): Boolean =
    pkg == "framework.realize" || pkg.startsWith("framework.realize.")

  private def artifact(sym: Symbol)(using Context): Option[String] =
    if !sym.exists then None
    else
      val file = sym.binaryFile
      if file == null then None else file.underlyingSource.map(_.path)

  private def trusted(sym: Symbol)(using Context): Boolean =
    val file = if sym.exists then sym.binaryFile else null
    val runtimeImage = file != null && (file.jpath match
      case path: java.nio.file.Path => path.getFileSystem.provider().getScheme == "jrt"
      case null                     => false)
    runtimeImage || artifact(sym).exists(trustedArtifacts) ||
    defn.AnyMethods.contains(sym) || defn.ObjectMethods.contains(sym) || sym == defn.throwMethod

  private def stored(sym: Symbol)(using ctx: Context): Option[Flow] =
    val carrier = getClassIfDefined("framework.dsl.familyFlow")
    if !carrier.exists then None
    else
      sym.getAnnotation(carrier).map { annotation =>
        try
          val values = annotation.arguments.flatMap(Carrier.strings)
          require(
            values.headOption.contains("v1"),
            s"${sym.fullName}: unknown family provenance version"
          )
          require(values.size >= 2, "missing family result contract")
          val result = values(1).split(',').filter(_.nonEmpty).map(_.toInt).toSet
          val refused = values
            .drop(2)
            .map { value =>
              val split = value.indexOf(':')
              require(split > 0, "missing family refusal index")
              value.take(split).toInt -> value.drop(split + 1)
            }
            .toMap
          require(
            (result ++ refused.keySet).forall(i => i >= -2 && i < params(sym).size),
            "family parameter index is outside the helper signature"
          )
          Flow(result, refused)
        catch
          case NonFatal(error) =>
            report.error(
              s"${sym.fullName}: invalid family forwarding contract: ${error.getMessage}",
              ctx.compilationUnit.tpdTree.srcPos
            )
            Flow(Set(-2), Map.empty)
      }

  private def flow(sym: Symbol)(using ctx: Context): Option[Flow] =
    index()
    flows
      .get(sym)
      .orElse(stored(sym))
      .orElse(sites.get(sym).map { definition =>
        val arguments = params(sym)
        if !active.add(sym) then
          Flow(
            Set.empty,
            arguments.indices.map(_ -> "recursive helper has no family contract").toMap
          )
        else
          val rhs = definition match
            case d: DefDef => d.rhs
            case v: ValDef => v.rhs
            case _         => EmptyTree
          val contract = analyze(rhs, arguments, sym)
          flows(sym) = contract
          active.remove(sym): Unit
          contract
      })

  private def analyze(rhs: tpd.Tree, arguments: List[Symbol], sourceOwner: Symbol)(using
      Context
  ): Flow =
    val refused = mutable.Map.empty[Int, String]
    val lifted = getClassIfDefined(Carrier.marker)
    var insideCapture = lifted.exists && sourceOwner.hasAnnotation(lifted)
    def reject(origins: Set[Int], tree: tpd.Tree, operation: String): Unit =
      val where = s"${tree.srcPos.sourcePos.source.file.path}:${tree.srcPos.sourcePos.line + 1}"
      origins.foreach(i => refused.getOrElseUpdate(i, s"$operation at $where")): Unit
    def evaluate(tree: tpd.Tree, env: Map[Symbol, Set[Int]]): Set[Int] =
      def read(sym: Symbol): Set[Int] =
        if isFamily(sym) then Set(if vocabulary(packageOf(sourceOwner)) then -2 else -1)
        else
          env.getOrElse(
            sym, {
              val contract = flow(sym)
              if !insideCapture && !bindings(
                  sym
                ) && contract.isEmpty && sym.exists && sym.isTerm && !sym.is(
                  Module
                ) &&
                !sym.is(Param) && !trusted(sym)
              then reject(Set(-1), tree, s"${sym.fullName} has no validated model helper contract")
              contract.fold(Set.empty[Int])(_.result.filter(_ < 0))
            }
          )
      tree match
        case Literal(c) =>
          if c.value.isInstanceOf[String] &&
            c.stringValue.contains(reserved)
          then
            report.error(
              "reserved family provenance cannot be authored as a string literal",
              tree.srcPos
            )
          Set.empty
        case _: Ident                    => read(tree.symbol)
        case Typed(value, _)             => evaluate(value, env)
        case NamedArg(_, value)          => evaluate(value, env)
        case Inlined(_, bindings, value) => evaluate(Block(bindings, value), env)
        case Block(statements, value)    =>
          var closures = Set.empty[Int]
          val inner = statements.foldLeft(env) { (scope, statement) =>
            statement match
              case local: ValDef => scope.updated(local.symbol, evaluate(local.rhs, scope))
              case local: DefDef =>
                closures ++= evaluate(local.rhs, scope)
                scope
              case _ => evaluate(statement, scope): Unit; scope
          }
          evaluate(value, inner) ++ closures
        case If(condition, yes, no) =>
          reject(evaluate(condition, env), condition, "guard")
          evaluate(yes, env) ++ evaluate(no, env)
        case Match(selector, cases) =>
          reject(
            evaluate(selector, env),
            selector,
            "family-tainted pattern matching has no forwarding contract"
          )
          cases.flatMap { branch =>
            reject(evaluate(branch.guard, env), branch.guard, "pattern guard")
            evaluate(branch.body, env)
          }.toSet
        case SeqLiteral(values, _)           => values.flatMap(evaluate(_, env)).toSet
        case Closure(environment, method, _) =>
          environment.flatMap(evaluate(_, env)).toSet ++ evaluate(method, env)
        case application: Apply =>
          def clauses(tree: tpd.Tree): (tpd.Tree, List[tpd.Tree]) = tree match
            case Apply(fun, args) =>
              val (base, previous) = clauses(fun)
              (base, previous ++ args)
            case TypeApply(fun, _) => clauses(fun)
            case _                 => tree -> Nil
          val (base, arguments) = clauses(application)
          val method = base.symbol
          val marker = getClassIfDefined(Carrier.capture)
          val capture = marker.exists && method.paramSymss.flatten.exists(_.hasAnnotation(marker))
          def receiver(tree: tpd.Tree): Set[Int] = tree match
            case TypeApply(value, _) => receiver(value)
            case Select(value, _)    => evaluate(value, env)
            case _                   => Set.empty
          val recv = receiver(base)
          val previousCapture = insideCapture
          insideCapture ||= capture
          val args = arguments.map(evaluate(_, env))
          insideCapture = previousCapture
          val all = recv ++ args.flatten
          val owner = method.owner.fullName.toString.stripSuffix("$")
          val name = method.name.toString
          val concatenation = name == "+" && owner == "java.lang.String"
          val interpolation = name == "s" && owner == "scala.StringContext"
          val constructor = Set("apply", "<init>", "copy")(name) &&
            Set(
              "framework.realize.Realization",
              "framework.realize.Correlation",
              "framework.realize.Script",
              "framework.realize.Name",
              "framework.realize.Learned",
              "framework.realize.Actuator",
              "framework.realize.Evidence",
              "framework.realize.Command",
              "framework.realize.Instruction.Hold",
              "framework.realize.Instruction.Release",
              "framework.realize.Instruction.AwaitLearned",
              "framework.realize.Instruction.AwaitCommand"
            )(owner)
          val script = owner == "framework.realize.Scripts$package" && name == "script"
          val collection = name == "apply" && Set(
            "scala.collection.IterableFactory",
            "scala.collection.immutable.Vector",
            "scala.collection.SeqFactory"
          )(owner)
          val conversion = owner == "framework.realize.Conversion$package" &&
            (name == "toIR" || name == "toIR$default$3")
          if capture then
            reject(
              all.filter(_ != -1),
              application,
              "contextual family cannot enter captured function IR"
            )
            Set.empty
          else if conversion then Set.empty
          else if concatenation || interpolation || constructor || script || collection then all
          else if name == "<init>" then
            reject(all, application, s"${method.fullName} has no family constructor contract")
            if !insideCapture && !sites.contains(method) && !trusted(method) then
              reject(
                Set(-1),
                application,
                s"${method.fullName} has no validated model helper contract"
              )
            Set.empty
          else
            flow(method) match
              case Some(contract) =>
                reject(recv, application, s"receiver of ${method.fullName}")
                contract.refused.foreach { (i, reason) =>
                  if i >= 0 then
                    args.lift(i).foreach(origins => reject(origins, arguments(i), reason))
                }
                contract.result.flatMap {
                  case i if i < 0 => Set(i)
                  case i          => args.lift(i).getOrElse(Set.empty)
                }
              case None =>
                reject(all, application, s"${method.fullName} has no family forwarding contract")
                if !insideCapture && !trusted(method) then
                  reject(
                    Set(-1),
                    application,
                    s"${method.fullName} has no validated model helper contract"
                  )
                Set.empty
        case Select(_, _) if isFamily(tree.symbol) => read(tree.symbol)
        case Select(value, _)                      =>
          val recv = evaluate(value, env)
          if tree.symbol.is(Accessor) || tree.symbol.is(Param) || tree.symbol.is(ParamAccessor) then
            recv ++ read(tree.symbol)
          else
            reject(recv, tree, tree.symbol.fullName.toString)
            read(tree.symbol)
        case TypeApply(value, _) => evaluate(value, env)
        case _                   =>
          var origins = Set.empty[Int]
          new tpd.TreeTraverser:
            def traverse(child: tpd.Tree)(using Context): Unit =
              if child eq tree then traverseChildren(child)
              else origins ++= evaluate(child, env)
          .traverse(tree)
          reject(origins, tree, "unsupported family dataflow")
          Set.empty
    val env = arguments.zipWithIndex.map((p, i) => p -> Set(i)).toMap
    val result = if rhs.isEmpty then Set.empty[Int] else evaluate(rhs, env)
    Flow(result, refused.toMap)

  private def retain(tree: tpd.MemberDef)(using Context): Unit =
    index()
    if !tree.symbol.is(Param) && !closureMethods(tree.symbol) then
      flow(tree.symbol).foreach { contract =>
        contract.refused
          .filter(_._1 < 0)
          .values
          .foreach(reason =>
            report.error(s"family cannot be inspected or used as data: $reason", tree.srcPos)
          )
        val carrier = getClassIfDefined("framework.dsl.familyFlow")
        if carrier.exists then
          val values = List("v1", contract.result.toList.sorted.mkString(",")) ++
            contract.refused.toList.filter(_._1 >= 0).sortBy(_._1).map((i, reason) => s"$i:$reason")
          tree.symbol.addAnnotation(
            Annotation(carrier.asClass, List(Carrier.literals(values)), tree.span)
          )
      }

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    retain(tree)
    tree

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    retain(tree)
    tree

  override def transformTemplate(tree: tpd.Template)(using ctx: Context): tpd.Tree =
    index()
    templates.getOrElse(ctx.owner, tree).body.filterNot(_.isInstanceOf[tpd.MemberDef]).foreach {
      expression =>
        val contract = analyze(expression, Nil, ctx.owner)
        contract.refused
          .filter(_._1 < 0)
          .values
          .foreach(reason =>
            report.error(s"family cannot be inspected or used as data: $reason", expression.srcPos)
          )
    }
    tree

  private def reference(tree: tpd.Tree)(using ctx: Context): tpd.Tree =
    if !isFamily(tree.symbol) then tree
    else
      val pkg = packageOf(ctx.owner)
      val value = if vocabulary(pkg) then contextual else pkg
      Literal(Constant(value)).withSpan(tree.span)

  override def transformIdent(tree: tpd.Ident)(using Context): tpd.Tree = reference(tree)
  override def transformSelect(tree: tpd.Select)(using Context): tpd.Tree = reference(tree)

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
    val references = new tpd.TreeMap:
      override def transform(tree: tpd.Tree)(using Context): tpd.Tree = tree match
        case _: tpd.Ident | _: tpd.Select => super.transform(reference(tree))
        case _                            => super.transform(tree)
    val args = tree.args.map:
      case arg: tpd.NamedArg => cpy.NamedArg(arg)(arg.name, references.transform(arg.arg))
      case arg               => arg
    cpy.Apply(tree)(tree.fun, args)
