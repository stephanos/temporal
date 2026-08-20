# Operating Umpire3

Choose the least-authoritative eligible profile. Local and CI profiles may bind in-process adapters;
hard-budget and canary profiles require a killable worker command. Remote configuration requires an
HTTPS origin, build/configuration attestation, unique namespace and task queue, explicit capability
intersection, and separately supplied authentication. Secrets are excluded from profile values and
artifacts.

Run `make umpire3-check` for model, generation, dependency, and Go gates. Run
`make umpire3-integration` for real local Temporal Nexus and Workflow Task protocol evidence. Before
qualifying a release, execute its external qualification entries unchanged in the named deployment,
retain the semantic result and attestation, and review omissions, cleanup, and evidence retention.

Use the unified command for deployment execution:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 run \
  -experiment <experiment.json> -address <host:port> -namespace <namespace> \
  -task-queue <unique-queue> -build-id <build> -profile remote-deployment \
  -output <result.json> -bundle-output <bundle.json>
```

Supply Nexus endpoint, service, and operation together or omit all three. Supply authentication only
through `UMPIRE3_TEMPORAL_API_KEY`. A replay uses the same connection flags and obtains semantic
source, seed, bounds, baseline claim, and expected capabilities from its bundle. Profile or
capability drift is reported as realization drift.

The local and remote SDK runner currently advertises no fault authority. An experiment declaring a
fault therefore fails closed before actions unless its environment supplies a scoped `fault.Realizer`
and positive occurrence evidence. The root Nexus harness has such a local realizer; the production
canary worker deliberately refuses faulted experiments until a deployment-owned realizer is wired
through its approval policy.

Stop on unsupported capability, evidence loss, identity or lineage ambiguity, contradictory facts,
configuration drift, budget exhaustion, or cleanup uncertainty. A conforming monitor verdict with
incomplete cleanup is inconclusive.
