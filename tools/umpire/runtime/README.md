# Umpire local execution

The current executable slice is an ordinary generated Go test. There is no `umpire-local-run`,
`umpire-run-local`, `umpire-run-tests`, or Make execution wrapper.

From the repository root, regenerate and run the first checked test with:

```sh
go run ./tools/umpire/cmd/umpire-gen-tests-go \
  tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json \
  --output tools/umpire/temporal/nexus
go test -count=1 ./tools/umpire/temporal/nexus/... \
  -run '^TestHermeticCIPortability$'
```

`umpire-gen-tests-go` is generation-only. It strictly admits the complete two-member Artifact set,
checks that the current closed Nexus adapter accepts it, and renders a deterministic `_test.go`.
The generated test embeds the exact two-space pretty JSON plus one terminal LF and retains literal
bindings for the set identity, set checksum, manifest SHA-256, and both member checksums and Behavior
Fingerprints, plus the required local authority capabilities. Any byte, version, closure, authority,
or digest drift fails before runtime IO.

The planning `ExperimentSpec` remains Lean-owned and byte-identical to the output selected by
`umpire-gen-tests`. `Temporal.System.Execution` separately owns the local profile, participant
program, RuntimeConfiguration meaning, and evidence-source contract. The Go test does not rebuild
setup, action, order, observation, termination, or cleanup intent.

## Hermetic CI portability

The ordinary pinned CI command is:

```sh
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$'
```

`make umpire-check-regression` is the aggregate repository gate and invokes that same generated Go
test. Both commands consume the byte-identical canonical v2 `ExperimentSpec` used by the local path,
check its format, Artifact Checksum, Definition IDs, and Behavior Fingerprints before runtime IO,
and compose the invocation-owned loopback runner with the shared Run Evaluation authority. The CI
workflow supplies orchestration only; it does not construct or reinterpret semantic declarations.

The portability proof compares stable typed semantic meaning: operational, Observation Evaluation,
Implementation Link, Property and clause, Limit, Known Gap, and cleanup outcomes. Fresh local and CI
executions may have different runtime-scoped transport identities such as run, correlation, trace,
Evidence Link, output-set, and destination identities. Those differences do not permit the input
Artifact Checksum or Behavior Fingerprints to drift and do not turn a semantic or operational
non-success into portable success.

Evaluation Profiles, Evaluation Receipts, provenance schemas, new artifact-set versions, Claim
Assessment, remote, canary, and release work are excluded from this bounded hermetic proof.

## Library boundary

`tools/umpire/runner` is the reusable generated-test handoff. It accepts an already-admitted exact
two-member set, a generated digest binding, a run identity, and one closed adapter. It verifies the
input binding before the adapter constructs authority, then verifies that authority binding before
participant or environment construction. It runs the bounded phase engine in memory and returns an
admitted four-member set:

1. `umpire-experiment/v2`
2. `umpire-runtime-configuration/v2`
3. `umpire-experiment-run/v2`
4. `umpire-raw-evidence/v2`

The runner does not read or normalize bytes, publish Artifacts, select an environment, map evidence,
or evaluate a Property. `tools/umpire/runtime` owns the domain-neutral five-phase engine;
`tools/umpire/temporal/local` owns the sole invocation-local loopback authority; and
`tools/umpire/temporal/nexus` owns the exact System-derived caller-closure binding.

The profile has these fixed phase budgets; callers cannot override them:

| Phase | Deadline | Attempts | Record limit | Byte limit |
| --- | ---: | ---: | ---: | ---: |
| Preparation | 30 seconds | 1 | 128 | 1 MiB |
| Realization | 30 seconds | 1 | 128 | 1 MiB |
| Observation | 30 seconds | 1 | 3,584 | 12 MiB |
| Isolation | 15 seconds | 1 | 128 | 1 MiB |
| Cleanup | 15 seconds | 1 | 128 | 1 MiB |

The aggregate ceiling is 120 seconds, 5 attempts, 4,096 records, and 16 MiB. The four bounded
sources are participant output, workflow history, the control receipt, and cleanup. The raw field
dispositions are closed too:

- Retain only event type and ID, workflow/run/operation correlation IDs, command kind and status,
  cancellation callback count, open-handle count, and closed error codes.
- Represent namespace, task-queue, and endpoint identities only as named SHA-256 digest tokens.
- Never retain headers, credentials, raw Nexus or SDK payload bytes, stack traces, or arbitrary
  error text; their presence is omitted, redacted, or represented by a named SHA-256 digest token
  where the closed evidence contract calls for one.

A started operational failure or incomplete run remains an admitted Run plus RawEvidence when its
closure is valid. Cleanup is still attempted exactly once through its independent bounded context.

The runtime and runner libraries prove operational execution only. The generated portability test
composes their output with the existing downstream Observation Evaluation, Implementation Link,
Property evaluation, and Run Evaluation boundary; the libraries do not acquire that semantic
authority themselves.
