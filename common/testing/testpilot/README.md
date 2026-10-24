# Testpilot

Testpilot is what remains of the Go test runtime after fn-228 made the Runner the one host loop.
The Model judges every observation and answers every callback; the Runner
(`umpire.local/model/runner`, see `model/umpire/hosts/go/README.md`) asks it to Propose,
Judge and Answer, drives a Driver and records a Run. Testpilot keeps only the Temporal mechanics
that loop acts through.

| Package | Job |
| --- | --- |
| `temporal/act` | The Runner's Temporal Driver: sends each proposed request over the generated WorkflowService API, reports what the server answered as correlated events, and lets one in-process SDK worker take only the tasks a read instruction admits. It never decides what a message means. `Options.Transport` carries a credentialed connection (the canary's); the zero value is plaintext. |
| `temporal/provision` | Registers and removes a namespace for a Driver configured to create one. |
| `temporal/control` | `DeliveryGate`, which holds one activity dispatch at the server's test hook until released. |
| `duration` | Checked conversion between protobuf durations and whole milliseconds. |

The package root holds tests only: `TestTestpilotDependencyBoundary`, which keeps every
`umpire.local` import inside `temporal/act` and keeps Testpilot from importing `tests/`,
`tools/umpire` or `tools/canary`, and the protocol schema checks `make umpire-check-testpilot-protocol`
runs over `proto/internal/temporal/server/api/testpilot/v1`.

fn-228.6 retired the Case runtime: the root facade (`Prepare`, `PreparedCase`), the Program
interpreter (`internal/execution`), the CEL expression IR (`internal/ir`, `cel`, `predicate`),
Contract verification (`internal/verification`, `contract`), `casefile`, `recordedrun`,
`evaluation`, `publish`, `campaign`, `replay` and the Case Temporal Driver (`temporal`,
`temporal/{server,worker,binding,internal}`). The coverage they carried and its Runner
replacements are mapped in fn-228.6's coverage map; workflow, Nexus and workflow-scheduled
activity behaviour has no Runner Driver yet.
