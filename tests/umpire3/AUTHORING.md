# Authoring Umpire3 regressions

Write semantic intent with generated resources, actions, properties, and structural combinators.
Keep cluster setup and credentials in an environment factory. `RequireRegression` compiles first,
rejects unsupported capabilities before allocation, runs every deterministic path, qualifies the
generated monitor from evidence, cleans up, and prints a replay-oriented failure.

Use `OnePath` for one intended trace, `AllPaths(AnyOrder(...))` for every bounded valid ordering,
`Before` for an author constraint, `During` for a scoped fault, `Bind` for a runtime identity, and
`WithArgument` for typed values. Use `Explain` in a focused test when changing a scenario: it exposes
completed actions, inferred semantic edges, identity projections, omissions, bounds, and digests
without allocating an environment.

An ordinary regression should contain only the user-visible setup, the behavior that threatens the
contract, and `Require<Property>()`. Do not copy transition mechanics into Go, inspect adapter-local
booleans, use sleeps, or turn unsupported/inconclusive into a skip. Promoted campaign source uses
this same API and must compile without campaign-only support.

Generated domain handles under `regress/nexus`, `regress/workflow`, `regress/activity`, and
`regress/callback` are the preferred entry point. Their methods expose compatible semantic actions,
create runtime-learned identities where needed, and preserve the author call site. Use the lower
level generated `regress` constructors only when adding a new domain package or testing the compiler.

On failure, configure `WithCorpus(artifact.NewFileCorpus(...))`. `RequireRegression` retains a
redacted digest-bound bundle and reports its path. Inspect a checked experiment without allocation
with `umpire3 explain`; replay a retained failure with `umpire3 replay -bundle ...` plus the same
least-authority connection profile. Replay drift is classified instead of collapsed into a generic
test mismatch.

Adding a protobuf-backed behavior starts in `model/Temporal/API/selection.json`. Generation imports
the transitive descriptor closure, produces wire declarations and fixtures, and leaves every
selected field with an explicit semantic, opaque, ignored, or rejected disposition. Authors do not
transcribe protobuf messages or field numbers into Lean or Go.
