# Fn-109.7 source-control admissions

These admissions retain source acceptance; neither grants native qualification,
changes production support, nor turns an earlier failed command into a pass.

## Supported-source controls

The actual host is linux/arm64. The ordinary baseline's unsupported-host failures
remain retained. Controlled tests may use isolated compile-time overlays selecting
the existing linux/amd64 source profile and matching target-host constants, with a
counted stock-Go driver cross-building actual linux/amd64 binaries. Those binaries
are not executed. Production Default, host refusal and validation remain unchanged;
any composition seam is private and per call. Bind overlay, fixture, tool, source
and cache inputs to each result. This is not full-host or native execution.

## Original comparison harness compatibility

The coherent original caller tree at `a3b9f80efa` predates
`toolchain/installation`. The first new test-only harness imported that later
package and failed before tests. Root inspected
`../source-acceptance-20261008/original-callers-first-receipt.json` and its raw
log: exit 1, no named test outcomes, runner setup failure from the missing package;
raw SHA256 `111aaac0f9bcbb7fc48bd66c7b5c610d5e1052a3f83613ab5957fa3ff7d2a6e1`.

The narrow corrected input is an identical test-only harness on original and
current trees using existing `target.ReadToolchainIdentity` and literal
fixture-owned cache paths. Keep the original production callers, dependencies and
four preserved preimages byte-exact. No installation import backport, dependency
upgrade or old-caller adaptation is admitted. Retain the first failure and corrected
harness identity, count setup and retry, and assert actual old/current caller
invocation, fixed identities and cache states. A spelled-protocol probe alone
cannot replace these controls. No corrected-run success is claimed here.

## Original unexecuted execution compilation

The corrected harness also failed before tests: the coherent original execution
package names `syscall.Dup2` at `bootstrap_unix.go:161` and
`launch_plan_unix.go:334`, unavailable when compiling on the actual linux/arm64
host. Root inspected both original files and the corrected raw compile log.
`git show a3b9f80efa:tools/gomad3/go.mod` requires only x/mod v0.37.0, so the
proposed x/sys/unix substitution is not admitted as an existing dependency.

The narrow alternative is ORIGINAL-only test compile overlays replacing exactly
those two references with a standard-library-only fail-closed fixture function.
The comparison must fail if either path is reached. Preserve original on-disk
caller, dependency and preimage bytes; bind the exact overlay diffs and retain
both setup-red receipts. The actual Explore control must demonstrably stop at
ProgressRunning before target execution. This cannot prove execution equivalence,
native support or a baseline production fix; no silently executing substitute is
admitted. These fixture inputs do not extend to any other source task or test.

## Caller-fixture source cache

The fail-closed compatible original run reached named tests, then failed with
ENOSPC in capability `go list` under `/home/agent/.cache/go-build`. Root inspected
`original-callers-compatible.log` and its receipt: exit 1, 2 named passes and
3 named failures, including the parent; raw SHA256
`43fbda93fde1e2617e022a4eda558bb752bcee6b85345691f1d280564f62ec64`.
The original sanitized subprocess drops inherited GOCACHE, while the initial
fixture driver only selected the cross-build platform.

The admitted changed input is identical pinned-driver bytes for original/current
caller fixtures explicitly selecting an owned workspace `toolchain/source-cache`
as GOCACHE for every driver invocation. Bind the updated driver identity and
control manifests before execution; record initial cache absence and each side's
actual source-cache state, including later warmth, separately from prepared-cache
fresh/hit/release assertions. Retain the failed command and count the changed-input
retry. No global cache deletion, HOME change, production caller adaptation or
task-wide exception follows. The earlier fn113.4 site-specific XDG admission is
not reused. This admission makes no corrected-run pass claim.

## Authoritative static package inventory

Unrestricted linux/amd64 `./...` list/vet attempts traversed
`toolchain/runtime/overlay/src` as nested-module packages and failed on forbidden
stdlib/compiler internal imports before any assertions. Preserve those red raw
commands. Root inspected the list log, existing `TestHostPackageVet`, and
`architecture.Discover`; the latter explicitly inventories host packages while
retaining established runtime-overlay/fixture exclusions and checking uncovered
sources, unclassified modules and stale exclusions.

Use the unchanged authoritative inventory, existing `TestHostPackageVet`, and
exact enumerated package-list commands. Each supported source platform and the
actual host must have a bound nonempty inventory with zero discovery findings and
its complete list/vet results retained. No source or exclusion change is admitted.
This is a correction to the test surface, not a new gate waiver: all portable
assertions remain required, failed unrestricted commands remain red, and static
coverage is not native qualification. No corrected static-pass claim is made here.

## Portable qualification prune diagnostic

The ordinary `./qualification/...` command is red: root independently decoded
its full retained JSON stream and verified raw SHA256
`eb0c576640072e08086de906721fcf8769758e5a2e2f677537e39b0d27a7eeb6`
against the receipt. Exit 1, 213 named passes, 11 named failures, no skips. Ten
analysis/capability cases stop at the unchanged unsupported-host profile guard;
the separate `TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact`
fails a portable assertion after the final artifact prune. It is not transferred
to a native owner and cannot be excused by the other ten cases.

