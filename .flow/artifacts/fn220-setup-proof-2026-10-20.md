# fn-220 one-command Temporal setup proof

The one-command setup, retained `TestFrontendSystemInfo` execution and exact
replay passed on clean integrated candidate
`29f49b17bc00f567c8ab3e660f6413988404c3f4` on Linux/aarch64. This is fn-220's
consumer proof, not the separate fn-222 one-box qualification or support for any
other Temporal test. The integrated batch review belongs to the conductor.

## Decisions and requirement coverage

- R1 / task .1–.2: extend `install` with `--temporal-checkout` and
  `--temporal-package`, keeping the existing explicit compiler, helper, runtime,
  cache and output inputs. `--config` and `--profile` cannot accompany derivation.
  The new path refuses every inherited `GOMAD_*` variable. Existing manual
  configurations and CLI defaults remain available.
- R1 / task .1: preserve the prototype's 256 MiB output, 1 GiB transcript,
  2^28 calls, 4096 descriptors, 100000 files, 512 MiB filesystem and 4096 pending
  events. Network defaults preserve the inspection fixture's 128 listeners,
  4096 connections, 128 MiB total, 4 MiB per connection, 10-second stall bound
  and 2^20 transitions. Default memory is 4 GiB and fuel is 2^50.
- R3 / task .1: the command and both named fixtures use `wasi.TemporalConfig`
  with explicit complete `TemporalPolicy` inputs. Shared production derivation
  owns all network/environment/entropy/argv/writable-root/output/transcript
  policy fields; callers make no post-derivation adjustments. A unit test
  compares complete serialized prototype configuration bytes with the prior
  fixture expression, and verifies schema mutation changes captured identity.
  For identical checkout/directory/policy/argument inputs the command and fixture
  derivation produce identical final serialized configurations, compared without
  removing or replacing any field. Separate complete historical-policy tests
  preserve prototype nil network, inspection nil argv/zero entropy/debug
  environment/reduced bounds, and qualification nil argv for Runner. The
  installation policy preserves the bounded network used by the consumer proof.
  Independent expectations check literal network limits and every policy's final
  bytes, including the historical inspection and qualification configurations.
  Installation tags are optional manifest inputs; preparation merges and seals
  their effective selection into its existing source/build identity.
- R2 / task .2: one seed passed, retained one success (215546369 bytes), and a
  fresh replay reported `reproduced=true diagnostic=false result=success
  choice-replay=exact`. The complete trace has 3943 records and 2943 decisions.
- R4 / task .3: README and WASM CLI guide contain install/explore/replay commands,
  retention flags, capacities and the named-test support boundary.

## Commands and exits

Source commits: `fb8044eca5` (.1), `7c8ef42504` (.2), `ff719ea3b4` and
`8896829710` (.3). Root integration produced the clean candidate above before
the guest was built. All commands below exited 0 unless explicitly noted.

Focused checks, run from `tools/gomad` before integration:

```sh
go test -tags test_dep ./internal/backends/wasm/wasi -run '^TestTemporalConfigPreservesPrototypeBytes$' -count=1
go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./cmd/gomad/internal/cli -count=1
GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go GOMAD_WASM_TEST_HELPER=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost go test -tags test_dep ./internal/backends/wasm/backend -run '^TestBuildSelectionChangesInvalidateCompilationCache$' -count=1
```

The compiler-backed test proves installed tags select the same closure and
build identity as explicit tags. Command tests exercise install/discover,
missing compiler/helper/schema, inherited configuration and package escapes.
The complete command/installation/shared CLI packages passed again on the
integrated candidate with `GOMAXPROCS=2`.

From the repository root,
`make lint-code-fast GOLANGCI_LINT_BASE_REV=9cb5bc3149 GOLANGCI_LINT_FIX=false`
exited 0. The first lint attempt exited 2 for five capitalized error strings;
those were corrected before the successful rerun. An initial broad WASI/backend
test attempt without `GOMAD_STOCK_GO` exited 1 because the existing integration
gates require that explicit compiler; it is not a full-suite pass.

Pinned builds on the integrated candidate:

