# Temporal SDK worker Driver

This package owns the SDK-worker half of the Temporal Testpilot Driver. A `Driver` keeps compatible
queue registrations alive across Runs, while each `Session` owns its prepared entrypoints,
reservation-delivery ledger, completion bridge, failure state, and bounded diagnostics.

At construction, the Driver freezes the Profile binding snapshot. `Validate` compares the prepared binding
IDs for StartWorkflow, GetHistory, and StartNexus carriers before any registry or target effect;
equal resolved strings reached through different IDs still reject. `Open` derives the physical
namespace, queues, and named Nexus routes from the validated prepared roles. The SDK client and
worker lifecycle configuration remain caller-owned physical inputs.

Task-queue registrations are complete before a worker starts. A registration consists of the
allowlisted workflow types and Nexus service/operation pairs assigned to that physical queue.

A Program that declares no `InjectFault` shares compatible registrations across Runs, as above.
Deliberate outages belong to the registry's `outage` module. Definition preparation calls
`PlanOutages` once, and `Validate` and `Open` read the same `OutagePlan`: it resolves each fault's
task-queue role to its queue and refuses a fault on a queue no entrypoint of the Program registers a
worker on. A plan that declares a fault `Requires` a dedicated worker group keyed by its Run instead,
so an outage this Run asks for can never reach another Run's workers. `Session.InjectFault` asks the
Run's `Outage` to `Begin` the transition, which records it under the registry lock and returns the
`Settle` the effect handle's `Wait` runs: `FAULT_KIND_WORKER_STOP` stops that group's SDK worker and
suppresses its fatal-failure callback for the stop window, and `FAULT_KIND_WORKER_RESUME`
re-registers with the same structural signature; both are bounded by the instruction's own timeout.
A second transition in the same direction conflicts, and a fault on a group that is not dedicated is
an unsupported operation. Closing the Session calls `Restore`, which refuses any transition it did
not start, resumes a worker still stopped, and always reaches the registry, so a resume that cannot
finish is reported as a failed cleanup rather than leaving the Run's hold behind. Tasks queued
during the stop window wait in matching and dispatch after the resume. A transition the Driver
refuses outright, or cannot complete, is a failed instruction outcome plus a Driver invariant
diagnostic: the Run records that the fault was requested and not realized, and the Verdict is left
to the Contract.
The typed worker instructions (fn-85 R10) reach the SDK through `typed.go`: a `WorkflowCommand`
carrying `ScheduleNexusOperationCommandAttributes` becomes `workflow.ExecuteNexusOperation` with the
carried payload as an unconverted `converter.RawValue`, the carried timeouts, and the carried Nexus
header merged under the Run's routing header; a `NexusHandlerReply` becomes the handler's return, a
synchronous payload unconverted, an asynchronous reply through the completion authority the Session
publishes under its own token, a handler error with its type and retry behavior, or a failed start
as an operation error; and a `NexusOperationCompletion` becomes the completion callback's body, a
payload verbatim or a failure converted as the Nexus SDK converts a Temporal failure, canceled when
its failure info says so. Which fields of each message reach the SDK is the Driver-reach table in
`internal/execution/typed.go`; the interpreter reads only the fields that table names realized, and
`CommandTypes` names the command types this Driver realizes for `DeriveProfile`. Every registered
Nexus operation reads its input as a `RawValue`, since a schedule command carries any payload.
The workflow implementation receives arbitrary SDK arguments through `converter.EncodedValues`,
then rejects workflow types outside that allowlist before reservation admission.

Controller code reserves worker activations before dispatch and creates a `Carrier` from the
prepared carrier plan. `Carrier` delegates route injection, start-response pinning, trigger
terminal release, parent terminal release, and quarantine to the delivery ledger. The worker
validates callback URLs, resolves the SDK system callback against the trusted configured base, and
builds the protocol completion effect. Only that generic effect crosses the package boundary through
`CapabilityFactory`; its callback data remains opaque and the resulting opaque handle is published
through the Run's handle bridge.

Each actual workflow or Nexus-handler interpretation constructs a fresh private
`temporal/internal/activation.State` from its prepared entrypoint. `Evaluate` owns guard/input
reference resolution; `Admit` validates and atomically retains owned outcome fields. Both operations
consume the public plan methods' returned work, including failure charges, within the prepared
runtime ceiling. The state contains no SDK objects or completion handles and is used serially.

The worker traverses the prepared DAG and owns SDK dispatch, Nexus futures, Await timeouts,
cancellation and terminal responses. Workflow interpretation keeps SDK context checks and uses a
background Go context for bounded pure operations, including admission of canceled SDK outcomes.
Replay reconstructs activation state from the prepared plan and replayed SDK results; Nexus
redelivery uses the existing result cache without creating another interpretation. Delivery identity,
Stop, late publication checks and opaque completion authority remain with the Session and ledger.
