// The kit's evidence modules: what several Temporal realizations declared alike by hand, declared
// once. A request base carries the fields every call on a role assigns; a described status reads a
// describe method's status for each fact a table lists; a history declares a workflow's
// history-event kinds. The lifter writes each module's declarations by name, as the core records
// they stand for (model/framework/export/Realizations.scala), so a module's body is what the IR says too.
package temporal.realize

import io.grpc.MethodDescriptor
import scalapb.{GeneratedEnum, GeneratedMessage, GeneratedMessageCompanion}
import io.temporal.api.history.v1.HistoryEvent
import framework.dsl.{Action, Class, Input, Machine}
import framework.realize.*

// The fields every call on `role` assigns, each a request field by its protobuf name and the
// operand it gets: `RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" ->
// run)`. A call on the base, `rpc(base, method) { … }` or `await(evidence, base)(until) { … }`, is a
// call on its role whose request assigns the base's fields first, in order, then its own. A field
// is the request's own field of that name, or the one field of that name in one of the request's
// message fields, such as `execution.workflow_id`. A call that assigns a base field itself
// overrides it, and an override equal to the base is refused.
final case class RequestBase(role: Addressee, fields: (String, TypedOperand[String])*)
    extends Addressee:
  def id: String = role.id

// The status a describe method reports for each fact of `machine` that `entries` lists, read from
// the response's `info` message, keyed to its operation by `operation` and compared at `status`:
// one declaration of what a realization reads back of a described operation, its calls made on
// `calls`. Its facts are the machine's, so a fact of another machine does not compile. It yields
// the evidence of each listed fact, `evidence`, in table order; one fact's, `described(fact)`; and
// a read that waits until the description reports the fact's status, `described.await(fact)`,
// named `await-<status>` after the status value's own name (`ACTIVITY_EXECUTION_STATUS_TIMED_OUT`
// awaits as `await-timed-out`). Each fact is listed once.
final class DescribedStatus[
    F <: AnyRef,
    Req <: GeneratedMessage,
    Rsp <: GeneratedMessage,
    Info <: GeneratedMessage,
    V <: GeneratedEnum
](
    val machine: Machine[?, ?, F],
    val calls: RequestBase,
    val method: MethodDescriptor[Req, Rsp],
    val info: Field[Rsp, Info],
    val operation: Field[Info, ?],
    val status: Field[Info, V]
)(val entries: (F | EveryValue, V)*)(using
    GeneratedMessageCompanion[Req],
    GeneratedMessageCompanion[Rsp],
    GeneratedMessageCompanion[Info]
):
  // What each listed fact reads as, as a status table.
  def table: StatusTable[V] = statusTable(entries*)

  // The evidence of every fact the table lists, in its order.
  def evidence: Vector[EvidenceRef[Req, Info]] = entries.map((fact, _) => apply(fact)).toVector

  // The evidence of one fact: the describe method's status, read once the operation stays in it.
  def apply(fact: F | EveryValue): EvidenceRef[Req, Info] = Evidence.read(
    id = evidenceId(fact),
    records = fact,
    source = sourceId(fact),
    from = Recorded.single(method, info),
    operation = operation,
    commitment = Commitment.reported
  )

  // Reads the description until it reports the status the table lists for `fact`.
  def await(fact: F | EveryValue): Instruction =
    val reported = entries.find((f, _) => java.util.Objects.equals(f, fact)).get._2
    Instruction.readUntil(apply(fact), calls)(
      Vector.empty,
      Condition.equal(status, Operand.enumValue[V, V](reported))
    )

// One history-event kind: the fact a history event confirms, and the member of the event's
// attributes it is recorded in.
final class HistoryKind(
    val fact: Fact,
    val attributes: HistoryEvent => Option[GeneratedMessage],
    val selection: Field[HistoryEvent, Option[GeneratedMessage]]
)

object HistoryKind:
  inline def apply(
      fact: Fact,
      inline attributes: HistoryEvent => Option[GeneratedMessage]
  ): HistoryKind =
    val selection = Field(attributes)
    new HistoryKind(fact, selection.select, selection)

// A workflow's history-event kinds, each keyed to the operation by the field `key` of its
// attributes, such as `scheduled_event_id`, and named after its fact without the prefix
// `factPrefix` the kinds' facts share (`nexusOperationTimedOut` is the kind `timedOut`). The history
// is read once the workflow has closed, when it holds every event an operation will ever have, so
// each kind is exhaustive, and the read that names `evidence` in its `closes` is their closing read.
// Each kind counts in the one source `historySource`.
final case class HistoryEvidence(key: String, factPrefix: String)(val kinds: HistoryKind*):
  // The evidence of every kind, in order.
  def evidence: Vector[TypedEvidence[HistoryEvent]] = kinds.map { k =>
    val name = k.fact.toString.stripPrefix(factPrefix)
    Evidence.keyed(
      id = evidenceId(name.head.toLower +: name.tail),
      records = k.fact,
      source = historySource,
      from = WorkflowHistory.keyedEvent(k.selection, key),
      operation = k.selection,
      commitment = Commitment.reported,
      exhaustive = true
    )
  }.toVector

// The one source a workflow's history-event kinds count in.
def historySource = sourceId("history")

// The request field, of a message `M`, that an input of an action sets when its deadline expires:
// `scheduleToStart.sets(_.getScheduleToStartTimeout)`.
final class DeadlineField[M](
    val input: Input[?],
    val field: M => com.google.protobuf.duration.Duration,
    val selection: Field[M, com.google.protobuf.duration.Duration]
)

object DeadlineField:
  inline def apply[M](
      input: Input[?],
      inline field: M => com.google.protobuf.duration.Duration
  ): DeadlineField[M] =
    val selection = Field(field)
    new DeadlineField(input, selection.select, selection)

extension (input: Input[?])
  // The field of `M` the input's deadline sets.
  inline def sets[M](inline field: M => com.google.protobuf.duration.Duration): DeadlineField[M] =
    DeadlineField(input, field)

final case class DeadlineBindings[M](
    action: Action[?] | Class,
    call: Command | Instruction,
    value: TypedProto[com.google.protobuf.duration.Duration],
    unset: Option[(Input[?], Command | Instruction)],
    fields: Vector[DeadlineField[M]]
) extends PerformanceBinding

// The bindings of every class of `action` the deadlines `fields` declare, one `perform` item:
// `deadlines[StartActivityExecutionRequest](client.start, startActivity, duration(2))(
// scheduleToStart.sets(_.getScheduleToStartTimeout), startToClose.sets(_.getStartToCloseTimeout))`.
// Each class sets the inputs it expires among the declared ones, each a `Timeout` of the action,
// and no other: its command is `call` with each such input's field set to `value`, in the order the
// fields are declared, keeping the call's name, as `withFields` does. A class that leaves the input
// `unset` names unset is performed from that entry's command instead, since the server refuses the
// call without it. A class that expires an input the declaration leaves out is unrealizable: no
// binding performs it, and the coverage report shows it. An action with non-Timeout inputs must
// supply a Class preset: those inputs keep its values, while its Timeout inputs must be at their
// defaults. Several presets extend the action's catalog without repeating a class. `M` is the message of `call` the fields
// are of: its request, or the protobuf a worker command carries.
object deadlines:
  def apply[M](
      action: Action[?] | Class,
      call: Command | Instruction,
      value: TypedProto[com.google.protobuf.duration.Duration],
      unset: Option[(Input[?], Command | Instruction)] = None
  )(fields: DeadlineField[M]*): Item =
    Item(binding = Some(DeadlineBindings(action, call, value, unset, fields.toVector)))
