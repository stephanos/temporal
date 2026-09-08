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
The workflow implementation receives arbitrary SDK arguments through `converter.EncodedValues`,
then rejects workflow types outside that allowlist before reservation admission.

Controller code reserves worker activations before dispatch and creates a `Carrier` from the
prepared carrier plan. `Carrier` delegates route injection, start-response pinning, trigger
terminal release, parent terminal release, and quarantine to the delivery ledger. The worker
validates callback URLs, resolves the SDK system callback against the trusted configured base, and
builds the protocol completion effect. Only that generic effect crosses the package boundary through
`CapabilityFactory`; its callback data remains opaque and the resulting capability is published
through the Run's bridge.
