# fn-189 R2 compatibility-pack source reconciliation

Source candidate: `bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe` plus the uncommitted
`wasm-milestones-pins` worktree diff. Host: Linux/arm64. Compiler:
`/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go`
(`go1.27.1`). This is source review and generation evidence only. Native
execution/qualification remains deferred under fn-128/fn-149. No native target
was built or executed; portable stock-Go host tools/tests were built and run.

## Source authoring route

`compatibility-pack discover` accepts an explicit `--source-platform`,
`--source-go` and `--source-inventory`. The requested platform must equal the
request's single declared platform and belong to the generated native platform
allowlist. Source listing uses its GOOS/GOARCH, exact request package/build tags,
cgo-disabled environment and pinned Go version. Pure capability evaluation uses
the same declared platform. Every selected adapter retains its exact module sum,
anchored rewrites and whole-module source inventories; its prepared source set is
verified using `build.AdapterPreparedSourceSetSHA256` against that platform's
existing pin. Adapter bindings use the existing platform profile golden.

The route returns a source inventory, never an executable transcript profile.
`transcript.Default`, its host refusal, runtime preparation and execution
admission are unchanged. Inventory schema
`gomad.compatibility-pack-source-review/v1` explicitly records
`native_execution_qualified:false`. Discovery/review/generation use the existing
`authoring.Discover`, `PublishReview`/`RenderReview`, and `Generate` owners and
require each exact new approval digest.

Root review identified an executable-alias path assumption in adapter source
preparation. A regression first reproduced failures for a pinned compiler
symlink outside `GOROOT/bin` on both declared platforms (exit 1,
`source-alias-red.log`). Source resolution now queries the pinned compiler's
own `GOVERSION` and `GOROOT` with the canonical scrubbed environment, a 30-second
bound and 64-KiB output bound. It validates the exact Go release and absolute
root before module-source resolution. Direct and aliased compiler cases then
pass for both platforms (exit 0, `source-alias-green.log`), with identical
adapter source-set pins and profile goldens. Invalid compiler identities also
reject. This correction changes no generated request, fact or execution profile.

## Reviewed changes

All five requests retain exactly the same fact lists and dispositions as HEAD:
zero added allowed facts, zero changed allowed facts, zero changed denied facts.
The Linux compute request's existing denied `import:golang.org/x/sys/cpu`
remains denied. New discoveries still default to denial. There is no generic
syscall, os/exec, os/signal or x/sys grant.

| Request | Exact pin/source reconciliation | New review SHA-256 |
| --- | --- | --- |
| `modernc-libc-xsys-v047-isatty-v021` | x/sys v0.47.0 → v0.48.0; isatty sources/facts unchanged; libc inventory domain receipt below | `a26c2983b331d670e89b87a7837abe77a64968206fa1f69d6c783f0a313c81d0` |
| `temporal-functional-compute-darwin-arm64` | x/crypto v0.55.0 → v0.57.0; selected Go/assembly bytes unchanged | `1b36534e78a78803fda22089d9f83e8ddac23b5e26728134ea21d600ff359a52` |
| `temporal-functional-compute-linux-amd64` | x/crypto v0.55.0 → v0.57.0 and x/sys v0.47.0 → v0.48.0; crypto/assembly/linkname bytes unchanged; CPU Go changes reviewed below | `98d856b738bf8e03b678690768c4054367ca895ddf12a8d27eba4ae3c355e0f2` |
| `temporal-functional-tests-darwin-arm64` | Prometheus v1.21.0 → v1.21.1; selected Go changes reviewed below | `8c8baa5627451b01217a1cdb9f0f91ca959407fd90f45f6435ab4255baf94e39` |
| `temporal-leaf-xsys-darwin-arm64` | Prometheus v1.21.0 → v1.21.1 and x/sys v0.47.0 → v0.48.0; exact unix Go changes reviewed below | `e8306298fcb518239d731bcf8f36484e029ede7ddfe72ab7b9268fddb1d9aabd` |

Source classifications:

- x/sys CPU `cpu.go` adds PPC64/RISC-V metadata and a nil-checked derived-feature
  hook; the only assignment to that hook occurs in `cpu_linux_riscv64.go`, which
  is excluded from the reviewed linux/amd64 source set. `cpu_other_x86.go` adds a
  NetBSD build exclusion. Linux/amd64 imports and assembly are unchanged.
- x/sys Darwin unix `syscall.go` replaces pointer arithmetic with `unsafe.Add`;
  `syscall_bsd.go` uses `range n`. Its assembly and exact runtime/syscall
  linkname directives are unchanged.
- Prometheus removes `atomic_update.go` and replaces its float helper calls in
  counter/gauge/histogram/summary with inline CAS loops. Darwin collector source
  adds `!ios`; the no-cgo memory stub is renamed and adds that same exclusion.
  The stub still returns not-implemented. No new foreign source or capability
  fact is introduced.
