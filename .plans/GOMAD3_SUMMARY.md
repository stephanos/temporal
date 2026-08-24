# Gomad v3: Executive State Assessment

**Assessment date:** 2026-08-13
**Scope:** Current working tree, `tools/gomad3`, its Temporal integration, qualification manifests, CI workflow, and retained local qualification evidence.

## Executive assessment

Gomad v3 is a technically substantial but very young **internal preview** for finding and exactly replaying concurrency failures in trusted, pure-Go tests on one qualified platform: **Go 1.26.4 on `darwin/arm64`**. Within that envelope it has a credible end-to-end design: a pinned patched toolchain, seeded single-P scheduling, virtual time, a reviewed partial I/O boundary, fresh-process isolation between seeds, bounded evidence, content-addressed artifacts, resume, inspection, and replay.

It is **not ready for broad or default Temporal adoption**. It is not a model checker, does not systematically enumerate schedules, cannot reproduce schedules across program or toolchain changes, and supports only a narrow subset of Go programs and I/O. The representative Temporal qualification currently executes only three of five cases; the other two are counted as successful expected rejections. Several correctness and crash-recovery defects also remain in replay, resume, publication, and virtual networking.

The appropriate current positioning is:

- **Use now:** opt-in experiments and targeted deterministic tests whose dependency closure and I/O fit the documented contract, on trusted `darwin/arm64` machines.
- **Do not use as:** a general Temporal test runner, a cross-platform CI dependency, proof that a concurrent program is correct, a production runtime, a security sandbox, or a realistic distributed-systems/network simulator.
- **Readiness:** strong research prototype; not yet a stable product or release-grade platform.

## State at a glance

| Area | State | Assessment |
| --- | --- | --- |
| Seeded runtime determinism | Implemented, bounded | Repeats supported runtime choices for an unchanged toolchain, architecture, target, input, and seed. |
| Virtual time | Implemented | Covers native timers and deadlines when the runtime can establish quiescence; CPU loops and unsupported blocking I/O require a wall watchdog. |
| Failure evidence and replay | Strong foundation, known gap | Evidence integrity and I/O transcript replay are strong; World replay does not currently install its transition plan before execution. |
| Campaign execution | Implemented | Parallel fresh-process seed runs, failure policies, resume, retention, semantic probes, and guidance corpus exist. |
| Exploration power | Limited | Randomized seed sampling, not exhaustive or coverage-directed schedule exploration. |
| I/O model | Partial | Deterministic entropy, bounded in-memory filesystem, read-only mounts, basic loopback TCP, and one exact libc adapter. |
| Go compatibility | Narrow and fail-closed | Pure-Go/internal linking only, with three exact compatibility packs and one exact build adapter. |
| Temporal compatibility | Limited | Current representative set is three supported cases and two expected unsupported cases. |
| Platform support | Blocker | Full qualification is `darwin/arm64` only. |
| Security isolation | Not provided | Trusted tests only; raw syscalls can escape the reviewed boundary. |
| CI/release posture | Preview | Core checks run on changes, but real Temporal qualification is scheduled/manual; no release packaging or durable qualification attestation. |
| Maturity | Very early | Large test surface, but only 16 Gomad v3 commits over 2026-08-10 through 2026-08-13 in this branch and active uncommitted changes. |

## What Gomad v3 can do

### Run deterministic campaigns

The public CLI implements `explore`, `qualify`, `resume`, `replay`, `doctor`, and `inspect`. Targets can be one `go-run` package, one `go-test` package, or a prepared executable with matching provenance. `explore` supports seed ranges, parallel worker processes, per-run and overall timeouts, multiple failure policies, bounded success retention, explicit environment values, read-only mounts, semantic probes, and a guidance corpus.

Each seed runs in a fresh process with one P. The patched runtime seeds scheduling-related choices, virtualizes native time, disables asynchronous preemption and the system monitor, and advances time when no goroutine is runnable. This makes supported executions repeatable and lets timer-heavy tests run without host-time sleeps.

### Capture and reproduce failures

The runner retains full-output hashes plus bounded output, target/toolchain/profile identities, outcome classifications, I/O transcripts, optional World records, and mount snapshots. Artifacts are private, content-addressed, validated on open, and published with their manifest last. Replay validates the stored target and execution envelope and detects divergence in I/O, outcome, output, transcript, and final World evidence.

Campaigns have an append-only journal and can resume after interruption. Failure artifacts are deduplicated by signature; successes can be retained under a byte budget. This is a meaningful operational capability, not only a runtime experiment.

