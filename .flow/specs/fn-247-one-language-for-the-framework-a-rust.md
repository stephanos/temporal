# One language for the framework: Rust process ownership and Cargo development tools

> Local HTML render lens: open `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/spec.html` after generation. Rendering is pending the conductor's safe review boundary; Markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-24): "why is there backends/tla and backends/tlc ?" then, on moving process supervision out of the emitter: "yes"
> user (2026-10-24): "can w delete all __pycache__ folders?"
> user (2026-10-24): "why do we have python scripts at all?"
> user (2026-10-24): "yes file"

## Goal & Context

Remove Python from the active Model/framework development toolchain while preserving its existing behavior. Shared process ownership belongs in `umpire-backend-process`, separate from TLA+ emission. A host-only `xtask` replaces build helpers, their embedded Python glue and controls. Rust replaces the independent retained-original verifier without changing its authority or any retained artifact. Go stays where the host requires it; Bash remains orchestration.

Developers keep the existing backend APIs, guest command arguments, statuses, deterministic bytes and safety controls. The post-fn-237.3 integrated guest is the comparison baseline. This work changes no Model, checker meaning, protocol, IR or Temporal feature behavior.

## Architecture & Data Models

`umpire-backend-process` owns the existing `Limits`, `Refusal`, `Scratch`, `ProcessFailure`, Java admission, process execution and bounded-text contracts. TLA+, TLC and Apalache use it directly where they need those contracts; Apalache acquires no TLC dependency. TLA+ retains binding, emission, Snowcat and typed-value admission. Preserve existing caller signatures and compatibility reexports used by consumers, including their diagnostic wording.

Use an allocation-free process supervisor inside the process crate. Prepare executable resolution, argument/environment bytes, descriptors and bounded buffers before forking. Every path in the forked supervisor and its pre-exec target child is restricted to audited libc calls and stack-only scalar/byte operations; it cannot allocate, format, lock, run Rust destructors, unwind or call general `std` process/filesystem helpers. All child exits use `_exit`. This is the production path for focused Cargo tests and library consumers; no sibling executable, runtime Cargo invocation or warm workspace build is required. A standalone Rust helper is an alternative only if a reviewed build/discovery contract first proves all those use cases.

Linux retains the private supervisor group, target session, subreaper and adopted-descendant cleanup, including escaped process groups. Darwin retains non-root admission and descendant prevention. Other platforms refuse at admission. Signals and owner-liveness EOF preserve the existing outcome mapping. Safe Rust retains capture handles, inode checks, output concatenation, dump admission and scratch cleanup. Limits keep their existing units and enforcement: Java heap configuration is not an OS memory cap, and polling a regular output file is not a strict kernel output cap.

`xtask` is a native development-profile binary, excluded from guest/wasm dependency closures. Its existing named commands retain their positional arguments, stdout/status contracts and declared diagnostics. Internal commands also replace cache-key calculation, launcher publication and Cargo-artifact selection formerly embedded in the guest shell. The shell preserves its public interface and build/check behavior. Compile xtask before setting the guest wrapper and encoded guest flags; the compiler wrapper directly executes the built native tool, never recursively starts Cargo. Preserve the previous wrapper, jobserver descriptors and macOS loader environment.

## API Contracts

Preserve `run_process` and cancellable execution's distinction between refusal, child status and retained output. Preserve timeout/cancellation output, first-signal handling, owner EOF, target signal mapping, `125` owner-loss status and `128+signal` outcomes as applicable. Do not rename `TLC refusal` as incidental cleanup.

Guest identity schemas and all hash-input encodings stay unchanged: Python-compatible ASCII escaping, sorted compact JSON for portable identities/closures, default-spaced JSON for the existing host-variant/cache encodings, and existing sidecar record bytes. Sidecars publish before compilation and disappear on failure. Closure identity retains checksums, non-dev edges across all platforms, build dependencies, edge structure, ambiguity and shared-feature unification behavior.

Guest check mode returns 0 for fresh, 2 for stale/missing and 1 for tool/build/discovery failure, reports sizes and the first differing offset, and never publishes the candidate. Freshness disables automatic installs; absent CI permits a missing-prerequisite warning/skip, while even empty CI requires failure. Probe/metadata/build/staleness errors always fail.

The retained-original verifier still checks the pinned original commit, exactly 15 manifest entries, the independent authoritative checksum manifest, every listed original and the same success statement. Neither implementation convenience nor a regenerated artifact may replace that authority.

## Edge Cases & Constraints

Execution admission requires fn-237.3 done, with its implementation and Flow closure commits present in `umpire` ancestry. Begin from that integrated tip and retain its exact guest/artifact hashes before changing tools. A native ready result alone does not clear this gate. Do not add a whole-fn-237 dependency: its later tasks follow this spec and fn-246. Preserve fn-246's whole-spec dependency on this spec. The separate fn-246/whole-fn-243 scheduling correction belongs to that plan.

