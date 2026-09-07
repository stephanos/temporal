# Temporal worker Driver

This package owns the SDK-worker half of the Temporal Testpilot Driver. A `Driver` keeps compatible
queue registrations alive across Runs, while each `Session` owns its prepared entrypoints,
reservation-delivery ledger, completion bridge, failure state, and bounded diagnostics.

At construction, a Profile with environment bindings selects symbolic mode and requires empty legacy
namespace, task-queue, and Nexus endpoint options. A Profile without bindings selects legacy mode and
supports only literal Case 1.0 resources. In symbolic mode, `Validate` compares the prepared binding
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
terminal release, parent terminal release, and quarantine to the delivery ledger. Callback URL,
headers, operation token, and start time cross the package boundary only through
`CompletionCapabilityFactory`; the resulting capability remains opaque and is published through
the Run's bridge.