### Enforce a reviewed compatibility boundary

Target preparation walks the complete dependency closure and rejects unapproved cgo, external/foreign code, direct syscall-related packages, subprocesses, signals, plugins, `go:linkname` use, and `x/sys` dependencies. Exact compatibility packs allow only reviewed exceptions. This is maintenance-heavy, but it prevents unsupported dependencies from silently receiving a support claim.

The default I/O profile provides:

- deterministic entropy;
- a bounded in-memory filesystem;
- captured read-only host mounts;
- basic loopback TCP (`tcp`/`tcp4`);
- an exact adapter for `modernc.org/libc v1.72.3`;
- recorded I/O transcripts for exact replay.

The reviewed Go 1.26.4 boundary contains 129 `os`/`net` intercepts: 62 modeled, 64 explicitly denied, and 3 delegated. Denial is an important capability because it makes most boundary gaps visible rather than accidentally host-dependent.

### Provide an explicit state-modeling library

`World` is a bounded deterministic event/state module with requests, readiness, cancellation, quiescence, snapshots, records, restoration, and replay validation. It can support adapters written specifically for it; mailbox is the current pilot. Applications must integrate explicitly through `world/child`. Transparent filesystem and loopback networking are separate mechanisms and are not World fault models.

## What Gomad v3 cannot do

### It cannot prove concurrency correctness

Different seeds sample different runtime choices, but there is no exhaustive enumeration, DPOR, state-space search, stable choice numbering, forced decision sequence, or coverage proof. More seeds improve sampling only; a clean campaign does not establish absence of a bug.

Guidance ranks already-realized seeds and transcripts. It does not mutate program inputs or scenarios, inject faults, force runtime choices, shrink failures, or collect code-edge coverage. Its current “smaller” preference is artifact/payload ranking, not causal minimization.

### It cannot preserve a schedule across changes

Reproducibility requires the same Go patch and overlay, architecture, target binary/source, deterministic inputs, and seed. Program changes can shift the shared runtime random stream and every later decision. Artifacts are therefore excellent for exact reproduction of the captured binary, but they are not durable schedule specifications for a modified program.

### It cannot run general Go or Temporal workloads

The supported contract excludes cgo, external linking, race mode, plugins, foreign threads, signals, finalizers, subprocesses, arbitrary host I/O, DNS, UDP, Unix sockets, non-loopback networking, and many raw descriptor/filesystem operations. Calling `runtime.GOMAXPROCS` to raise parallelism after startup is unsupported.

This is already visible in the representative Temporal set:

- clock, future, and timer cases qualify;
- an Activity API case remains unsupported after the exact x/net adapter removes its former `go:linkname` boundary, because other closure blockers remain;
- the SQLite persistence case is expected to be unsupported because a transitive cloud-auth dependency imports `os/exec`.

The aggregate report still says `qualified=true` because all five cases matched their expected disposition. That means **3/5 supported and 5/5 expectations met**, not 5/5 Temporal support.

### It cannot simulate a distributed system realistically

The transparent network is only in-memory loopback TCP with a small API surface. There is no DNS, UDP, Unix sockets, interfaces, partitions, latency model, packet loss, reordering, bandwidth control, or multi-host behavior. World currently has only a mailbox pilot and cannot itself prove native runtime quiescence. Gomad v3 can expose concurrency behavior inside a process; it is not a replacement for multi-service or fault-injection testing.

### It cannot provide security containment

The project explicitly targets trusted tests and is not an OS sandbox. Direct raw syscalls can bypass the standard-library interception layer. Deterministic map seeds remove a production security defense. Gomad mode must never be enabled for production or treated as isolation from hostile code.

## Principal flaws and risks

### High: exact World replay is not wired end to end

Recorded World transitions are validated while opening an artifact, but replay sends only the expected initial World snapshot to the child. The child calls `world.Restore(initial, nil)`, so the recorded `ReplayPlan` is never attached. A divergent modeled operation can mutate state and run to completion before the runner notices a final evidence mismatch. This contradicts the documented guarantee that each external transition is checked before application.

**Impact:** replay still detects many divergences after execution, but it does not provide the advertised pre-apply World replay safety or first-transition guarantee. Existing replay tests use a fake executor and do not verify plan transport.

### High: deduplicated failures can make an interrupted campaign unresumable

Failure storage deduplicates by failure signature. When a later seed has the same signature, its journal entry references the first seed's artifact. Resume validation requires the referenced artifact's seed and selection ordinal to match the current journal entry, so a batch interrupted after repeated equivalent failures can fail resume validation.

