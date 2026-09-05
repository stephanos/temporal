# Temporal Umpire Host

This package composes the controller transport in `server` with the SDK activation runtime in
`worker`. `New` freezes one Profile, keeps endpoint addresses and credentials in Host configuration,
and delegates each operation to the package that owns its authority. The composite adds no Case or
scenario interpretation.

`Open` creates the server Session first, then opens a worker Session only when the prepared Program
contains workflow, activity, or Nexus-handler entrypoints. The server Session supplies the private
capability bridge and completion-capability factory. The worker Session owns reservations and SDK
routes; the server Session owns RPC effects and opaque Nexus completion claims. Composite Close and
quarantine preserve that split.

Reserved `StartWorkflowExecution` calls pass through a worker `Carrier` before server dispatch. The
Carrier validates the prepared reservation topology and physical workflow binding, injects only the
reserved delivery header, checks the final request size, and pins the returned Temporal Run ID.
Calls without a declared carrier retain ordinary RPC request and response behavior.

The SDK's system callback identifier is resolved only against the trusted
`SystemCallbackBaseURL` supplied to `Options`. Runtime data may select the exact system callback path
but cannot supply its scheme, authority, user information, fragment, or base. Existing absolute
HTTP and HTTPS callback URLs continue through the server package's validation.

`NewWorkflowServiceCatalog` freezes the public WorkflowService descriptor closure used by Lean Case
artifacts and Go preparation. The retained fixtures in `testdata` are canonical ProtoJSON generated
from `Temporal.CaseRuntime`; `caseartifact.DecodeProtoJSON` is the strict wire-boundary decoder.
