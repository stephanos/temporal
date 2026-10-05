# tools/umpire

The Go tooling of the Temporal behavior model. Everything here works from the Umpire IR, the
checked-in files under `model/ir` that the Scala Models are lifted to. No package runs Scala or
reads a Scala source. [model/README.md](../../model/README.md) explains the whole pipeline and its
terms; this page says what each package is for.

## Packages

| Package | Job |
| --- | --- |
| `model` | The reader: loads and validates an IR Model, interprets it into tables, and answers its Properties, Queries, refinements and progress claims. Its table checker is private, in `model/internal/checker` |
| `lower` | Lowering: turns a `find` Query's witness, through the realization its machine declares, into a Testpilot Case, and generates the managed Case trees. Its Program and Contract builder is private, in `lower/internal/producer` |
| `conformance` | Model assessment: says whether a Run's evidence is explained by the Model, and what the Query's Property is on the executions that explain it |
| [`export`](export/README.md) | Writes the IR for Quint and P and compares their answers with the reader's |
| `lint` | Model lint: reports what a Model declares that nothing reaches, takes, asks, evidences or realizes, and its specification holes, with a coverage count per kind and each machine's per-operation modality table. It reads lowering only through what its command hands it |
| `explore` | Enumerates the candidates a Query's exploration declares, lowers each one, and serves them over the campaign and replay bridge protocol |
| `internal/cli` | What the commands share at their edge: interruption, output lines, and the rule that nothing is written under the model |
| `internal/golden` | Test support for the frozen migration goldens in `model/testdata/migration` and `lower/testdata/migration` |

The module map, [.plans/UMPIRE_MODULES.md](../../.plans/UMPIRE_MODULES.md), states each package's
public interface and what it may import. `model/ownership_test.go` enforces those import rules: the
reader imports nothing of Testpilot, `export` and `lint` depend on the reader alone, and every
package here has a live caller.

The Case runtime is not here. It is Testpilot, in
[common/testing/testpilot](../../common/testing/testpilot/README.md), and it imports nothing from
this tree.

## Commands

Each command is `tools/umpire/cmd/<name>`. The Make targets build into `.build/`.

| Command | What it does | Make target |
| --- | --- | --- |
| `umpire-gen-cases` | Lowers every Query of `model/ir` and checks or rewrites a managed Case tree: `--kind model` (`model/cases`), `functional` (`tests/testcore/testpilot/testdata/generated`) or `canary` (`tools/canary/casebinding/testdata`) | `umpire-check-cases`, `umpire-gen-cases`, `umpire-check-fixtures`, `umpire-gen-fixtures`, `canary-check-case`, `canary-gen-case` |
| `umpire-lint` | Lints every IR file of `model/ir`, or the files named, and prints each file's findings and coverage summary; `--tables` adds the per-operation modality tables, each followed by the laws its machine is held to (read from the law sidecar), `--must-not-pinned` the H5 kind. It fails on a finding no acceptance matches and on a stale acceptance, never on a count. The model gate runs it | `umpire-check-lint` |
| `umpire-run` | Runs one Case against a Temporal deployment and reports its Verdict; with `--model`, also its Model's assessment of the Run | `umpire-run` |
| `umpire-assess` | Assesses one recorded Run of one Case under an Evaluation Profile and publishes a receipt; with `--model`, also its Model's assessment of the recorded Run | `umpire-assess`, `umpire-assess-run` |
| `umpire-ir-bridge` | Serves exploration candidates and replay reductions from `model/ir` to the two commands below | `umpire-ir-bridge` |
| `umpire-fuzz` | Runs one bounded exploration campaign against a deployment | `umpire-fuzz`, `umpire-fuzz-run` |
| `umpire-replay` | Replays one violated Run and reduces its Query | `umpire-replay`, `umpire-replay-run` |
| `umpire-repeat` | Runs a selection of the live tests many times and counts failures by signature | `umpire-repeat`, `umpire-repeat-run` |

## Checks

```sh
go test -tags test_dep ./tools/umpire/...   # every package, from the checked-in IR; no JVM, no server
make umpire-check-model                     # the model gate; it runs the line above as its last step
make umpire-check-cases                     # model/cases equals what lowering produces
make umpire-check-lint                      # every finding of model/ir is fixed or accepted
make umpire-check-exploration-bridge        # the bridge's campaign protocol
make umpire-check-replay-bridge             # the bridge's replay protocol
make umpire-check-backends                  # Quint and P against the reader; needs the tools export/README.md names
```

A finding is accepted in `<file>.lint.json` beside its IR file, by its kind, its machine or
composition and its subject, each acceptance with the reason it is accepted. The IR generator writes no
such file; an author does, and one beside no IR file fails the run.

The tests under `model/testdata/migration` and `lower/testdata/migration` hold the reader's and the
lowering's output to 1,411 frozen snapshots. A change that moves one of them changes what a Model
means or what a Case contains, and has to be intended.
