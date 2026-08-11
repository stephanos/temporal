# Gomad v3 Testing Gaps

## Purpose

Gomad v3 demonstrates its core runtime hypothesis: for deterministic external
inputs, a fixed Go 1.26.4 toolchain, program, architecture, and `GOMADSEED`
produce repeatable runtime-controlled choices. The current suite provides
strong coverage for small scheduling, `select`, map, channel, synchronization,
`go run`, and `go test` fixtures. Every schedulable child is bounded by a
process-tree watchdog, invalid seeds are checked in a prebuilt binary, and
seeds `0`, `1`, and maximum `uint64` receive full repeatability coverage.

This document records the remaining test gaps. A gap is not evidence that the
implementation is wrong; it identifies behavior whose failure would not be
reliably detected by the checked-in suite.

Post-v3 capabilities are deliberately excluded. See `GOMAD_NEXT.md` for that
roadmap and `GOMAD_ALT.md` for the v3 contract.

## Testing principles

- Keep behavior fixtures outside the Go source patch and runtime overlay.
- Test observable output and exit status across fresh processes.
- Put a watchdog around every process that can block or schedule goroutines.
- Require same-seed repeatability and require cross-seed diversity only when a
  fixture presents more than one observable choice.
- Validate logical results separately from their ordering. Repeatable data loss
  is still a failure.
- Exercise the root Make targets in addition to calling the custom Go binary
  directly.
- Keep Windows excluded. Run supported-platform coverage on Unix-like hosts.
- Use isolated toolchain roots for destructive builder and failure-injection
  tests; never corrupt or republish the developer's stable `.toolchain`.
- Run repository-focused Go tests with `-tags test_dep`.

## Priority 1: close before relying on v3 broadly

### Exercise local run-queue boundary states

The deterministic `runqget` branch chooses and removes an entry from Go's
256-slot circular local run queue. Existing fixtures stage at most twelve
goroutines and do not wrap the queue or force local-to-global overflow.

Add fixtures that:

- exercise empty, singleton, and multiple-candidate queues;
- advance `runqhead` and `runqtail` through at least one circular wrap;
- create more than 256 runnable goroutines so local overflow and global refill
  paths execute;
- use multiple waves so a wrapped queue is reused rather than tested only at
  process startup; and
- assert that every worker appears exactly the expected number of times before
  comparing order.

Acceptance criteria:

- each same-seed workload repeats across at least 100 fresh processes;
- multiple seeds produce more than one order where alternatives exist;
- no worker is lost, duplicated, or executed after the fixture completes; and
- every invocation has a watchdog.

### Exercise automatic GC during deterministic scheduling

Current fixtures are small enough that they do not prove automatic GC ran.
The address-layout test intentionally uses `GOGC=off`, and upstream runtime
tests run with Gomad disabled.

Add an enabled fixture that allocates while several goroutines repeatedly
block, become runnable, and yield. Run it with the default GC setting and a low
`GOGC` value that reliably causes multiple cycles. Record stable logical output
such as per-worker allocation counts and a scheduling checksum; do not expose
wall time, heap addresses, or GC timing in the expected output.

Acceptance criteria:

- the fixture proves that at least two GC cycles completed;
- logical results and scheduling output repeat for the same seed;
- different seeds retain schedule diversity;
- the workload repeats under bounded unrelated host CPU load; and
- finalizers, host timers, and memory-pressure-dependent `GOMEMLIMIT` behavior
  remain outside the claim.

### Test valid build-key invalidation

The builder rejects invalid patches and overlays and safely reuses the current
key, but the suite does not change a valid input and prove that the key changes.

Add a key-only test seam or an isolated builder harness that varies one input at
a time:

- patch contents;
- overlay path and contents;
- Go version and source checksum;
- host OS and architecture;
- bootstrap Go version; and
- canonical build-recipe revision.

The unchanged input set must produce the same key. Every contracted change must
produce a different key. Path names must be part of overlay identity, not just
file contents.

### Test interrupted and concurrent uncached builds

Current concurrency coverage uses an already-complete cache key. It does not
exercise two builders compiling the same new key or termination during build
and publication.

