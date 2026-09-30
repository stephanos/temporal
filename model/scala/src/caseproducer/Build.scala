package caseproducer

import temporal.server.api.testpilot.v1.Correlated.{CorrelatedEvidenceProjection, CorrelatedEvidenceRule}
import temporal.server.api.testpilot.v1.ExpressionOuterClass.*
import temporal.server.api.testpilot.v1.InstructionOuterClass.*
import temporal.server.api.testpilot.v1.ProgramOuterClass.{Observation, Role, RoleKind, Slot}
import temporal.server.api.testpilot.v1.ValueOuterClass.*

import scala.jdk.CollectionConverters.*

/** Builders for the Program and Contract messages, mirroring model/lean/Testpilot/Authoring.lean and
  * model/lean/Temporal/Testpilot/CaseSupport.lean, so a realization reads the same in all three. */
object Build:
  def text(value: String): Value = Value.newBuilder().setTextValue(value).build()
  def bool(value: Boolean): Value = Value.newBuilder().setBoolValue(value).build()
  /** A signed integer in the protocol's decimal spelling. */
  def signedInteger(value: Long): Value = Value.newBuilder().setSignedIntegerValue(value.toString).build()
  def enumValue(name: String): Value = Value.newBuilder().setEnumValue(EnumValue.newBuilder().setName(name)).build()

  def literal(v: Value): Expression = Expression.newBuilder().setLiteral(v).build()
  private def reference(r: Reference.Builder): Expression = Expression.newBuilder().setReference(r).build()
  /** One symbolic environment resource. */
  def environment(bindingID: String): Expression = reference(Reference.newBuilder().setEnvironmentBindingId(bindingID))
  /** The current Run. */
  def run: Expression = reference(Reference.newBuilder().setRun(RunReference.getDefaultInstance))
  /** The value an evidence lift or poll is projecting. */
  def projectedValue: Expression = reference(Reference.newBuilder().setProjectedValue(ProjectedValueReference.getDefaultInstance))
  def correlatedStep(field: CorrelatedStepField, definitionID: String): Expression =
    reference(Reference.newBuilder().setCorrelatedStep(CorrelatedStepReference.newBuilder().setField(field).setDefinitionId(definitionID)))

  /** Reads the value at a path out of an operand. */
  def path(operand: Expression, path: String): Expression =
    Expression.newBuilder().setPath(PathExpression.newBuilder().setOperand(operand).setPath(path)).build()
  def present(operand: Expression): Expression =
    Expression.newBuilder().setPresent(PresentExpression.newBuilder().setOperand(operand)).build()
  def equal(left: Expression, right: Expression): Expression = Expression.newBuilder().setCompare(CompareExpression.newBuilder()
    .setOperator(ComparisonOperator.COMPARISON_OPERATOR_EQUAL).setLeft(left).setRight(right)).build()

  /** One path segment, spelled as `Testpilot.Authoring.Path.Segment.render` spells it. */
  opaque type Segment = String
  def field(name: String): Segment = name
  /** Fans out over every element of a repeated field. */
  def repeated(name: String): Segment = s"$name[*]"
  /** Selects a oneof only when its active field has this name. */
  def oneofMember(name: String, member: String): Segment = s"$name<$member>"
  def makePath(segments: Segment*): String = segments.mkString(".")

  def assign(target: String, value: Expression): RequestAssignment =
    RequestAssignment.newBuilder().setTarget(target).setValue(value).build()

  /** Invokes one unary method on an endpoint role. */
  def invokeRPC(endpointRoleID: String, method: String, assignments: Seq[RequestAssignment], reads: Seq[ResponseRead]): Instruction =
    Instruction.newBuilder().setInvokeRpc(InvokeRpc.newBuilder().setEndpointRoleId(endpointRoleID).setMethod(method)
      .addAllRequestAssignments(assignments.asJava).addAllResponseReads(reads.asJava)).build()

  /** Polls the read an evidence declaration names until the condition holds. */
  def readEvidence(evidenceID: String, endpointRoleID: String, assignments: Seq[RequestAssignment], until: Expression,
      pollIntervalMilliseconds: Long): Instruction =
    Instruction.newBuilder().setReadEvidence(ReadEvidence.newBuilder().setEvidenceId(evidenceID).setEndpointRoleId(endpointRoleID)
      .addAllRequestAssignments(assignments.asJava).setUntil(until).setPollIntervalMilliseconds(pollIntervalMilliseconds)).build()

  /** One instruction node, with an optional dispatch timeout and guard. */
  def node(id: String, instruction: Instruction, timeoutMilliseconds: Option[Long] = None, guard: Option[Expression] = None): InstructionNode =
    val b = InstructionNode.newBuilder().setInstructionId(id).setInstruction(instruction)
    timeoutMilliseconds.foreach(ms => b.setLimits(InstructionLimits.newBuilder().setTimeoutMilliseconds(ms)))
    guard.foreach(b.setGuard)
    b.build()

  def responseRead(path: String, cardinality: ReadCardinality, targets: ReadTarget*): ResponseRead =
    ResponseRead.newBuilder().setPath(path).setCardinality(cardinality).addAllTargets(targets.asJava).build()

  /** Writes into a declared Observation. */
  def observationTarget(id: String): ReadTarget = ReadTarget.newBuilder().setObservationId(id).build()

  /** A history read's lift target: the history kinds among the resolved rules, each a rule naming its
    * declaration, in the order the rules name them (`Temporal.Case.Evidence.target`). */
  def evidenceTarget(observationID: String, rules: Seq[EvidenceRule]): ReadTarget =
    ReadTarget.newBuilder().setCorrelatedEvidence(CorrelatedEvidenceProjection.newBuilder().setObservationId(observationID)
      .addAllRules(rules.filter(_.readsHistory).map(r => CorrelatedEvidenceRule.newBuilder().setEvidenceId(r.source.kindID).build()).asJava))
      .build()

  def role(id: String, kind: RoleKind, namespaceBinding: String = "", resourceBinding: String = ""): Role =
    Role.newBuilder().setRoleId(id).setKind(kind).setNamespaceBindingId(namespaceBinding).setResourceBindingId(resourceBinding).build()

  /** An opaque handle slot. */
  def handleSlot(id: String): Slot = Slot.newBuilder().setSlotId(id).setOpaqueHandle(OpaqueHandleType.getDefaultInstance).build()

  /** An Observation of one protobuf message type. */
  def messageObservation(id: String, protobufType: String): Observation = Observation.newBuilder().setObservationId(id)
    .setType(ValueType.newBuilder().setSingular(SingularType.newBuilder().setMessage(NamedType.newBuilder().setProtobufType(protobufType))))
    .build()
