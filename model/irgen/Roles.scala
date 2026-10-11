package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Pattern.Kind as P

// The roles the cases of one phase enum take (model/framework/Roles.scala), by class names alone, so
// that it reads no TASTy: for every role a case carries, the cases that have it, inheritance
// included, in the enum's declaration order, and the cases whose roles conflict. A role is a
// framework role or a trait that extends one; a Model's own role takes part in a conflict only
// through the framework roles it extends.
private[irgen] object Roles:
  private val live = "framework.Live"
  private val closed = "framework.Closed"

  // The framework's roles, in the order model/framework/Roles.scala declares them.
  val framework: Seq[String] =
    Seq(
      "Live",
      "Waiting",
      "Retrying",
      "Held",
      "Suspended",
      "Closed",
      "Succeeded",
      "Failed",
      "Canceled",
      "Terminated",
      "TimedOut"
    ).map("framework." + _)

  // The roles a phase has at most one of, each with what the refusal says of them.
  private val exclusive: Seq[(Seq[String], String)] = Seq(
    Seq(live, closed) -> "a phase is live or closed, never both",
    Seq("Waiting", "Held", "Suspended").map("framework." + _) ->
      "a live phase is at most one of Waiting, Held and Suspended",
    Seq("Succeeded", "Failed", "Canceled", "Terminated", "TimedOut").map("framework." + _) ->
      ("a closed phase has at most one closure role of Succeeded, Failed, Canceled, Terminated " +
        "and TimedOut")
  )

  // A case whose roles conflict: its name, the roles it declares (those no other of its roles
  // implies), the framework roles that conflict and what the refusal says of them.
  final case class Conflict(phase: String, declared: Seq[String], roles: Seq[String], rule: String)

  // The roles of one enum: the cases of each role, in declaration order, and the conflicts.
  final case class Closure(holding: Map[String, Seq[String]], conflicts: Seq[Conflict]):
    def cases(role: String): Seq[String] = holding.getOrElse(role, Nil)

  // Whether a class is a role, given every class it derives from, itself included.
  def isRole(bases: Seq[String]): Boolean = bases.exists(framework.contains)

  // The closure of an enum's cases, each in declaration order with every class it derives from;
  // `bases` gives the classes a class derives from, itself included.
  def closure(cases: Seq[(String, Seq[String])], bases: String => Seq[String]): Closure =
    val roled = cases.map((phase, classes) => phase -> classes.filter(c => isRole(bases(c))))
    val holding = mutable.LinkedHashMap.empty[String, Vector[String]]
    for (phase, roles) <- roled; role <- roles do
      holding(role) = holding.getOrElse(role, Vector.empty) :+ phase
    val conflicts = roled.flatMap: (phase, roles) =>
      exclusive.collectFirst:
        case (group, rule) if group.count(roles.contains) > 1 =>
          val declared = roles.filterNot(r => roles.exists(o => o != r && bases(o).contains(r)))
          Conflict(phase, declared, group.filter(roles.contains), rule)
    Closure(holding.toMap, conflicts)

  // A role as a refusal names it: by its simple name.
  def named(role: String): String = role.split('.').last

