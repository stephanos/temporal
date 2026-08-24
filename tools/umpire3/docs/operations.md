# Operating Umpire3

Choose the least-authoritative eligible Deployment profile. Local and CI profiles may bind in-process Environment adapters;
hard-budget and canary profiles require a killable worker command. Remote configuration requires an
HTTPS origin, build/configuration attestation, unique namespace and task queue, explicit capability
intersection, and separately supplied authentication. Secrets are excluded from profile values and
artifacts.

Run `make umpire3-check` for model, generation, dependency, and Go gates. Run
`make umpire3-integration` for real local Temporal Nexus and Workflow Task protocol evidence. Before
qualifying a release, execute its external qualification entries unchanged in the named deployment,
retain the semantic result and attestation, and review omissions, cleanup, and evidence retention.

`make umpire3-check` also validates and freshly repeats the bounded 10-replica native benchmark.
Use `make umpire3-record-native-benchmark` only to replace the retained performance measurement;
review timing, peak memory, state/certificate bytes, checker identity, worker determinism, and both
recovery fields in the resulting report before accepting it.

Production canary approval is a v3 Ed25519-signed immutable allowlist. Configure the controller with
the reviewed signer identity and PKIX public key through `-approval-authority` and
`-approval-public-key`; keep the private approval key in the separate approval service. Execution and
cleanup both reject an unsigned approval, a different signer, or recovery metadata that does not
match the signed approval.

Provision each external gate with its deployment owner's reviewed authority identity and Ed25519
public key before running it. Keep the matching PKCS#8 private key outside the repository and pass
only its file path with `-signing-key` inside the authorized qualification job.

Run `umpire3 qualify` beside each external deployment result to produce a v3 signed receipt bound to the
canonical candidate release, immutable Experiment, complete Result artifact, evidence graph, build,
and configuration identity. Use the same Experiment digest for local in-process, CI test cluster,
remote, public-gRPC-only, and canary runs. After all vision evidence is passing, `umpire3 promote`
consumes exactly one receipt for each required profile and emits a qualified release that retains
those receipt bindings; missing, duplicate, unsigned, wrong-authority, drifted, or mixed-scenario
receipts fail closed.

Use the unified command for deployment execution:

```sh
go run -tags test_dep ./tools/umpire3/cmd/umpire3 run \
  -experiment <experiment.json> -address <host:port> -namespace <namespace> \
  -task-queue <unique-queue> -build-id <build> -profile remote-deployment \
  -output <result.json> -bundle-output <bundle.json>
```

Supply Nexus endpoint, service, and operation together or omit all three. Supply authentication only
through `UMPIRE3_TEMPORAL_API_KEY`. A replay uses the same connection flags and obtains semantic
source, seed, bounds, baseline claim, and expected capabilities from its Replay bundle. Deployment profile or
capability drift is reported as realization drift.

The local and remote Temporal Environment adapter currently advertises no fault authority. An Experiment declaring a
fault therefore fails closed before actions unless its Environment supplies a scoped `fault.Realizer`
and positive occurrence evidence. The root Nexus harness has such a local realizer; the production
canary worker deliberately refuses faulted experiments until a deployment-owned realizer is wired
through its approval policy.

Stop on unsupported capability, evidence loss, identity or lineage ambiguity, contradictory facts,
configuration drift, budget exhaustion, or cleanup uncertainty. A conforming monitor verdict with
incomplete cleanup is inconclusive.
