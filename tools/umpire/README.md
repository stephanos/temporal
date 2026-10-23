# tools/umpire

The Go tooling of the Temporal behavior model. Everything here works from the Umpire IR, the
checked-in files under `model/ir` that the Scala Models are lifted to. No package runs Scala or
reads a Scala source. [model/README.md](../../model/README.md) explains the whole pipeline and its
terms; this page says what each package is for.

## Packages

| Package | Job |
| --- | --- |
| `ir` | Loads and validates IR Models, computes totals and admits exploration declarations |
| `interp` | Interprets Values, Machines, Channels and decision traces into finite tables |
| `check` | Binds and answers Properties, Queries, compositions, refinements and progress claims, a composition's over its composed table and scoped fairness; owns `Realizer` and the bound declarations used by lowering |
| `realization` | Admits executable realizations and types their guards, operands and payloads |
| `internal/engine` | Evaluates finite table claims; private to `interp` and `check` |
| `lower` | Lowering: turns a `find` Query's witness, through the realization its machine declares, into a Testpilot Case, and generates the managed Case trees. Its Program and Contract builder is private, in `lower/internal/producer` |
| `conformance` | Model assessment: says whether a Run's evidence is explained by the Model, and what the Query's Property is on the executions that explain it |
| [`export`](export/README.md) | Writes the IR for Quint and compares its answers with the reader's |
| `lint` | Model lint: reports what a Model declares that nothing reaches, takes, asks, evidences or realizes, and its specification holes, with a coverage count per kind and each machine's per-operation modality table. It reads lowering only through what its command hands it |
| `explore` | Enumerates the candidates a Query's exploration declares, lowers each one, and serves them over the campaign and replay bridge protocol |
| `internal/cli` | What the commands share at their edge: interruption, output lines, and the rule that nothing is written under the model |

The module map, [.plans/UMPIRE_MODULES.md](../../.plans/UMPIRE_MODULES.md), states each package's
public interface and what it may import. `ir/ownership_test.go` enforces the reader's layers:
`realization` imports `interp`, `ir` imports both, and `check` imports `ir` and `interp`. None imports
Testpilot. `export` and `lint` depend only on `ir`, `interp` and `check`, and every package here has
a live caller.

The Case runtime is not here. It is Testpilot, in
[common/testing/testpilot](../../common/testing/testpilot/README.md), and it imports nothing from
this tree.

## Commands

Each command is `tools/umpire/cmd/<name>`. The Make targets build into `.build/`.

| Command | What it does | Make target |
| --- | --- | --- |
| `umpire-gen-cases` | Lowers every Query of `model/ir` and checks or rewrites a managed Case tree: `--kind model` (`model/cases`), `functional` (`tests/testcore/testpilot/testdata/generated`) or `canary` (`tools/canary/casebinding/testdata`) | `umpire-check-cases`, `umpire-gen-cases`, `umpire-check-fixtures`, `umpire-gen-fixtures`, `canary-check-case`, `canary-gen-case` |
| `umpire-lint` | Lints every IR file of `model/ir`, or the files named, and prints each file's findings and coverage summary; `--tables` adds the per-operation modality tables, `--must-not-pinned` the H5 kind. It fails on a finding no acceptance matches and on a stale acceptance, never on a count. The model gate runs it | `umpire-check-lint` |
| `umpire` | The one host loop over a Model guest: `question`, `explore`, `reduce`, `replay` or `repeat`; it records and publishes Runs (see `model/umpire/hosts/go/README.md`), acting through the Temporal Driver configured below | `umpire` |
| `umpire-ir-bridge` | Serves exploration candidates and replay reductions from `model/ir` | `umpire-ir-bridge` |

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
go test -tags test_dep ./tools/umpire/...   # every package, from the checked-in IR; no JVM, no server
make umpire-check-model                     # the model gate; it runs the line above as its last step
make umpire-check-cases                     # model/cases equals what lowering produces
make umpire-check-lint                      # every finding of model/ir is fixed or accepted
make umpire-check-exploration-bridge        # the bridge's campaign protocol
make umpire-check-replay-bridge             # the bridge's replay protocol
make umpire-check-backends                  # Quint against the reader; needs the tools export/README.md names
```

A finding is accepted in `<file>.lint.json` beside its IR file, by its kind, its machine or
composition and its subject, each acceptance with the reason it is accepted. Authors own finding
acceptances; the model gate also maintains `capability-waiver` metadata there from each
`capabilities` section. Lint preserves that metadata without reporting or judging it, and an
accepted file beside no IR file fails the run.

The reader's tests pin what each Model of `model/ir` means, and `umpire-check-cases` holds every
managed Case tree to what lowering produces, byte for byte. A change that moves either changes what
a Model means or what a Case contains, and has to be intended.
