# Gomad downstream findings

**Measurement date:** 2026-09-30. **Scope:** darwin/arm64, linked mode, an
in-process downstream cluster with a local server replacement.

The Gomad-side work in
[fn-104](../.flow/specs/fn-104-gomad-run-a-downstream-cell-under-the.md) is complete.
The consumer still needs its own source seams and injected filesystem/membership
transport. Classified blockers are not a claim that the cluster qualifies.

[fn-107](../.flow/specs/fn-107-gomad-finish-downstream-cell.md) owns the remaining
downstream implementation and successful qualification on both platforms,
including the closure-mode, Linux, and seam-guide follow-ups D8/D9/D10.

Current downstream-module behavior, `--working-dir`, forced build environment,
exact adapters, and external packs are documented in the
[README](../tools/gomad3/README.md) and
[architecture](../tools/gomad3/ARCHITECTURE.md#targets-outside-this-repository).
Packs remain downstream-owned, authored through `--compatibility-root`, and loaded
with `GOMAD3_COMPATIBILITY_PACKS`; their exact identities bind preparation and replay.

## Measurement (2026-09-30)
Run from the downstream checkout against the `gomad` branch on darwin/arm64, with the server
replaced by this checkout, in linked mode, the tags `test_dep`, `integration`, and `gomad`, and no
downstream source change. The raw reports name downstream packages and are not retained here.

| Step | Live blockers | Change |
| --- | --- | --- |
| Baseline (2026-09-29) | 78 | 11 `remain_unsupported`, 34 `model_operation`, 33 `add_exact_pack` |
| Address-library adapter and standalone `x/sys` pack | 67 | the address library's `os/exec` and 10 `x/sys` facts admitted |
| Downstream-owned external pack (23 facts, 17 packages) | 44 | every `add_exact_pack` fact admitted from a directory outside this repository |

The 44 that remain are all downstream-owned: 10 `remain_unsupported` (five of the downstream's own
subprocess and signal sites; five cloud credential chains reached through its CLI package and blob
store provider) and 34 `model_operation`, each with the disposition and injection point recorded
in the boundary table below. `gomad qualify --repeat 2 --capability-mode=linked` on the in-process cluster smoke test
classifies seeds 11 and 17 as `unsupported_target` at the first live boundary, a credential chain's
`os/exec` import. This is a capability blocker. Qualification is the next step once the downstream cuts its
source seams and injects the filesystem and membership transport; nothing on the Gomad side
remains for that.

The external pack went through `discover`, `review`, `generate --approve-review`, and `check`
against `--compatibility-root`, which exercised the whole external flow. One Gomad-side note from
the run: an artifact root under macOS's `/tmp` symlink is refused as not a directory; `/private/tmp`
works, as the store requires non-symlinked roots.

## Boundary dispositions

`TestProfileNetworkBindContract` in `runner/internal/execution` runs the `net_bind`
fixture with the deterministic profile for two seeds. The following dispositions
record the evidence and the consumer's injection points.

| Operation | Disposition | Evidence and injection point |
| --- | --- | --- |
| All-interface binds (`":port"`, `0.0.0.0`) | modeled | The in-memory network binds the loopback listener at `127.0.0.1`; no host interface is reached. `net_bind` asserts it. |
| Concrete listener types (`*net.TCPListener`, `*net.TCPAddr`) | modeled | `net.Listen` returns a real `*net.TCPListener` over an in-memory descriptor and `Addr()` a `*net.TCPAddr`; `net_bind` asserts both. |
| Port probing (listen, close, re-bind) | modeled | Port 0 allocates sequentially, a closed port can be bound again, a second bind fails; the transcript is seed-independent in `net_bind`. |
| Datagram sockets (UDP listen, UDP address resolution) | denied (`network.udp-listen`, `network.resolve-udp-address`) | Target-injectable: the membership layer accepts a transport, so the target supplies a TCP-only one. Loopback UDP is not modeled. |
| Advisory file locks, `chown`, `link`, deadlines, raw descriptors, `ReadFrom`/`WriteTo` | denied (unmodeled, and `filesystem.*` findings in linked mode) | Target-injectable: the storage engine takes its filesystem through an interface, so the target injects an in-memory filesystem, which removes all nine filesystem findings at once. |
| `statfs` | denied (the target package's own `x/sys/unix` import) | Target seam: a `gomad`-tagged file answers capacity from configuration. |
| Interface enumeration, non-literal address resolution | denied (`network.interface-addresses-*`, `network.resolve-*`) | Target-injectable: bind and advertise addresses are configured as IP literals, so the address library never enumerates interfaces. |
| DNS lookups | modeled for `localhost`, otherwise denied | Reachable through gRPC's DNS resolver and cloud metadata clients but not called when targets dial IP literals or use the passthrough resolver; the server's own qualified closure carries the same resolver. |
| Process metrics | linux: admitted (`procfs` in `temporal-functional-tests-linux-amd64`); darwin: denied at run time | Target seam: the process collector is not registered under the `gomad` tag; the collector's `x/sys` imports at the downstream version belong in the downstream's external pack. |
| `process.kill`, `process.signal` | denied | Target seam: the harness's subprocess mode and the certificate proxy stay out of the in-process closure under the `gomad` tag. |
| Long readiness waits | modeled (virtual clock) | A polling wait advances virtual time directly when nothing is runnable, so it completes as soon as the system is ready or reaches its deadline logically; `--clock-tick=forward` separates timestamps but does not shorten waits. |

## Downstream responsibilities

- Inject a filesystem and TCP-only membership transport. Configure bind/advertise
  addresses as IP literals and remove process collectors where the platform needs it.
- Keep volume-discovery, repository-root, and cluster-harness subprocesses, signal
  handlers, CLI-only providers, and cloud credential discovery out of the in-process
  target through `gomad`/`!gomad` source seams or dependency injection. Default
  production builds retain their behavior.
- Use the Gomad CLI or a qualification manifest instead of a wrapper that forces
  `-race`; race detection is a separate unsupported execution profile.
- Keep external base stores, log proxies, coordination services, and blob stores
  outside the support claim. Gomad supplies no general subprocess or external-service
  model. An in-process substitute is a downstream architecture decision.

Adapters remain embedded and exact, one version per module. Local replacement
findings cannot be admitted by an exact dependency pack; the special server-main
module simulation linkname allowance does not apply to a downstream target.
The [milestone constraints](../MILESTONES.md) govern every extension.

Closure-mode support, Linux downstream packs, and the seam guide remain D8, D9,
and D10 in [deferred follow-ups](../MILESTONES.md#f10-follow-ups-deferred-scope).
The completed capability design and September 29 closure-analysis baseline remain
in fn-104 and Git history.
