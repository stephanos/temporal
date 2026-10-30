# tools/umpire

The Go tooling for live Temporal Model runs: the `umpire` Runner command and the
Temporal Driver composition. [model/README.md](../../model/README.md) explains the
Model and host pipeline; [.plans/UMPIRE_MODULES.md](../../.plans/UMPIRE_MODULES.md)
states module ownership and permitted imports.

## Packages

| Package | Job |
| --- | --- |
| `cmd/umpire` | Run the shared host loop over a Model guest and publish Runs. |
| `temporaldriver` | Compose the Testpilot Temporal Driver with the raw and SDK reference workers. |

## Commands

Each command is `tools/umpire/cmd/<name>`. The Make targets build into `.build/`.

| Command | What it does | Make target |
| --- | --- | --- |
| `umpire` | The one host loop over a Model guest: `question`, `explore`, `reduce`, `replay` or `repeat`; it records and publishes Runs (see `model/umpire/hosts/go/README.md`), acting through the Temporal Driver configured below | `umpire` |

### The `umpire` command's Temporal Driver

Explore, reduce and repeat act through `common/testing/testpilot/temporal/act`, configured by the
environment; question and replay need none of it:

| Variable | Meaning |
| --- | --- |
| `UMPIRE_TEMPORAL_ADDRESS` | The frontend's `host:port` (required) |
| `UMPIRE_TEMPORAL_NAMESPACE` | The namespace the worker polls; the World's Start request must name it (required) |
| `UMPIRE_TEMPORAL_TASK_QUEUE` | The task queue the worker polls; the World's Start request must name it (required) |
| `UMPIRE_TEMPORAL_CREATE` | `true` registers the namespace before the first Run and deletes it after the last |
| `UMPIRE_TEMPORAL_RPC_TIMEOUT` | Bound of each call that is not a read (default 10s) |
| `UMPIRE_TEMPORAL_WORKER_POLL_TIMEOUT` | Bound of each worker poll the Driver admits (default 5s); shorter than the frontend's 2s is refused |
| `UMPIRE_TEMPORAL_REDACT` | Comma-separated fields removed from the Run once judged, each a registered message's full name and a field path in it; an unknown one is refused |

An invalid configuration exits 3 before any Run. `make umpire-check-model-slice` runs the live
activity slice through the command.

## Checks

```sh
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire ./tools/umpire/temporaldriver ./common/testing/testpilot/...
make umpire-check-live-tests
```
