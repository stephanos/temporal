# Gomad v3 Next: Temporal and Platform Compatibility

> **Status note:** This is the detailed track design. Current implementation status and cross-track ordering live in [GOMADv3_NEXT.md](GOMADv3_NEXT.md). The capability designs, invariants, verification plan, and exit criteria here remain normative.

## Goal

Increase the amount of valuable Temporal code that can execute under Gomad without weakening its fail-closed deterministic contract. Compatibility work should be driven by observed workload blockers, not by attempting to model every Go or operating-system API.

The recommended progression is:

> explain rejection → measure support → unlock high-value closures → qualify another platform

The completed first deliverables were the structured target compatibility analyzer and unambiguous support reports. The next slice is governed compatibility extension driven by the measured Temporal corpus.

## What success means

Gomad should be able to answer:

- Will this package/test execute under a named Gomad platform bundle?
- If not, which dependency path, source file, linkname, foreign source, or boundary blocks it?
- Is the blocker modeled, denied, delegated, or eligible for an exact compatibility pack?
- Which Temporal workload would a proposed pack, adapter, or I/O model unlock?
- How much of the representative corpus is actually supported on each platform?
- Did a dependency or Go upgrade reduce support or change a reviewed boundary?

## Non-goals

- Running arbitrary Go programs.
- Generic subprocess, cgo, plugin, signal, or hostile-code support.
- Permitting host I/O because a forbidden operation appears unlikely to execute.
- Cross-platform replay of the same artifact.
- API-count parity with `os`, `net`, or `x/sys`.

## COMPAT-1: `gomad analyze`

Expose capability-closure review as a read-only user command:

```text
gomad analyze [target flags] -- go-test ./path/to/package --test TestName
```

The result should include:

- target and platform-bundle identity;
- supported/unsupported classification;
- canonical dependency path from target to each blocker;
- package, module version/sum, source-set digest, source file, and directive where relevant;
- exact denied capability, boundary operation, or unapproved linkname;
- selected compatibility packs and why they activated;
- deterministic profile features the target would require;
- machine-readable remediation categories: add an exact pack, add an adapter, model an operation, remove dependency, or remain unsupported.

This should reuse `ReviewCapabilityClosure` and compatibility policy data through a reporting projection. The CLI must not duplicate policy logic or turn human strings into a second source of truth.

Add `--format=json` with a versioned schema so qualification tooling can aggregate blockers. Human output should group repeated blockers and show the shortest dependency path first.

## COMPAT-2: Unambiguous support matrices

Replace the overloaded set-level meaning of `qualified` with separate evidence:

- selected and completed workloads;
- expectation matches and mismatches;
- actually supported, unsupported, failed, and infrastructure-error counts;
- replay-qualified count;
- platform bundle and corpus identities;
- reviewed boundary-diff approval state.

Add a public command to run and compare qualification sets, for example:

```text
gomad qualify-set --manifest temporal.json --output report.json
gomad compare-support --baseline old.json --candidate new.json
```

Comparison should fail CI on an unexpected support regression, a changed blocker without manifest review, replay divergence, or an unapproved boundary change. Expected unsupported cases remain useful evidence, but never contribute to the supported count.

## COMPAT-3: Tiered Temporal corpus

Grow the corpus around architectural value rather than raw test count.

### Tier 1: deterministic primitives

Retain clock, timer, future, channel, cancellation, filesystem, loopback, libc, and SQLite smoke cases. Add adversarial boundary and replay cases. These certify the mechanism.

### Tier 2: package-level Temporal behavior

Select pure-Go, single-process tests from high-value areas such as workflow state transitions, task queues, caches, retry/backoff logic, matching decisions, history state machines, persistence serialization, and worker coordination. Each selected test must document the invariant it represents and why Gomad adds value beyond ordinary repetition.

### Tier 3: composed in-process scenarios

Build purpose-specific harnesses that compose several Temporal components using explicit deterministic adapters. Keep external services out of process. These scenarios should exercise cancellation, timeout, duplicate delivery, retry, and shutdown interactions under controlled schedules and faults.

The first functional gate is `tests/gomadfunctional.TestFrontendSystemInfo`.
It starts the existing local one-box cluster and completes a frontend system-info
RPC with `test_dep,disable_grpc_modules`. It now also executes successfully under
guarded Gomad mode. The functional option uses in-memory SQLite and static
membership, server construction skips host resource detection when tracing has
no span processors, and the exact `os/signal.Stop` cleanup path is a
deterministic no-op. Unmodeled operations in `os/signal` and every other guarded
package still terminate at the runtime boundary.

Continue that gate in this order:

1. Run the functional probe through `gomad qualify` with exact successful
   replay, then add it to the checked corpus.
2. Keep guarded package linkage fail-closed and add exact runtime-negative
   coverage whenever another operation from an allowed package is modeled.
3. Separate optional providers when doing so materially shrinks the local
   target or unlocks another workload; do not make that separation a
   prerequisite for this already-executable gate.
4. Add exact packs or deterministic operations only for boundaries reached by
   another selected local workload.