Introduce test-only failure injection at phase boundaries after lock
acquisition, extraction, overlay copy, patch application, compilation, build
directory publication, stamp publication, and launcher publication. Run these
tests against an isolated toolchain root.

Acceptance criteria:

- an interrupted build never publishes a stamp for an incomplete tree;
- the previously stable launcher continues to work after every injected
  failure;
- a later invocation reclaims a stale lock and completes successfully;
- two uncached same-key builders publish one immutable build without racing;
- temporary work, owner, and incomplete-build paths are cleaned or safely
  recoverable; and
- no test relies on PID `999999` being absent without checking first.

### Run a supported-platform CI job

No checked-in GitHub workflow currently runs the Gomad v3 suite. This means the
scope validator, patch application, source build, upstream checks, and
black-box fixtures are not enforced on pull requests.

Add a Linux/amd64 job that:

1. prepares the Go version selected by the root `go.mod`;
2. runs `make -C tools/gomadv3 test` from a clean checkout;
3. preserves the Go source archive and immutable toolchain build as keyed
   caches without treating a partial build as valid; and
4. runs a second cached build to cover the fast path.

Darwin/arm64 remains useful local coverage. Other supported Unix hosts may be
added after the Linux job is stable. Windows must remain excluded.

### Broaden disabled-mode upstream compatibility

The suite currently runs the upstream runtime package tests with
`GOMADSEED` absent and compares representative custom and stock commands. It
does not run upstream compiler, linker, `cmd/go`, or wider standard-library
tests through the custom distribution.

Use `go tool dist test -list` from the pinned Go 1.26.4 tree to select explicit
disabled-mode shards for the runtime, compiler, linker, `cmd/go`, and standard
library. Keep a focused subset in pull-request CI and run the complete relevant
distribution suite on a scheduled or pre-merge tier if its duration is too
large for every change.

Every command must remove `GOMADSEED`; compatibility tests must never pass by
exercising deterministic mode accidentally.

## Priority 2: fill contract and harness gaps

### Complete the `select` matrix

The current fixture covers two ready buffered receives. Add separately labeled
cases for:

- two ready receives;
- a ready send competing with a ready receive;
- multiple ready sends;
- a default case when no communication can proceed;
- nil channels mixed with ready or default cases, proving they are disabled;
  and
- repeated selects after channel state changes.

Only cases with multiple ready alternatives require cross-seed diversity.
Prebuild the fixture and repeat it under explicit allocation-layout
perturbations with automatic GC disabled for that isolated check.

### Complete channel contention coverage

Current observable output covers multiple blocked unbuffered senders and
receivers woken by close. Buffered channels are used only for bookkeeping.

Add output-bearing cases for:

- multiple blocked receivers on an ordinary send;
- senders blocked behind a full buffered channel;
- receivers blocked behind an empty buffered channel;
- close with waiting receivers and buffered values still present; and
- repeated contention waves using the same channels.

Each case must validate the complete value multiset separately from wake order.

### Validate map semantics and every lifecycle independently

Current assertions prove repeatability and per-family diversity but derive the
baseline from the same implementation and do not validate every logical entry.
The create, clone, clear, and standalone NaN lines are not included in the
per-family diversity check.

Add normalized semantic assertions for expected key/value sets, then require
repeatability and applicable diversity independently for:

- initial creation;
- consecutive iterations of the same unchanged map;
- clone;
- clear and repopulation;
- growth and deletion;
- small-map and multi-table sizes;
- floating and complex NaNs; and
- interface, array, and struct hashing.

Extend the regular-memory audit to 8-bit, 16-bit, and variable-length key
layouts if v3 intends to claim every runtime hashing path rather than only the
families listed in `GOMAD_ALT.md`.

### Isolate synchronization choices from scheduler arrival order

The mutex fixture releases all contenders together. Its output can therefore
become diverse solely because the patched scheduler changes waiter arrival
order, even if the synchronization path stops consuming seeded randomness.

