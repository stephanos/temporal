package framework.`export`

import framework.dsl.{Capabilities, IrFile, Query}
import framework.realize.ConstructedRealization
import framework.realize.ConstructedRealization.ActionDependency
import io.temporal.server.api.umpire.v1 as ir
import scala.collection.mutable
import scala.util.control.NonFatal

private[`export`] object Constructed:
  def read(root: String, loader: ClassLoader): ConstructedRealization =
    try
      value(root, loader) match
        case result: ConstructedRealization => result
        case _                              =>
          throw IllegalArgumentException(
            "the declaration is not a constructed realization"
          )
    catch
      case error: IllegalArgumentException
          if Option(error.getMessage).exists(
            _.startsWith("while constructing root ")
          ) =>
        throw error
      case NonFatal(error)     => throw refused(root, error)
      case error: LinkageError => throw refused(root, error)

  private[`export`] def value(root: String, loader: ClassLoader): Any =
    val moduleName = root.replace("$.", "$") + "$"
    try
      val cls = java.lang.Class.forName(moduleName, true, loader)
      cls.getField("MODULE$").get(null) // scalafix:ok DisableSyntax.null
    catch
      case _: ClassNotFoundException =>
        val split = root.lastIndexOf('.')
        require(split > 0, s"$root has no declaring object")
        val owner = root.take(split).replace("$.", "$")
        val cls = java.lang.Class.forName(owner, true, loader)
        val module =
          cls.getField("MODULE$").get(null) // scalafix:ok DisableSyntax.null
        val getter = cls.getDeclaredMethod(root.drop(split + 1))
        require(
          getter.trySetAccessible(),
          s"$root has no accessible declaring getter"
        )
        getter.invoke(module)

  private def refused(
      root: String,
      error: Throwable
  ): IllegalArgumentException =
    val cause =
      Iterator.iterate(error)(_.getCause).takeWhile(_ != null).toVector.last
    IllegalArgumentException(
      s"while constructing root $root, declaration $root failed: ${cause.getMessage}",
      error
    )