AWS, GCP, Kubernetes control planes, external credential discovery, and
external-service emulation are non-goals. Their linked code may remain guarded,
but it is not approved, adapted, modeled, or exercised for Gomad.

For every tier, record multiple seeds or choice prefixes, replay qualification, actual support state, execution time, artifact growth, and blocker classification. The corpus should be required on Gomad changes and on dependency changes that alter its closure.

## COMPAT-4: Compatibility-pack development kit

Implementation uses a v2-only, exact-source policy and a governed
discover/review/approve/generate/check/qualify workflow. Active migration
requests and unapproved candidate reports live under
`tools/gomadv3/target/internal/compatibility`; exact review digests, rather
than boolean approval, gate publication. Registered adapter evidence replaces
the former arbitrary-local-replacement representation. The backoff candidate
was retired after the exact gRPC adapter removed its blockers and qualified
the workload. The Activity candidate remains denied pending the COMPAT-6 or
deterministic adapter evidence described below.

Turn exact compatibility packs into a governed extension point rather than hand-authored exceptions. Provide a generator/validator that:

- captures module path, version, sum, replacement state, package source-set digest, and exact source hashes;
- inventories requested forbidden imports, foreign files, and linknames;
- produces a human-readable diff and justification template;
- generates positive, changed-source, changed-version, and unavailable-pack tests;
- records owner, unlocked workloads, review date, and platform scope;
- refuses wildcard versions, missing sums, local replacements, or unreviewed directives.

Packs should approve narrow ABI or source facts. They must not grant generic `syscall`, `os/exec`, or `x/sys` access. If the approved package needs deterministic behavior, pair the pack with an I/O adapter and include both identities in the artifact.

The original `x/net/internal/socket` pack candidate was retired after the exact x/net adapter removed its linknames, excluded its Darwin assembly bridge, and modeled raw socket options as deterministic denials. Remaining Activity blockers must be ranked from fresh analyzer evidence rather than reviving that obsolete allowance request.

## COMPAT-5: Targeted deterministic adapters and I/O models

Use analyzer data to rank missing operations by the Temporal workloads they unlock. Each new model should have a small semantic contract, hard resource bounds, transcript coverage, exact replay, and adversarial conformance tests.

Plausible candidates, subject to corpus evidence:

- deterministic hostname lookup from an explicit hosts manifest, with no live DNS fallback;
- additional read-only filesystem metadata used by dependency initialization;
- pipes or descriptor-like primitives implemented entirely in memory;
- Unix-domain stream sockets implemented as a separate bounded in-memory model;
- package adapters that replace exact third-party time, entropy, filesystem, or SQLite boundary behavior.

Do not implement UDP, general host networking, subprocesses, or broad raw-descriptor emulation merely for API completeness. Their state spaces and host semantics are large, and subprocesses violate the current single-target containment model.

The first evidence-ranked adapter is implemented for exact
`google.golang.org/grpc@v1.80.0`. Its private bounded module replacement removes
only the Unix `net.Dialer.Control` keepalive callback and its `syscall` and
`x/sys/unix` imports while preserving the negative `KeepAlive` value. Exact
module, source, original/replacement inventory, rewritten package source-set,
and profile identities fail closed. The existing `temporal-backoff-overflow`
workload now executes and exactly replays in closure mode, raising the tier-2
baseline to 5/16 without a compatibility allowance or host fallback.

## COMPAT-6: Safer handling of transitive forbidden dependencies

The current package-level closure intentionally rejects a package that imports a forbidden package even if the target does not call the offending function. This is safe but rejects useful dependency graphs, including the current persistence case.

Do not replace it with an unsound source call graph. Evaluate a stricter, compiler-backed capability manifest:

- the pinned compiler/linker emits the exact forbidden packages, directives, foreign objects, and live boundary references included in the prepared binary;
- initialization paths remain part of the manifest;
- every listed capability must be denied, modeled, or approved by an exact pack;
- denied entry points remain deterministic fail-fast stubs if an approved package can reach them;
- the complete manifest is artifact identity and is revalidated for replay.

Keep closure review as the default until this mechanism demonstrates that it accepts real targets while still rejecting deliberately hidden init-time and indirect calls. A package-specific adapter or dependency refactor is preferable to a generic exemption.

An experimental `linked` mode now embeds a bounded canonical record in the
prepared Mach-O executable, retains the exact payload in evidence, and
revalidates both before replay. Direct, init, function-value, interface,
reflection, and inlined paths remain visible while unreachable forbidden
imports are eliminated. The current Temporal candidates still retain live
assembly, linkname, `syscall`, or forbidden-import blockers, so the qualified
baseline remains 5/16 and closure remains the default. COMPAT-6 is not complete
until a real unsupported workload qualifies without widening policy.

## COMPAT-7: Platform bundles

Represent platform support as a versioned deep module rather than scattered `GOOS` branches. A platform bundle should own:

- Go archive and checksum;
- patch and overlay identity;
- supported `GOOS/GOARCH`;
- boundary manifest and compiler fingerprints;
- deterministic I/O profile and adapters;
- runtime, host-clock, containment, and I/O qualification gates;
- compatibility packs scoped to that platform;
- immutable release and qualification evidence.

