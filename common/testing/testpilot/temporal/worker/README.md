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

A Program that declares no `InjectFault` shares compatible registrations across Runs, as above. One
that declares an `InjectFault` instruction opens a dedicated worker group keyed by its Run instead,
so an outage this Run asks for can never reach another Run's workers. `Session.InjectFault` realizes
`FAULT_KIND_WORKER_STOP` by stopping that group's SDK worker and suppressing its fatal-failure
callback for the stop window, and `FAULT_KIND_WORKER_RESUME` by re-registering with the same
structural signature; both are bounded by the instruction's own timeout. Releasing the group resumes
a worker still stopped, and always reaches the registry, so a resume that cannot finish is reported
as a failed cleanup rather than leaving the Run's hold behind. Tasks queued during the stop window
wait in matching and dispatch after the resume. A transition the Driver refuses outright, or cannot
complete, is a failed instruction outcome plus a Driver invariant diagnostic: the Run records that
the fault was requested and not realized, and the Verdict is left to the Contract.
The workflow implementation receives arbitrary SDK arguments through `converter.EncodedValues`,
then rejects workflow types outside that allowlist before reservation admission.

Controller code reserves worker activations before dispatch and creates a `Carrier` from the
prepared carrier plan. `Carrier` delegates route injection, start-response pinning, trigger
terminal release, parent terminal release, and quarantine to the delivery ledger. The worker
validates callback URLs, resolves the SDK system callback against the trusted configured base, and
builds the protocol completion effect. Only that generic effect crosses the package boundary through
`CapabilityFactory`; its callback data remains opaque and the resulting capability is published
through the Run's bridge.

Each actual workflow or Nexus-handler interpretation constructs a fresh private
`temporal/internal/activation.State` from its prepared entrypoint. `Evaluate` owns guard/input
reference resolution; `Admit` validates and atomically retains owned outcome fields. Both operations
consume the public plan methods' returned work, including failure charges, within the prepared
runtime ceiling. The state contains no SDK objects or completion capabilities and is used serially.

The worker traverses the prepared DAG and owns SDK dispatch, Nexus futures, Await timeouts,
cancellation and terminal responses. Workflow interpretation keeps SDK context checks and uses a
background Go context for bounded pure operations, including admission of canceled SDK outcomes.
Replay reconstructs activation state from the prepared plan and replayed SDK results; Nexus
redelivery uses the existing result cache without creating another interpretation. Delivery identity,
Stop, late publication checks and opaque completion authority remain with the Session and ledger.
