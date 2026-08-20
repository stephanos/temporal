# Umpire3

Umpire3 is an independent Lean-first semantic modeling and implementation-conformance system. Lean
owns product meaning, selected distributed-system behavior, executable exploration, and refinement.
The Go runtime consumes versioned experiments and never reimplements their state machines.

Run the complete focused gate from the repository root:

```sh
make umpire3-check
```

`make umpire3-integration` additionally runs the exported Nexus cancellation and Workflow Task
acknowledgement traces through a real Temporal cluster. `make umpire3-root` runs the retained
Umpire2 files and the separately implemented Umpire3 copies. The Umpire3 root factory uses generated
participant programs, SDK workflows, activities, Updates, Nexus operations, public history, and a
scoped local Nexus fault realizer. The cancellation/retry copy starts an asynchronous Nexus operation,
issues a real Nexus cancellation, and holds the modeled fault interval until the selected
`OnCancelOperation` occurrence is observed. Adapter unit tests remain negative controls rather than
release qualification.

Use `make umpire3-gen` after an intentional semantic-source or selected Temporal protobuf descriptor
change. It regenerates the catalog, typed Go identifiers, experiment schema, proof manifests,
experiments, typed regression constructors, and selected API projection. Review generated diffs as semantic changes; do not
hand-edit generated files. `make umpire3-check-generated` checks the same artifacts for drift.

## Write a regression

Most tests use three Umpire3 packages: structural combinators, one typed domain package, and the test
facade. Cluster setup stays in the environment factory and is not part of the scenario.

```go
func TestNexusCancellationRetry(t *testing.T) {
    operation := nexus.Operation("operation")
    scenario := nexus.Regression("nexus-cancellation", operation,
        regress.OnePath(
            operation.CancelWithRetry(),
            operation.CancellationSafety(),
        ),
    )

    umpire3test.RequireRegression(t, scenario,
        umpire3test.WithEnvironment(newTestNexusFactory(t)))
}
```

The same scenario can run against another eligible environment by changing only
`WithEnvironment`. The compiler infers dependencies and capabilities from the generated catalog,
grounds identities projected by live actions, emits a deterministic path suite, and observes the
generated property monitor after realization. Unsupported capabilities fail before environment
allocation.

Use `regress.AllPaths` with `regress.AnyOrder` when every bounded linearization matters. Compilation
fails instead of truncating if explicit path, action, state, memory, or time limits are exhausted.
`umpire3test.Explain` returns the completed paths, semantic and causal edges, grounded identity plan,
bounded omissions, and catalog digest without allocating an environment.

## Explain, run, replay, and campaign

The unified command emits `umpire3/diagnostic/v1` JSON for every subcommand:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 explain \
  -experiment tests/umpire3/testdata/update-lifecycle.json

go run -tags test_dep ./tests/umpire3/cmd/umpire3 run \
  -experiment <experiment.json> -address <host:port> -namespace <namespace> \
  -task-queue <isolated-queue> -build-id <build> -profile remote-deployment \
  -output <result.json> -bundle-output <bundle.json>

go run -tags test_dep ./tests/umpire3/cmd/umpire3 replay \
  -bundle <bundle.json> -address <host:port> -namespace <namespace> \
  -task-queue <isolated-queue> -build-id <build>
```

`replay` strictly decodes the redacted bundle, verifies its experiment digest, reuses its seed and
bounds, executes through the normal SDK runner, and classifies semantic, realization, schedule,
observation, and evidence drift. `campaign` deterministically mutates typed values, schedules,
fault occurrence/scope, and bounded topology, then executes the selected candidates through that
same runner. `make umpire3-mutation-gate` verifies that an approved cross-layer seed is discovered,
minimized, bundled, replayed, and emitted as normal `RequireRegression` source.

Set `UMPIRE3_TEMPORAL_API_KEY` outside experiment, result, and replay files when a remote endpoint
requires authentication. The CLI never accepts credentials in serialized semantic input.

## Authoring a model

1. State the user-visible contract in `model/Temporal/Product` without task, shard, persistence, or
   ownership mechanics.
2. Add the smallest threatening mechanism under `model/Temporal/System` and name every assumption.
3. Represent executable transitions with a proved equivalence to the relational `Step`.
4. Carry product traces in system transition results, then prove the system-to-product relation.
5. Add permitted examples, forbidden examples, and a mutation that fails a theorem or emits a
   counterexample.
6. Export only well-formed traces with bounds, assumptions, theorem identity, source hash, and tool
   version.
7. Add transition dependencies, projections, property hashes, and monitor declarations to the Lean
   catalog; run `make umpire3-gen` rather than adding Go vocabulary by hand.
8. Add a Go adapter only for stable semantic actions and normalize evidence independently of the
   expected checkpoint.

The selected protobuf manifest under `model/Temporal/API/selection.json` is the only wire import
surface. Add a message or field disposition there, implement its Lean interpretation, regenerate,
and review the descriptor closure, redaction classification, conformance fixtures, and typed fuzz
domain. Never copy generated protobuf structures into Lean by hand.

Proof, bounded exploration, and one implementation run are different claims. Missing identity,
causality, ordering, capability, cleanup, or source evidence must remain unsupported or
inconclusive.

The checked release manifest is `testdata/umpire3-1.2.json`. It links every vision goal to executable
evidence and lists the remote, gRPC-only, and production qualifications that deployment owners must
run before changing the release status from candidate to qualified. Local implementation does not
fabricate those receipts. Its v2 parity and v3 migration ledgers also preserve current partial
fidelity: candidate validation accepts declared gaps, while qualified validation requires complete,
profile-qualified parity and no partial root behavior. See `AUTHORING.md`, `MODELING.md`,
`OPERATIONS.md`, `SECURITY.md`, and `INCIDENT_RECOVERY.md` for the supported workflows.