- libc's original/replacement source bytes and prepared package pin are
  unchanged. Its old whole-module inventory values used the historical
  `gomad3.adapter-source-inventory/v1` domain, while current source uses
  `gomad.adapter-source-inventory/v1`. Rehashing the same 4,161 original files
  gives old `7455942bdfcf64ff4d46cd874f1f6e80a79e4ebe6f4d98a9c2d1ae6aaabb59da`
  and current `40c930bdf315e3150b38b58b66cc55b60e236f551b5980720d44ee52cf501bbb`;
  the same 4,163 prepared files give old
  `94c69a969829324cfbcfe9382b5abe2ad73009fbbc94bd469f903d9ea0304681` and
  current `12c9adff0eea66b77c12819f7aafe7109e9a3a9804865fc8e00da94efc3e7fdc`.
  The prepared package pin stays `39d7fabdd955565c06edef231c35d384595799edd349d6ab8fc9278c46da8500`.

All seven untouched request/report/pack triples (21 files), including the
historical v0.41/v0.47 fixture requests, are byte-identical to HEAD. Their module
mappings, activation meaning and exact historical evidence remain unchanged.

## Retained source evidence and commands

Local evidence remains under `/home/agent/wt/wasm-milestones-pins/.flow/tmp/fn189-pins/`.
`reconciliation-receipt.json`, generated by `node receipt.mjs`, contains complete
old/new activation/module evidence, per-package Go/foreign deltas, every retained
fact, exact approval digests, all 21 preservation hashes, inventory hashes/counts
and the independently reproduced libc domain hashes. The five source inventories
retain the full capability closure: imports and nonstandard selected Go/foreign
source hashes/linknames; standard-library membership uses the pinned Go release.

| Inventory | Packages | Go sources | Foreign sources | Imports | Linknames |
| --- | ---: | ---: | ---: | ---: | ---: |
| `libc-source-inventory.json` | 1551 | 6892 | 12 | 17877 | 47 |
| `compute-darwin-source-inventory.json` | 1473 | 6726 | 11 | 16444 | 47 |
| `temporal-functional-compute-linux-amd64-source-inventory.json` | 1455 | 6763 | 22 | 16370 | 43 |
| `temporal-functional-tests-darwin-arm64-source-inventory.json` | 1053 | 4465 | 10 | 12254 | 47 |
| `temporal-leaf-xsys-darwin-arm64-source-inventory.json` | 866 | 3425 | 6 | 8802 | 8 |

`gomadtool-final` is the locally built stock-Go source authoring executable.
For each request, run from the worktree root:

```sh
gomadtool-final compatibility-pack discover --root="$PWD/tools/gomad" \
  --request=internal/backends/native/compatibilitypack/requests/ID.json \
  --working-dir="$PWD" --source-platform=DECLARED_OS/ARCH \
  --source-go=PINNED_STOCK_GO --source-inventory=ABSOLUTE_INVENTORY_JSON
gomadtool-final compatibility-pack review --root="$PWD/tools/gomad" \
  --request=internal/backends/native/compatibilitypack/requests/ID.json \
  --output=internal/backends/native/compatibilitypack/reports/ID.md
gomadtool-final compatibility-pack generate --root="$PWD/tools/gomad" \
  --request=internal/backends/native/compatibilitypack/requests/ID.json \
  --approve-review=sha256:EXACT_REVIEW_DIGEST
```

Every discover/review/generate command exited 0. The libc final recapture returned
the same approval digest as its first source discovery.

| Check | Exit | Evidence/result |
| --- | ---: | --- |
| `compatibility-pack check --root=.../tools/gomad` | 0 | All generated packs/reports/index/request approvals current |
| `compatibility-pack refresh --root=.../tools/gomad --go=PINNED_STOCK_GO` | 0 | `linux/arm64: 0 requests selected`; existing per-module mapped impact gate has no stale mapped request |
| `go test -p=1 -tags=gomad_native,test_dep -count=1 -run 'TestSourceReview\|TestInspectSource\|TestCompatibilityPackSource\|TestRunCompatibilityPackRefresh\|TestProjectCapabilityReview\|TestReviewGoCapabilityReview' ./build ./transcript ./preparation ./qualification/analysis ./cmd/gomadtool` | 0 | Fresh final run after alias correction, `source-regressions-alias-final.log`, `GOMAXPROCS=2`; source-platform/tag inventories, direct/aliased exact adapters/profile goldens, source-only provenance, explicit-input rejection, host execution refusal, existing collection/project/refresh controls |
| Canonical affected native-source `make lint-code`, existing Linux tools, exact relative patch, five changed packages, fixes disabled | 0 | Fresh final run after alias correction, `native-lint-alias-final.log`; zero reported issues, followed by successful canonical errortype vet; `GOFLAGS='-p=1 -tags=gomad_native' GOMAXPROCS=2` |
| Expanded selection adding `TestCapabilityReviewGoldenSemantics` | 1 | `source-regressions.log`; six inherited digest-golden failures |
| Same golden test at baseline `bad3c401b5` | 1 | `baseline-golden.log`; all six candidate and baseline got/want JSON pairs identical; baseline has only another worker's unrelated `wasmhost/tests/cluster.rs` edit |
| Raw root `pin-impact --root=.../tools/gomad --go=PINNED_STOCK_GO --json --output=root-pin-impact.json` | 1 | Exactly 20 invalidated historical pack-rule rows across four historical fixture variants, no changed activation semantics |
| `git diff --check` and historical fixture diff | 0 | No whitespace findings; preserved files verified by receipt |