```sh
cd /home/agent/wt/fn220/tools/gomad
GOMAXPROCS=2 go build -o /home/agent/wt/fn220-gomad ./cmd/gomad
CARGO_BUILD_JOBS=2 cargo +1.99.1 build --manifest-path internal/backends/wasm/wasmhost/Cargo.toml --release --locked --target-dir /home/agent/.cache/fn220-helper
cd /home/agent/wt/fn220
CARGO_BUILD_JOBS=2 CARGO_TARGET_DIR=/home/agent/.cache/fn220-helper make -C tools/gomad/internal/backends/wasm helper
```

Direct Cargo output had link count 2 and installation correctly refused it
(exit 2). The normal `make helper` publication copied it to the required regular
single-link executable. No configuration JSON was manually authored.

The actual successful operator commands, from `/home/agent/wt/fn220`:

```sh
env -i /home/agent/wt/fn220-gomad install --temporal-checkout /home/agent/wt/fn220 --temporal-package ./tests/gomadfunctional --compiler /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go --helper /home/agent/wt/fn220/tools/gomad/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost --runtime-root /home/agent/wt/fn220/tools/gomad/internal/model/runtimesource --cache /home/agent/wt/fn220-proof.L2aUaa/integrated-preparation --compile-cache /home/agent/wt/fn220-proof.L2aUaa/integrated-engine --output /home/agent/wt/fn220-proof.L2aUaa/integrated-installation.json
/home/agent/wt/fn220-gomad --installation /home/agent/wt/fn220-proof.L2aUaa/integrated-installation.json discover > /home/agent/wt/fn220-proof.L2aUaa/integrated-discover.json
GOMAXPROCS=2 /home/agent/wt/fn220-gomad --installation /home/agent/wt/fn220-proof.L2aUaa/integrated-installation.json explore --working-dir /home/agent/wt/fn220 --env=TEMPORAL_ROOT=/workspace --env=TMPDIR=/tmp --env=TEMPORAL_TEST_LOG_LEVEL=warn --choices --choice-bytes=64MiB --output-limit=16MiB --seeds=0 --parallel=1 --keep-successes=all --success-limit=1 --success-bytes=4GiB --execution-timeout=15m --overall-timeout=30m --artifacts /home/agent/wt/fn220-proof.L2aUaa/integrated-artifacts go-test ./tests/gomadfunctional -- '-test.run=^TestFrontendSystemInfo$' -test.v -test.count=1 -test.timeout=72h -persistenceType=sql -persistenceDriver=sqlite > /home/agent/wt/fn220-proof.L2aUaa/explore.stdout 2> /home/agent/wt/fn220-proof.L2aUaa/explore.stderr
GOMAXPROCS=2 /home/agent/wt/fn220-gomad --installation /home/agent/wt/fn220-proof.L2aUaa/integrated-installation.json replay /home/agent/wt/fn220-proof.L2aUaa/integrated-artifacts/v1/campaign-20261020T153731.584924878Z-2cdd5bcf9444e63abbd7986154b0cfc1/successes/sha256-f23f559f9f5f54eb8f4b0af76cb7444a > /home/agent/wt/fn220-proof.L2aUaa/replay.stdout 2> /home/agent/wt/fn220-proof.L2aUaa/replay.stderr
```

## Retained identities and handover

Go: go1.27.1 linux/arm64; Rust: 1.99.1; engine: pinned Wasmtime 47.0.3.
The retained guest stdout is:

```text
=== RUN   TestFrontendSystemInfo
--- PASS: TestFrontendSystemInfo (0.00s)
PASS
```

SHA-256 values (without the `sha256:` prefix):

| Input/evidence | Digest |
| --- | --- |
| Compiler | `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64` |
| Helper | `59ee602eabb6204bf5ded6c9b5c3cfb167f6df13d9e07b20d3d1759654201ee4` |
| Model | `f9607ca9acb77614c5b247563bd586a5b1c5a3333bffead91392848fdcb14f29` |
| Runtime implementation | `4ac4238ff22066e38a7e67356f1388bb8d09995a3fc9f2bea2c57fce8c284ac4` |
| Installation JSON | `34096860a2f8783e4dfbaa7052dcad7b7b8ec6fe647666fdc57073582eaabff2` |
| Discover JSON | `6fa8920c21b27ef8e55c59dd361254290148f9fb47f533835b793ac0c3e91b01` |
| Explore stdout | `c89b04d1b8d4fcc8ecedffb2ded84c7897fa5297cb08ea770e1df3b479a7833b` |
| Explore stderr | `f790172e020bd8c4a917c877ffc9c0472162c682e97c49c939e79d3c6d26d379` |
| Replay stdout | `831124bf0b78915683d88a5ae976f76cc3d215be7f5a63d51307b3f37fd272cb` |
| Replay stderr (empty) | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Retained module | `29f90f0aca2dedcf1aabe262ed50744edaaaa0d31119be4413772e8c11a366be` |
| Complete choice trace | `07f521ac663a0cf165e82ee7768d2f5b4d44d500a7c74124b929fb6bb594cb1f` |
| Forced-choice tape | `c26f9b9848eb8ffa6e8848c8f25b3708b36830d6b378f8695e4b41e66b31591d` |

