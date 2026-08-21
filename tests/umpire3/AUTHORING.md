# Authoring Umpire3 regressions

Write a sparse Scenario with generated resources, actions, properties, and structural combinators.
Keep cluster setup and credentials in an Environment factory. `RequireRegression` compiles the
Scenario into validated Experiments first,
rejects unsupported capabilities before allocation, runs every deterministic path, qualifies the
generated monitor from evidence, cleans up, and prints a Replay-bundle-oriented failure.

Select the model family with a generated `<Family>Regression` constructor. Use `OnePath` for one
intended trace, `AllPaths(AnyOrder(...))` for every bounded valid ordering, `Before` for an author
constraint, and `During` with a generated fault such as `Drop`. Generated parameter options such as
`WithReason`, public value constructors, outcome alternatives, and fault selectors keep protocol
records out of tests. Domain handles bind runtime identities internally; lower-level code can use
`Identity` and `BindIdentity` without importing transport types. `Explain` exposes completed actions,
inferred semantic edges, identity projections, omissions, bounds, and digests without allocating an
environment.

An ordinary regression should contain only the user-visible setup, the behavior that threatens the
contract, and `Require<Property>()`. Do not copy transition mechanics into Go, inspect adapter-local
booleans, use sleeps, or turn unsupported/inconclusive into a skip. Promoted campaign source uses
this same API and must compile without campaign-only support.

Regressions normally require conformance. A deliberate negative control uses
`umpire3test.ExpectViolation()` with the same `RequireRegression` entry point; it still requires a
qualified violating observation and the flagged outcome, so an evidence failure cannot satisfy it.

Domain handles under `scenario/nexus` and `scenario/workflow` are the preferred entry point. Their
methods expose compatible semantic actions, create runtime-learned identities where needed, and
preserve the author call site. The generated top-level facade is the normal API for families without
a deeper domain handle; it supplies typed family, resource, action, property, parameter, outcome,
and fault constructors.

On failure, configure `WithCorpus(replay.NewFileCorpus(...))`. `RequireRegression` retains a
redacted digest-bound Replay bundle containing the Experiment and Result, and reports its path. Inspect a checked Experiment without allocation
with `umpire3 explain`; replay a retained failure with `umpire3 replay -bundle ...` plus the same
least-authority connection profile. Replay drift is classified instead of collapsed into a generic
test mismatch.

Read a failure from the top down: compilation categories identify the originating scenario source;
`unsupported` names the missing profile capability before allocation; `unknown` or `conflict` names
the incomplete evidence or identity obligation; a violation names its path, checkpoint, support
facts, cleanup result, retained artifact, and replay command. Do not diagnose a trust badge as a
verdict: kernel proof, checked certificate, reconstructed proof, trusted solver, tested instance,
and live conformance describe different evidence boundaries.

Adding a protobuf-backed behavior starts in `model/Temporal/API/selection.json`. Generation imports
the transitive descriptor closure, produces wire declarations and fixtures, and leaves every
selected field with an explicit semantic, opaque, ignored, or rejected disposition. Authors do not
transcribe protobuf messages or field numbers into Lean or Go.
