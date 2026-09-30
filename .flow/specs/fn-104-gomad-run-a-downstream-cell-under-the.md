# Gomad: run a downstream cell under the deterministic contract

**Plan date:** 2026-09-29

## Goal & Context
<!-- scope: business -->

Gomad v3 qualifies the Temporal functional package today: an in-process one-box cluster with
SQLite in the in-memory filesystem and loopback gRPC, built from the server's own module
(`.plans/GOMAD_MILESTONES.md`, F5–F7). The next consumer is a downstream Go module that embeds
the Temporal server as a library and adds its own services around it: a replicated storage layer
on an embedded key-value engine, a control plane that uses gossip membership, a replication
sidecar that talks to a blob store, and an integration harness that boots all of it together with
several server nodes in one test process. Nothing in Gomad was designed for a target outside the
server repository, and an assessment of such a module on 2026-09-29 found the gap is not one
thing but a set of capabilities, each small, that together decide whether the storage services
can run in-process under Gomad at all.

The assessment also found a hard wall: the downstream integration harness in its normal
configuration depends on services that run as separate processes (a wide-column base store,
write-ahead-log proxies, a coordination store). Gomad has no non-loopback network, no
subprocesses, and no model of those services. This spec delivers the capabilities that make the
downstream module's own in-process services a supported Gomad target; it records the external
service topology as a boundary, not as work.

## Architecture & Data Models
<!-- scope: technical -->

The work is organized by which layer owns the missing capability.

**C1. Downstream-module targets in the Runner.** `explore`, `qualify`, and `analyze` take the
target module from `os.Getwd()` (`cmd/gomad/internal/cli/cli.go:591`), and adapter selection reads
`<cwd>/go.mod` directly (`deterministicio/adapter_registry.go:151`) instead of the module root that
`target/internal/build/context.go` resolves. Only `qualify-set` has `--working-dir`. The three
commands gain `--working-dir`, adapter selection uses the resolved module root, and the schema
read-only mount source (`--io-ro-mount ./schema=/go.temporal.io/server/schema` in the manifests)
resolves from the server's module directory — a local `replace` or the module cache — rather than
from the working directory. A downstream module replaces `go.temporal.io/server` with the branch
that carries the `gomad` build seams; the capability review already records such a module as a
local replacement and builds it (`target/capability.go:914`). The Runner documents the environment
it forces on the build (`GOWORK=off`, `GOFLAGS=` cleared, `GOENV=off`, `CGO_ENABLED=0`,
`GOEXPERIMENT=nogreenteagc`, `-mod=readonly`): a workspace file is not honored, vendoring is
unsupported, and private-module settings must be exported variables. The toolchain root and
`gomad3-install.json` bundle discovery must work when the working directory is outside this
repository.

**C2. Compatibility packs for a foreign closure.** Packs are embedded in the `gomad` binary
(`internal/compatibilitypack/policy.go`) and bind exact module versions, sums, and source-set
digests, so every downstream dependency bump of a packed module means a new pack and a rebuilt
binary. The downstream closure reaches `syscall` and `golang.org/x/sys` through libraries no
Temporal pack covers: the embedded storage engine's filesystem layer, a structured-error library,
a DNS library, an error-reporting client, a request validator, a parser runtime, terminal
detection, `x/net` IPv4/IPv6 socket options, a metrics library, a cloud blob SDK, and a
Kubernetes network utility. Two modules the existing packs pin are already at newer versions
downstream (the Prometheus client and terminal detection). The pack flow (`discover`, `review`,
`generate --approve-review`, `check`, `qualify`) already accepts `--working-dir`; what is missing
is a reviewed downstream pack per platform, the re-pinned versions, and a decision on whether a
downstream repository can supply reviewed packs from a directory bound by digest instead of
rebuilding the binary. The default remains embedded packs; an external pack directory is a
design consideration, not a requirement of this spec.

**C3. Adapters for `remain_unsupported` imports the seams cannot reach.** The gossip membership
layer pulls in a metrics library that registers a signal handler (`os/signal`) and an address
library that shells out (`os/exec`). Both are `remain_unsupported`, so no pack may admit them,
and they live in third-party modules a downstream build tag cannot touch. The linker removes the
metrics library's import (see the baseline measurement), so linked mode needs only the
address-library adapter; closure mode, which the qualification manifests use today, needs both.
Each becomes an exact, digest-anchored adapter in the shape of the fx, SDK, and otel adapters:
version pinned in
`toolchain/version/version.json`, per-file source and replacement digests, original and
replacement inventory, per-platform prepared source-set pins. The adapter registry keeps one
version per module; a downstream module pinning a different version fails closed, and that is
documented rather than widened.

