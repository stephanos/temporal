// -*- mode: quint -*-
//
// # The Nexus caller-side Model
//
// One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
// operation does, the protocol machine says how the server gets there and refines it, and the
// functional set runs one Query per side effect that settles the operation, once per value of
// the implementation switch.
//
// Read from top to bottom: vocabulary → the two machines → what they promise → what the set
// asks. Each machine is two modules: a `Table` module of types and pure step functions, which
// the pins and the refinement walk without dragging state along, and the machine module that
// holds the `var`s and turns the table into actions.
//
// Typecheck:  quint typecheck nexus_caller.qnt
// Scenarios:  quint test nexus_caller.qnt --main nexusProtocol
// Explore:    quint run nexus_caller.qnt --main nexusProtocol --invariant properties --mbt
// Verify:     quint verify nexus_caller.qnt --main nexusProtocol --invariant refinesProduct

// ### Entities and the input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes
// one class per assignment of them: `HandlerError(retryable)` is one constructor and two classes,
// which is the granularity an example is written at and what mirrors a protobuf oneof.
//
// Quint constructor labels are module-global, so `Resolution`'s members carry a `Resolved`
// prefix to live beside the phases of the same name; see README.
module nexusCallerVocabulary {
  import umpire.* from "./umpire"

  // An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
  // event: every history event of the operation carries that event's id.
  pure val workflow: Entity = { name: "workflow", key: "", refer: Set() }
  pure val operation: Entity =
    { name: "operation", key: "scheduledEvent", refer: Set(("caller", "workflow")) }

  type Reply = SyncSuccess | Async | OperationFailed | OperationCanceled | HandlerError(bool)

  type Resolution = ResolvedSucceeded | ResolvedFailed | ResolvedCanceled

  pure val replies: Set[Reply] =
    Set(SyncSuccess, Async, OperationFailed, OperationCanceled, HandlerError(true),
        HandlerError(false))

  pure val resolutions: Set[Resolution] =
    Set(ResolvedSucceeded, ResolvedFailed, ResolvedCanceled)

  // ### Actions
  //
  // Parties are names the feature declares by using them: `caller`, `handler`, `network`,
  // `worker`. The reserved party `system` is the server. A fault is an ordinary action of a
  // declared party, and a timer is `system` behavior the machine owns, so neither is a separate
  // kind. The action *types* are per machine (each machine's `Action` sum type); this catalog is
  // the part Quint cannot check: party, entity, schema, examples.
  pure val actions: str -> ActionDecl = Map(
    "schedule" -> { party: Caller, creates: "operation", on: "",
      schema: "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
      examples: Set() },
    "handlerReply" -> { party: Handler, creates: "", on: "operation",
      schema: "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError",
      examples: Set(("HandlerError(false)", "BadRequest"), ("HandlerError(true)", "Internal")) },
    // The Nexus HTTP completion carries no protobuf message, so it declares no schema and its
    // classes are names the realization interprets.
    "complete" -> decl(Handler).with("on", "operation"),
    "transportFault" -> decl(Network).with("on", "operation"),
    // The handler's worker stops polling. An action that names no entity is behavior no entity
    // records: the Run records the fault, but nothing recorded names the operation, so the
    // machines keep their state and record nothing at it.
    "workerStop" -> decl(Worker),
  )

  // ### The derived observation
  //
  // A retryable attempt failure writes no history event, so the attempt count is read back
  // through `DescribeWorkflowExecution`. Every other evidence name resolves against the
  // realization's catalog, which is why only a derived observation is declared.
  pure val pendingAttempts: (str, str) = ("operation", "attempts")
}
