package framework.dsl

object IllustrationStateFixture:
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

  object Subject extends Machine[Record, Boolean, Nothing]:
    def init: Record = throw AssertionError("init evaluated")
    def end(state: Record): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  object SumSubject extends Machine[Variant, Boolean, Nothing]:
    def init: Variant = throw AssertionError("init evaluated")
    def end(state: Variant): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

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

  object OpaqueOwner extends Machine[OpaqueRecord, Boolean, Nothing]:
    def init: OpaqueRecord = throw AssertionError("init evaluated")
    def end(state: OpaqueRecord): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  class MailOwner(mapping: Int) extends Machine[Mailbox, Boolean, Nothing]:
    if mapping != 0 then
      inbox(_.direct, wire)
      inbox(_.nested.get.box, if mapping == 3 then wire else other)
      if mapping == 2 then inbox(_.direct, wire)
      if mapping == 4 then inbox(_.direct, other)
    def init: Mailbox = throw AssertionError("init evaluated")
    def end(state: Mailbox): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  enum MailState:
    case idle
    case active(nested: Held)

  object BranchOwner extends Machine[MailState, Boolean, Nothing]:
    inboxCase[MailState.active, Boolean](_.nested.box, wire)
    def init: MailState = throw AssertionError("init evaluated")
    def end(state: MailState): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  enum Packet[+A]:
    case empty
    case full(value: A)

  object PacketOwner extends Machine[Packet[UpTo[2]], Boolean, Nothing]:
    def init: Packet[UpTo[2]] = throw AssertionError("init evaluated")
    def end(state: Packet[UpTo[2]]): Boolean = false

    object rules extends Bindings()

  object PacketMailOwner extends Machine[Packet[Held], Boolean, Nothing]:
    inboxCase[Packet.full[Held], Boolean](_.value.box, wire)
    def init: Packet[Held] = throw AssertionError("init evaluated")
    def end(state: Packet[Held]): Boolean = false

    object rules extends Bindings()