// The lifting of role tests: `p.isInstanceOf[R]`, `p.in[R]`, a rule's `when[R]` and the type
// pattern `case _: R` of a phase, each lowered to the cases of the phase's enum that have the role,
// as `when(...)` and the alternatives of case literals written by hand lift; and the conflict
// check of every enum whose cases take roles, run as its type is declared.
private[irgen] trait PhaseRoles:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  private val closures = mutable.Map.empty[Symbol, Roles.Closure]

  // The roles of an enum's cases. A case with fields is a class; a case without is a value, whose
  // type carries the roles its extends clause names.
  private def closureOf(e: Symbol): Roles.Closure = closures.getOrElseUpdate(
    e, {
      val classes = e.children.map: c =>
        c -> (if c.isClassDef then c.typeRef else c.termRef.widen).baseClasses
      val bases = classes
        .flatMap(_._2)
        .distinct
        .map(b => b.fullName -> b.typeRef.baseClasses.map(_.fullName))
        .toMap
      // Reversed, the linearization lists a case's parents in the order its extends clause names
      // them, the order a refusal names its roles in.
      Roles.closure(
        classes.map((c, cs) => c.name -> cs.reverse.map(_.fullName)),
        b => bases.getOrElse(b, Nil)
      )
    }
  )

  // Hook: the refusal of an enum some case of which takes conflicting roles, at that case. Core
  // form: none of its own; `declareType` runs it on every enum it declares.
  def checkRoles(e: Symbol, at: Tree): Unit =
    for c <- closureOf(e).conflicts.headOption do
      val tree = e.children
        .find(_.name == c.phase)
        .flatMap(s => scala.util.Try(s.tree).toOption)
        .getOrElse(at)
      fail(
        tree,
        s"${c.phase} of ${e.fullName} is ${c.roles.map(Roles.named).mkString(" and ")}, through " +
          s"its roles ${c.declared.map(Roles.named).mkString(", ")}: ${c.rule}"
      )

  // The cases of the enum of `value` that have the role `role`, in declaration order. Refused: a
  // type that is no role, a value of no enum, a role no case of the enum has, and a role a case with
  // fields has.
  private def roleCases(
      value: TypeRepr,
      role: TypeRepr,
      at: Tree,
      reader: Option[String] = None
  ): List[Symbol] =
    val r = role.dealias.typeSymbol
    if !(r.isClassDef && Roles.isRole(r.typeRef.baseClasses.map(_.fullName))) then
      fail(
        at,
        s"${role.show} is no role, so a test against it has no IR form: test a phase against a " +
          "role of model/framework/Roles.scala, or a trait that extends one"
      )
    val e = value.widen.dealias.baseClasses
      .find(b => b.flags.is(Flags.Enum) && !b.flags.is(Flags.Case))
      .getOrElse(
        fail(
          at,
          s"a role test reads a phase, a value of a finite enum, and ${value.widen.show} is " +
            s"none: test the phase, such as `s.phase.in[${r.name}]`"
        )
      )
    val names = closureOf(e).cases(r.fullName)
    if names.isEmpty then
      for name <- reader do fail(at, s"$name's phase type ${e.fullName} has no ${r.name} case")
      fail(
        at,
        s"no case of ${e.fullName} is ${r.name}, so the test never holds: give the role to the " +
          s"phases it names, `case p extends ${e.name}, ${r.name}`, or test another role"
      )
    val cases = names.toList.map(n => e.children.find(_.name == n).get)
    // A case with fields is no one value, so its literal is no member of the set.
    for c <- cases.find(_.isClassDef) do
      fail(
        at,
        s"${c.name} of ${e.fullName} is ${r.name} and has fields, so the cases of ${r.name} are no " +
          "set of values: give the role to cases without fields"
      )
    cases

  // The list of the cases of `phase` that have `role`, as `when(...)` lists them in a rule.
  def roleSet(
      phase: TypeRepr,
      role: TypeRepr,
      at: Term,
      reader: Option[String] = None
  ): ir.Expr =
    list(roleCases(phase, role, at, reader).map(enumLiteral(_, at)), at)

  // `value.isInstanceOf[R]` and `value.in[R]`, as `value.in(<the cases of R>)` lifts.
  def roleTest(value: Term, role: TypeRepr, at: Term): ir.Expr =
    val cases = roleSet(value.tpe, role, at)
    binary(ir.Binary.Op.OP_CONTAINS, lift(value), cases, at)

  // The type pattern `case _: R` of a `scrutinee`, as the alternatives of the cases of R written
  // by hand lift: `case Phase.done | Phase.failed`, and one case as its literal alone.
  def rolePattern(scrutinee: TypeRepr, role: TypeRepr, at: Tree): P =
    val literals = roleCases(scrutinee, role, at).map: c =>
      declareType(enumOf(c), at)
      ir.Pattern(P.Literal(ir.Value(enumValue(irTypeName(enumOf(c)), c.name))))
    literals match
      case List(one) => one.kind
      case _         => P.Alternatives(ir.Alternatives(literals))