Promote `libc` 0.2.191 and, under the conductor's explicit autonomous R18 amendment, the already resolved `toml` 1.1.8+spec-1.1.0 for host-only xtask parsing. The latter retains checksum `20489e00e4d8741d6be680764cc12e270655e375a20d1011e844a9c3379e678d`, disables default features and requests only existing `std`, `parse` and `serde` features. Record before/after package and feature inventories, including the guest closure. Resolve no new third-party package/version and infer no broader transitive admission. Keep workspace `unsafe_code=forbid`. The process crate alone may follow the existing runtime precedent: mirror the workspace lint policy with crate-local `unsafe_code=deny`, allowing unsafe only in its audited syscall module. All other modules forbid unsafe; no workspace-wide relaxation is allowed. Reuse safe standard-library facilities where their behavior is equivalent and use syscall wrappers only where required.

Full TOML parsing now uses the narrowly authorized existing TOML package. The no-argument installer exposes one fixed SHA-pinned archive, not a caller-supplied archive grammar. Preserve that production universe and every existing installer control. Use established system archive tools with exact-known-member filtering before numeric verbose listing, complete selected-record validation and stdout-only extraction, followed by the unchanged jar hash. Ambiguous listings and unchecked extraction remain forbidden. Prove real pinned-release parity and appropriately test GNU/BSD command behavior before switching the caller. BSD executable tests, primary-source evidence and native macOS execution are distinct receipts. If BSD tooling is absent, report the precise remaining contract evidence and consider a disposable test executable without adding a project prerequisite. Unavailable hardware does not create a new gate beyond the captured platform contract. Existing SHA subprocess support may be reused with explicit prerequisites.

The disposable archive experiment established that Python selects the last duplicate and accepts contiguous-file type7. It does not reject duplicates. Plain stdout extraction concatenates duplicate data, and first-mode-byte checks alone do not prove regular-member parity. Synthetic malformed gzip trailers also expose lazy Python versus strict tool-reader differences, but all synthetic alternatives fail the actual production archive hash before parsing. These mock-pinned research cases do not create full arbitrary-tar input parity as a new obligation. If the actual pinned-archive/system-tool proof fails, return its concrete counterexample before changing dependencies. Additional Rust tar/gzip packages or system libarchive FFI remain separately reviewed alternatives, not presumed admissions. Earlier process/guest work continues independently.

Literal equality covers deterministic artifacts, hash inputs, sidecars and declared diagnostic fixtures. Runtime, PID, absolute scratch path, compiler stack and curl progress are observed as dynamic values with structural checks, not whole-transcript byte promises. This distinction never permits normalizing guest/golden/hash differences. Source-derived build cache identifiers can change when the tool changes; the normalized guest bytes cannot. Retain historical Python commands as historical evidence outside the active toolchain.

## Acceptance Criteria

- **R1:** `umpire-backend-process` owns the moved public support and a Rust supervisor; TLA+ contains no process implementation or Python; TLA+, TLC and Apalache use the new owner without an Apalache-to-TLC dependency. Existing signatures, statuses, diagnostics, Java pin, capture/dump/scratch contracts and platform behavior are equivalent. Errors: unsupported platforms and unsafe Darwin identity refuse at admission; launch, signal, timeout, cancellation and owner EOF controls retain their outcomes; actual Linux direct/escaped descendants are gone before return; output substitution, malformed UTF-8, bounded reads and cleanup failures retain their checks. The unsafe policy compiles without weakening other crates.
- **R2:** xtask replaces the active helper and embedded Python responsibilities with equivalent arguments, statuses, deterministic output bytes and compiler-input identities; shell/Make callers use it; post-fn-237.3 guest bytes compare exactly and freshness passes without refreshing the pin. Errors: malformed inputs/TOML/JSON/sidecars, missing/ambiguous closures, native-macro collisions, missing/probe-failed tools, stale artifacts, launch/compiler failures, installation hashes/locks/symlinks/concurrency and malformed retired-name streams retain their previous fail/skip behavior; a changed deterministic byte fails the equivalence harness.
- **R3:** Every existing Python unittest and Rust control that constructs Python children has a Rust-owned counterpart; the ordinary gate executes those counterparts and invokes no Python; no tracked Python file remains anywhere under the active Model tree. The retained-original verifier preserves its exact 15-original authority and failure controls. Errors: missing/extra/changed originals or authority mismatch fail; platform-limited controls are reported honestly; no deletion of a test, verifier responsibility or frozen artifact can satisfy this criterion.
- **R4:** Current prerequisites, crate/module ownership documentation and the gate describe the Rust toolchain and process platform/safety policy; design evidence records the one-language decision, libc/R18 accounting, byte-equivalence proof and measured `make model-time`. Errors: unmet dependency admission remains explicit; missing platform evidence and timing overruns are reported rather than passed off as certification; no error surface beyond the documented gates.

## Early proof point

Task fn-247-one-language-for-the-framework-a-rust.4 proves cold focused-consumer execution plus actual orphan cleanup after .1 pins public process behavior and establishes the extraction. If .4 fails, reconsider the supervisor's fork-safety/production path before porting build tools in .2 and its successors.