**Impact:** a core recovery feature breaks on a normal campaign pattern—multiple seeds finding the same bug. There is no regression test for this case.

### High: batch publication has an unrecoverable crash window

Publication removes `.prepared`—including the resume plan and prepared target—before atomically writing `batch.json`. A crash, cancellation, disk-full condition, or fsync failure between those operations leaves neither a published batch nor a resumable batch.

**Impact:** the journal is not crash-consistent at its final commit boundary. The campaign can be rerun from scratch, but the built-in recovery path cannot recover it.

### High: the virtual TCP model has incorrect close/data race semantics

A peer can enqueue final bytes and then close its write side between the reader's initial nonblocking receive and blocking `select`. Both data and close are then ready; Go may select close and return EOF before delivering the queued bytes. Related readiness races exist around write, accept, and dial.

**Impact:** valid stream data can be truncated deterministically for some seeds. The single happy-path TCP qualification does not cover close/write races.

### High: qualification labels can create false confidence

Set-level `qualified=true` means “all observed results matched the manifest,” including expected unsupported results. Upgrade dossier qualification likewise does not require an approved or empty boundary diff. These meanings are internally consistent but unsafe for dashboards or adoption decisions.

**Needed distinction:** publish separate fields for `expectations_met`, `supported`, `unsupported`, and `boundary_changes_approved`.

### High: supported surface and evidence are too small for broad Temporal use

Core qualification is five smoke workloads, one seed, and two repetitions. The representative Temporal set is five cases, only three of which execute successfully. It does not cover real frontend, history, matching, worker, persistence, or multi-service behavior at meaningful breadth. Two equal repetitions can falsify observed determinism, but cannot establish it generally.

Real Temporal qualification runs only weekly or manually in CI, not on every affected pull request. Dependency or server changes can therefore merge without exercising the compatibility claim.

### High: host-safety cleanup trusts unvalidated process-group reports

On a supervisor protocol error, cleanup includes every positive process-group ID in the unvalidated reports, even when those IDs fail the trusted target-identity check. A corrupt or buggy supervisor report could therefore cause signals to be sent to an unrelated host process group.

**Impact:** this is a high-consequence containment risk in an error path. The supervisor is a trusted Gomad component, so this is not a direct target-code escape, but cleanup should use only the independently trusted identity.

### Medium: writer and reader limits disagree

`runs.jsonl` can grow without a campaign-level byte cap, but batch open and resume reject journals larger than 64 MiB. A sufficiently large campaign can therefore publish a batch that inspection cannot open and an interrupted batch that resume cannot continue.

The virtual network also increments listener and client ephemeral ports forever without reuse or exhaustion handling. Long or connection-heavy tests eventually receive ports greater than 65535. Per-listener and per-connection channel bounds exist, but there is no global network resource budget.

### Medium: cancellation is reported as timeout

The current runner treats any overall context error, including caller cancellation, as `overall_timeout`. Current edited tests explicitly cancel contexts while asserting timeout. This loses an important operational distinction and weakens the tests for actual deadline expiration.

### Medium: unsupported behavior is not uniformly fail-closed

Dependency closure review catches much of the unsupported surface, but some assumptions remain behavioral: trusted code must not make raw syscalls, must not raise `GOMAXPROCS`, and must not rely on finalizers or host-dependent readiness. Busy loops and polling prevent virtual-time advancement and are only stopped by an external wall watchdog.

### Medium: artifacts may retain sensitive data without governance

Explicit environment values, stdout/stderr, I/O transcripts, and captured mount data can be retained in artifacts. CI uploads qualification evidence for 90 days. No redaction, secret classification, retention policy, or user-facing sensitive-data warning was found.

### Medium: the toolchain is safe to reproduce but costly to maintain

The exact Go archive, checksum, patch, overlay, compiler fingerprints, boundary manifest, compatibility packs, and adapters are tightly coupled. This gives strong reproducibility, but every Go or dependency upgrade requires specialized regeneration and review. Only one platform is fully qualified, and the privileged DTrace clock-escape audit needs host capabilities that are unavailable in the current local environment.

## Brittle assumptions