The raw root all-pack report deliberately still compares historical v0.41/v0.47
alternatives with the root module's current pins. Its 20 rows belong to
`golang-x-sys-v047-darwin-arm64`, `modernc-libc-xsys-v041`,
`modernc-libc-xsys-v047`, and `modernc-libc-xsys-v047-linux-amd64`; their own mapped
fixture modules retain the exact historical pins. The existing mapped gate in
`compatibility_pack_refresh.go:packPinImpact` remains unchanged. A green mapped
reconciliation does not redefine the raw root report as passing.

An earlier owner stop interrupted the first lint attempt during tool installation
(session `31412`, exit 143; `native-lint.log`). That attempt supplies no pass. The
resumed source-only scope used the already installed Linux tools through an
explicit Make assignment, avoiding installation and broad package loading:

```sh
GOENV=off GOWORK=off GOTOOLCHAIN=local GOMAXPROCS=2 \
GOFLAGS='-p=1 -tags=gomad_native' PATH="PINNED_STOCK_BIN:$PATH" \
make lint-code LOCALBIN=PRIMARY/.flow/tmp/fn189-linux/source-gates/lint-bin \
  LINT_CODE_DIR="$PWD/tools/gomad/internal/backends/native" \
  'LINT_CODE_TARGETS=./build ./transcript ./preparation ./qualification/analysis ./cmd/gomadtool' \
  ALL_TEST_TAGS=gomad_native,test_dep GOLANGCI_LINT_BASE_REV=bad3c401b5 \
  GOLANGCI_LINT_PATCH="$PWD/.flow/tmp/fn189-pins/native-lint-resumed.patch" \
  GOLANGCI_LINT_FIX=false
```

The fresh maintainer-tool build, generated pack check and mapped refresh also
exited 0 using `gomadtool-alias-final`. The refresh again reported
`linux/arm64: 0 requests selected`. The staged review diff contains exactly 27
task-owned source/generated/documentation files, including all six new source
and test files; this handover remains separate. `git diff --cached --check`
exited 0. The alias correction is included in the frozen source diff; generated
inputs remain unchanged. A freshly generated `reconciliation-receipt-resumed.json`
is byte-identical to the retained receipt, preserving the five exact approvals,
zero fact/disposition delta and all 21 historical file hashes.
The exported staged diff `review-ready.patch` has SHA-256
`ded9f6e83df44eb9d4a6952474f1e2f78cc7d90a2e735c6ae4e5948bba2efbe0`;
the resumed reconciliation receipt has SHA-256
`e9063f29bcb91109864fe8e2c042622c059f49759af8a62091873e42f80fcfa9`.

Independent fresh-context source review and integrated `make -C tools/gomad
validate` belong to the root conductor before acceptance. This source-ready
checkpoint makes no guest/runtime/resource qualification or broad WASM claim.

The worktree and ignored evidence/cache directories are retained for source
review/integration and exact reruns. No commit, push, PR, CI, native execution,
runtime semantic source change or Flow task-state write occurred.

## Integrated source checkpoint

Root integrated all 27 reviewed source files into the primary `gomad` worktree
at `95eed5c83a` plus the uncommitted batch; SHA-256 comparisons matched every
worker file exactly. `make -C tools/gomad validate` exited 0 with pinned stock
Go 1.27.1, `GOMAXPROCS=2` and serial package compilation. Primary log:
`.flow/tmp/fn189-pins-integrated-validate.log`.

The Flow implementation-review wrapper examined only committed `base..HEAD`,
which was empty, so its three SHIP responses are explicitly **not acceptance**.
The workflow could not cover the owner-requested uncommitted patch. A fallback
fresh-context Codex reviewer (`gpt-6.1-sol`, high, tool-enforced read-only,
same model family as writer) instead inspected `git diff --cached HEAD`,
confirmed all 27 paths and six new source/test files, and returned SHIP with no
actionable findings. It explicitly checked compiler aliases, source-set pins,
unchanged capability dispositions, unchanged execution admission, 21 historical
fixture files and the inherited golden mismatch. Review output remains at
`/home/agent/wt/wasm-milestones-pins/.flow/tmp/fn189-pins/staged-source-review.md`.
No commit, native execution or workload/performance qualification occurred.
Fn-189 remains in progress; this checkpoint proves only the source batch.

The primary worktree also reran the five-package focused source regression
selection above with `-p=1 -tags=gomad_native,test_dep -count=1` and
`GOMAXPROCS=2`: exit 0. Log:
`.flow/tmp/fn189-pins-integrated-source-tests.log`. Integrated `git diff --check`
also exited 0. Affected-source lint was run against the byte-identical reviewed
worker files and is not claimed as a full repository lint pass.
