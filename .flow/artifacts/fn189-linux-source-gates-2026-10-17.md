# fn-189 Linux source gates, stopped checkpoint

The owner stopped this batch before source acceptance completed. Fresh stock-Go
build, vet, integrated validation, the default harness including archcheck,
World race tests and portable adapter source checks passed on the source rooted
at `bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe`. Full WASM and canonical lint
remain incomplete. This report supplies no native execution qualification.

Commands, individual exits and raw logs remain in
`../tmp/fn189-linux/source-gates/`. `run.sh`, `extra.sh` and `commands.tsv`
retain the command lines. The compiler is the pinned Linux Go 1.27.1 executable
at `tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go`, with GOROOT unset,
GOENV=off, GOWORK=off and GOTOOLCHAIN=local. Initial checks used GOMAXPROCS=4
and `GOFLAGS=-p=4`; later checks used 2 for both limits.

| Gate | Exit | Evidence and limit |
| --- | ---: | --- |
| Default `go -C tools/gomad build ./...` | 0 | `build.log` |
| Default `go -C tools/gomad vet ./...` | 0 | `vet.log` |
| `make -C tools/gomad validate` | 0 | `validate.log`; generated/toolchain/pack/inventory checks |
| `make -C tools/gomad test-harness` | 0 | `harness.log`; archcheck package passed in 420.943 seconds |
| `make -C tools/gomad world-test` | 0 | `world.log`; World, mailbox and process race suites passed |
| Portable adapter source/pin/preservation/regeneration/refusal selectors | 0 | `adapter-source.log`; stock-Go source checks passed in 46.843 seconds, including both declared source platforms |
| Initial canonical default and selected lint | 2 each | `lint.log`, `lint-selected.log`; inherited Darwin lint executable refused on Linux |
| Canonical default lint with isolated Linux tools | 143 | `lint-linux.log`, `lint-linux-cancellation.log`; cold package loading took 8m56.821s, internal 10m deadline expired, owner cancelled its remaining analysis after more than 16m elapsed |
| Full `make -C tools/gomad test-wasm` | 143 | `wasm.log`, `user-stop-processes.log`; user stopped its process group before completion |
| Full portable compatibility-pack package tests | Interrupted | Began compiling when the user stopped the source wrapper, whose collected exit was 143; no package pass is claimed |

The Linux lint executable and errortype tool were built at their pinned versions
under `source-gates/lint-bin`, preserving the inherited `.bin` executables.
Canonical lint kept its 10m timeout and `GOLANGCI_LINT_FIX=false`. The warmed
rerun, later selected native lint, upstream seam checks and compile-only
`./tests` check were not launched before the stop.

The interrupted WASM target completed all three named slim-provider refusal
tests, the namespace and volume conformance suite, generated projection guards
and the WASM architecture package. Its remaining backend and command packages
were still active. The installed operator typed scenario replay and Linux
kill/resume gates have no completed result in this log.

## Pin evidence

`mapped-pins.json` records fresh module graphs and exact activation comparisons
for all twelve requests in the canonical working-directory map. Every graph
command exited 0. Historical x/sys v0.47 and libc v0.41/v0.47 fixture mappings
still matched their exact activation versions and sums.

`mapped-impact.json` retains canonical pin-impact commands, full reports,
stderr and exits for each mapped module. The root report exited 1 with forty
invalidated pack rules out of fifty-four, zero invalidated adapters and zero
unknowns. The historical checkpoint also recorded forty invalidated rules;
this worker did not complete a fresh row-by-row historical equality comparison.

Filtering the fresh reports by the same mapped-request ownership used by
`compatibility_pack_refresh.go` leaves twenty affected rules in five root
requests. The mapped historical fixture requests have zero affected rows.

| Mapped root request | Activation mismatch |
| --- | --- |
| modernc-libc-xsys-v047-isatty-v021 | x/sys v0.47.0 to v0.48.0 |
| temporal-functional-compute-darwin-arm64 | x/crypto v0.55.0 to v0.57.0 |
| temporal-functional-compute-linux-amd64 | x/crypto v0.55.0 to v0.57.0 and x/sys v0.47.0 to v0.48.0 |
| temporal-functional-tests-darwin-arm64 | prometheus/client_golang v1.21.0 to v1.21.1 |
| temporal-leaf-xsys-darwin-arm64 | prometheus/client_golang v1.21.0 to v1.21.1 and x/sys v0.47.0 to v0.48.0 |

These are retained source obligations. Native execution deferral supplies no
pin-validation waiver. A separate owner prepared reconciliation in an isolated
worktree; this worker supplies no integrated repaired-source acceptance.

## Source boundary and stopped processes

The 5,149-file baseline manifest check exited 1 for exactly one change,
`wasmhost/tests/cluster.rs`. The owner identified and authorized its fixture
loop correction from 2,000 to 4,000 iterations. `source-baseline.log` retains
that diagnostic. Production Go/helper sources remained unchanged by this
worker. A final integrated manifest was not created here.

The earlier lint group 128695 and queued source wrapper were terminated and
reaped with exit 143. At the user stop, owned WASM group 139722 and source
group 249702 were validated with their descendants, terminated and reaped with
exit 143. The following process inventory found no residual members of these
groups. `user-stop-processes.log` preserves the validated commands. This
worker has no remaining active execution handle and created no worktree.
Existing worktrees, logs, caches and incomplete test temporary files remain
with their owners. No revert, delete, commit, push, PR, CI or Flow state write
was performed by this worker.