Add fixed-order waiter staging and repeated contention. Validate repeatability
for mutex and semaphore-backed behavior without requiring diversity from a
path whose documented behavior is FIFO. Add observable `sync.Cond` and
`sync.RWMutex` cases only when they present a runtime-controlled choice covered
by the v3 contract.

### Apply address and host-load perturbations consistently

Explicit allocation-layout and host-load checks currently cover selected map
and scheduler fixtures. Prebuild the supported `select`, channel, and
synchronization fixtures and run them through the same bounded perturbation
helpers. Strip only diagnostic layout markers; compare all logical output and
exit status.

### Complete validator negative cases

Add exact-diagnostic cases for:

- a complete-header modification to an existing prohibited runtime file;
- nested runtime paths;
- generated markers in patches and overlays;
- binary patches and NUL-containing overlays;
- malformed or inconsistent patch headers; and
- unexpected regular runtime files that require explicit scope review.

Every rejected input must leave the stable toolchain and build key unchanged.

### Prove `GOMADSEED` is the only feature gate

Run disabled fixtures with `GOMAXPROCS=1` and
`GODEBUG=asyncpreemptoff=1` together while leaving `GOMADSEED` absent. The
supporting settings have their ordinary upstream effects, but map entropy and
Gomad scheduler randomization must remain disabled.

### Exercise the root Make targets

Add automated cases for:

- missing `GOMADSEED`;
- missing `GOMADV3_RUN` and `GOMADV3_PACKAGES`;
- successful `gomadv3-go`, `gomadv3-run`, and `gomadv3-test` invocations;
- argument forwarding through `GOMADV3_ARGS`;
- use of the stable custom Go launcher; and
- inclusion of `-tags test_dep` for tests.

The success cases should use small fixtures and should not rebuild the
toolchain independently for every assertion.

### Test stock-toolchain resolution failures

Use small executable doubles to cover missing stock Go, wrong Go version,
invalid `GOROOT`, and a stock command that resolves to the custom GOROOT. Each
case must fail before compatibility output is compared and must emit its
specific diagnostic.

## Priority 3: protect repository integration

### Pin generated-toolchain pruning

Root source discovery, shell discovery, `goimports`, and formatting prune
`tools/gomadv3/.toolchain`. Add a sentinel under an isolated toolchain-shaped
directory and prove every discovery path ignores it. This prevents generated
GOROOT files from entering repository formatting, lint, test dependency, or
shellcheck inputs.

### Split the suite into explicit tiers

The single shell driver performs validation, builder checks, black-box runtime
checks, stock comparisons, and upstream tests. As coverage grows, preserve one
top-level gate but expose focused tiers so failures are faster to reproduce:

- `validate`: patch and overlay scope;
- `test-builder`: cache, lock, failure, and publication behavior;
- `test-runtime`: enabled and disabled black-box fixtures;
- `test-upstream`: disabled Go distribution compatibility; and
- `test`: all tiers in dependency order.

Shared helpers should own child execution, watchdogs, output comparison,
temporary toolchain roots, and process cleanup. The public `make -C
tools/gomadv3 test` command remains the complete local gate.

## Recommended implementation order

1. Complete `select`, channel, map, and synchronization fixtures.
2. Add run-queue boundary and enabled-GC stress fixtures.
3. Add isolated build-key and failure-injection tests.
4. Exercise root Make targets and pruning.
5. Add Linux/amd64 CI and broader disabled upstream shards.

The harness now reliably reports hangs and child failures before the larger
stress and build-lifecycle suites depend on it.

## Completion criteria

The v3 testing backlog is complete when:

- every production runtime hunk has normal, boundary, and failure-path
  coverage through an external fixture;
- enabled GC, run-queue wraparound, and queue overflow repeat for one seed;
- invalid seed handling is proven in prebuilt binaries before user `init`;
- all documented `select`, channel, map, and synchronization cases are pinned;
- all child processes are bounded, reaped, and status-checked;
- valid build inputs invalidate the cache and injected failures cannot publish
  incomplete state;
- supported Linux CI runs the complete v3 gate;
- disabled upstream runtime and toolchain compatibility is exercised; and
- the root developer commands are covered end to end.
