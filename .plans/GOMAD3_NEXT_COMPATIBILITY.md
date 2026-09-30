# Gomad v3 remaining compatibility work

Unlock valuable workloads without weakening the deterministic boundary.
[GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) governs the functional-test goal;
[qualification manifests](../tools/gomad3integration/README.md) and reports own
actual support and expected rejections.

The analyzer, support comparison, exact-pack authoring, platform support, and
closure/linked/guarded modes are documented in the
[README](../tools/gomad3/README.md), [CLI](../tools/gomad3/CLI.md), and
[specification](../tools/gomad3/SPEC.md#targetcapability-capability-review).
Use those sources instead of the old roadmap's platform and support-count tables.

## COMPAT-3: Tiered Temporal corpus

Keep primitive conformance, package behavior, and composed functional scenarios
separate. Select new scenarios for the invariants they exercise, such as timeout,
cancellation, retries, duplicate delivery, recovery, and shutdown.

Record platform, seeds/prefixes, exact replay, required probes, execution cost,
artifact growth, and blockers. A matching expected rejection contributes blocker
evidence, not supported coverage. The full generated functional set remains an
on-demand gate under the milestone validation rules.

## COMPAT-5: Targeted deterministic adapters and I/O models

Rank operations and exact adapters by named workloads unlocked. A new modeled
operation requires:

- a semantic contract and explicit differences from host behavior;
- hard resource bounds and typed capacity outcomes;
- transcript coverage and exact replay;
- positive, negative, error, deadline, and capacity tests;
- host-escape canaries and performance evidence.

Candidates include explicit hosts records, filesystem metadata, in-memory pipes,
Unix-domain streams, and exact dependency adapters. Require retained analyzer
findings before choosing one. Broad UDP, networking, raw descriptors, or subprocess
support needs its own model and containment design.

Packs approve exact source/ABI facts. An exception binds module version, sums,
source hashes, platform, owner, workload, and adapter identity where behavior changes.
Discovery, review, exact-digest approval, generation, checking, and qualification
remain mandatory. A pack cannot grant generic `syscall`, `x/sys`, `os/exec`, or
`os/signal` access; unsupported operations stay fail-closed.

## COMPAT-6: Safer handling of transitive forbidden dependencies

Extend compiler/linker-backed capability handling only from fresh live-boundary
findings. Initialization, indirect calls, interfaces, reflection, and inlined paths
must remain visible, and replay must revalidate the prepared binary's capability
identity before execution. Human remediation strings and report formatting cannot
grant capabilities.

Prefer an exact adapter or provider seam when it closes the workload. The remaining
downstream closure-mode adapter is deferred as D8 in
[GOMAD_FOLLOWUPS.md](GOMAD_FOLLOWUPS.md); the
[downstream measurement](GOMAD_CLOUD.md) records linked-mode limits.

## COMPAT-7: Platform bundles

The current qualified tuples and limitations belong to the README. Additional
platforms, including linux/arm64, need their own source/patch/overlay identity,
boundary inventory, adapters, packs, containment tests, host-clock audit, and core
and Temporal qualification. A portable host-package test is insufficient.

Artifacts replay on their exact platform identity. Cross-platform conformance
compares declared semantics and support; it does not require identical schedules.
Dynamic Linux clock auditing and downstream Linux packs remain D11 and D9 in the
follow-ups. The [system-level study](../docs/research/gomad/GOMAD3_OS.md) describes a
separate linux/arm64 firewall proposal; it is not a claim of implemented support.

## COMPAT-8: Dependency and Go upgrade impact reports

Build on the existing upgrade dossier with workload-level support and behavior
differences, changed pack/adapter identities, and an addressable qualified rollback
bundle. An exact reviewed boundary diff and baseline qualification are release
requirements; a completed but unapproved dossier is not a qualified release.

Cache reuse must revalidate immutable source and build identities. Analysis
uncertainty stays unsupported, unavailable audits stay unqualified, and
infrastructure failures remain separate from target outcomes.