Keep worktree `/home/agent/wt/fn220`, executable `/home/agent/wt/fn220-gomad`
and scratch `/home/agent/wt/fn220-proof.L2aUaa` for conductor review and retained
replay. Bulk module, artifact, engine cache and raw logs remain local. The private
566 MiB Cargo cache `/home/agent/.cache/fn220-helper` was removed after verifying
the independently published helper digest; it is reproducible by rebuilding.
No Flow task state, PR, push, CI run or qualification claim was made by this worker.

## Review coverage correction

The first review follow-up added independent literal network expectations and
historical inspection byte coverage. Completion review correctly found that its
normalized base comparison did not satisfy R3. That comparison has been replaced:
the complete policy is now an explicit production derivation input, and the
command test compares final serialized configurations without normalization.
`TestTemporalConfigPoliciesPreserveCompleteBytes` independently verifies all four
complete policies; the inspection and qualification historical-byte tests remain.

Full command/installation/shared CLI packages and the focused prototype,
inspection and qualification smoke configuration tests passed (exit 0).
Temporarily changing the shared network connection limit from 4096 to 4095 made
both command and inspection tests fail (exit 1); restoring it made both pass
(exit 0). The mutation log is retained at
`/home/agent/wt/fn220-proof.L2aUaa/network-mutation-test.log`.
`make lint-code-fast GOLANGCI_LINT_BASE_REV=fc04600872
GOLANGCI_LINT_FIX=false` also passed (exit 0, zero issues).

After the complete-policy change, rebuilding the launcher and invoking the same
successful installation command with output
`/home/agent/wt/fn220-proof.L2aUaa/policy-installation.json` exited 0. `cmp` against
the retained `integrated-installation.json` also exited 0: the entire manifest,
not just its configuration, is byte-identical to candidate `29f49b17bc`'s input.
Both have SHA-256 `34096860a2f8783e4dfbaa7052dcad7b7b8ec6fe647666fdc57073582eaabff2`.
This preserves the actual consumer proof inputs without relabeling its candidate.

Complete-policy follow-up checks (all exit 0):

```sh
cd tools/gomad
GOMAXPROCS=2 go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./cmd/gomad/internal/cli -count=1
GOMAXPROCS=2 go test -tags test_dep ./internal/backends/wasm/wasi ./internal/backends/wasm/backend ./internal/backends/wasm/qualification -run 'TestTemporalConfig|TestTemporalInspectionConfigPreservesFixtureBytes|TestOneboxSmokeConfigurationUnchangedByFamilyCells' -count=1
cd ../..
make lint-code-fast GOLANGCI_LINT_BASE_REV=93b12381b7 GOLANGCI_LINT_FIX=false
```

The updated launcher is retained at `/home/agent/wt/fn220-gomad-policy` for review.

## Completion review

The first independent completion review found the normalized configuration
comparison insufficient for R3. Commit `ba57de4e5b` introduced complete explicit
policies and final-byte comparisons. The resumed same-session review returned
`SHIP` on 2026-10-20T16:09:30Z (exit 0), with all four requirements covered.
Flow records `completion_review_status: ship`, all three tasks done, and the
spec closed as done. The receipt and complete transcript remain in
`.flow/tmp/completion-review-receipt-fn-220.json` and
`.flow/tmp/fn220-completion-rereview.log`. Reviewer and writer are from the same
model family. Full changed-candidate WASM qualification remains under fn-219.4
and fn-222; this record does not claim that separate gate passed.
