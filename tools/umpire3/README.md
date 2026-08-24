# Umpire3

Umpire3 checks selected Temporal behaviors against Lean-owned semantic models. Authors describe
sparse intent with a `scenario.Scenario`; the regression facade compiles it into versioned
Experiments, executes them through an Environment, and reports only claims earned by the observed
evidence.

## Start here

- [Architecture](docs/architecture.md) defines the domain vocabulary, trust boundaries, and Go/Lean
  seam.
- [Authoring](docs/authoring.md) explains Scenario and Regression developer workflows.
- [Modeling](docs/modeling.md) explains the canonical Feature, independent System, and Refinement
  family shape.
- [Generation](docs/generation.md) defines generated, retained, fixture, experiment, and cache
  ownership.
- [Operations](docs/operations.md), [security](docs/security.md), and
  [recovery](docs/recovery.md) cover deployment and failure handling.
- [Support](docs/support.md), [verification](docs/verification.md), and the
  [wire specification](docs/spec.md) define supported interfaces and claim boundaries.
- [Veil](docs/veil.md) documents the Lean extension and its retained evidence.

## Commands

Run commands from the repository root:

```sh
make umpire3-explain EXPERIMENT=tools/umpire3/testdata/generated/update-lifecycle.json
make umpire3-check-family FAMILY=nexus-cancellation
make umpire3-check-generated
make umpire3-check
make umpire3-integration
make umpire3-clean
```

The supported CLI is `go run -tags test_dep ./tools/umpire3/cmd/umpire3`. Separate canary,
canary-worker, and participant binaries remain only where deployment or process isolation requires
them. TLA+/TLC/Apalache and the isolated Lentil project are not Umpire3 checkers.
