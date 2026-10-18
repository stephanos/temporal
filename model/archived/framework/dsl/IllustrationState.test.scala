package framework.dsl

import framework.dsl.Finite.given

object IllustrationStateFixture:
  given Ok[Boolean] = Ok(true)
  final case class Inner(count: UpTo[2], values: List[Int])
  final case class Record(inner: Inner, optional: Option[UpTo[2]], entries: Map[Int, Set[UpTo[2]]])
  enum Variant:
    case idle
    case active(inner: Inner)

  given Finite[Inner] = new Finite[Inner]:
    def values = throw AssertionError("nested state catalog evaluated")
  given Finite[Record] = new Finite[Record]:
    def values = throw AssertionError("whole state catalog evaluated")
  given Finite[Variant] = new Finite[Variant]:
    def values = throw AssertionError("sum state catalog evaluated")

  object Subject extends Machine[Boolean, Nothing]:
    var record = Record(Inner(UpTo[2](0), Nil), None, Map.empty)
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  object SumSubject extends Machine[Boolean, Nothing]:
    var phase: Variant = Variant.idle
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  final case class Held(box: Inbox[Boolean])
  final case class Mailbox(direct: Inbox[Boolean], nested: Option[Held])
  val wire =
    channel[Boolean](capacity = 2, order = Order.unordered, loss = Loss.reliable, duplicates = 1)
  val other =
    channel[Boolean](capacity = 2, order = Order.unordered, loss = Loss.reliable, duplicates = 1)
  given Finite[Inbox[Boolean]] = new Finite[Inbox[Boolean]]:
    def values = throw AssertionError("Inbox contents catalog evaluated")

  object Positive:
    opaque type Value <: Int = Int
    def apply(value: Int): Value =
      require(value > 0)
      value
    given StateData[Value] = summon[StateData[Int]]
    given Data.Structural[Value] = summon[Data.Structural[Int]]

  final case class OpaqueRecord(value: Positive.Value)

  object OpaqueOwner extends Machine[Boolean, Nothing]:
    var record = OpaqueRecord(Positive(1))
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  class MailOwner(mapping: Int) extends Machine[Boolean, Nothing]:
    var mailbox = Mailbox(wire.empty, None)
    if mapping != 0 then
      inbox(mailbox.direct, wire)
      inbox(mailbox.nested.get.box, if mapping == 3 then wire else other)
      if mapping == 2 then inbox(mailbox.direct, wire)
      if mapping == 4 then inbox(mailbox.direct, other)
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  enum MailState:
    case idle
    case active(nested: Held)

  object BranchOwner extends Machine[Boolean, Nothing]:
    var phase: MailState = MailState.idle
    inbox(
      phase match
        case active: MailState.active => active.nested.box
        case _                        => throw AssertionError("selector executed")
      ,
      wire
    )
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  enum Packet[+A]:
    case empty
    case full(value: A)

  object PacketOwner extends Machine[Boolean, Nothing]:
    var packet: Packet[UpTo[2]] = Packet.empty
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

  object PacketMailOwner extends Machine[Boolean, Nothing]:
    var packet: Packet[Held] = Packet.empty
    inbox(
      packet match
        case full: Packet.full[Held] => full.value.box
        case _                       => throw AssertionError("selector executed")
      ,
      wire
    )
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions

class IllustrationStateSuite extends munit.FunSuite:
  import IllustrationStateFixture.{*, given}

  private val context = "property sample illustration boundary at Example.scala:17:3"
  private def outside: UpTo[2] = 3.asInstanceOf[UpTo[2]] // scalafix:ok DisableSyntax.asInstanceOf

  private def validate(
      owner: MachineDeclaration[State, Boolean, Nothing],
      updates: (String, Any)*
  ): Unit =
    val supplied = owner.init.copy.applyDynamicNamed("apply")(updates*)
    StructuralValue.capture[State].validate(supplied, owner, context)

  test("structural state checks ignore whole and nested catalogs and impose no Int or list cap"):
    val value = Record(Inner(UpTo[2](2), List(Int.MinValue, Int.MaxValue)), None, Map.empty)
    validate(Subject, "record" -> value)
    validate(SumSubject, "phase" -> Variant.active(value.inner))
    validate(SumSubject, "phase" -> Variant.idle)

  test("known bounds are checked throughout records active branches and native containers"):
    val valid = Record(Inner(UpTo[2](0), Nil), None, Map.empty)
    val invalid = Vector(
      valid.copy(inner = Inner(outside, Nil)) -> "inner.count",
      valid.copy(inner = Option.empty[Inner].orNull) -> "inner",
      valid.copy(optional = Some(outside)) -> "optional",
      valid.copy(entries = Map(7 -> Set(outside))) -> "entries"
    )
    invalid.foreach: (state, field) =>
      val error = intercept[IllegalArgumentException]:
        validate(Subject, "record" -> state)
      assert(error.getMessage.contains(context), error.getMessage)
      assert(error.getMessage.contains(field), error.getMessage)
    val branch = intercept[IllegalArgumentException]:
      validate(SumSubject, "phase" -> Variant.active(Inner(outside, Nil)))
    assert(branch.getMessage.contains("active"), branch.getMessage)

  test("public direct nested optional and branch Inbox coordinates admit exact receivers"):
    val owner = new MailOwner(1)
    val value = Mailbox(wire.empty.send(false).send(false), Some(Held(other.empty.send(true))))
    validate(owner, "mailbox" -> value)
    validate(owner, "mailbox" -> value.copy(nested = None))
    validate(BranchOwner, "phase" -> MailState.active(Held(wire.empty)))
    validate(BranchOwner, "phase" -> MailState.idle)

  test("Inbox admission rejects crossed empty receivers malformed bounds messages and order"):
    val owner = new MailOwner(1)
    val cases = Vector(
      other.empty,
      Inbox(wire, List.fill(3)(Delivery(false, 0))),
      Inbox(wire, List(Delivery(false, -1))),
      Inbox(wire, List(Delivery(false, 2))),
      Inbox(wire, List(Delivery(true, 0), Delivery(false, 0)))
    )
    cases.foreach: supplied =>
      val error = intercept[IllegalArgumentException]:
        validate(owner, "mailbox" -> Mailbox(supplied, None))
      assert(error.getMessage.contains(context), error.getMessage)
      assert(error.getMessage.contains("direct"), error.getMessage)

  test(
    "mapping acquisition rejects missing repeated conflicting and duplicate storage declarations"
  ):
    for mapping <- Vector(0, 2, 3, 4) do
      val error = intercept[IllegalArgumentException]:
        validate(new MailOwner(mapping), "mailbox" -> Mailbox(wire.empty, None))
      assert(error.getMessage.contains("direct"), error.getMessage)
      assert(error.getMessage.contains(context), error.getMessage)
      if mapping != 0 then
        assert(error.getMessage.contains("IllustrationState.test.scala"), error.getMessage)
    object SplitOwner extends Machine[Boolean, Nothing]:
      var first = wire.empty
      var second = wire.empty
      inbox(first, wire)
      inbox(second, wire)
      object transitions extends Transitions
    val duplicate = intercept[IllegalArgumentException]:
      validate(SplitOwner)
    assert(
      duplicate.getMessage.contains("duplicate Inbox storage coordinate"),
      duplicate.getMessage
    )

  test(
    "actual Built and Derived lineage retains Inbox declarations without evaluating end or catalogs"
  ):
    val owner = new MailOwner(1)
    val built = owner.unmonitored
    object DerivedOwner extends Derived(built)
    val supplied = Mailbox(wire.empty, Some(Held(other.empty)))
    validate(built, "mailbox" -> supplied)
    validate(DerivedOwner, "mailbox" -> supplied)

  test("same state type does not borrow another machine's Inbox declarations"):
    val declared = new MailOwner(1)
    validate(declared, "mailbox" -> Mailbox(wire.empty, None))
    val foreign = new MailOwner(0)
    val _ = intercept[IllegalArgumentException]:
      validate(foreign, "mailbox" -> Mailbox(wire.empty, None))
    foreign.retainNativeInbox(declared.nativeInboxes.head)
    val error = intercept[IllegalArgumentException]:
      validate(foreign, "mailbox" -> Mailbox(wire.empty, None))
    assert(error.getMessage.contains("foreign Inbox association owner"), error.getMessage)

  test("opaque state validity requires selected leaf evidence and enforces its membership"):
    val unsupported = intercept[IllegalArgumentException]:
      validate(OpaqueOwner, "record" -> OpaqueRecord(Positive(1)))
    assert(unsupported.getMessage.contains("value"), unsupported.getMessage)
    locally:
      given Finite[Positive.Value] = Finite.of(Positive(1))
      class SelectedOwner extends Machine[Boolean, Nothing]:
        var record = OpaqueRecord(Positive(1))
        override def end: Boolean = throw AssertionError("end evaluated")
        object transitions extends Transitions
      val owner = new SelectedOwner
      validate(owner, "record" -> OpaqueRecord(Positive(1)))
      val _ = intercept[IllegalArgumentException]:
        validate(owner, "record" -> OpaqueRecord(Positive(2)))
    locally:
      @scala.annotation.unused
      given scala.deriving.Mirror.SumOf[Positive.Value] = new scala.deriving.Mirror.Sum:
        @scala.annotation.unused
        type MirroredType = Positive.Value
        type MirroredMonoType = Positive.Value
        type MirroredLabel = "Pretend"
        @scala.annotation.unused
        type MirroredElemTypes = EmptyTuple
        type MirroredElemLabels = EmptyTuple
        def ordinal(value: Positive.Value): Int = 0
      class ForgedOwner extends Machine[Boolean, Nothing]:
        var record = OpaqueRecord(Positive(1))
        object transitions extends Transitions
      val error = intercept[IllegalArgumentException]:
        validate(new ForgedOwner, "record" -> OpaqueRecord(Positive(1)))
      assert(error.getMessage.contains("unsupported"), error.getMessage)

  test("absent optional Inbox needs no inferred association and present value requires one"):
    final case class OptionalMail(nested: Option[Held])
    object Owner extends Machine[Boolean, Nothing]:
      var mailbox = OptionalMail(None)
      override def end: Boolean = throw AssertionError("end evaluated")
      object transitions extends Transitions
    validate(Owner, "mailbox" -> OptionalMail(None))
    intercept[IllegalArgumentException]:
      validate(Owner, "mailbox" -> OptionalMail(Some(Held(wire.empty))))

  test("Inbox messages use actual selected channel catalog independently of ambient witness"):
    val selected =
      new Channel[Boolean]("selected", 2, Order.fifo, Loss.reliable, 0)(using Finite.of(false))
    object Owner extends Machine[Boolean, Nothing]:
      var box = selected.empty
      inbox(box, selected)
      override def end: Boolean = throw AssertionError("end evaluated")
      object transitions extends Transitions
    validate(Owner, "box" -> selected.empty.send(false))
    val error = intercept[IllegalArgumentException]:
      validate(Owner, "box" -> selected.empty.send(true))
    assert(error.getMessage.contains("selected Channel domain"), error.getMessage)
    val bounded = new Channel[UpTo[2]]("bounded", 1, Order.fifo, Loss.reliable, 0)(using
      Finite.of[UpTo[2]](UpTo[2](0), outside)
    )
    object BoundedOwner extends Machine[Boolean, Nothing]:
      var box = bounded.empty
      inbox(box, bounded)
      object transitions extends Transitions
    validate(BoundedOwner, "box" -> bounded.empty.send(UpTo[2](0)))
    val malformed = intercept[IllegalArgumentException]:
      validate(BoundedOwner, "box" -> bounded.empty.send(outside))
    assert(malformed.getMessage.contains("UpTo bounds"), malformed.getMessage)
    val reversed =
      new Channel[Int]("reversed", 2, Order.unordered, Loss.reliable, 0)(using Finite.of(2, 0))
    object UnorderedOwner extends Machine[Boolean, Nothing]:
      var box = reversed.empty
      inbox(box, reversed)
      object transitions extends Transitions
    for values <- List(List(0, 2), List(0, 0)) do
      validate(UnorderedOwner, "box" -> Inbox(reversed, values.map(Delivery(_, 0))))
    val unordered = intercept[IllegalArgumentException]:
      validate(UnorderedOwner, "box" -> Inbox(reversed, List(Delivery(2, 0), Delivery(0, 0))))
    assert(unordered.getMessage.contains("canonical structural order"), unordered.getMessage)

  test(
    "nested sum Inbox declarations accept an exact rooted typed branch without selector execution"
  ):
    final case class Envelope(phase: MailState)
    object Owner extends Machine[Boolean, Nothing]:
      var envelope = Envelope(MailState.idle)
      inbox(
        envelope.phase match
          case active: MailState.active => active.nested.box
          case _                        => throw AssertionError("selector executed")
        ,
        wire
      )
      object transitions extends Transitions
    validate(Owner, "envelope" -> Envelope(MailState.active(Held(wire.empty))))
    validate(Owner, "envelope" -> Envelope(MailState.idle))
    val crossed = intercept[IllegalArgumentException]:
      validate(Owner, "envelope" -> Envelope(MailState.active(Held(other.empty))))
    assert(crossed.getMessage.contains("phase.<active>.nested.box"), crossed.getMessage)

  test("Inbox selectors refuse computed unrooted and ambiguous field coordinates"):
    val computed = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.IllustrationStateFixture.*
      object Owner extends Machine[Boolean, Nothing]:
        var mailbox = Mailbox(wire.empty, None)
        inbox(if mailbox.direct.isEmpty then mailbox.direct else mailbox.nested.get.box, wire)
        object transitions extends Transitions
    """)
    assert(computed.contains("Inbox association"), computed)
    val unrooted = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.IllustrationStateFixture.*
      object Owner extends Machine[Boolean, Nothing]:
        var mailbox = Mailbox(wire.empty, None)
        inbox(wire.empty, wire)
        object transitions extends Transitions
    """)
    assert(unrooted.contains("Inbox association"), unrooted)
    val foreign = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.IllustrationStateFixture.{*, given}
      object Foreign extends Machine[Boolean, Nothing]:
        var box = wire.empty
        object transitions extends Transitions
      object Owner extends Machine[Boolean, Nothing]:
        var box = wire.empty
        inbox(Foreign.box, wire)
        object transitions extends Transitions
    """)
    assert(foreign.contains("this Machine's owning state var"), foreign)

  test("generic sum cases retain concrete nested bounds without consulting their catalogs"):
    validate(PacketOwner, "packet" -> Packet.full(UpTo[2](2)))
    validate(PacketOwner, "packet" -> Packet.empty)
    val error = intercept[IllegalArgumentException]:
      validate(PacketOwner, "packet" -> Packet.full(outside))
    assert(error.getMessage.contains("<full>.value"), error.getMessage)
    final case class Envelope(packet: Option[Packet[UpTo[2]]])
    object Owner extends Machine[Boolean, Nothing]:
      var envelope = Envelope(None)
      object transitions extends Transitions
    validate(Owner, "envelope" -> Envelope(Some(Packet.full(UpTo[2](2)))))
    val nested = intercept[IllegalArgumentException]:
      validate(Owner, "envelope" -> Envelope(Some(Packet.full(outside))))
    assert(nested.getMessage.contains("UpTo bounds"), nested.getMessage)

  test("generic Inbox branches retain their exact concrete field and Channel association"):
    validate(PacketMailOwner, "packet" -> Packet.full(Held(wire.empty)))
    validate(PacketMailOwner, "packet" -> Packet.empty)
    val error = intercept[IllegalArgumentException]:
      validate(PacketMailOwner, "packet" -> Packet.full(Held(other.empty)))
    assert(error.getMessage.contains("<full>.value.box"), error.getMessage)