**C4. Boundary operations the in-process services perform.** Closure analysis does not see these;
each terminates the process at run time or produces unrepeatable evidence.
- **Datagram sockets.** The membership transport opens UDP alongside TCP. UDP is denied
  (`target/internal/livecap/protocol_generated.go`). The supported answer is a TCP-only transport
  supplied by the target; loopback UDP becomes a modeled operation only with COMPAT-5 evidence
  (contract, bound, transcript coverage, exact replay, negative test) and is not assumed here.
- **Advisory file locks.** The storage engine takes a lock file (`flock`/`fcntl`) on open. Either
  the target injects a lock-free or in-memory filesystem, or Gomad models advisory locks on the
  in-memory volume with the same COMPAT-5 evidence. The analysis records which.
- **Filesystem statistics.** A restore path calls `statfs` and treats failure as fatal. Needs a
  modeled answer over the in-memory volume or a target-side seam.
- **Concrete listener types.** Target code asserts `*net.TCPListener` and `*net.TCPAddr`. The
  in-memory network's listeners and addresses must satisfy those assertions or the analysis
  records the exact call sites as target findings.
- **All-interface binds.** A metrics endpoint listens on `":port"`. The boundary either
  normalizes an unspecified bind address to loopback deterministically or denies it with a clear
  finding; today's behavior is undocumented.
- **Port probing.** Listen, close, re-bind on the same port is the downstream allocation pattern
  and must be deterministic across repetitions under the in-memory network.
- **Process metrics.** The process collector reads `/proc` on linux (admitted by the existing
  linux pack) and host APIs on darwin (needs the re-pinned Prometheus client pack).
- **Long readiness waits.** The harness polls with `require.Eventually` at 180 s to 10 min. Under
  the virtual clock these are logical deadlines and can fire immediately when nothing is
  runnable; `fn-103` (seeded virtual-clock ticks) governs the policy, and this spec records the
  observed behavior per wait rather than changing the clock.

**C5. Guidance for downstream source seams.** The server closed its own closure with `gomad`
build-tag seams (`temporal/interrupt_gomad.go`, `common/config/persistence_password_gomad.go`,
`common/archiver/provider/provider_cloud_gomad.go`, `tests/testcore/flag_sql_gomad.go`, …). The
downstream module needs the same pattern for: a volume-discovery subprocess, `signal.Ignore` and
`signal.NotifyContext` in service lifecycle and test helpers, a `git` subprocess that locates the
repository root, a subprocess execution mode of its cluster harness, a CLI package (command
framework, port forwarding, hostname and name lookups) that leaks into the in-process closure,
and cloud credential chains reached through blob-store and metrics providers. A documented
convention (tag name, `_gomad.go` / `!gomad` pairing, default build unchanged, linked mode as the
measurement of what the linker already removes) lets the downstream repository do this without
reading Gomad internals.

**C6. Test-driver independence.** The downstream test wrapper hard-wires `-race`; Gomad refuses
the race detector. The qualification path is the `gomad` CLI or a manifest, never the wrapper.

## Edge Cases & Constraints
<!-- scope: technical -->

- The constraints of `.plans/GOMAD_MILESTONES.md` apply unchanged: no policy widening, no source
  translation or test rewriting, fail-closed stays, evidence over narration, per-platform
  qualification, bounded server source changes. A downstream seam is the downstream
  repository's change; this spec provides the convention and the analysis that names the sites.
- Every adapter version in the downstream module matches today (gRPC, fx, SDK, otel/sdk, x/net,
  libc, memory). That is coincidence, not contract. A version drift on either side fails
  preparation closed; the spec documents the one-version-per-module rule and does not add
  multi-version adapters.
- A local replacement of the server means no pack can admit findings inside it. The server has no
  assembly or linknames outside `tools/`, so this is harmless today; the analysis must fail loudly
  if that changes. The built-in `tools/gomad3sim` linkname allowance requires the server to be the
  main module and is unavailable downstream.