private[`export`] trait Constructed:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*
  import Equality.given

  final private[`export`] case class Pending(
      root: String,
      declaration: String,
      at: Tree,
      sourcePackage: Option[String],
      value: ConstructedRealization
  )

  private lazy val constructedClass = Symbol.requiredClass(
    "framework.realize.ConstructedRealization"
  )

  def constructedRoot(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    (!cls.isNoSymbol && cls.typeRef
      .derivesFrom(constructedClass)) || defs.get(sym).exists {
      case value: ValDef =>
        value.tpt.tpe.widen.dealias.derivesFrom(constructedClass)
      case _ => false
    }

  def constructedRoot(root: String): Boolean = constructedRoot(rootSymbol(root))

  def prepareConstructed(root: String): Pending =
    val sym = rootSymbol(root)
    val at = valDef(sym, sym.tree, "a constructed realization")
    validateContract(sym, at, root)
    val declaration = definitionId(sym, at)
    try
      val value = Constructed.read(root, runtimeLoader)
      val machineId = IrFile.identity(value.machine)
      val machine = defs.keys
        .find(s => s.fullName.replace("$.", ".").stripSuffix("$") == machineId)
        .getOrElse(
          fail(
            at,
            s"constructed machine $machineId has no signature declaration"
          )
        )
      machineOf(machine, at): Unit
      val dependencies = value.dependencies
      dependencies.channels.distinct.sorted.foreach(channel =>
        channelOf(sourceDeclaration(channel, at), at): Unit
      )
      dependencies.actions.foreach:
        case ActionDependency.Declaration(identity) =>
          action(Ref(sourceDeclaration(identity, at))): Unit
        case ActionDependency.Channel(identity, operation) =>
          channelAction(sourceDeclaration(identity, at), operation, at): Unit
      Pending(root, declaration, at, Some(ctx.familyOf(sym)), value)
    catch
      case error: LiftError if error.message.startsWith("while constructing root ") =>
        throw error
      case error: LiftError =>
        throw LiftError(
          error.position,
          failureMessage(root, declaration, error.message)
        )
      case NonFatal(error)     => throw failed(root, declaration, at, error)
      case error: LinkageError => throw failed(root, declaration, at, error)

  def completeConstructed(pending: Seq[Pending]): Seq[LiftError] =
    val converted = mutable.ArrayBuffer.empty[(Pending, ir.Realization)]
    val refused = mutable.ArrayBuffer.empty[LiftError]
    val metadata = constructionMetadata
    pending.foreach: p =>
      try
        val context = ConstructedRealization.Context(
          p.declaration,
          metadata,
          p.sourcePackage
        )
        val result = ConstructedRealization.resolveFamily(
          p.value.toIR(context),
          context
        )
        require(
          result.id == p.declaration,
          s"constructed realization id ${result.id} differs from ${p.declaration}"
        )
        require(
          result.queries.isEmpty,
          "constructed realization conversion sets no Queries; the export driver owns selection"
        )
        converted += p -> result.withPosition(pos(p.at)).withQueries(runNames(p, result.machine))
      catch
        case error: LiftError if error.message.startsWith("while constructing root ") =>
          refused += error
        case error: LiftError =>
          refused += LiftError(
            error.position,
            failureMessage(p.root, p.declaration, error.message)
          )
        case NonFatal(error) =>
          refused += failed(p.root, p.declaration, p.at, error)
        case error: LinkageError =>
          refused += failed(p.root, p.declaration, p.at, error)
    if refused.isEmpty then
      val names = mutable.Map.empty[String, (String, Pending)]
      val runBy = mutable.Map.empty[String, (String, Pending)]
      converted.foreach: (pending, realization) =>
        names
          .get(realization.name)
          .foreach: (other, _) =>
            refused += LiftError(
              where(pending.at),
              s"two realizations are named ${realization.name}: $other and ${realization.id}, and the IR names them by name"
            )
        names(realization.name) = realization.id -> pending
        realization.queries.foreach: query =>
          runBy
            .get(query)
            .foreach: (other, _) =>
              if other != realization.name then
                refused += LiftError(
                  where(pending.at),
                  s"$query is named by the realizations $other and ${realization.name}: one realization runs it"
                )
          runBy(query) = realization.name -> pending
    if refused.isEmpty then
      converted.foreach((pending, realization) => realizations(pending.declaration) = realization)
    refused.toSeq

  private def rootSymbol(root: String): Symbol = defs.keys
    .find(s => s.isValDef && s.fullName == root)
    .getOrElse(
      throw LiftError(
        s"root $root",
        "names no declaration of the lifted sources"
      )
    )

  private def validateContract(sym: Symbol, at: Tree, root: String): Unit =
    def strings(tree: Tree): List[String] = tree match
      case Literal(StringConstant(value)) => List(value)
      case Typed(value, _)                => strings(value)
      case Repeated(values, _)            => values.flatMap(strings)
      case Apply(_, args)                 => args.flatMap(strings)
      case _                              => Nil
    val carrier = Symbol.requiredClass("framework.dsl.familyFlow")
    val symbols = Seq(sym, moduleClassOf(sym)).filterNot(_.isNoSymbol).distinct
    val contract = symbols.iterator
      .flatMap(_.getAnnotation(carrier))
      .map(strings)
      .find(_.nonEmpty)
      .getOrElse(Nil)
    val valid = contract match
      case List("v1", result) =>
        result.split(',').filter(_.nonEmpty).forall(Set("-1", "-2"))
      case _ => false
    if !valid then
      fail(
        at,
        s"while constructing root $root: missing or invalid compiler source provenance; compile the declaration with the reify plugin"
      )

  private def sourceDeclaration(sourceId: String, at: Tree): Symbol = defs.keys
    .find(s =>
      s.fullName
        .split('.')
        .filterNot(_.endsWith("$package$"))
        .map(_.stripSuffix("$"))
        .mkString(".") == sourceId
    )
    .getOrElse(
      fail(
        at,
        s"retained source declaration $sourceId has no signature declaration"
      )
    )

  private def constructionMetadata: ir.Model =
    def sorted[K: Ordering, V](values: collection.Map[K, V]): Seq[V] =
      values.toSeq.sortBy(_._1).map(_._2)
    ir.Model(
      types = sorted(types),
      functions = sorted(functions),
      actions = sorted(actions),
      machines = machines.values.toSeq.sortBy(_.name),
      channels = sorted(channels),
      monitors = sorted(monitors),
      assumptions = sorted(assumptions),
      holes = sorted(holes),
      compositions = compositions.values.toSeq.sortBy(_.name),
      properties = sorted(properties),
      scenarios = sorted(scenarios),
      queries = sorted(queries),
      progress = sorted(progress)
    )

  private def runNames(pending: Pending, machine: String): Seq[String] =
    val found = pending.value.runs.flatMap(runSelection(pending, _)).distinct
    found.foreach: name =>
      val query = queries.getOrElse(
        name,
        throw LiftError(
          where(pending.at),
          s"the realization runs no lifted Query $name"
        )
      )
      val owner = query.getScenario.machine
      if owner != machine then
        throw LiftError(
          where(pending.at),
          s"the realization realizes $machine and names $name, a Query of $owner"
        )
    found.filter(name => queries(name).form == ir.Query.Form.FORM_FIND)

  private def runSelection(pending: Pending, value: Any): Seq[String] =
    value match
      case query: Query                        => Seq(runtimeQueryName(pending, query))
      case values: Seq[?]                      => values.flatMap(runSelection(pending, _))
      case capabilities: Capabilities[?, ?, ?] =>
        val machine = capabilities.model.name
        queries.values
          .filter(q =>
            q.getProperty.machine == machine &&
              properties
                .get((machine, q.getProperty.name))
                .exists(_.origin.nonEmpty)
          )
          .map(_.name)
          .toSeq
          .sorted
      case section: AnyRef =>
        val values = IrFile.queriesOf(section)
        if values.isEmpty then
          throw LiftError(
            where(pending.at),
            s"the realization runs Queries, a queries or a capabilities section, not ${section.getClass.getName}"
          )
        values.flatMap(runSelection(pending, _))
      case other =>
        throw LiftError(
          where(pending.at),
          s"the realization runs Queries, not ${Option(other).fold("null")(_.getClass.getName)}"
        )

  private def runtimeQueryName(pending: Pending, query: Query): String =
    if query.name.nonEmpty && queries.contains(query.name) then query.name
    else
      defs.keys.iterator
        .flatMap(sym =>
          folded.get(sym).collect {
            case Decl.Declared(name) if queries.contains(name) =>
              sym -> name
          }
        )
        .collectFirst {
          case (sym, name)
              if scala.util
                .Try(Constructed.value(sym.fullName, runtimeLoader))
                .toOption
                .exists {
                  case candidate: Query => candidate eq query
                  case _                => false
                } =>
            name
        }
        .getOrElse(
          throw LiftError(
            where(pending.at),
            "the realization runs a Query with no constructed declaration identity"
          )
        )

  private def failed(
      root: String,
      declaration: String,
      at: Tree,
      error: Throwable
  ): LiftError =
    val cause =
      Iterator.iterate(error)(_.getCause).takeWhile(_ != null).toVector.last
    LiftError(
      where(at),
      failureMessage(root, declaration, cause.getMessage)
    )

  private def failureMessage(
      root: String,
      declaration: String,
      message: String
  ): String =
    s"while constructing root $root, declaration $declaration failed: $message"
