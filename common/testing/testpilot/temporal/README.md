# Temporal Testpilot Driver

This package composes the controller transport in `server` with the SDK activation runtime in
`sdkworker`. `New` freezes one Profile, keeps transport addresses and credentials in Driver
configuration, and delegates each operation to the package that owns its authority. The composite
adds no Case or scenario interpretation.

The Driver has two explicit resource modes. Symbolic mode accepts a Profile with environment
bindings and rejects legacy namespace, task-queue, and Nexus endpoint maps; `Validate` and `Open`
derive request carriers and SDK worker resources from the immutable prepared roles. Legacy mode accepts
only literal Case 1.0 Programs and requires the existing physical resource options. The modes cannot
be mixed, and legacy options never fill a missing symbolic binding. `WorkerRoleID`, transport
connections, SDK clients, callback authority, HTTP clients, credentials, and lifecycle timeouts
remain explicit environment-owned inputs in both modes.

After Testpilot has checked the complete Driver identity, it calls the composite no-I/O `Validate`
hook before Monitor creation and `Open`. `Open` creates the server Session first, then opens an SDK worker
Session only when the prepared Program contains workflow, activity, or Nexus-handler entrypoints. The server Session supplies the private
capability bridge and completion-capability factory. The SDK worker Session owns reservations and SDK
routes; the server Session owns RPC effects and opaque Nexus completion claims. Composite Close and
quarantine preserve that split.

Reserved `StartWorkflowExecution` calls pass through an SDK worker `Carrier` before server dispatch. The
Carrier validates the prepared reservation topology and physical workflow binding, injects only the
reserved delivery header, checks the final request size, and pins the returned Temporal Run ID.
Calls without a declared carrier retain ordinary RPC request and response behavior.

The SDK's system callback identifier is resolved only against the trusted
`SystemCallbackBaseURL` supplied to `Options`. Runtime data may select the exact system callback path
but cannot supply its scheme, authority, user information, fragment, or base. Existing absolute
HTTP and HTTPS callback URLs continue through the server package's validation.

`NewWorkflowServiceCatalog` freezes the public WorkflowService descriptor closure used by Lean Case
artifacts and Go preparation. The retained fixtures in `tests/testcore/testpilot/testdata` are
canonical ProtoJSON generated from `Temporal.Testpilot` through `Testpilot.Authoring` and the
`Protobuf.Json`-backed `Testpilot.ProtoJSON` policy; `testpilot.DecodeCaseProtoJSON` is the strict wire-boundary decoder,
and `testpilot.Prepare` owns descriptor, bounds, identity, scope, and environment admission.
The async Nexus fixture is one Case 1.1 byte sequence. Its live test prepares that sequence against
two Profiles, runs both physical environments, and proves namespace isolation plus the same
correlated satisfied Contract; the binding fingerprints and Driver identities differ.
