# Native development tools

From the framework root, `cargo xtask` is the general native command front door. `scripts/xtask.sh` bootstraps both native tools into `<target>/xtask-bootstrap/debug`, clearing guest wrappers and flags before building the locked development profile.

`cargo xtask gate` runs the complete framework gate. `gate --list` prints its stages;
`gate --stage <name>` runs one real stage for focused feedback. `fmt` formats the
workspace, `update` refreshes snapshots, `time` times the gate and checker benchmark,
and `validate` forces required TLC before the timed gate. Stage failure preserves
the child status and identifies its stage. Full gate children receive
`UMPIRE_QUICK_CACHE=0` and `UMPIRE_CACHE_DISABLE=1`, including backend provisioning.

Guest operations execute the dedicated [guest driver](../guest-driver/README.md). The CLI first builds the current driver sources with the local locked manifest, then replaces itself with that executable, retaining raw argv, environment, loader variables and jobserver descriptors. Its own executable is never the compiler wrapper or a guest cache input.

`retired-names <POSIX ERE>` reads the raw `git grep -n -z` stream: NUL-terminated
path, NUL-terminated nonempty ASCII decimal line number, then a newline-terminated
line (the last line may end at EOF). Paths and line contents preserve arbitrary bytes.
Each forbidden record prints `path:number:line` with a final newline. An empty stream
returns 0, forbidden matches return 1, and malformed records or I/O/launch failures
return 2. External grep failures propagate their exit code; a signal returns 2.

