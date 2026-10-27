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

`test-run` is an internal no-argument subprocess route used by the gate's `tests`
stage and `update`. It inherits their working directory and admitted environment,
and currently runs exactly one `mise exec -- cargo test --workspace --locked --quiet`,
including Cargo's doctests. Test sequencing belongs in `src/test_run.rs`; callers
only select this route. Child exit codes and signal statuses propagate to the gate.

## Stage graph and host admission

`src/gate/tasks.toml` is the framework stage DAG. The framework mise config includes it; the Temporal root includes the same file and adds its own stages and cross-workspace resource edges. Native stage listing, receipt order and standalone completeness read actual mise task metadata. Mise 2026.10.11 is the verified minimum; production commands do not probe capabilities.

One supervisor retains the host lease through each public gate, quick or test invocation and descendant cleanup. Leased descendants receive a validated file descriptor, including the clean standalone child. Mise owns dependency scheduling; native supervision polls failed stage receipts and cancels owned work when descendant pipes keep mise pending. Independent read-only checks can overlap. Shared Cargo targets, generated outputs, JVM phases and Go work retain conservative graph barriers until combined admission proves additional overlap.

Invocation records preserve public stage names and qualify their scope as `framework/<stage>` or `temporal/<stage>`. Every expected stage receives a success, failure, skipped, cancelled, not-started or partial record in logical order. Missing or invalid successful receipts fail. Parent and isolated-child suites remain separate authorities. Raw stage timings and counts feed the later budget task; inclusive parent time is not the sum of overlapping stage times.

`test-run --package NAME` uses the selected runner's exact package inventory, nextest policy and eligible scoped Cargo doctests. Full workspace test execution retains its separate doctest phase.
