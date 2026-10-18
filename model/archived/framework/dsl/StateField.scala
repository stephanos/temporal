package framework.dsl

import scala.deriving.Mirror
import scala.quoted.*

final private[framework] class StateField[S, A](
    val ownerType: String,
    val fieldType: String,
    val path: List[String],
    val location: String,
    val rootFields: Vector[List[String]],
    validity: A => Option[Boolean],
    getter: S => A,
    writer: (S, A) => S,
    rebuilder: (S, Vector[(List[String], Any)]) => S
):
  def get(state: S): A = NativeRuntime.installed.invoke(getter, state)
  def write(state: S, value: A): S = NativeRuntime.installed.invoke(writer, state, value)
  def rebuild(state: S, replacements: Vector[(List[String], Any)]): S =
    if replacements.isEmpty then state
    else NativeRuntime.installed.invoke(rebuilder, state, replacements)
  def valid(value: A): Option[Boolean] = NativeRuntime.installed.invoke(validity, value)

@scala.annotation.publicInBinary
private[framework] object StateField:
  inline def capture[S, A](inline selector: S => A): StateField[S, A] = ${ captured('selector) }

  def make[S, A](
      ownerType: String,
      fieldType: String,
      path: List[String],
      location: String,
      rootFields: Vector[List[String]],
      validity: A => Option[Boolean],
      getter: S => A,
      writer: (S, A) => S,
      rebuilder: (S, Vector[(List[String], Any)]) => S
  ): StateField[S, A] =
    new StateField(
      ownerType,
      fieldType,
      path,
      location,
      rootFields,
      validity,
      getter,
      writer,
      rebuilder
    )

  def replace[S](state: S, index: Int, value: Any)(using mirror: Mirror.ProductOf[S]): S =
    NativeRuntime.installed.replace(state, index, value, mirror)

  private def validator[T: Type](using Quotes): Option[Expr[T => Boolean]] =
    import quotes.reflect.*
    def defaultBounded(term: Term): Boolean = term match
      case Inlined(_, _, body)    => defaultBounded(body)
      case Apply(function, _)     => defaultBounded(function)
      case TypeApply(function, _) => defaultBounded(function)
      case other => other.symbol.fullName == "framework.dsl.Finite.given_Finite_UpTo"
    def bounded: Option[Expr[T => Boolean]] = Type.of[T] match
      case '[UpTo[n]] =>
        TypeRepr.of[n] match
          case ConstantType(IntConstant(high)) =>
            Some('{ (value: UpTo[n]) =>
              value >= 0 && value <= ${ Expr(high) }
            }.asExprOf[T => Boolean])
          case _ => None
      case _ => None
    Expr.summon[Finite[T]] match
      case Some(finite) if defaultBounded(finite.asTerm) && bounded.nonEmpty => bounded
      case Some(finite)                                                      =>
        Some('{ (value: T) =>
          $finite.values.exists(candidate => $finite.equivalent(candidate, value))
        })
      case None =>
        Type.of[T] match
          case '[Int]       => Some('{ (_: Int) => true }.asExprOf[T => Boolean])
          case '[Boolean]   => Some('{ (_: Boolean) => true }.asExprOf[T => Boolean])
          case '[String]    => Some('{ (value: String) => value != null }.asExprOf[T => Boolean])
          case '[Option[a]] =>
            validator[a].map(check =>
              '{ (value: Option[a]) => value.forall($check) }.asExprOf[T => Boolean]
            )
          case '[List[a]] =>
            validator[a].map(check =>
              '{ (value: List[a]) => value.forall($check) }.asExprOf[T => Boolean]
            )
          case '[Set[a]] =>
            validator[a].map(check =>
              '{ (value: Set[a]) => value.forall($check) }.asExprOf[T => Boolean]
            )
          case '[Map[k, v]] =>
            for key <- validator[k]; value <- validator[v] yield '{ (entries: Map[k, v]) =>
              entries.forall((k, v) => $key(k) && $value(v))
            }.asExprOf[T => Boolean]
          case _ =>
            val record = TypeRepr.of[T].dealias
            if !record.typeSymbol.flags.is(Flags.Case) then None
            else
              val checks = record.typeSymbol.caseFields.map: field =>
                record.memberType(field).asType match
                  case '[member] => validator[member].map(field -> _)
              if checks.exists(_.isEmpty) then None
              else
                def checked(value: Expr[T]): Expr[Boolean] =
                  checks.flatten.foldLeft(Expr(true)): (previous, entry) =>
                    val (field, check) = entry
                    record.memberType(field).asType match
                      case '[member] =>
                        val typed = check.asExprOf[member => Boolean]
                        val selected = Select.unique(value.asTerm, field.name).asExprOf[member]
                        '{ $previous && $typed($selected) }
                Some('{ (value: T) => ${ checked('value) } })

  // The state var a durability classification names, `durable(p1)`, as a field of its Machine's
  // State: its path is the var's name.
  inline def captureVar[A](
      inline variable: A,
      owner: MachineDeclaration[State, ?, ?]
  ): StateField[State, A] = ${ capturedVar[A]('variable, 'owner) }

  private[framework] def capturedVar[A: Type](
      variable: Expr[A],
      owner: Expr[MachineDeclaration[State, ?, ?]]
  )(using Quotes): Expr[StateField[State, A]] =
    import quotes.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, _, body) => unwrap(body)
      case Typed(body, _)      => unwrap(body)
      case other               => other
    def selected(term: Term): (Ref, List[String]) = unwrap(term) match
      case reference: Ref
          if reference.symbol.flags.is(Flags.Mutable) && reference.symbol.owner.isClassDef =>
        (reference, Nil)
      case Select(receiver, name)
          if receiver.tpe.widen.dealias.typeSymbol.flags.is(Flags.Case) &&
            receiver.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        val (root, path) = selected(receiver)
        (root, path :+ name)
      case _ => report.errorAndAbort("a state field names a state var of its Machine", variable)
    val (root, path) = selected(variable.asTerm)
    val name = root.symbol.name
    val receiver = root match
      case Select(value, _) => value.asExprOf[Model]
      case _                => This(root.symbol.owner).asExprOf[Model]
    val at = Position.ofMacroExpansion
    val location = s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"
    val field = if path.isEmpty then
      '{
        stateVar[A](
          $owner,
          ${ Expr(name) },
          ${ Expr(TypeRepr.of[A].show) },
          ${ Expr(location) },
          ${ validity[A] }
        )
      }
    else
      root.tpe.widen.asType match
        case '[r] =>
          val selector = Lambda(
            Symbol.spliceOwner,
            MethodType(List("value"))(_ => List(TypeRepr.of[r]), _ => TypeRepr.of[A]),
            (_, arguments) =>
              path.foldLeft(arguments.head.asInstanceOf[Term])((value, name) =>
                Select.unique(value, name)
              )
          ).asExprOf[r => A]
          '{ nestedVar[r, A]($owner, ${ Expr(name) }, ${ captured[r, A](selector) }) }
    '{
      require(Vars.of($receiver) eq Vars.of($owner), "a state field belongs to another Machine")
      $field
    }

  def nestedVar[R, A](
      owner: MachineDeclaration[State, ?, ?],
      name: String,
      nested: StateField[R, A]
  ): StateField[State, A] =
    val vars = Vars.of(owner)
    val index = vars.schema.indexOf(name)
    require(index >= 0, s"$name is not a state var of ${owner.name}")
    def root(state: State): R =
      vars.owned(state).values(index).asInstanceOf[R] // scalafix:ok DisableSyntax.asInstanceOf
    def updated(state: State, replacements: Vector[(List[String], Any)]): State =
      val changes = replacements.collect {
        case (head :: tail, value) if head == name => tail -> value
      }
      val replacement =
        changes.find(_._1.isEmpty).map(_._2).getOrElse(nested.rebuild(root(state), changes))
      vars.updated(state, Vector(name -> replacement))
    make[State, A](
      "framework.dsl.State",
      nested.fieldType,
      name :: nested.path,
      nested.location,
      vars.schema.vars.flatMap(variable =>
        if variable.name == name then nested.rootFields.map(name :: _)
        else Vector(List(variable.name))
      ),
      nested.valid,
      state => nested.get(root(state)),
      (state, value) => updated(state, Vector((name :: nested.path) -> value)),
      updated
    )

  def stateVar[A](
      owner: MachineDeclaration[State, ?, ?],
      name: String,
      fieldType: String,
      location: String,
      validity: A => Option[Boolean]
  ): StateField[State, A] =
    val vars = Vars.of(owner)
    val index = vars.schema.indexOf(name)
    require(index >= 0, s"$name is not a state var of ${owner.name}")
    def updated(state: State, replacements: Vector[(List[String], Any)]): State =
      vars.updated(
        state,
        replacements.map:
          case (List(selected), value) => selected -> value
          case (path, _)               =>
            throw IllegalArgumentException(
              s"durability field ${path.mkString(".")} at $location: a Machine classifies whole vars"
            )
      )
    make[State, A](
      "framework.dsl.State",
      fieldType,
      List(name),
      location,
      vars.schema.vars.map(v => List(v.name)),
      validity,
      state =>
        vars.owned(state).values(index).asInstanceOf[A], // scalafix:ok DisableSyntax.asInstanceOf
      (state, value) => updated(state, Vector(List(name) -> value)),
      updated
    )

  private def validity[A: Type](using Quotes): Expr[A => Option[Boolean]] = validator[A] match
    case Some(check) => '{ $check.andThen[Option[Boolean]](value => Some(value)) }
    case None        => '{ (_: A) => None }

  private[framework] def captured[S: Type, A: Type](selector: Expr[S => A])(using
      Quotes
  ): Expr[StateField[S, A]] =
    import quotes.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, _, body) => unwrap(body)
      case Typed(body, _)      => unwrap(body)
      case other               => other
    def fields(term: Term, root: Symbol): Option[List[String]] = unwrap(term) match
      case parameter: Ident if parameter.symbol.equals(root) => Some(Nil)
      case Select(receiver, name)
          if receiver.tpe.widen.dealias.typeSymbol.flags.is(Flags.Case) &&
            receiver.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        fields(receiver, root).map(_ :+ name)
      case _ => None
    val (path, selectedType) = unwrap(selector.asTerm) match
      case Lambda(List(parameter), body)
          if TypeRepr.of[S].dealias.typeSymbol.flags.is(Flags.Case) =>
        fields(body, parameter.symbol)
          .filter(_.nonEmpty)
          .map(_ -> unwrap(body).tpe.widen)
          .getOrElse:
            report.errorAndAbort(
              "a state field requires a rooted case-class field selector",
              selector
            )
      case _ =>
        report.errorAndAbort("a state field requires a rooted case-class field selector", selector)
    if !(selectedType =:= TypeRepr.of[A]) then
      report.errorAndAbort("a writable state field requires its exact field type", selector)

    def updated[T: Type](state: Expr[T], names: List[String], value: Expr[A]): Expr[T] =
      val owner = TypeRepr.of[T].dealias
      val members = owner.typeSymbol.caseFields
      val index = members.indexWhere(_.name == names.head)
      val field = members(index)
      val mirror = Expr
        .summon[Mirror.ProductOf[T]]
        .getOrElse:
          report.errorAndAbort(
            "a writable case-class field selector requires a product mirror",
            selector
          )
      val replacement =
        if names.tail.isEmpty then value.asExprOf[Any]
        else
          owner.memberType(field).asType match
            case '[nested] =>
              val current = Select.unique(state.asTerm, field.name).asExprOf[nested]
              updated[nested](current, names.tail, value).asExprOf[Any]
      '{ replace[T]($state, ${ Expr(index) }, $replacement)(using $mirror) }

    val writer = '{ (state: S, value: A) => ${ updated[S]('state, path, 'value) } }
    def rebuilt[T: Type](
        state: Expr[T],
        replacements: Expr[Vector[(List[String], Any)]],
        seen: List[TypeRepr]
    ): Expr[T] =
      val record = TypeRepr.of[T].dealias
      val mirror = Expr
        .summon[Mirror.ProductOf[T]]
        .getOrElse:
          report.errorAndAbort(
            "a writable case-class field selector requires a product mirror",
            selector
          )
      val values = record.typeSymbol.caseFields.map: field =>
        record.memberType(field).asType match
          case '[member] =>
            val current = Select.unique(state.asTerm, field.name).asExprOf[member]
            val matching = '{
              $replacements.collect {
                case (head :: tail, value) if head == ${ Expr(field.name) } => (tail, value)
              }
            }
            def fallback(changes: Expr[Vector[(List[String], Any)]]): Expr[Any] =
              val child = TypeRepr.of[member].dealias
              if child.typeSymbol.flags.is(Flags.Case) && !seen.exists(_ =:= child) then
                '{
                  if $changes.isEmpty then $current
                  else ${ rebuilt[member](current, changes, record :: seen) }
                }
              else current
            '{
              val changes = $matching
              changes.find(_._1.isEmpty).map(_._2).getOrElse(${ fallback('changes) })
            }
      '{ $mirror.fromProduct(Tuple.fromArray(Array[Any](${ Varargs(values) }*))) }
    val rebuilder = '{ (state: S, replacements: Vector[(List[String], Any)]) =>
      ${ rebuilt[S]('state, 'replacements, Nil) }
    }
    def leaves(owner: TypeRepr, prefix: List[String], seen: List[TypeRepr]): List[List[String]] =
      val record = owner.dealias
      if record.typeSymbol.flags.is(Flags.Case) && record.typeSymbol.caseFields.nonEmpty &&
        !seen.exists(_ =:= record)
      then
        record.typeSymbol.caseFields.flatMap: field =>
          leaves(record.memberType(field), prefix :+ field.name, record :: seen)
      else List(prefix)

    val rootFields =
      Expr.ofList(leaves(TypeRepr.of[S], Nil, Nil).map(p => Expr.ofList(p.map(Expr(_)))))
    val at = Position.ofMacroExpansion
    val location = s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"
    '{
      make[S, A](
        ${ Expr(TypeRepr.of[S].show) },
        ${ Expr(selectedType.show) },
        ${ Expr.ofList(path.map(Expr(_))) },
        ${ Expr(location) },
        $rootFields.toVector,
        ${ validity[A] },
        $selector,
        $writer,
        $rebuilder
      )
    }
