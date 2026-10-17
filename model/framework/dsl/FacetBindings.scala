package framework.dsl

import java.util.Base64
import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[framework] object FacetBindings:
  final case class Definition(traitName: String, namespace: String, rules: String)
  final case class Origin(book: RuleBook[?, ?, ?], source: FacetSource)
  final case class Site(
      kind: String,
      owner: Option[String],
      call: ir.Call,
      state: ir.TypeRef,
      phase: ir.TypeRef,
      result: ir.TypeRef,
      placement: Option[ir.Position]
  )
  final case class Template(
      definition: Definition,
      packet: RuleBindings.Packet,
      sites: Vector[Site]
  )
  final case class Retained(origin: Origin, template: Template)
  final case class Context(
      source: RecordMachine[?, ?, ?],
      owner: String,
      family: String,
      machine: String,
      projection: String,
      writer: String,
      state: ir.TypeRef,
      phase: Option[ir.TypeRef],
      projectionResult: Option[ir.TypeRef],
      writerResult: Option[ir.TypeRef],
      model: ir.Model,
      positions: Vector[ir.Type],
      proofs: Vector[CoreBindings.CurriedCall]
  )

  private def fail(message: String): Nothing =
    throw IllegalArgumentException(s"Machine Facets: $message")
  private def requireThat(condition: Boolean, message: String): Unit =
    if !condition then fail(message)
  private def read[A](text: String)(decode: Array[Byte] => A): A =
    decode(Base64.getDecoder.decode(text))
  private def decoding[A](body: => A): A =
    try body
    catch
      case scala.util.control.NonFatal(error) =>
        throw IllegalArgumentException(
          s"Machine Facets: malformed retained metadata: ${error.getMessage}",
          error
        )

  final private class Cursor(data: Seq[String]):
    private var index = 0
    def next(): String =
      requireThat(index < data.size, "incomplete retained payload")
      val result = data(index)
      index += 1
      result
    def count(): Int =
      val size = next().toInt
      requireThat(size >= 0 && size <= data.size - index, "invalid retained payload length")
      size
    def chunks(): String = Vector.fill(count())(next()).mkString
    def done(): Unit = requireThat(index == data.size, "unrelated retained payload fields")

  def unwrap(data: Seq[String], version: String): (Definition, Vector[String], Vector[Site]) =
    decoding {
      val c = new Cursor(data)
      requireThat(c.next() == version, "unsupported facet template version")
      val definition = Definition(c.next(), c.next(), c.next())
      requireThat(
        definition.traitName.nonEmpty && definition.namespace.nonEmpty && definition.rules.nonEmpty,
        "missing exact facet definition identity"
      )
      val ordinary = Vector.fill(c.count())(c.next())
      val sites = Vector.fill(c.count()) {
        Site(
          c.next(),
          Option(c.next()).filter(_.nonEmpty),
          read(c.next())(ir.Call.parseFrom),
          read(c.next())(ir.TypeRef.parseFrom),
          read(c.next())(ir.TypeRef.parseFrom),
          read(c.next())(ir.TypeRef.parseFrom),
          Option(c.next()).filter(_.nonEmpty).map(read(_)(ir.Position.parseFrom))
        )
      }
      c.done()
      (definition, ordinary, sites)
    }

  def sites(data: Seq[String]): Vector[Site] = decoding {
    val c = new Cursor(data)
    val result = Vector.fill(c.count()) {
      Site(
        c.next(),
        Option(c.next()).filter(_.nonEmpty),
        read(c.next())(ir.Call.parseFrom),
        read(c.next())(ir.TypeRef.parseFrom),
        read(c.next())(ir.TypeRef.parseFrom),
        read(c.next())(ir.TypeRef.parseFrom),
        Option(c.next()).filter(_.nonEmpty).map(read(_)(ir.Position.parseFrom))
      )
    }
    c.done()
    result
  }

  def template(data: Seq[String]): Template =
    val (definition, ordinary, sites) = unwrap(data, "facet-rule-v1")
    val packet = RuleBindings.packet(ordinary, "Machine Facets template")
    requireThat(packet.owner == definition.traitName, "stale facet template owner")
    Template(definition, packet, sites)

  final class Metadata:
    private var retained = Option.empty[Context]
    def value: Option[Context] = retained
    def retain(source: RecordMachine[?, ?, ?], data: Seq[String]): Unit = synchronized {
      val context = decoding {
        val c = new Cursor(data)
        requireThat(c.next() == "facet-context-v1", "unsupported assembly context version")
        val owner = c.next()
        val family = c.next()
        val machine = c.next()
        val projection = c.next()
        val writer = c.next()
        val state = read(c.next())(ir.TypeRef.parseFrom)
        def optionalType(): Option[ir.TypeRef] =
          Option(c.next()).filter(_.nonEmpty).map(read(_)(ir.TypeRef.parseFrom))
        val phase = optionalType()
        val projectionResult = optionalType()
        val writerResult = optionalType()
        val model = read(c.chunks())(ir.Model.parseFrom)
        val locations = read(c.chunks())(ir.Model.parseFrom)
        requireThat(
          locations.copy(types = Nil).equals(ir.Model()),
          "unrelated context type positions"
        )
        val proofs = Vector.fill(c.count()) {
          CoreBindings.CurriedCall(
            read(c.next())(ir.Call.parseFrom),
            read(c.next())(ir.Function.parseFrom),
            c.next().split(',').filter(_.nonEmpty).map(_.toInt).toVector,
            Option(c.next()).filter(_.nonEmpty)
          )
        }
        c.done()
        Context(
          source,
          owner,
          family,
          machine,
          projection,
          writer,
          state,
          phase,
          projectionResult,
          writerResult,
          model,
          locations.types.toVector,
          proofs
        )
      }
      requireThat(retained.forall(_.equals(context)), "conflicting assembly context")
      retained = Some(context)
    }

  def checkedOrigin(source: RecordMachine[?, ?, ?], origin: Origin, definition: Definition): Unit =
    val identity = origin.source.identity.getOrElse(fail("missing live facet source identity"))
    requireThat(identity.source.eq(source), "foreign facet assembly source")
    requireThat(identity.namespace ne null, "facet namespace is not initialized")
    requireThat(
      (identity.traitClass ne null) && identity.traitClass.isInterface && identity.traitClass
        .isInstance(source),
      "facet trait token does not own this assembly"
    )
    requireThat(
      source.ownsFacetNamespace(
        identity.namespace,
        identity.traitName,
        identity.namespaceName,
        identity.traitClass
      ),
      "foreign or unregistered facet namespace receiver"
    )
    requireThat(
      identity.traitName == definition.traitName &&
        identity.namespaceName == definition.namespace && identity.rulesName == definition.rules,
      "stale facet trait/namespace/Rules definition provenance"
    )
    requireThat(origin.book.facetOrigin.exists(_.eq(origin)), "stale facet RuleBook source token")

  def contributors(
      source: RecordMachine[?, ?, ?],
      book: RuleBook[?, ?, ?]
  ): Option[Vector[Origin]] =
    book match
      case facets: Facets[?, ?, ?] =>
        requireThat(facets.assembly.eq(source), "foreign Facets assembly")
        val origins = facets.contributors.map(
          _.facetOrigin.getOrElse(fail("Facets contributor has no live facet provenance"))
        )
        origins.foreach { origin =>
          val i = origin.source.identity.getOrElse(fail("missing facet identity"))
          checkedOrigin(source, origin, Definition(i.traitName, i.namespaceName, i.rulesName))
        }
        Some(origins)
      case direct if isDirectRuleBook(direct) => None
      case _ => fail("non-direct Rules contributions are unsupported")

  private def walk(value: Any)(visit: Any => Unit): Unit =
    visit(value)
    value match
      case seq: Seq[?]      => seq.foreach(walk(_)(visit))
      case product: Product => product.productIterator.foreach(walk(_)(visit))
      case _                => ()

  private def patternNames(pattern: ir.Pattern): Set[String] = pattern.kind match
    case ir.Pattern.Kind.Bind(bind) =>
      bind.pattern.fold(Set.empty[String])(patternNames) + bind.name
    case ir.Pattern.Kind.Case(c)         => c.fields.flatMap(patternNames).toSet
    case ir.Pattern.Kind.Alternatives(a) => a.patterns.flatMap(patternNames).toSet
    case _                               => Set.empty

  private def children(value: ir.Expr, next: ir.Expr => ir.Expr): ir.Expr =
    import ir.Expr.Kind as E
    value.copy(kind = value.kind match
      case E.Call(c)      => E.Call(c.copy(args = c.args.map(next)))
      case E.Field(f)     => E.Field(f.copy(base = f.base.map(next)))
      case E.Construct(c) => E.Construct(c.copy(args = c.args.map(next)))
      case E.Copy(c)      =>
        E.Copy(
          c.copy(
            base = c.base.map(next),
            updates = c.updates.map(u => u.copy(value = u.value.map(next)))
          )
        )
      case E.Unary(u)  => E.Unary(u.copy(operand = u.operand.map(next)))
      case E.Binary(b) => E.Binary(b.copy(left = b.left.map(next), right = b.right.map(next)))
      case E.If(b)     =>
        E.If(
          b.copy(
            condition = b.condition.map(next),
            `then` = b.`then`.map(next),
            `else` = b.`else`.map(next)
          )
        )
      case E.Match(m) =>
        E.Match(
          m.copy(
            scrutinee = m.scrutinee.map(next),
            cases = m.cases.map(c => c.copy(guard = c.guard.map(next), body = c.body.map(next)))
          )
        )
      case E.Let(l)    => E.Let(l.copy(value = l.value.map(next), body = l.body.map(next)))
      case E.List(l)   => E.List(l.copy(items = l.items.map(next)))
      case E.Lambda(l) => E.Lambda(l.copy(body = l.body.map(next)))
      case E.Inbox(i)  =>
        E.Inbox(i.copy(contents = i.contents.map(next), message = i.message.map(next)))
      case other => other)

  private def free(value: ir.Expr): Set[String] =
    import ir.Expr.Kind as E
    value.kind match
      case E.Var(name) => Set(name)
      case E.Let(l)    =>
        l.value.fold(Set.empty[String])(free) ++ (l.body.fold(Set.empty[String])(free) - l.name)
      case E.Lambda(l) => l.body.fold(Set.empty[String])(free) -- l.params.map(_.name)
      case E.Match(m)  =>
        m.scrutinee.fold(Set.empty[String])(free) ++ m.cases.flatMap(c =>
          (c.guard.fold(Set.empty[String])(free) ++ c.body.fold(Set.empty[String])(
            free
          )) -- c.pattern.fold(Set.empty[String])(patternNames)
        )
      case _ =>
        var result = Set.empty[String]
        children(
          value,
          e =>
            result ++= free(e); e
        ): Unit
        result

  def substitute(
      value: ir.Expr,
      replacements: Map[String, ir.Expr],
      preservePositions: Set[String] = Set.empty
  ): ir.Expr =
    substituted(value, replacements, preservePositions)._1

  private def substituted(
      value: ir.Expr,
      replacements: Map[String, ir.Expr],
      preservePositions: Set[String]
  ): (ir.Expr, Vector[(ir.Call, ir.Call)]) =
    val calls = mutable.ArrayBuffer.empty[(ir.Call, ir.Call)]
    val taken = mutable.Set.from(replacements.keySet ++ replacements.values.flatMap(free))
    walk(value) {
      case e: ir.Expr if e.kind.isVar => taken += e.getVar: Unit
      case p: ir.Param                => taken += p.name: Unit
      case l: ir.Let                  => taken += l.name: Unit
      case b: ir.Bind                 => taken += b.name: Unit
      case _                          => ()
    }
    def fresh(): String =
      val name = LazyList.from(1).map(i => s"$$facetLocal$i").find(n => !taken(n)).get
      taken += name
      name
    def renamePattern(p: ir.Pattern, names: Map[String, String]): ir.Pattern =
      p.copy(kind = p.kind match
        case ir.Pattern.Kind.Bind(b) =>
          ir.Pattern.Kind.Bind(
            b.copy(
              name = names.getOrElse(b.name, b.name),
              pattern = b.pattern.map(renamePattern(_, names))
            )
          )
        case ir.Pattern.Kind.Case(c) =>
          ir.Pattern.Kind.Case(c.copy(fields = c.fields.map(renamePattern(_, names))))
        case ir.Pattern.Kind.Alternatives(a) =>
          ir.Pattern.Kind.Alternatives(a.copy(patterns = a.patterns.map(renamePattern(_, names))))
        case other => other)
    def bind(
        names: Iterable[String],
        values: Map[String, ir.Expr],
        used: Set[String]
    ): (Map[String, ir.Expr], Map[String, String]) =
      val remaining = (values -- names).filter((name, _) => used(name))
      val captures = remaining.values.flatMap(free).toSet
      (remaining, names.toVector.sorted.iterator.filter(captures).map(_ -> fresh()).toMap)
    def loop(e: ir.Expr, values: Map[String, ir.Expr], lexical: Map[String, String]): ir.Expr =
      import ir.Expr.Kind as E
      e.kind match
        case E.Var(name) =>
          values
            .get(name)
            .fold(e.copy(kind = E.Var(lexical.getOrElse(name, name))))(v =>
              if preservePositions(name) then v.copy(position = e.position) else v
            )
        case E.Call(call) =>
          val result = call.copy(args = call.args.map(loop(_, values, lexical)))
          calls += ((call, result))
          e.copy(kind = E.Call(result))
        case E.Let(l) =>
          val (remaining, names) = bind(List(l.name), values, l.body.fold(Set.empty[String])(free))
          e.copy(kind =
            E.Let(
              l.copy(
                name = names.getOrElse(l.name, l.name),
                value = l.value.map(loop(_, values, lexical)),
                body = l.body.map(loop(_, remaining, (lexical - l.name) ++ names))
              )
            )
          )
        case E.Lambda(l) =>
          val (remaining, names) =
            bind(l.params.map(_.name), values, l.body.fold(Set.empty[String])(free))
          e.copy(kind =
            E.Lambda(
              l.copy(
                params = l.params.map(p => p.copy(name = names.getOrElse(p.name, p.name))),
                body = l.body.map(loop(_, remaining, (lexical -- l.params.map(_.name)) ++ names))
              )
            )
          )
        case E.Match(m) =>
          e.copy(kind =
            E.Match(
              m.copy(
                scrutinee = m.scrutinee.map(loop(_, values, lexical)),
                cases = m.cases.map { c =>
                  val bound = c.pattern.fold(Set.empty[String])(patternNames)
                  val (remaining, names) = bind(
                    bound,
                    values,
                    c.guard.fold(Set.empty[String])(free) ++ c.body.fold(Set.empty[String])(free)
                  )
                  c.copy(
                    pattern = c.pattern.map(renamePattern(_, names)),
                    guard = c.guard.map(loop(_, remaining, (lexical -- bound) ++ names)),
                    body = c.body.map(loop(_, remaining, (lexical -- bound) ++ names))
                  )
                }
              )
            )
          )
        case _ => children(e, loop(_, values, lexical))
    val result = loop(value, replacements, Map.empty)
    result -> calls.toVector

  def renameRoot(packet: RuleBindings.Packet, names: Map[String, String]): RuleBindings.Packet =
    val replacements = names.map((from, to) => from -> ir.Expr(kind = ir.Expr.Kind.Var(to)))
    val (guard, guardCalls) = substituted(packet.guard, replacements, names.keySet)
    val (effect, effectCalls) = substituted(packet.effect, replacements, names.keySet)
    val calls = guardCalls ++ effectCalls
    val proofs = packet.curriedCalls.flatMap { proof =>
      if proof.owner.nonEmpty then Vector(proof)
      else
        val matches = calls.collect {
          case (before, after) if before.equals(proof.call) => proof.copy(call = after)
        }
        requireThat(matches.nonEmpty, "stale fragment curried-call proof")
        matches
    }
    packet.copy(guard = guard, effect = effect, curriedCalls = proofs.distinct)

  private def placed(value: ir.Expr, position: ir.Position): ir.Expr =
    children(value, placed(_, position)).copy(position = Some(position))

  private def reserved(call: ir.Call): Boolean = call.function.startsWith("$facet-context-v1.")

  def instantiate(source: RecordMachine[?, ?, ?], template: Template): RuleBindings.Packet =
    instantiate(source, template.packet, template.sites, template.definition.traitName)

  def instantiate(
      source: RecordMachine[?, ?, ?],
      packet: RuleBindings.Packet,
      sites: Vector[Site],
      fragmentOwner: String
  ): RuleBindings.Packet =
    val context =
      source.facetContextMetadata.getOrElse(fail("missing assembly projection/writer context"))
    requireThat(context.source.eq(source), "stale assembly context source")
    requireThat(
      context.owner.nonEmpty && source.getClass.getName == context.owner.replace("$.", "$"),
      "stale assembly context owner"
    )
    requireThat(
      source.headerIdentity.contains(context.family -> context.machine),
      "pending or stale assembly header"
    )
    requireThat(
      packet.params.headOption.exists(_.getType.equals(context.state)),
      "wrong template state type"
    )
    requireThat(
      context.model.copy(functions = Nil, types = Nil).equals(ir.Model()),
      "unrelated assembly context declarations"
    )
    requireThat(sites.distinct.size == sites.size, "duplicate contextual sites")
    val roots = Set(context.projection, context.writer).filter(_.nonEmpty)
    requireThat(
      context.projection.isEmpty || context.projection == s"$$facet-context-v1.${context.owner}.projection",
      "stale assembly projection root identity"
    )
    requireThat(
      context.writer.isEmpty || context.writer == s"$$facet-context-v1.${context.owner}.writer",
      "stale assembly writer root identity"
    )
    requireThat(
      context.projection.isEmpty || context.projection != context.writer,
      "conflicting context roots"
    )
    val seen = mutable.Set.empty[Site]
    val additions = mutable.ArrayBuffer.empty[CoreBindings.CurriedCall]
    val originalFunctions = packet.model.functions.map(f => f.name -> f).toMap
    val contextFunctions = context.model.functions.map(f => f.name -> f).toMap
    requireThat(
      contextFunctions.size == context.model.functions.size,
      "conflicting assembly context definitions"
    )
    val reachedContext = mutable.Set.empty[String]
    def contextFunction(name: String): Unit =
      if name.nonEmpty && reachedContext.add(name) then
        val function = contextFunctions.getOrElse(name, fail("missing assembly context dependency"))
        walk(function) {
          case call: ir.Call => contextFunction(call.function)
          case _             => ()
        }
    sites.foreach(s =>
      contextFunction(if s.kind == "read" then context.projection else context.writer)
    )
    val selectedContext = context.model.functions.filter(f => reachedContext(f.name))
    val selectedProofs = context.proofs.filter(p => p.owner.exists(reachedContext))
    if reachedContext.nonEmpty then
      CoreBindings.dependencies(
        Nil,
        reachedContext.filter(roots).toVector,
        selectedContext,
        context.model.types,
        "Machine Facets assembly context",
        selectedProofs
      ): Unit
    def projection(site: Site): ir.Function =
      val read = site.kind == "read"
      requireThat(read || site.kind == "write", "unknown contextual site kind")
      requireThat(
        reserved(site.call) && site.call.function.endsWith("." + site.kind),
        "wrong contextual call slot"
      )
      if site.owner.isEmpty then
        requireThat(
          site.call.function == s"$$facet-context-v1.$fragmentOwner.${site.kind}",
          "foreign fragment contextual owner"
        )
      requireThat(
        site.state.equals(context.state) && context.phase.exists(_.equals(site.phase)),
        "wrong contextual state/phase type"
      )
      requireThat(
        site.result.equals(if read then site.phase else site.state),
        "wrong contextual result type"
      )
      val name = if read then context.projection else context.writer
      requireThat(
        name.nonEmpty,
        s"missing concrete phase ${if read then "projection" else "writer"}"
      )
      val function = contextFunctions.getOrElse(name, fail("missing assembly context root"))
      val expected = if read then Vector(site.state) else Vector(site.state, site.phase)
      requireThat(
        function.params.map(_.getType).equals(expected) && site.call.args.size == expected.size,
        "wrong contextual parameter slot/type/arity"
      )
      requireThat(
        (if read then context.projectionResult else context.writerResult)
          .exists(_.equals(site.result)),
        "wrong assembly context result type"
      )
      requireThat(
        function.body.nonEmpty && function.requires.isEmpty,
        "missing or contracted assembly context body"
      )
      requireThat(
        function.params.map(_.name).distinct.size == function.params.size && function.params.forall(
          _.name.nonEmpty
        ),
        "invalid context parameters"
      )
      requireThat(
        free(function.getBody).subsetOf(function.params.map(_.name).toSet),
        "free assembly context variable"
      )
      function
    val rewrittenCalls = mutable.ArrayBuffer.empty[(Option[String], ir.Call, ir.Call)]
    def transform(value: ir.Expr, owner: Option[String]): ir.Expr =
      value.kind match
        case ir.Expr.Kind.Call(call) if reserved(call) =>
          val site = sites
            .find(s => s.owner == owner && s.call.equals(call))
            .getOrElse(fail("missing exact contextual site"))
          val function = projection(site)
          seen += site: Unit
          val placement = site.placement.map { definition =>
            requireThat(
              site.kind == "read" && (definition.equals(ir.Position.defaultInstance) ||
                (definition.file.nonEmpty && definition.line > 0)),
              "malformed contextual heading placement"
            )
            val position = function.position.getOrElse(fail("missing concrete heading position"))
            requireThat(
              position.equals(ir.Position.defaultInstance) ||
                (position.file.nonEmpty && position.line > 0),
              "malformed concrete heading position"
            )
            position
          }
          val body = placement.fold(function.getBody)(placed(function.getBody, _))
          val args = call.args.map { argument =>
            val instantiated = transform(argument, owner)
            placement.fold(instantiated)(placed(instantiated, _))
          }
          val replacements = function.params.map(_.name).zip(args).toMap
          val preserve =
            if site.kind == "write" && owner.isEmpty && args.headOption.exists(_.kind.isVar)
            then Set(function.params.head.name)
            else Set.empty[String]
          val (instantiated, calls) = substituted(body, replacements, preserve)
          selectedProofs.filter(_.owner.contains(function.name)).foreach { proof =>
            requireThat(
              proof.function.equals(
                contextFunctions.getOrElse(proof.function.name, fail("stale context callee proof"))
              ),
              "stale context callee proof"
            )
            val expression = ir.Expr(kind = ir.Expr.Kind.Call(proof.call))
            val at = placement.fold(expression)(placed(expression, _))
            val matches = calls.collect {
              case (before, after) if before.equals(at.getCall) =>
                proof.copy(owner = owner, call = after)
            }
            requireThat(matches.nonEmpty, "stale context root curried-call proof")
            additions ++= matches
          }
          instantiated
        case ir.Expr.Kind.Call(call) =>
          val result = call.copy(args = call.args.map(transform(_, owner)))
          rewrittenCalls += ((owner, call, result))
          value.copy(kind = ir.Expr.Kind.Call(result))
        case _ => children(value, transform(_, owner))
    val functions = packet.model.functions.map(f =>
      f.copy(
        body = f.body.map(transform(_, Some(f.name))),
        requires = f.requires.map(transform(_, Some(f.name)))
      )
    )
    val guard = transform(packet.guard, None)
    val effect = transform(packet.effect, None)
    requireThat(sites.forall(seen), "stale contextual site owner or Call")
    val allFunctions = functions ++ selectedContext.filterNot(f => roots(f.name))
    val byName = allFunctions.map(f => f.name -> f).toMap
    val proofs = packet.curriedCalls.flatMap { proof =>
      requireThat(
        originalFunctions.get(proof.function.name).exists(_.equals(proof.function)),
        "stale facet callee proof"
      )
      val matches = rewrittenCalls.collect {
        case (owner, before, after) if owner == proof.owner && before.equals(proof.call) =>
          proof.copy(
            call = after,
            function =
              byName.getOrElse(proof.function.name, fail("missing instantiated proof callee"))
          )
      }
      requireThat(matches.nonEmpty, "stale facet curried-call proof owner or Call")
      matches
    } ++ selectedProofs.filterNot(p => p.owner.exists(roots)) ++ additions
    packet.copy(
      owner = context.owner,
      family = context.family,
      machine = context.machine,
      guard = guard,
      effect = effect,
      model = ir.Model(functions = allFunctions, types = packet.model.types ++ context.model.types),
      typePositions = packet.typePositions ++ context.positions,
      curriedCalls = proofs
        .map(p =>
          p.copy(function = byName.getOrElse(p.function.name, fail("missing context proof callee")))
        )
        .distinct
        .toVector
    )