The author facade's test-directory token is exempt only at path/token boundaries.
The native boundary grammar uses ASCII whitespace and the existing punctuation
boundaries; Python's Unicode whitespace classification is not part of this native
byte-stream contract (owner's 2026-10-24 native grammar amendment). After replacing
exempt tokens, external `grep -E -q -- <pattern>` rechecks the remaining line using
POSIX ERE syntax. This detects a forbidden token beside an exempt token and keeps
POSIX character classes. The shell resolves the compiled tool once through
`scripts/xtask.sh`; `UMPIRE_XTASK` can select an absolute prebuilt executable for
isolated caller controls. `cargo test -p xtask --test main --locked` covers
the stream contract, active spellings, historical exclusions and tool failures.

## Apalache installer

`fetch-apalache [framework-root]` admits the official pinned Apalache jar. The
optional root defaults to the admitted framework tool root. Local Cargo, mise and
Rust-toolchain configuration must exist; parent directories are never searched. Missing
cache components are created. Success
prints the absolute jar path followed by a newline and exits 0. Failures exit 1
with `xtask: <native error>` on stderr; curl can also write its own diagnostics.
Paths are OS arguments, including non-UTF-8 paths on Unix. No Python diagnostic,
exception type or output encoding compatibility is promised.

The installer requires curl and Unix filesystem admission. It stores the jar at
`build/apalache/apalache-<version>/lib/apalache.jar`, locks a persistent
regular `.install.lock` using `File::lock()`, and refuses symlink components,
wrong existing jars and partial installations. Download and validation use a
private 0700 staging directory. The archive hash is checked before gzip/tar
parsing, the exact regular jar member is hashed before publication, and the
validated directory is renamed into place while holding the lock. Staging is
removed on success and failure. Installation pins live only in
`src/apalache_pins.rs`, included by the runtime backend as well as xtask.
The production command accepts no URL, version, archive or hash overrides.

`test-run` serves the gate's `tests` stage and `update`. It inherits their working
directory and admitted environment, runs unit/integration suites through pinned
nextest, then runs Cargo doctests separately. Test sequencing belongs in
`src/test_run.rs`; callers select this route. Child failures retain their public
exit codes, including shell signal status `128 + signal`.

## Stage graph and host admission

`src/gate/tasks.toml` is the framework stage DAG. The framework mise config includes it; a system root includes the same file and adds its own stages and cross-workspace resource edges. Native stage listing, receipt order and standalone completeness read actual mise task metadata. Mise 2026.10.11 is the verified minimum; production commands do not probe capabilities.

One supervisor retains the host lease through each public gate, quick or test invocation and descendant cleanup. Leased descendants receive a validated file descriptor, including the clean standalone child. Mise owns dependency scheduling; native supervision polls failed stage receipts and cancels owned work when descendant pipes keep mise pending. Independent read-only checks can overlap. Shared Cargo targets, generated outputs, JVM phases and Go work retain conservative graph barriers until combined admission proves additional overlap.

Substantive leaf controllers use native ownership. Guardians share root cancellation and retain parent completion custody through direct-child waits; clean standalone commands preserve that verified context. Checker Java receives no lease or lifecycle descriptors. Linux retains subreaper cleanup for escaped descendants. Darwin relies on admitted controllers forwarding cancellation and waiting for their own groups; its non-root runtime proof remains open. Descriptor closure or an elapsed grace period alone does not establish reaping.

Invocation records preserve public stage names and qualify them with `framework/` or the consumer-defined scope prefix. Every expected stage receives a success, failure, skipped, cancelled, not-started or partial record in logical order. Missing or invalid successful receipts fail. Parent and isolated-child suites remain separate authorities. Raw stage timings and counts feed the later budget task; inclusive parent time is not the sum of overlapping stage times.

`test-run --package NAME` uses the selected runner's exact package inventory, nextest policy and eligible scoped Cargo doctests. Full workspace test execution retains its separate doctest phase.

Scheduled receipts contain `counts` (`discovered`, `executed`, `ignored`, `skipped`, `tlc_launches`, `tlc_configurations`), `count_error`, and `count_authority: "exclusive-stage"`. Unobserved values are `null`. Successful `tests` stages require completed producer records; malformed, duplicate and unfinished producers refuse success while a real failing runner retains its original status. Producer event files live only beneath the current invocation's `work/<scope>/<stage>/` directory. Current producers cover native Rust nextest/doctests and TLC. Go host test counts and other unadapted modalities remain unobserved; budget admission must add their actual producers before admitting work-count tables for those stages.

Nextest discovery uses its actual JSON list under the run's selection and ignored policy. Scoped runners hand their already-admitted inventory directly to the observer; full-workspace runners perform one owned list before running. `executed` counts distinct successful or failed terminal test identities, never selected tests. Discovery entries excluded for the ignored policy count as `ignored`; other filter mismatches count as `skipped`. An ignored test selected with `--run-ignored all` counts as executed after completion. Repeated nextest suite wrappers are not summed. Cargo doctests have a separate producer, pairing each `running N tests` start with its complete result summary, including zero-test suites. Both modalities contribute to the stage total: `discovered = executed + ignored + skipped`. ANSI styles are removed only from the private doctest observation; human output retains its original bytes.

TLC counts come from actual supervised attempts confirmed by TLC's startup output. Launch identities are process/sequence pairs; distinct configurations hash the exact emitted TLA+ source and configuration bytes. Repeated launches of the same configuration increment launches once each and configurations once total. The confirmed `tlc2.TLC -h` admission JVM counts as a launch with no model configuration. Missing completions refuse a complete receipt; unconfirmed starts or configurations remain unknown. The standalone parent's `child_count_receipts` embed the child's ordered rows with their source path and digest. `path_bytes` retains exact Unix path bytes; `path` is its optional UTF-8 rendering. Event framing encodes paths without record separators. Child counts are authoritative there and are not added to the parent's own counts or inclusive elapsed time.

Ordinary quick and test-run commands without scheduled receipt authority perform no observation discovery and retain their existing output. Native commands freeze their verified controller role at startup, so Cargo replacing their executable cannot omit a later count phase. Test binaries inherit backend observation scope but keep their rejected test-run interception role even if configured executable paths change.

Global quick also compiles the explicitly admitted zero-harness facade, IR derive and guest packages with `cargo check --all-targets`. Their eligible doctests remain selected, and any newly discovered tests run. Unknown empty packages and missing target/name selections fail admission.

Required and quick validation cap each Cargo queue at one build job and Go at `-p=1` with `GOMAXPROCS=1`; other caller Go flags remain intact. Clean standalone children receive only their explicit clean settings and these caps. Backend-skipped native commands keep caller CPU defaults. The DAG admits at most one external heavy phase beside up to three read-only task owners. Within a required nextest phase, four test slots permit at most four nested one-job compiler queues, or two JVM owners with two recovery JVMs each and up to two other test owners. Cargo may wait for a build script’s Go child; that live ancestry does not add an independent compiler queue. These are queue and JVM-owner bounds, not CPU affinity or total RSS limits. `admission.json` records the actual child compiler/Go settings; scoped stage reports record outcomes separately.

Building the initial native executable before it starts is caller preparation, outside its already-running supervisor lease. Cold bootstrap work requires its own caller admission. The graph does not claim to constrain arbitrary Cargo commands launched before that entry point exists.
