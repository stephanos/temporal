# Umpire local execution

The current executable slice is an ordinary generated Go test. There is no `umpire-local-run`,
`umpire-run-local`, `umpire-run-tests`, or Make execution wrapper.

From the repository root, regenerate and run the first checked test with:

```sh
go run ./tools/umpire/cmd/umpire-gen-tests-go \
  tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json \
  --output tools/umpire/temporal/nexus
go test -count=1 ./tools/umpire/temporal/nexus/... \
  -run '^TestGeneratedWorkflowNexusQueryExactActionCallerClosureExecutesLocally$'
```

`umpire-gen-tests-go` is generation-only. It strictly admits the complete two-member Artifact set,
checks that the current closed Nexus adapter accepts it, and renders a deterministic `_test.go`.
The generated test embeds the exact two-space pretty JSON plus one terminal LF and retains literal
bindings for the set identity, set checksum, manifest SHA-256, and both member checksums and Behavior
Fingerprints. Any byte, version, closure, or digest drift fails before runtime IO.

The planning `ExperimentSpec` remains Lean-owned and byte-identical to the output selected by
`umpire-gen-tests`. `Temporal.System.Execution` separately owns the local profile, participant
program, RuntimeConfiguration meaning, and evidence-source contract. The Go test does not rebuild
setup, action, order, observation, termination, or cleanup intent.

## Library boundary

`tools/umpire/runner` is the reusable generated-test handoff. It accepts an already-admitted exact
two-member set, a generated digest binding, a run identity, and one closed adapter. It verifies the
binding before the adapter can construct authority or participant state, runs the bounded phase
engine in memory, and returns an admitted four-member set:

1. `umpire-experiment/v2`
2. `umpire-runtime-configuration/v2`
3. `umpire-experiment-run/v2`
4. `umpire-raw-evidence/v2`

The runner does not read or normalize bytes, publish Artifacts, select an environment, map evidence,
or evaluate a Property. `tools/umpire/runtime` owns the domain-neutral five-phase engine;
`tools/umpire/temporal/local` owns the sole invocation-local loopback authority; and
`tools/umpire/temporal/nexus` owns the exact System-derived caller-closure binding.

The profile has fixed single-attempt preparation, realization, observation, isolation, and cleanup
budgets. It retains exactly four bounded sources: participant output, workflow history, the control
receipt, and cleanup. A started operational failure or incomplete run remains an admitted Run plus
RawEvidence when its closure is valid. Cleanup is still attempted exactly once through its
independent bounded context.

This slice proves operational execution only. It does not perform Observation Evaluation,
Implementation Link checking, Property evaluation, semantic Run Evaluation, remote execution, CI
qualification, replay, promotion, or Claim Assessment. Those remain downstream components.
