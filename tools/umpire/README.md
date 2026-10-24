# tools/umpire

The Go tooling of the Temporal behavior model: the `umpire` command, the one host loop over a Model
guest, and the Scala-era IR reader. The reader's packages work from the Umpire IR the archived
Scala Models were lifted to; that IR is no longer checked in, so they keep only their own test
fixtures. No package runs Scala or reads a Scala source. [model/README.md](../../model/README.md) explains the whole pipeline and its
terms; this page says what each package is for.

## Packages

| Package | Job |
| --- | --- |
| `ir` | Loads and validates IR Models, computes totals and admits exploration declarations |
| `interp` | Interprets Values, Machines, Channels and decision traces into finite tables |
| `check` | Binds and answers Properties, Queries, compositions, refinements and progress claims, a composition's over its composed table and scoped fairness; owns `Realizer` and the bound declarations |
| `realization` | Admits executable realizations and types their guards, operands and payloads |
| `internal/engine` | Evaluates finite table claims; private to `interp` and `check` |
| [`export`](export/README.md) | Writes the IR for Quint and compares its answers with the reader's |
| `lint` | Model lint: reports what a Model declares that nothing reaches, takes, asks, evidences or realizes, and its specification holes, with a coverage count per kind and each machine's per-operation modality table |

The module map, [.plans/UMPIRE_MODULES.md](../../.plans/UMPIRE_MODULES.md), states each package's
public interface and what it may import. `ir/ownership_test.go` enforces the reader's layers:
`realization` imports `interp`, `ir` imports both, and `check` imports `ir` and `interp`. None imports
Testpilot. `export` and `lint` depend only on `ir`, `interp` and `check`.

fn-228.6 retired Case lowering, conformance, IR-bridge exploration and their commands
(`umpire-gen-cases`, `umpire-lint`, `umpire-ir-bridge`) with the Testpilot Case runtime; the Runner
judges every observation through the Model instead.

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
go test -tags test_dep ./tools/umpire/...   # every package; no JVM, no server
make umpire-check-backends                  # Quint against the reader; needs the tools export/README.md names
```

Many of the reader's tests still read the retired `model/ir`, `model/irgen` and
`model/temporal/features` trees and fail for that reason alone; they are not part of
`make model-check`.
