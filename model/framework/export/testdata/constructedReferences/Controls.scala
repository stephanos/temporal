package framework.references

import framework.dsl.*
import framework.dsl.Finite.given

enum Grade:
  case low, high
final case class Cell(flag: Boolean)
final case class Pair(left: Cell, right: Cell, extra: Cell)

object Controls:
  val send = action("send", Actor("caller"))
  val fill = action("fill", Actor("caller")).input[Grade]("grade")
  val drain = action("drain", Actor("caller"))
  val foreign = action("send", Actor("caller"))
  val unexpected = hole
  var calls = 0
  var classCalls = 0
  var selectors = 0
  var order = Vector.empty[String]
  def selected: Action[EmptyTuple] =
    calls += 1
    send
  def selectedClass: Class =
    classCalls += 1
    fill(Grade.high)
  def receiver: Pipeline.type =
    order = order :+ "receiver"
    Pipeline
  def argument: Action[EmptyTuple] =
    order = order :+ "argument"
    send
  object Tank extends Machine[Cell, Boolean, Nothing]:
    def init = throw new AssertionError("reference inspected starts")
    def end(s: Cell) = throw new AssertionError(s"reference inspected ends $s")
    object rules
        extends Bindings(
          send ~> ((s: Cell) => if s.flag then unexpected.reached else unexpected.reached),
          fill ~> ((s: Cell, g: Grade) =>
            if s.flag && g == Grade.low then unexpected.reached else unexpected.reached
          ),
          drain ~> ((s: Cell) => if s.flag then unexpected.reached else unexpected.reached)
        )
  object Pipeline
      extends Composition[Pair](
        s =>
          selectors += 1;
          s.left -> Tank
        ,
        _.right -> Tank
      ):
    object syncs extends Syncs:
      sync(
        "transfer",
        s =>
          selectors += 1; s.left -> send
        ,
        _.right -> fill
      )
  object Missing extends Composition[Pair](_.left -> Tank, _.right -> Tank)
  object Ambiguous extends Composition[Pair](_.left -> Tank, _.right -> Tank):
    object syncs extends Syncs:
      sync("first", _.left -> send, _.right -> fill)
      sync("second", _.left -> send, _.right -> drain)
  object Duplicate extends Composition[Pair](_.left -> Tank, _.right -> Tank):
    object syncs extends Syncs:
      sync("same", _.left -> send, _.right -> fill)
      sync("same", _.left -> drain, _.right -> send)
  object EmptyName extends Composition[Pair](_.left -> Tank, _.right -> Tank):
    object syncs extends Syncs:
      sync("", _.left -> send, _.right -> drain)
  object SameField extends Composition[Pair](_.left -> Tank):
    object syncs extends Syncs:
      sync("sameField", _.left -> fill, _.left -> send)
  var prereadCalls = 0
  object PreRead
      extends Composition[Pair](
        s =>
          prereadCalls += 1;
          s.left -> Tank
        ,
        s =>
          prereadCalls += 1; s.right -> Tank
      )
  var memberOrder = Vector.empty[String]
  def member(label: String, model: Model): Model =
    memberOrder = memberOrder :+ label
    model
  object Ordered
      extends Composition[Pair](_.left -> member("left", Tank), _.right -> member("right", Tank))
  object OrderedReplacement
      extends Composition(Ordered.withMember(_.right -> member("replacement", RestrictedTank)))
  object OpaqueSync extends Composition[Pair](_.left -> Tank, _.right -> Tank):
    val endpoint: Pair => (Any, Action[?]) = _.left -> send
    object syncs extends Syncs:
      sync("opaque", endpoint, _.right -> drain)
  object DuplicateMembers extends Composition[Pair](_.left -> Tank, _.left -> Tank)
  object NormalTank extends Machine[Cell, Boolean, Nothing]:
    val init = Cell(false)
    def end(s: Cell) = s.flag
    def rules: RuleBook[Cell, Boolean, Nothing] = Tank.rules
  object RestrictedTank extends Derived(NormalTank.restrict(fill, drain))
  object DroppedSync extends Derived(NormalTank.restrict(drain))
  val limited = action("limited", Actor("caller")).input[Grade]("grade")(using Finite.of(Grade.low))
  val throwing: Finite[Grade] = new Finite[Grade]:
    def values = throw new AssertionError("bare reference enumerated input catalog")
  val allClasses = action("allClasses", Actor("caller")).input[Grade]("grade")(using throwing)
  object CatalogTank extends Machine[Cell, Boolean, Nothing]:
    val init = Cell(false)
    def end(s: Cell) = s.flag
    object rules
        extends Bindings(
          limited ~> ((s: Cell, g: Grade) => List(Step(g == Grade.low, s))),
          allClasses ~> ((s: Cell, g: Grade) => List(Step(g == Grade.low, s)))
        )
  object Catalog extends Composition[Pair](_.left -> CatalogTank)
  object Replaced extends Composition(Pipeline.withMember(_.right -> RestrictedTank))
  object Broken extends Composition(Pipeline.withMember(_.right -> DroppedSync))
  object Inherited extends Composition(Pipeline)
  object Solo extends Composition[Pair](_.left -> Tank, _.right -> Tank)
  private def refuses(expected: String)(body: => Any): Unit =
    try
      body: Unit
      throw new AssertionError(s"accepted $expected")
    catch
      case error: IllegalArgumentException =>
        assert(error.getMessage.contains("controls"), error.getMessage)
        assert(error.getMessage.contains(expected), error.getMessage)
  def main(args: Array[String]): Unit =
    val lazyMove = Pipeline.synced(_.left -> selected)
    assert(calls == 0)
    val bare = lazyMove.toIR("controls")
    assert(bare.getSync == "transfer")
    assert(calls == 1)
    assert(selectors == 2)
    val explicit = Pipeline.synced(_.right -> selectedClass)
    assert(classCalls == 0)
    assert(explicit.toIR("controls").selection.isSyncClass)
    assert(classCalls == 1)
    explicit.toIR("controls"): Unit
    assert(classCalls == 1)
    assert(lazyMove.toIR("controls").getSync == "transfer")
    assert(calls == 1)
    assert(selectors == 2)
    assert(Pipeline.own(_.right, send).toIR("controls").member == "right")
    assert(
      Pipeline
        .synced(_.right -> fill(Grade.high))
        .toIR("controls")
        .getSyncClass
        .inputs
        .head
        .getEnum
        .`case` == "high"
    )
    assert(Solo.own(_.left, fill).toIR("controls").getAction.endsWith(".fill"))
    assert(Solo.own(_.left, send()).toIR("controls").selection.isActionClass)
    assert(Solo.own(_.left, send).toIR("controls").selection.isAction)
    assert(Catalog.own(_.left, allClasses).toIR("controls").selection.isAction)
    assert(
      Catalog
        .own(_.left, limited(Grade.low))
        .toIR("controls")
        .getActionClass
        .inputs
        .head
        .getEnum
        .`case` == "low"
    )
    refuses("native action input catalog")(
      Catalog.own(_.left, limited(Grade.high)).toIR("controls")
    )
    assert(EmptyName.synced(_.left -> send).toIR("controls").getSync == "")
    assert(EmptyName.synced(_.left -> send).toIR("controls").selection.isSync)
    assert(
      SameField.synced(_.left -> fill(Grade.high)).toIR("controls").getSyncClass.inputs.size == 1
    )
    refuses("takes inputs")(SameField.synced(_.left -> send()).toIR("controls"))
    assert(Inherited.synced(_.left -> send).toIR("controls").getSync == "transfer")
    assert(Replaced.own(_.right, drain).toIR("controls").member == "right")
    refuses("binds no action")(Replaced.own(_.right, send).toIR("controls"))
    assert(Replaced.synced(_.left -> send).toIR("controls").getSync == "transfer")
    refuses("binds no action")(Broken.synced(_.left -> send).toIR("controls"))
    refuses("binds no action")(Pipeline.own(_.right, foreign).toIR("controls"))
    refuses("no member")(Pipeline.own(_.extra, drain).toIR("controls"))
    refuses("only with its pair")(Pipeline.own(_.left, send).toIR("controls"))
    refuses("no sync")(Missing.synced(_.left -> send).toIR("controls"))
    refuses("each pair")(Ambiguous.synced(_.left -> send).toIR("controls"))
    refuses("duplicate sync")(Duplicate.own(_.right, drain).toIR("controls"))
    refuses("retained direct member")(OpaqueSync.synced(_.left -> send).toIR("controls"))
    refuses("duplicate member")(DuplicateMembers.own(_.left, drain).toIR("controls"))
    refuses("takes inputs")(Pipeline.synced(_.left -> send()).toIR("controls"))
    val opaque: Pair => Any = _.left
    refuses("retained direct member")(Solo.own(opaque, send).toIR("controls"))
    refuses("retained direct member")(Solo.own(_.left.flag, send).toIR("controls"))
    refuses("retained direct member")(
      Solo.synced(s => s.left -> (if s.left.flag then send else drain)).toIR("controls")
    )
    receiver.own(_.right, argument): Unit
    assert(order == Vector("receiver", "argument"))
    val left = Solo.own(_.left, send)
    val right = Missing.own(_.right, drain)
    val mixed = assume("mixed").fair(send).fair(left).fair().fair(right, left).fair(fill)
    assert(mixed.fairness == List(send.decl, fill.decl))
    assert(
      mixed.composed.size == 3 && mixed.composed
        .zip(List(left, right, left))
        .forall((a, b) => a eq b)
    )
    assert(mixed.fairSegments.size == 5)
    val reversed = assume("reversed").fair(left, left).fair(send).fair(right)
    assert(
      reversed.composed.size == 3 && reversed.composed
        .zip(List(left, left, right))
        .forall((a, b) => a eq b)
    )
    assert(reversed.fairness == List(send.decl))
    PreRead.members: Unit
    assert(prereadCalls == 2)
    val preread = PreRead.own(_.right, send)
    preread.toIR("controls"): Unit
    preread.toIR("controls"): Unit
    assert(prereadCalls == 2)
    PreRead.members: Unit
    assert(prereadCalls == 4)
    OrderedReplacement.own(_.right, drain).toIR("controls"): Unit
    assert(memberOrder == Vector("left", "right", "replacement"), memberOrder)
    OrderedReplacement.own(_.right, drain).toIR("controls"): Unit
    assert(memberOrder.size == 3)
    preread.toIR("controls"): Unit
    assert(prereadCalls == 4)
    limited.decl.schemaKeys = Vector(None)
    refuses("retained typed schema")(Catalog.own(_.left, limited(Grade.low)).toIR("controls"))