## Boundaries

No Model, DSL, protocol, IR, checker semantics or Temporal feature refactor; only the exact compatibility verifier migration reaches the feature tree. No new public tool format, guest pin refresh, golden acceptance or compiler/dependency upgrade. No frozen-archive or retained-original modification. No change to historical Flow Python evidence or Flow tooling. No Windows support claim, remote operation or history rewrite. rIC3 remains deferred and consumes the process/xtask patterns only when revived.

## Decision Context

2026-10-24, [owner]: extract process ownership from the TLA+ emitter and remove the Model toolchain's Python helpers. libc promotion is expressly approved. The previous claim that this work does not overlap fn-237 is replaced by the committed fn-237.3 admission gate.

2026-10-24, [conductor] under the owner's autonomous milestone goal: amend the libc-only direct-promotion wording for exactly the already resolved host-only TOML package/version/checksum and minimal existing parse features above. This preserves full input grammar and authorizes no new resolved package/version. The same-family primary review must examine that amendment and its before/after feature and guest-byte evidence. Registry source establishes feature names; a live documentation lookup was unavailable during planning, so no release-date verification is claimed.

[research] Preserve API/diagnostic spellings and source authority; keep Python during an intermediate extraction until its Rust replacement passes; port the full control inventory before deleting its sources; compare against the actual post-predecessor artifact; serialize Cargo, lock, helper dispatch and codegen ownership. Reject an unbuilt runtime sibling binary and a TOML subset as unsupported shortcuts. [conductor-accepted research] The28-case disposable archive experiment distinguishes the fixed production pin from mock-pinned generic-reader behavior. No extra package is admitted, no duplicate-refusal or arbitrary-tar contract is added, and actual pinned-release/GNU/BSD proof remains required as described above.

## Quick commands

```bash
cd model && mise exec -- cargo test -p umpire-backend-process --locked
cd model && mise exec -- cargo test -p xtask --locked
```

Run these separately from the repository root after implementation. Focused tasks own their listed checks; the final task runs the complete default/required-backend gates and timing once after integration.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `umpire-backend-process` owns the moved public support and a Rust supervisor; TLA+ contains no process implementation or Python; TLA+, TLC and Apalache use the new owner without an Apalache-to-TLC dependency. Existing signatures, statuses, diagnostics, Java pin, capture/dump/scratch contracts and platform behavior are equivalent. Errors: unsupported platforms and unsafe Darwin identity refuse at admission; launch, signal, timeout, cancellation and owner EOF controls retain their outcomes; actual Linux direct/escaped descendants are gone before return; output substitution, malformed UTF-8, bounded reads and cleanup failures retain their checks. The unsafe policy compiles without weakening other crates. | fn-247-one-language-for-the-framework-a-rust.1, fn-247-one-language-for-the-framework-a-rust.4 | — |
| R2 | xtask replaces the active helper and embedded Python responsibilities with equivalent arguments, statuses, deterministic output bytes and compiler-input identities; shell/Make callers use it; post-fn-237.3 guest bytes compare exactly and freshness passes without refreshing the pin. Errors: malformed inputs/TOML/JSON/sidecars, missing/ambiguous closures, native-macro collisions, missing/probe-failed tools, stale artifacts, launch/compiler failures, installation hashes/locks/symlinks/concurrency and malformed retired-name streams retain their previous fail/skip behavior; a changed deterministic byte fails the equivalence harness. | fn-247-one-language-for-the-framework-a-rust.2, fn-247-one-language-for-the-framework-a-rust.5, fn-247-one-language-for-the-framework-a-rust.6, fn-247-one-language-for-the-framework-a-rust.7 | — |
| R3 | Every existing Python unittest and Rust control that constructs Python children has a Rust-owned counterpart; the ordinary gate executes those counterparts and invokes no Python; no tracked Python file remains anywhere under the active Model tree. The retained-original verifier preserves its exact 15-original authority and failure controls. Errors: missing/extra/changed originals or authority mismatch fail; platform-limited controls are reported honestly; no deletion of a test, verifier responsibility or frozen artifact can satisfy this criterion. | fn-247-one-language-for-the-framework-a-rust.2, fn-247-one-language-for-the-framework-a-rust.3, fn-247-one-language-for-the-framework-a-rust.4, fn-247-one-language-for-the-framework-a-rust.5, fn-247-one-language-for-the-framework-a-rust.6, fn-247-one-language-for-the-framework-a-rust.7 | — |
| R4 | Current prerequisites, crate/module ownership documentation and the gate describe the Rust toolchain and process platform/safety policy; design evidence records the one-language decision, libc/R18 accounting, byte-equivalence proof and measured `make model-time`. Errors: unmet dependency admission remains explicit; missing platform evidence and timing overruns are reported rather than passed off as certification; no error surface beyond the documented gates. | fn-247-one-language-for-the-framework-a-rust.1, fn-247-one-language-for-the-framework-a-rust.3 | — |
