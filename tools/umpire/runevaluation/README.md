# Local Run Evaluation

The local Run Evaluation command checks one already-executed caller-closure run without contacting
Temporal. Its two inputs are an admitted four-member execution-set directory and a separate,
existing output-root directory. The execution set must contain exactly `experiment.json`,
`runtime-configuration.json`, `experiment-run.json`, and `raw-evidence.json` plus its manifest.

From the repository root, the supported build-and-run interface is:

```sh
mkdir -p /tmp/umpire-local-results
make umpire-check-local-run-evaluation \
  SET=/absolute/path/to/admitted-four-member-run-set \
  OUTPUT_ROOT=/tmp/umpire-local-results
```

The direct command has the same two inputs:

```sh
umpire-local-run-evaluation \
  --set /absolute/path/to/admitted-four-member-run-set \
  --output-root /tmp/umpire-local-results
```

The direct form requires `umpire-local-run-evaluation` and
`temporal-run-evaluation-checker` to be installed as the verified sibling pair produced by the Make
target. There is no checker path, profile selector, network option, retry, timeout override, or
arbitrary executable hook. Input and output directories must resolve to distinct, non-overlapping
physical directories.

On successful checking the command publishes and reopens an admitted six-member set. It preserves
the four input members byte-for-byte and adds `evidence.json` and `result.json`. Publication is
immutable at:

```text
<output-root>/sets/<manifest-sha256-without-prefix>
```

Repeated evaluation of the same exact four-member set produces the same Evidence, Result,
manifest, one-line summary, and destination. The summary reports the run identity, operational,
Observation Evaluation, and semantic statuses, both new Artifact Checksums, the Lean-owned
evaluation-outcome checksum when available, the complete set checksum, manifest digest, and
destination.

The exit statuses are:

- `0`: operational `succeeded`, Observation Evaluation `accepted`, and semantic `satisfied`.
- `2`: checking and publication completed, but at least one of those statuses is not successful.
- `1`: admission, checker, construction, publication, or reporting failed.

A published semantic non-success is therefore not a tooling failure. Structured failures use
`umpire-local-run-evaluation-error/v2` on stderr and distinguish whether checking or publication
occurred. A reporting failure after publication retains the complete published destination
identity.

## Independent result dimensions

Operational success means the bounded five-phase run completed and cleanup closed with zero open
handles. It does not prove product behavior. Run Evaluation keeps these authorities separate:

1. checked Observation maps the four closed raw sources to one Evidence-backed System trace;
2. the checked Implementation Link must apply before an authoritative Feature trace exists; and
3. the unchanged Feature Property and strict Query summary decide semantic satisfaction.

`Result` retains operational, Observation Evaluation, Implementation Link, cleanup, and semantic
statuses independently. Unknown, conflict, unsupported, missing-link, or incomplete outcomes do
not become a Property violation or success.

The checker accepts at most 32 MiB in each protocol direction and has a fixed 30-second deadline.
The checked caller-closure Observation plan admits at most 4096 evidence records, and its Query
search Limit is eight candidate evaluations. These are separate from the runtime phase Limits
already bound into the input artifacts.

Raw fields retain their admitted `plain`, `redacted`, `sha256`, or `rejected` disposition. The
checked mapping declares `retain`, `redact`, `hash`, or `reject` handling for each present field;
only approved normalized values, contribution markers, or named-policy digest tokens may support a
Model coordinate. The current successful caller-closure proof contains plain mechanical values and
SHA-256 identity tokens. Cleanup facts participate in trace/source closure but do not support a
Feature Property clause.

Checking fails closed for malformed or noncanonical sets, binding or fingerprint drift, unknown
sources/kinds/fields, count or closure mismatch, invalid dispositions, missing causal support,
ambiguity, incompatible evidence, a missing/tampered/misdirected checker, timeout, stderr or
nonzero checker output, an oversized or noncanonical response, invalid Evidence/Result closure,
unsafe paths, and failed publication. No partial set is exposed before immutable publication.

The bounded live regression runs the corruption and ambiguity controls before producing a real
fn-19 execution, invokes the root command twice, reopens the six-member output, and independently
checks cleanup/source closure and the API/history-backed caller-closure Property:

```sh
go test -count=1 -run '^TestBoundedLiveCallerClosureEvaluation$' \
  ./tools/umpire/runevaluation
```

This proves one invocation-owned loopback caller-closure scenario. It does not provide replay,
minimization, promotion, formal-checker integration, remote/staging/canary execution, CI
qualification, release eligibility, or Claim Assessment.