Admit only bounded owned-scratch standard-library/filesystem link-count
diagnostics while source remains frozen. Bind prune/test/dependency identities,
actual `os.Root.Lstat` Sys type/Nlink and remaining file/link paths. Do not retry
the unchanged gate, change expectations, infer a metadata cause without observing
it, or claim full-source green. A proved product/fixture defect requires root
owner disposition and a separately routed correction before source acceptance.

Root then read the complete standard-library probe and all six observations,
and independently verified its raw SHA256
`080342a49a6d2676209aed9a71c4263d0f22fd3bcd916ed9c9283a7fb6dff0fd`.
Both removal APIs remove campaign/link paths, leaving only `targets/shared`,
but both stat APIs report `*syscall.Stat_t` with Nlink 3 at every phase.
The retained probe supports a workspace-filesystem metadata limitation, not a
prune-code defect or permission to weaken its conservative safety rule.

Admit a separate bounded filesystem-capability probe in a fresh private
task-specific `/dev/shm` scratch, only after space/capability checks. Keep GOCACHE
and GOTMPDIR on existing workspace paths. The same probe must actually show
3→2→1 through both stat APIs with deleted paths absent. Only if that succeeds,
one changed-filesystem retry of the exact prune selector may use command-local
TMPDIR at that scratch. Retain filesystem/mount/state, source and command inputs,
the original red command and workspace diagnostic. Do not retry the full gate,
generalize tmpfs to other cases, change production or assertions, or claim native
qualification. An unsupported alternative remains a concrete residual, not a
reason to substitute weaker evidence. No alternative-filesystem pass is claimed
by this admission.

The tmpfs probe reports the required 3→2→1 through both APIs, with correct
remaining paths. Root read its complete df/mount-prefixed stream: 64 MiB available,
tmpfs mounted noexec. The initial root read-only parser expected JSON-only input
and exited 1 on that preserved prefix; no raw output or source was rewritten.
The permitted TMPDIR-only selector retry is itself red, not a pass.

Root inspected pinned Go1.27.1 `src/testing/testing.go:1573,1613`, SHA256
`8d2437cfdcfb82fd8807e5063783de1d50f70e589e80cad10e17d8ef23f724d9`:
`t.TempDir` uses `os.MkdirTemp(os.Getenv("GOTMPDIR"), pattern)`. Therefore the
first retry kept its test fixture on the workspace despite changing TMPDIR.
Admit one corrected compile-then-run input: compile the tagged qualification/set
test binary in owned executable workspace with the existing workspace GOCACHE and
GOTMPDIR, then use test2json to run only the exact selector with child-only
GOTMPDIR and TMPDIR at the already-probed private tmpfs scratch. Keep compiler and
converter on workspace paths, bind binary/SDK/source/mount/actual child environment,
and count compilation and runtime separately. Retain the wrong-filesystem red.
No assertion, production, full-gate, native or task-wide filesystem exception
follows. This admission does not assert a successful corrected test.

The tagged binary compilation succeeded, but its first converter invocation
exited 2 before any named test: SIGBUS in the vendored telemetry mapped-file
counter. Root read that stack and verified raw SHA256
`33a41aa90bcc78a1ea10644a93bc647e6375582388046ca4d301521d6e456d9a`.
Zero named outcomes does not make this a pass; retain this separate setup-red.

Root inspected pinned SDK `cmd/internal/telemetry/counter/counter.go:24`, SHA256
`592d9ac24df7b6a04debca264ea5bcd85dffb128462c606d49fbd29a0f1eb8c2`:
its Open calls `counter.OpenDir(os.Getenv("TEST_TELEMETRY_DIR"))`. Admit only this
converter invocation's command-local TEST_TELEMETRY_DIR at its owned workspace
`test2json-telemetry` directory. Bind SDK/tool/converter bytes and actual environment,
and unset that variable in the test child, preserving its already-admitted private
tmpfs data directory and workspace GOCACHE. Reuse the unchanged bound compiled
binary and exact selector. Retain/count setup failure and changed-input retry.
No global telemetry mode, XDG/HOME mutation, source/assertion change or broad
exception follows, and no corrected-test success is asserted here.

The first telemetry-local attempt instead guessed a nonexistent SDK tool path
and exited 127 before tests. Root verified raw SHA256
`2fafe41b3f9135bb265745558210030dce579abeb2c98a0210e468628d9f7518`;
retain this separate setup failure. The established `go tool test2json` resolver
with the already-admitted command-local telemetry directory is the corrected
recipe; a direct tool path must first be resolved, never assumed. Unset
TEST_TELEMETRY_DIR in the actual test child as required above. No gate scope,
test input or assertion is weakened by correcting this executable path.

The resolved, telemetry-local converter subsequently records the exact selector
as exit 0, one named pass and its package pass. Root read the complete eight-event
raw stream and independently verified SHA256
`8bb7c53976aec8495ed8bb7b1f781cbbccb89122e1774f42fbd03b9f5ee1d86b`.
The child confirms private tmpfs TMPDIR/GOTMPDIR, workspace GOCACHE and telemetry
override unset. Runtime inputs were bound at 21:40:45.384Z, before execution:
compiled binary, runtime shim, both SDK control files and resolved cached
test2json. The original full qualification command remains exit 1 with
213 passes/11 failures. This exact corrected-runtime result resolves only the
portable prune assertion; it is not a full qualification or native pass.