| Assumption | Failure mode |
| --- | --- |
| One P exposes the concurrency bugs that matter | Bugs requiring true CPU parallelism or memory-race behavior remain invisible. |
| A runnable goroutine eventually blocks | Busy loops starve virtual-time advancement until the wall watchdog kills the process. |
| All external effects pass reviewed Go boundaries | Raw syscalls or unmodeled runtime behavior escape determinism. |
| Exact dependency versions stay fixed | Minor upstream changes can invalidate hashes, compatibility packs, linkname approvals, or adapters. |
| The modeled filesystem/TCP semantics are close enough | Tests may pass under simplified models but fail against real OS behavior, or vice versa. |
| Seed diversity approximates useful schedule diversity | Seeds may revisit equivalent behavior; there is no schedule-coverage metric. |
| Two equal qualification runs are sufficient evidence | Rare host escapes or nondeterminism can survive a two-run smoke check. |
| Artifact volume stays modest | Journals, transcripts, outputs, mounts, and World histories hit independent limits or create memory/disk pressure. |
| Targets and their inputs are trusted and non-sensitive | Raw syscall escape and retained evidence become security/data-handling problems. |
| `darwin/arm64` is an adequate execution venue | Most CI and developer environments cannot run the fully qualified tool directly. |

At 10× workload size, the most likely failures are artifact growth, the 64 MiB journal incompatibility, virtual-network resource/port exhaustion, longer global-lock pauses in World quiescence and snapshotting, watchdog sensitivity, and preparation/qualification timeouts. There is not yet performance or soak evidence establishing safe operating limits.

## Missing pieces and recommended order

### P0: correct the trust claims

1. Transport and attach the World replay plan; add an end-to-end test proving divergence is rejected before mutation.
2. Make failure deduplication compatible with resume, and test interruption after duplicate signatures.
3. Make batch publication crash-consistent; add fault-injection tests at each commit step.
4. Correct virtual TCP close/data ordering and add adversarial stream, accept, dial, and deadline tests.
5. Restrict process cleanup to independently validated target identities.
6. Split expectation matching from actual support in qualification and upgrade reports.

### P1: establish an adoptable contract

1. Define the product tier explicitly: opt-in deterministic unit-test runner versus general Temporal test infrastructure.
2. Expand the Temporal corpus to real package and service workloads; make it a required gate for Gomad, root dependency, and affected server changes.
3. Add qualified Linux and additional architecture support, or provide a managed hermetic Mac runner.
4. Convert additional unsupported runtime behavior into fail-fast checks where possible.
5. Bound campaign journal size before execution and align all writer/reader limits; add global network resource and port-exhaustion handling.
6. Distinguish cancellation from deadline expiry and restore real deadline tests.
7. Require an upgrade baseline plus explicit approval of boundary additions, removals, and semantic changes.
8. Publish a security and artifact-data policy covering trusted-code scope, raw syscalls, isolation, secrets, retention, and cleanup.

### P2: become release- and operations-ready

1. Produce versioned installation bundles with signed checksums/attestation, SBOM/third-party notices, commit-bound qualification evidence, and rollback instructions.
2. Add conventional `help` and `version`, clearer flag help, a minimal quickstart, and explicit notice that `doctor` writes an artifact-store probe.
3. Add nested-module lint, vet, vulnerability/license scanning, clean-host tests, multi-host soak tests, and artifact-growth benchmarks.
4. Add metrics/trend reporting, artifact pruning, compatibility-pack ownership, and a documented dependency-upgrade procedure.
5. Pursue systematic choice tracing, deterministic GC, compiler checkpoints, multi-P support, fault/input generation, and failure minimization only after the current correctness gaps are closed.

## Evidence and confidence

The assessment combined source, architecture, boundary, qualification, CI, test, and git-history review with focused current-source verification.

Verified on the current working tree:

- focused `process`, `runner`, and `testdriver` tests passed with the patched toolchain and `-tags test_dep`;
- generation, boundary, patch, and script validation passed;
- the five-case core qualification passed;
- the representative Temporal qualification reported `qualified=true`, comprising three qualified cases and two expected unsupported cases.

The ignored local upgrade dossier is not release evidence and remains `qualified=false` because the privileged DTrace host-clock audit could not run without root privileges. Live GitHub Actions history and required-check status could not be verified because the available `gh` credentials lack the Temporal organization's SAML authorization.

The working tree already contained five uncommitted Gomad v3 source/test changes before this report. Those changes were preserved and included in focused test execution. Their cancellation/timeout behavior is called out above.

Primary references:

- [Gomad v3 README](../tools/gomad3/README.md)
- [Architecture](../tools/gomad3/ARCHITECTURE.md)
- [Core qualification manifest](../tools/gomad3/qualification/core.json)
- [Temporal qualification manifest](../tools/gomad3integration/qualification/temporal.json)
- [Gomad v3 CI workflow](../.github/workflows/gomad3.yml)
