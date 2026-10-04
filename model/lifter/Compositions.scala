package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Compositions:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Compositions

  /** `s.field -> value`, the body of a selector. */
  def arrow(t: Term): (Term, Term) = t match
    case Apply(
          TypeApply(Select(Apply(TypeApply(Ident("ArrowAssoc"), _), List(k)), "->"), _),
          List(v)
        ) =>
      (k, v)
    case other => fail(other, s"expected `_.member -> value`, not ${other.show}")

  /** The parameter and body of a selector of the composed state, such as `_.queue -> enqueue`. */
  def selector(t: Term): Option[(ValDef, Term)] = lambda(t).collect { case (List(p), body) =>
    (p, body)
  }

  /**
   * The field of the composed state a selector's `key` reads, `_.queue`: one field, read off the
   * selector's parameter, refused at `at` otherwise.
   */
  def selectedField(param: ValDef, key: Term, at: Tree, what: String): String =
    fieldPath(param, key) match
      case Some(List(field)) => field
      case path              =>
        val state = param.tpt.tpe.widen.dealias.typeSymbol
        val example = state.caseFields.headOption.fold("field")(_.name)
        // The compiler names the parameter of `_.field`, so a path is shown as it was written.
        val written = path.fold(key.show)(fields => ("_" +: fields).mkString("."))
        fail(
          at,
          s"$what names a member by one field of ${state.name}, such as `_.$example`, not `$written`"
        )

  /** The member of `c` that fills `field`, refused at `at` where none does. */
  def filled(c: ir.Composition, field: String, at: Tree, what: String): ir.Member =
    c.members
      .find(_.field == field)
      .getOrElse(
        fail(
          at,
          s"$what names $field, which no member of ${c.name} fills: a selector names a member of " +
            "the composition"
        )
      )

  /**
   * The machine a selector puts in the field its `key` reads, refused at `at` where the machine's
   * state is not that field's type.
   */
  def memberMachine(field: String, key: Term, value: Term, at: Tree, of: String): ir.Machine =
    val m = machineOf(resolveSymbol(value), value)
    val state = typeRef(key.tpe.widen, key).getNamed
    if m.stateType != state then
      fail(
        at,
        s"$field of $of holds a $state, and ${m.name} is a machine of ${m.stateType}: a member is a " +
          "machine of its field's type"
      )
    m

  /** The action of `name` a machine binds, by its Definition ID. */
  def boundNamed(machine: String, name: String): Option[String] =
    machineNamed(machine).flatMap(
      _.steps.map(_.action).find(id => actions.get(id).exists(_.name == name))
    )

  /** Whether a machine binds the action of the Definition ID `id`: the declaration, not its name. */
  def binds(machine: String, id: String): Boolean =
    machineNamed(machine).exists(_.steps.exists(_.action == id))

  def compositionOf(sym: Symbol, at: Tree): ir.Composition =
    compositions.get(sym.fullName) match
      case Some(c) => c
      case None    =>
        if composing(sym) then
          fail(
            at,
            s"${sym.name} is derived from itself, through ${composing.map(_.name).mkString(", ")}: " +
              "a composition derives from a composition declared without it"
          )
        composing += sym
        val c =
          try composition(sym, valDef(sym, at, "a composition").rhs.get)
          finally composing -= sym
        distinctModelName(c.name, sym, c.getPosition)
        compositions(sym.fullName) = c
        c

  // The compositions whose declarations are being lifted, so one derived from itself is refused.
  private val composing = mutable.LinkedHashSet.empty[Symbol]

  /**
   * A composition, from `compose[S](members*)` named after its val `sym` in the given family, with
   * each member by a selector of its field; or another composition with one member replaced,
   * `c.withMember(_.field -> machine)`; and the syncs, ends and replacements chained onto it.
   */
  def composition(sym: Symbol, rhs: Term): ir.Composition =
    def declared(s: TypeTree, family: String, name: String, members: Term, t: Term) =
      ir.Composition(
        position = Some(pos(rhs)),
        family = family,
        name = name,
        stateType = typeRef(s.tpe, t).getNamed,
        members = varargs(members).map { m =>
          val (param, body) = selector(m).getOrElse(
            fail(m, s"compose names a member by a selector, `_.member -> machine`, not ${m.show}")
          )
          val (key, value) = arrow(body)
          val field = selectedField(param, key, m, "compose")
          ir.Member(field, memberMachine(field, key, value, m, name).name)
        }
      )
    // A typed move names its member by a selector and the very action the member binds.
    def move(c: ir.Composition, sync: String, t: Term): ir.SyncMove =
      val (param, body) = selector(t).getOrElse(
        fail(t, s"sync names a member by a selector, `_.member -> action`, not ${t.show}")
      )
      val (key, a) = arrow(body)
      val field = selectedField(param, key, t, s"sync $sync")
      val member = filled(c, field, t, s"sync $sync")
      val id = action(a)
      if !binds(member.machine, id) then
        fail(
          t,
          s"sync $sync pairs ${actions(id).name} of $field, and ${member.machine} binds no action " +
            s"of $id: a sync pairs the action its member binds, not one of another declaration"
        )
      ir.SyncMove(field, actions(id).name)
    // A sync named `n`, or after its first member's action, refused where another has its name.
    def sync(c: ir.Composition, n: String, first: Term, second: Term, at: Tree) =
      for s <- c.syncs.find(_.name == n) do
        fail(
          at,
          s"${c.name} pairs two syncs named $n, and the IR keys a sync by its name: a sync is " +
            "named after its first member's action unless it names itself, `.sync(\"name\", ...)`"
        )
      c.addSyncs(ir.Sync(n, Some(move(c, n, first)), Some(move(c, n, second))))
    def walk(t: Term): ir.Composition = t match
      case Apply(Select(inner, "sync"), List(name, first, second)) =>
        sync(walk(inner), constString(name), first, second, name)
      // Named after the first member's action, as its declaration names it.
      case Apply(Select(inner, "sync"), List(first, second)) =>
        val (_, body) = selector(first).getOrElse(
          fail(first, s"sync names a member by a selector, `_.member -> action`, not ${first.show}")
        )
        sync(walk(inner), actions(action(arrow(body)._2)).name, first, second, first)
      case Apply(Select(inner, "ends"), List(p))                 => walk(inner).withEnds(lift(p))
      case Apply(Select(inner, "replaces"), List(field, opaque)) =>
        val c = walk(inner)
        val replaced = machineOf(resolveSymbol(opaque), opaque).name
        val (param, body) = selector(field).getOrElse(
          fail(field, s"replaces names a member by a selector, `_.member`, not ${field.show}")
        )
        replaces(c, selectedField(param, body, field, "replaces"), replaced, field)
      case Apply(Apply(Select(inner, "withMember"), List(member)), List(family)) =>
        val base = inner match
          case r: Ref => compositionOf(resolveSymbol(r), r)
          case chain  => walk(chain)
        withMember(base, member).copy(
          family = constString(family),
          name = capturedName(sym, rhs, "a composition"),
          position = Some(pos(rhs))
        )
      case Apply(Apply(TypeApply(Ident("compose"), List(s)), List(members)), List(_, family)) =>
        declared(s, constString(family), capturedName(sym, rhs, "a composition"), members, t)
      case other => fail(other, s"not a part of a composition declaration: ${other.show}")
    val c = walk(rhs)
    val fields = c.members.map(_.field)
    for s <- c.syncs; m <- Seq(s.getFirst, s.getSecond) if !fields.contains(m.member) do
      fail(rhs, s"sync ${s.name} names ${m.member}, which is not a member of ${c.name}")
    c

  /** `c` with the member of `field` standing in for `opaque`, which its machine must refine. */
  def replaces(c: ir.Composition, field: String, opaque: String, at: Tree): ir.Composition =
    val i = c.members.indexWhere(_.field == field)
    if i < 0 then fail(at, s"$field replaces $opaque, and no member fills $field")
    val member = c.members(i).machine
    if !machineNamed(member).exists(_.getRefines.product == opaque) then
      fail(at, s"$field replaces $opaque, and its member $member does not refine $opaque")
    c.withMembers(c.members.updated(i, c.members(i).withReplaces(opaque)))

  /**
   * `base` with the member its selector names replaced in place by the selector's machine. That
   * machine binds the very actions the syncs of the member pair, and where the member stood in for
   * a machine, the new one stands in for the machine it declares it refines, which a provider over
   * another interface names differently.
   */
  def withMember(base: ir.Composition, sel: Term): ir.Composition =
    val (param, body) = selector(sel).getOrElse(
      fail(sel, s"withMember names a member by a selector, `_.member -> machine`, not ${sel.show}")
    )
    val (key, value) = arrow(body)
    val field = selectedField(param, key, sel, "withMember")
    val was = filled(base, field, sel, "withMember")
    val m = memberMachine(field, key, value, sel, base.name)
    for s <- base.syncs; mv <- Seq(s.getFirst, s.getSecond) if mv.member == field do
      (boundNamed(was.machine, mv.action), boundNamed(m.name, mv.action)) match
        case (_, None) =>
          fail(
            sel,
            s"${m.name} binds no action ${mv.action}, which sync ${s.name} of ${base.name} pairs " +
              s"for $field: a member's new machine binds the actions its syncs pair"
          )
        case (Some(id), Some(other)) if id != other =>
          fail(
            sel,
            s"${m.name} binds $other, and sync ${s.name} of ${base.name} pairs $id for $field: a " +
              "member's new machine binds the very actions its syncs pair, not ones spelled alike"
          )
        case _ => ()
    val replaces =
      if was.replaces.isEmpty then ""
      else
        m.getRefines.product match
          case "" =>
            fail(
              sel,
              s"$field of ${base.name} stands in for ${was.replaces}, and ${m.name} refines no " +
                s"machine: declare the machine it stands in for with `refines` in ${m.name}"
            )
          case product => product
    base.withMembers(
      base.members.map(mb => if mb.field == field then ir.Member(field, m.name, replaces) else mb)
    )

  // ### Composed classes: what `c.synced(...)` and `c.own(...)` select

  /** The receiver, the operation and the arguments of `c.synced(...)` or `c.own(...)`. */
  private def selection(t: Term): Option[(Term, String, List[Term])] = t match
    case Typed(e, _)        => selection(e)
    case Inlined(_, Nil, e) => selection(e)
    case Apply(Select(c, op @ ("synced" | "own")), args)
        if t.symbol.maybeOwner.fullName == "umpire.Composition" =>
      Some((c, op, args))
    case _ => None

  /** Whether a term selects a composed class or action: `c.synced(...)` or `c.own(...)`. */
  def composed(t: Term): Boolean = selection(t).isDefined

  /** The composed action key `whenAction(c.synced(...))` or `whenAction(c.own(...))` names. */
  def composedAction(t: Term, composition: String, env: Map[Symbol, Decl]): String =
    composedKey(t, composition, env, classes = false)

  /**
   * A Scenario's pinned schedule: a machine's classes, or a composition's composed class keys, each
   * selected by `c.synced(...)` or `c.own(...)`.
   */
  def scheduled(s: ir.Scenario, items: List[Term], env: Map[Symbol, Decl]): ir.Scenario =
    items.filter(composed) match
      case Nil        => s.addAllActions(items.map(classOf))
      case first :: _ =>
        if machineNamed(s.machine).nonEmpty then
          fail(
            first,
            s"Scenario ${s.name} of the machine ${s.machine} lists a composed class: a machine's " +
              "Scenario lists its own classes"
          )
        for plain <- items.find(!composed(_)) do
          fail(
            plain,
            s"Scenario ${s.name} of ${s.machine} lists ${actions(classOf(plain).action).name} beside " +
              "composed classes: a composition's Scenario selects each of its classes with " +
              "`.synced(_.member -> action)` or `.own(_.member, action)`"
          )
        s.addAllKeys(items.map(composedKey(_, s.machine, env, classes = true)))

  /**
   * The composed key `t` selects in the composition `of`, as SEMANTICS.md keys it: for a Scenario
   * (`classes`) the class, a sync's name or `<field>_<action>` followed by the class's inputs; for
   * `whenAction` the action, its name alone.
   */
  def composedKey(t: Term, of: String, env: Map[Symbol, Decl], classes: Boolean): String =
    val (receiver, op, args) = selection(t).get
    val name = modelName(fold(receiver, env), receiver)
    if name != of then
      if classes then
        fail(
          t,
          s"a Scenario of $of lists a class of $name: a Scenario selects the classes of the " +
            "composition it is declared on"
        )
      else
        fail(
          t,
          s"whenAction of a Property of $of names an action of $name: a Property names the actions " +
            "of the composition it is declared on"
        )
    val c = compositions.values
      .find(_.name == name)
      .getOrElse(fail(receiver, s"$name is no lifted composition"))
    // The class a member's action names, refused where it is not one of the member's own actions.
    def member(
        sel: Term,
        key: Term,
        param: ValDef,
        value: Term
    ): (String, ir.ActionClass, ir.Action) =
      val field = selectedField(param, key, sel, op)
      val m = filled(c, field, t, op)
      val cls = classOf(value)
      val a = actions(cls.action)
      if !binds(m.machine, cls.action) then
        fail(
          t,
          s"$op names ${a.name} of $field, and ${m.machine} binds no action of ${cls.action}: name " +
            s"the action $field binds, not one of another declaration spelled alike"
        )
      if classes && cls.inputs.size != a.inputs.size then
        fail(
          t,
          s"$op names the action ${a.name}, which takes inputs, and a Scenario lists classes: name " +
            s"one with `${a.name}(...)`"
        )
      if !classes && cls.inputs.nonEmpty then
        fail(
          t,
          s"whenAction names every class of an action, and $op names a class of ${a.name}: name the " +
            s"action ${a.name} without its inputs"
        )
      (field, cls, a)
    def inputs(cls: ir.ActionClass): String =
      if classes then cls.inputs.map("-" + valueKey(_)).mkString else ""
    def pairs(s: ir.Sync, field: String, action: String): Boolean =
      Seq(s.getFirst, s.getSecond).exists(mv => mv.member == field && mv.action == action)
    op match
      case "synced" =>
        val sel = args.head
        val (param, body) = selector(sel).getOrElse(fail(sel, "synced names `_.member -> action`"))
        val (key, value) = arrow(body)
        val (field, cls, a) = member(sel, key, param, value)
        c.syncs.filter(pairs(_, field, a.name)) match
          case Seq(s) =>
            val other = if s.getFirst.member == field then s.getSecond else s.getFirst
            val takes = boundNamed(filled(c, other.member, t, op).machine, other.action)
              .flatMap(actions.get)
              .exists(_.inputs.nonEmpty)
            if classes && takes then
              fail(
                t,
                s"sync ${s.name} pairs ${a.name} of $field with ${other.action} of ${other.member}, " +
                  "which takes inputs `synced` cannot name from one side"
              )
            s.name + inputs(cls)
          case Seq() =>
            fail(
              t,
              s"no sync of ${c.name} pairs ${a.name} of $field: name the step $field takes alone " +
                s"with `.own(_.$field, ${a.name})`"
            )
          case many =>
            fail(
              t,
              s"syncs ${many.map(_.name).mkString(", ")} of ${c.name} each pair ${a.name} of " +
                s"$field, and synced names one: a sync is selected by a member action it alone pairs"
            )
      case _ =>
        val (sel, value) = (args.head, args.last)
        val (param, body) = selector(sel).getOrElse(fail(sel, "own names `_.member`"))
        val (field, cls, a) = member(sel, body, param, value)
        for s <- c.syncs.find(pairs(_, field, a.name)) do
          fail(
            t,
            s"sync ${s.name} pairs ${a.name} of $field, which steps only with its pair: name it " +
              s"with `.synced(_.$field -> ${a.name})`"
          )
        s"${field}_${a.name}" + inputs(cls)