- Both downstream modules replace each other with local paths. Whichever is not the working
  directory is a local replacement; a linkname there that is only excluded by a `cgo` build
  constraint is one `CGO_ENABLED=1` away from a blocker.
- Packs are host-platform-reviewed. darwin/arm64 and linux/amd64 downstream packs are authored on
  their own hosts, as the Temporal packs were.
- The external service topology (base store, log proxies, coordination store) is out of scope;
  see Boundaries. Any in-process configuration of the downstream server that avoids those
  services is a downstream architecture decision that this spec neither makes nor waits for.

## Baseline measurement
<!-- scope: technical -->

Measured on 2026-09-29 on darwin/arm64 with the toolchain built from `gomad` at `3bc1fe643`
(go1.27.1, toolchain `ab2d4510…`, Runner `59e5be66…`), from the downstream module root, with the
server replaced by this checkout and the tags `test_dep`, `integration`, `gomad`. The replace
needed no other `go.mod` change: every shared dependency version was already identical. Both
analyses ran to completion; the raw reports name downstream packages and are not retained here.

**Closure mode** (23 s): 1,818 packages, `unsupported`, 82 blockers — 29 `remain_unsupported`,
45 `add_exact_pack`, 8 `model_operation` (the downstream module's own `syscall` and `x/sys`
imports). Three packs activated (reflect2, the darwin compute pack, the xxhash leaf pack) and
five adapters (otel/sdk, SDK, fx, x/net, gRPC). The libc and memory adapters did not activate
because the storage-only closure has no SQLite, and every `x/sys` pack binds its activation to
the libc adapter — so `x/sys/unix` and `x/sys/cpu` assembly, linknames, and `syscall` imports
that the server closure never sees as blockers are blockers here. Finding: `x/sys` admission is
coupled to the libc adapter and needs a standalone pack.

**Linked mode** (83 s, builds the test binary): `unsupported`, 78 live, 37 eliminated by the
linker. The linker removes the terminal, DNS-library, error-reporting, validator, parser-runtime,
and metrics-library `syscall` facts, both metrics-library `os/signal` imports, most CLI-only
`os/exec`, and 7 of the downstream module's 8 own `syscall` findings. What stays live:

- **11 `remain_unsupported`.** Five cloud credential chains (`os/exec`), all reached through the
  CLI package or the blob-store provider; the membership layer's address library (`os/exec`);
  and five downstream sites — the volume-discovery subprocess, the harness's subprocess mode,
  the repository-root subprocess, and two signal handlers (service lifecycle, sidecar server).
  These are the C5 seams plus one C3 adapter; under linked mode the metrics-library adapter is
  unnecessary.
- **34 `model_operation`.** The storage engine's filesystem layer accounts for nine: `chown`,
  `link`, read/write deadlines, raw descriptor and raw connection, `ReadFrom`/`WriteTo`. Raw
  descriptors also in a terminal-color library, procfs, and a downstream debug package;
  `readlink` in procfs and the blob transfer manager; DNS lookups in gRPC's DNS resolver and two
  cloud metadata clients (reachable, not necessarily called — the server's own closure carries
  the same resolver and qualifies); interface enumeration in the address library and the
  server's config package; address resolution in the validator, the address library, the
  downstream service-instance package, the membership layer (UDP), and a statsd client; one UDP
  listen; `process.kill` in a certificate proxy and `process.signal` in the harness; and the
  downstream `statfs`. This confirms C4 and adds the storage engine's `chown`/`link`/deadline
  surface, which favors injecting an in-memory filesystem over modeling each call.
