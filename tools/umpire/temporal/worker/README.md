# Temporal worker Host

This package owns the SDK-worker half of the Temporal Umpire Host. A `Host` keeps compatible
queue registrations alive across Runs, while each `Session` owns its prepared entrypoints,
reservation-delivery ledger, completion bridge, failure state, and bounded diagnostics.

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