class IllustrationStateSuite extends munit.FunSuite:
  import IllustrationStateFixture.*

  private val context = "property sample illustration boundary at Example.scala:17:3"
  private def outside: UpTo[2] = 3.asInstanceOf[UpTo[2]] // scalafix:ok DisableSyntax.asInstanceOf

  test("structural state checks ignore whole and nested catalogs and impose no Int or list cap"):
    val value = Record(Inner(UpTo[2](2), List(Int.MinValue, Int.MaxValue)), None, Map.empty)
    StructuralValue.capture[Record].validate(value, Subject, context)
    StructuralValue.capture[Variant].validate(Variant.active(value.inner), SumSubject, context)
    StructuralValue.capture[Variant].validate(Variant.idle, SumSubject, context)

  test("known bounds are checked throughout records active branches and native containers"):
    val valid = Record(Inner(UpTo[2](0), Nil), None, Map.empty)
    val invalid = Vector(
      valid.copy(inner = Inner(outside, Nil)) -> "inner.count",
      valid.copy(optional = Some(outside)) -> "optional",
      valid.copy(entries = Map(7 -> Set(outside))) -> "entries"
    )
    invalid.foreach: (state, field) =>
      val error = try
        StructuralValue.capture[Record].validate(state, Subject, context)
        fail(s"accepted malformed field $field")
      catch case error: IllegalArgumentException => error
      assert(error.getMessage.contains(context), error.getMessage)
      assert(error.getMessage.contains(field), error.getMessage)
    val branch = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[Variant]
        .validate(Variant.active(Inner(outside, Nil)), SumSubject, context)
    assert(branch.getMessage.contains("active"), branch.getMessage)

  test("public direct nested optional and branch Inbox coordinates admit exact receivers"):
    val owner = new MailOwner(1)
    val value = Mailbox(wire.empty.send(false).send(false), Some(Held(other.empty.send(true))))
    StructuralValue.capture[Mailbox].validate(value, owner, context)
    StructuralValue.capture[Mailbox].validate(value.copy(nested = None), owner, context)
    StructuralValue
      .capture[MailState]
      .validate(MailState.active(Held(wire.empty)), BranchOwner, context)
    StructuralValue.capture[MailState].validate(MailState.idle, BranchOwner, context)

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
        StructuralValue.capture[Mailbox].validate(Mailbox(supplied, None), owner, context)
      assert(error.getMessage.contains(context), error.getMessage)
      assert(error.getMessage.contains("direct"), error.getMessage)

  test(
    "mapping acquisition rejects missing repeated conflicting and duplicate storage declarations"
  ):
    for mapping <- Vector(0, 2, 3, 4) do
      val error = intercept[IllegalArgumentException]:
        StructuralValue
          .capture[Mailbox]
          .validate(Mailbox(wire.empty, None), new MailOwner(mapping), context)
      assert(error.getMessage.contains("direct"), error.getMessage)
      assert(error.getMessage.contains(context), error.getMessage)
      if mapping != 0 then
        assert(error.getMessage.contains("IllustrationState.test.scala"), error.getMessage)

  test("actual Built and Derived lineage retains Inbox declarations without reading init"):
    val owner = new MailOwner(1)
    val built = owner.unmonitored
    object DerivedOwner extends Derived(built)
    val supplied = Mailbox(wire.empty, Some(Held(other.empty)))
    StructuralValue.capture[Mailbox].validate(supplied, built, context)
    StructuralValue.capture[Mailbox].validate(supplied, DerivedOwner, context)

  test("same state type does not borrow another machine's Inbox declarations"):
    val declared = new MailOwner(1)
    StructuralValue.capture[Mailbox].validate(Mailbox(wire.empty, None), declared, context)
    val foreign = new MailOwner(0)
    val _ = intercept[IllegalArgumentException]:
      StructuralValue.capture[Mailbox].validate(Mailbox(wire.empty, None), foreign, context)
    foreign.retainNativeInbox(declared.nativeInboxes.head)
    val error = intercept[IllegalArgumentException]:
      StructuralValue.capture[Mailbox].validate(Mailbox(wire.empty, None), foreign, context)
    assert(error.getMessage.contains("foreign Inbox association owner"), error.getMessage)

  test("opaque state validity requires selected leaf evidence and enforces its membership"):
    val unsupported = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[OpaqueRecord]
        .validate(OpaqueRecord(Positive(1)), OpaqueOwner, context)
    assert(unsupported.getMessage.contains("value"), unsupported.getMessage)
    locally:
      given Finite[Positive.Value] = Finite.of(Positive(1))
      val _ = () => summon[Finite[Positive.Value]]
      StructuralValue
        .capture[OpaqueRecord]
        .validate(OpaqueRecord(Positive(1)), OpaqueOwner, context)
      val _ = intercept[IllegalArgumentException]:
        StructuralValue
          .capture[OpaqueRecord]
          .validate(OpaqueRecord(Positive(2)), OpaqueOwner, context)
    locally:
      given scala.deriving.Mirror.SumOf[Positive.Value] = new scala.deriving.Mirror.Sum:
        @scala.annotation.unused
        type MirroredType = Positive.Value
        type MirroredMonoType = Positive.Value
        type MirroredLabel = "Pretend"
        @scala.annotation.unused
        type MirroredElemTypes = EmptyTuple
        type MirroredElemLabels = EmptyTuple
        def ordinal(value: Positive.Value): Int = 0
      val _ = summon[scala.deriving.Mirror.SumOf[Positive.Value]]
      val error = intercept[IllegalArgumentException]:
        StructuralValue
          .capture[OpaqueRecord]
          .validate(OpaqueRecord(Positive(1)), OpaqueOwner, context)
      assert(error.getMessage.contains("unsupported"), error.getMessage)

  test("absent optional Inbox needs no inferred association and present value requires one"):
    final case class OptionalMail(nested: Option[Held])
    object Owner extends Machine[OptionalMail, Boolean, Nothing]:
      def init: OptionalMail = throw AssertionError("init evaluated")
      def end(state: OptionalMail): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    StructuralValue.capture[OptionalMail].validate(OptionalMail(None), Owner, context)
    intercept[IllegalArgumentException]:
      StructuralValue
        .capture[OptionalMail]
        .validate(OptionalMail(Some(Held(wire.empty))), Owner, context)

  test("Inbox messages use actual selected channel catalog independently of ambient witness"):
    val selected =
      new Channel[Boolean]("selected", 2, Order.fifo, Loss.reliable, 0)(using Finite.of(false))
    final case class SelectedMail(box: Inbox[Boolean])
    object Owner extends Machine[SelectedMail, Boolean, Nothing]:
      inbox(_.box, selected)
      def init: SelectedMail = throw AssertionError("init evaluated")
      def end(state: SelectedMail): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    StructuralValue
      .capture[SelectedMail]
      .validate(SelectedMail(selected.empty.send(false)), Owner, context)
    val error = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[SelectedMail]
        .validate(SelectedMail(selected.empty.send(true)), Owner, context)
    assert(error.getMessage.contains("selected Channel domain"), error.getMessage)
    val bounded = new Channel[UpTo[2]]("bounded", 1, Order.fifo, Loss.reliable, 0)(using
      Finite.of[UpTo[2]](UpTo[2](0), outside)
    )
    final case class BoundedMail(box: Inbox[UpTo[2]])
    object BoundedOwner extends Machine[BoundedMail, Boolean, Nothing]:
      inbox(_.box, bounded)
      def init: BoundedMail = throw AssertionError("init evaluated")
      def end(state: BoundedMail): Boolean = false
      object rules extends Bindings()
    StructuralValue
      .capture[BoundedMail]
      .validate(
        BoundedMail(bounded.empty.send(UpTo[2](0))),
        BoundedOwner,
        context
      )
    val malformed = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[BoundedMail]
        .validate(BoundedMail(bounded.empty.send(outside)), BoundedOwner, context)
    assert(malformed.getMessage.contains("UpTo bounds"), malformed.getMessage)
    val reversed =
      new Channel[Int]("reversed", 2, Order.unordered, Loss.reliable, 0)(using Finite.of(2, 0))
    final case class UnorderedMail(box: Inbox[Int])
    object UnorderedOwner extends Machine[UnorderedMail, Boolean, Nothing]:
      inbox(_.box, reversed)
      def init: UnorderedMail = throw AssertionError("init evaluated")
      def end(state: UnorderedMail): Boolean = false
      object rules extends Bindings()
    for values <- List(List(0, 2), List(0, 0)) do
      StructuralValue
        .capture[UnorderedMail]
        .validate(
          UnorderedMail(Inbox(reversed, values.map(Delivery(_, 0)))),
          UnorderedOwner,
          context
        )
    val unordered = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[UnorderedMail]
        .validate(
          UnorderedMail(Inbox(reversed, List(Delivery(2, 0), Delivery(0, 0)))),
          UnorderedOwner,
          context
        )
    assert(unordered.getMessage.contains("canonical structural order"), unordered.getMessage)

  test(
    "nested sum Inbox declarations accept an exact rooted typed branch without selector execution"
  ):
    final case class Envelope(phase: MailState)
    object Owner extends Machine[Envelope, Boolean, Nothing]:
      inbox(
        (s: Envelope) =>
          s.phase match
            case active: MailState.active => active.nested.box
            case _                        => throw AssertionError("selector executed")
        ,
        wire
      )
      def init: Envelope = throw AssertionError("init evaluated")
      def end(state: Envelope): Boolean = false
      object rules extends Bindings()
    StructuralValue
      .capture[Envelope]
      .validate(Envelope(MailState.active(Held(wire.empty))), Owner, context)
    StructuralValue.capture[Envelope].validate(Envelope(MailState.idle), Owner, context)
    val crossed = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[Envelope]
        .validate(Envelope(MailState.active(Held(other.empty))), Owner, context)
    assert(crossed.getMessage.contains("phase.<active>.nested.box"), crossed.getMessage)

  test("Inbox selectors refuse computed unrooted and ambiguous field coordinates"):
    assert(compileErrors("""
      import framework.dsl.*
      import framework.dsl.IllustrationStateFixture.*
      object Owner extends Machine[Mailbox, Boolean, Nothing]:
        inbox(s => if s.direct.isEmpty then s.direct else s.nested.get.box, wire)
        def init: Mailbox = throw AssertionError("init evaluated")
        def end(state: Mailbox): Boolean = false
        object rules extends Bindings()
    """).nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      import framework.dsl.IllustrationStateFixture.*
      object Owner extends Machine[Mailbox, Boolean, Nothing]:
        inbox(_ => wire.empty, wire)
        def init: Mailbox = throw AssertionError("init evaluated")
        def end(state: Mailbox): Boolean = false
        object rules extends Bindings()
    """).nonEmpty)

  test("generic sum cases retain concrete nested bounds without consulting their catalogs"):
    StructuralValue.capture[Packet[UpTo[2]]].validate(Packet.full(UpTo[2](2)), PacketOwner, context)
    StructuralValue.capture[Packet[UpTo[2]]].validate(Packet.empty, PacketOwner, context)
    val error = intercept[IllegalArgumentException]:
      StructuralValue.capture[Packet[UpTo[2]]].validate(Packet.full(outside), PacketOwner, context)
    assert(error.getMessage.contains("<full>.value"), error.getMessage)
    final case class Envelope(packet: Option[Packet[UpTo[2]]])
    object Owner extends Machine[Envelope, Boolean, Nothing]:
      def init: Envelope = throw AssertionError("init evaluated")
      def end(state: Envelope): Boolean = false
      object rules extends Bindings()
    StructuralValue
      .capture[Envelope]
      .validate(Envelope(Some(Packet.full(UpTo[2](2)))), Owner, context)
    val nested = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[Envelope]
        .validate(Envelope(Some(Packet.full(outside))), Owner, context)
    assert(nested.getMessage.contains("UpTo bounds"), nested.getMessage)

  test("generic Inbox branches retain their exact concrete field and Channel association"):
    StructuralValue
      .capture[Packet[Held]]
      .validate(Packet.full(Held(wire.empty)), PacketMailOwner, context)
    StructuralValue.capture[Packet[Held]].validate(Packet.empty, PacketMailOwner, context)
    val error = intercept[IllegalArgumentException]:
      StructuralValue
        .capture[Packet[Held]]
        .validate(Packet.full(Held(other.empty)), PacketMailOwner, context)
    assert(error.getMessage.contains("<full>.value.box"), error.getMessage)
