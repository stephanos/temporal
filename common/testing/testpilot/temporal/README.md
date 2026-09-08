# Temporal Testpilot Driver

This package composes the controller transport in `server` with the SDK activation runtime in
`worker`. `New` freezes one Profile, keeps transport addresses and credentials in Driver
configuration, and delegates each operation to the package that owns its authority. The composite
adds no Case or scenario interpretation.

The Driver accepts exact Case 1.0 and derives request carriers and SDK worker resources from the
immutable prepared roles. The Profile binding snapshot is the sole source of namespace, task-queue,
and named Nexus endpoint values. `WorkerRoleID`, transport
connections, SDK clients, callback authority, HTTP clients, credentials, and lifecycle timeouts
remain explicit environment-owned inputs.

After Testpilot has checked the complete Driver identity, it calls the composite no-I/O `Validate`
hook before Monitor creation and `Open`. `Open` creates the server Session first, then opens an SDK worker
Session only when the prepared Program contains workflow, activity, or Nexus-handler entrypoints. The server Session supplies the private
capability bridge and generic capability factory. The SDK worker Session owns reservations, SDK
routes, callback validation, and Nexus completion transport. The server Session owns RPC effects and
generic opaque capability claims. Composite Close and quarantine preserve that split.

Reserved `StartWorkflowExecution` calls pass through an SDK worker `Carrier` before server dispatch. The
Carrier validates the prepared reservation topology and physical workflow binding, injects only the
reserved delivery header, checks the final request size, and pins the returned Temporal Run ID.
Calls without a declared carrier retain ordinary RPC request and response behavior.

The SDK's system callback identifier is resolved only against the trusted
`SystemCallbackBaseURL` supplied to `Options`. Runtime data may select the exact system callback path
but cannot supply its scheme, authority, user information, fragment, or base. Existing absolute
HTTP and HTTPS callback URLs continue through the worker package's validation.

`NewWorkflowServiceCatalog` freezes the public WorkflowService descriptor closure used by Lean Case
artifacts and Go preparation. The retained fixtures in `tests/testcore/testpilot/testdata` are
canonical ProtoJSON generated from `Temporal.Testpilot` through `Testpilot.Authoring` and the
`Protobuf.Json`-backed `Testpilot.ProtoJSON` policy; `testpilot.DecodeCaseProtoJSON` is the strict wire-boundary decoder,
and `testpilot.Prepare` owns descriptor, bounds, identity, scope, and environment admission.
The async Nexus fixture is one Case 1.0 byte sequence. Its live test prepares that sequence against
two Profiles, runs both physical environments, and proves namespace isolation plus the same
correlated satisfied Contract; the binding fingerprints and Driver identities differ.