- **33 `add_exact_pack`.** Seven arm64 assembly files (the storage engine's compression and
  prefix helpers, a compression library, `x/sys/unix` and `x/sys/cpu`), nine `syscall` and
  `x/sys` imports (Prometheus client at the newer version, terminal detection at the newer
  version, the storage engine's filesystem layer, a structured-error library, procfs, `x/sys`),
  and seventeen linknames (a JSON library, the storage engine and its hash map, a concurrent map,
  `x/sys`). One reviewed darwin pack covers all of them.

Two Runner observations: `analyze` accepted a relative package source from the module root and a
local server replacement without complaint, so C1 is about the working-directory flag and the
schema mount, not module resolution; and `make gomad3` needs a host Go of at least 1.27.1 whose
`GOROOT` matches the `go` on `PATH` — a version manager that exports `GOROOT` for an older Go
fails the toolchain build with a compiler-version mismatch after `GOTOOLCHAIN=local` is forced.

## Scope cut (2026-09-29)
<!-- scope: both -->

F9 qualifies downstream targets in linked mode on darwin/arm64 only. Closure-mode support (D8),
linux/amd64 packs (D9), and the seam guide (D10) moved to
`fn-105-gomad-follow-ups-deferred-scope`. The spec no longer depends on F7: nothing here needs the
CI gate. C5 below stays as analysis context; its guide is not a deliverable.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `gomad explore`, `qualify`, and `analyze` accept `--working-dir`; adapter selection and
  the schema mount source resolve from the module root and the server's module directory; a
  `go-test` target in a module outside this repository that replaces `go.temporal.io/server` with
  a local path prepares and analyzes. Errors: a relative or missing working directory, a working
  directory that is not a module root, and a schema source that cannot be resolved are invalid
  input with a named reason. README documents the forced build environment and the private-module
  and vendoring limits.
- **R2:** `gomad analyze --capability-mode=linked` over the downstream in-process cluster test
  package reports `supported` on darwin/arm64 with a reviewed downstream pack, the re-pinned
  Prometheus-client and terminal-detection packs, a standalone `x/sys` pack that does not bind
  the libc adapter, and the address library's exact adapter (closure mode and its second adapter
  moved to `fn-105` D8); the adapter carries per-file digests, inventories, per-platform pins, and a negative test that
  fails the build on an upstream edit. `compatibility-pack-qualification` qualifies the new
  requests on the host.
- **R3:** Each C4 boundary operation has a disposition recorded in the analysis with evidence:
  modeled (with the COMPAT-5 set), target-injectable (with the injection point named), or denied
  (with the exact finding the target sees). Unspecified bind addresses and concrete listener-type
  assertions have documented, deterministic behavior. No C4 item is left as "unknown".
- **R4:** *Moved to `fn-105` D10 on 2026-09-29.* A downstream-seam guide (README section or `docs/`) states the tag convention, the
  pairing rule, the default-build invariant, and how to measure closure-versus-linked
  elimination; it lists the seam classes from C5 without naming a downstream repository.
- **R5:** The downstream in-process cluster smoke test passes `gomad qualify --repeat 2` on two
  seeds on darwin/arm64 with exact replay, or every non-qualified outcome is classified as
  capability blocker, unmodeled operation, watchdog, or evidence divergence, with the finding
  recorded. This is the measurement that closes the spec; it is run from the downstream checkout
  against this branch's toolchain and is reported here, not committed there.
- **R6:** `.plans/GOMAD_MILESTONES.md` work tracking lists this spec; README and ARCHITECTURE
  record downstream-module support as a supported target shape with its limits.

## Boundaries
<!-- scope: business -->

- No model of out-of-process services (wide-column stores, log proxies, coordination stores,
  object stores). A downstream harness that requires them is unsupported under Gomad until the
  downstream module offers an in-process configuration; that configuration is not this spec's
  work.
- No multi-node simulation claims; `tools/gomad3sim` and `GOMAD3_NEXT_SIM.md` remain separate.
- No multi-version adapters, no generic `syscall`/`x/sys`/`os/exec`/`os/signal` grants, no
  external pack loading unless a later decision adds it with its own review.
- No change to the virtual-clock policy; that is `fn-103`.
- No downstream source changes are made from this repository.

## Decision Context
<!-- scope: both -->

2026-09-29, assessment of a downstream module against Gomad v3 on the `gomad` branch. Every
deterministic-I/O adapter version matched; the closure over the in-process cluster test package
held roughly two thousand non-standard packages, about forty importers of `os/exec`, `os/signal`,
or `os/user`, and about fifty importers of `syscall` or `golang.org/x/sys`. Most `os/exec` and
`os/user` importers are cloud credential chains and CLI helpers that build seams or the linker
remove, as they did for the server (F4). The residue that needs Gomad work is the Runner's
working-directory assumption, a downstream pack per platform, two adapters for
`remain_unsupported` imports under the membership layer, and the run-time boundary operations
in C4. The integration harness with embedded server nodes was judged out of reach because of its
external service topology; the in-process storage services alone were judged reachable, so the
spec is scoped to them. User direction: record capabilities only, name no downstream repository
or component.