Add platforms in this order unless repository usage data argues otherwise:

1. `linux/amd64`;
2. `linux/arm64`;
3. additional macOS/Go tuples as needed.

Artifacts remain replayable only on their exact platform bundle. Cross-platform qualification compares behavior and support matrices; it does not claim byte-identical scheduling.

For Linux, replace the DTrace-specific host-clock escape audit with an equivalent privileged audit appropriate to the host. Platform qualification must exercise the full runtime and I/O corpus, not only portable host packages.

## COMPAT-8: Dependency and Go upgrade impact reports

Extend the upgrade dossier to answer:

- which target packages, compatibility packs, and adapters changed identity;
- which supported Temporal workloads became unsupported or changed behavior;
- which boundary entries were added, removed, or changed disposition;
- whether every change has explicit reviewer approval;
- whether the prior qualified bundle remains available for rollback.

Require a baseline for release qualification. A dossier with an unreviewed boundary diff may be technically complete, but it is not approved.

## Module boundaries

- **Capability analyzer:** owns dependency paths and blocker evidence; depends on closure review, not CLI formatting.
- **Compatibility policy:** owns exact approvals and pack identities; never performs target execution.
- **I/O profile:** owns deterministic semantics and boundary inventory; does not decide corpus priority.
- **Platform bundle:** composes toolchain, profile, audits, and approved packs behind one immutable identity.
- **Qualification set:** owns workload expectations and observations; reports support separately from expectation matching.

These boundaries keep one dependency workaround from becoming a generic policy bypass.

## Data flow

```text
target specification + platform bundle
  → capability closure
  → canonical analyzer evidence
  → compatibility-pack selection and deterministic profile requirements
  → prepared target or precise unsupported result
  → qualification observation
  → per-platform Temporal support matrix and upgrade comparison
```

Only canonical analyzer and qualification evidence crosses layers. Human remediation text, CI conclusions, and dashboards are projections and cannot grant capabilities.

## Error handling and failure modes

- Analyzer uncertainty is unsupported, not best-effort supported.
- Missing module sums, local replacements, changed sources, or unknown linknames fail closed.
- An adapter transcript overflow is a typed capacity outcome.
- A platform audit that cannot run is “not qualified,” distinct from a behavior failure.
- Corpus infrastructure failures are separate from target unsupported/failed outcomes.
- A support regression identifies the shortest changed dependency or boundary path.

At 10× corpus size, analysis should cache immutable `go list`/source digests by target build identity, qualification should shard by manifest entry, and reports should stream rather than hold all child output in memory. Cache reuse must revalidate module and source identities.

## Trade-offs

- Fail-closed compatibility improves trust but raises maintenance cost and rejects some safe-but-unproven targets.
- More I/O models unlock tests while increasing semantic mismatch risk with real operating systems.
- Compiler-backed capability manifests can recover safe targets but expand the pinned patch and upgrade burden.
- New platforms increase adoption and multiply every runtime, boundary, adapter, and host-audit obligation.
- A larger corpus improves confidence but can make required CI slow; deterministic sharding and tiered gates are preferable to weakening coverage.

## Verification plan

1. Analyzer golden tests for dependency paths, blocker grouping, compatibility-pack selection, and JSON stability.
2. Deliberately hostile closures covering indirect imports, init-time calls, linkname changes, foreign sources, replacements, and missing sums.
3. Adapter contract tests for success, errors, deadlines, capacity, transcript replay, and host-escape canaries.
4. Support-matrix tests proving expected unsupported cases do not count as supported.
5. Corpus regression tests tied to exact dependency and platform identities.
6. Clean-host qualification for every platform bundle.
7. Cross-platform comparison tests that allow schedule differences but require declared outcome/invariant equivalence.
8. Upgrade tests proving unapproved boundary diffs cannot be release-qualified.

## Exit criteria

### Analyzer and reporting v1

- Every current Temporal qualification rejection has a structured dependency path and remediation category.
- Human and JSON output derive from the same canonical evidence.
- Reports clearly state 3/5 supported rather than only 5/5 expectations met.

### Temporal compatibility v1

- The corpus contains representative package-level workloads beyond primitive smoke tests.
- Activity and persistence blockers are either safely resolved or deliberately retained with precise explanations.
- Supported workload regressions gate affected changes.

### Multi-platform v1

- A Linux bundle passes the same full core contract and a declared Temporal support matrix.
- Installation, artifact identity, and replay reject the wrong platform bundle.
- Platform-specific host-escape audits run in CI with required privileges.

## Recommended next slice

Use the analyzer and sixteen-workload tier-2 corpus to rank the remaining eleven blockers by workloads unlocked. The compatibility-pack development kit and first exact gRPC adapter have raised actual support to 5/16. Select another exact pack, adapter, or deterministic I/O model only when retained evidence shows that it removes every blocker for a workload without a generic exemption or host fallback; otherwise begin composed tier-3 scenarios from the expanded supported set.
