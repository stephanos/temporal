package framework.dsl

import framework.realize.Construction
import io.temporal.server.api.umpire.v1 as ir

private[framework] object ComposedReference:
  final private case class Endpoint(field: String, action: ActionDecl, id: String)
  final private case class Sync(name: String, first: Endpoint, second: Endpoint):
    def pairs(endpoint: Endpoint): Boolean =
      List(first, second).exists(value => value.field == endpoint.field && value.id == endpoint.id)

  def convert(value: Composed, declaration: String): ir.ComposedActionRef =
    def fail(message: String): Nothing =
      throw IllegalArgumentException(
        s"$declaration: ${value.composition.name}.${value.operation}: $message"
      )
    val composition = value.composition match
      case c: Composition[?] => c
      case _                 => fail("selection has no composition owner")
    def field(selector: Any): String = MemberPath
      .field(selector)
      .getOrElse(
        fail(
          "no retained direct member selector metadata; nested paths and opaque selectors are unsupported"
        )
      )
    val (member, selected) = value.operation match
      case "own" =>
        value.selected match
          case (selector, action: (Class | Action[?])) => field(selector) -> action
          case _                                       => fail("own has no retained selection")
      case "synced" =>
        val member = field(value.selected)
        member -> value.retainedAction.getOrElse(
          fail("no retained action expression independent of the member selector")
        )
      case _ => fail("no retained selection operation")
    val members = try composition.memberFields
    catch case error: IllegalArgumentException => fail(error.getMessage)
    def endpoint(member: String, action: ActionDecl): Endpoint =
      val machine = members.find(_._1 == member).map(_._2) match
        case Some(m: Machine[?, ?, ?]) => m
        case Some(_)                   => fail(s"$member is not a machine member")
        case None                      => fail(s"no member $member of ${composition.name}")
      val id = action.sourceIdentity.getOrElse(
        fail(s"action ${action.name} has no retained source identity")
      )
      if !machine.table.exists((decl, _) => decl.sourceIdentity.contains(id)) then
        fail(
          s"$member (${machine.name}) binds no action $id; name the action this member binds, not another declaration spelled alike"
        )
      Endpoint(member, action, id)
    val action = selected match
      case a: Action[?] => a.decl
      case c: Class     => c.decl
    val chosen = endpoint(member, action)
    val syncs = composition.referenceSyncs.collect:
      case statement @ Syncs.Statement.Sync(name, first, second) =>
        val (firstField, secondField, explicit) = statement.endpoints.getOrElse(
          fail("sync has no retained endpoint metadata")
        )
        val left = endpoint(
          firstField.getOrElse(fail("sync has no retained direct member selector metadata")),
          first
        )
        val right = endpoint(
          secondField.getOrElse(fail("sync has no retained direct member selector metadata")),
          second
        )
        val effectiveName = if explicit then name
        else if first.name.nonEmpty then first.name
        else first.spelling.getOrElse(fail("sync action has no retained declaration name"))
        Sync(effectiveName, left, right)
    if syncs.map(_.name).distinct.size != syncs.size then fail("duplicate sync names")
    val paired = syncs.filter(_.pairs(chosen))
    val sync = value.operation match
      case "own" =>
        if paired.nonEmpty then
          fail(
            s"sync ${paired.head.name} pairs $member.${action.name}, which steps only with its pair"
          )
        None
      case _ =>
        paired match
          case Vector(single) => Some(single)
          case Vector()       => fail(s"no sync pairs $member.${action.name}")
          case many           =>
            fail(s"syncs ${many.map(_.name).mkString(", ")} each pair $member.${action.name}")
    val ref = ir.ComposedActionRef(composition = composition.name)
    selected match
      case _: Action[?] =>
        sync match
          case Some(s) => ref.withSync(s.name)
          case None    => ref.withMember(member).withAction(chosen.id)
      case c: Class =>
        for s <- sync do
          val other =
            if s.first.field == chosen.field && s.first.id == chosen.id then s.second else s.first
          if other.action.inputs.nonEmpty then
            fail(
              s"sync ${s.name} opposite endpoint ${other.field}.${other.action.name} takes inputs a one-sided class cannot name"
            )
        val cls = try
          val signature = Signature.collect(Vector(new Action[EmptyTuple](action)), Vector.empty)
          new Construction(declaration, signature, "").actionClass(c)
        catch case error: IllegalArgumentException => fail(error.getMessage)
        sync match
          case Some(s) => ref.withSyncClass(cls.copy(action = s.name))
          case None    => ref.withMember(member).withActionClass(cls)
