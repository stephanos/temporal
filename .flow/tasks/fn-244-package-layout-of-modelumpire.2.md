---
satisfies: [R2]
---
# fn-244-package-layout-of-modelumpire.2 Go module split into host, runner and gen; Kettle Go under reference/kettle

## Description
After fn-228.6, before fn-228.7. Split the Go module umpire.local/model: the wasm Host moves from the root package to hosts/go/host, the Runner to hosts/go/runner, generated protobuf packages to hosts/go/gen/...; the proto-go freshness check covers exactly gen/ with no exception. Move Kettle's Go server and Kettle-specific host tests under reference/kettle/ as their own Go module with a replace directive. Update all importers (tools/umpire/cmd/umpire, common/testing/testpilot/temporal/act, tools/canary, model/temporal Go), the fn-228 spec's import path and the module map.
## Acceptance
- [ ] host, runner and gen packages; freshness check without exceptions
- [ ] Kettle Go server and tests under reference/kettle
- [ ] Importers updated; Go tests, live slice and default model gate pass
## Done summary
Go callers now use separate `host`, `runner` and generated `gen/umpire/.../v1` packages. Kettle's independent Go server, generated API and parity/live tests live in module `umpire.local/kettle` under `reference/kettle`, with a local replacement onto the host module.

Updated every active importer, the Driver dependency allowlist and its controls, all protobuf generators, default model/parity gates, fn-228's import path, module map, README, MIGRATE and AGENTS paths. The freshness check compares the complete `gen/` tree and rejects symlinks without a Runner exception. Extra-file and symlink negative controls both failed as intended. Kettle parity assertions and all goldens remain unchanged; the rebuilt committed Temporal guest is byte-identical.

Verification passed at e104d5b199. Focused command/Testpilot/canary tests, generic host/runner tests, independent Kettle tests, all three protobuf freshness checks, `make umpire-guest`, `make model-check`, live slice and gaps, and `make lint-code-fast` passed. The default model gate took 579.51 seconds, exceeding the 60-second R30 soft target. Live gaps took 411.117 seconds; the combined live command took about 530 seconds. Logs are `.flow/tmp/fn2442-{focused,host,kettle,guest,model,live,lint}.log`.

Baseline `make model-check` passed. The broad pre-edit Go command was red in unchanged legacy tooling. `TestActivityEveryClaimDeclarationIsLifted` reads removed `model/temporal/features/activity/standalone/product/Product.scala` and `system/Timeouts.scala`; `TestDeadlineSelectorsAndBeforeStateAgreeWithTheReader` reads removed `model/irgen/testdata/lifts/expected/deadlineCapabilities.json`; interpreter fixtures also read removed `presence.json`. The check/export packages timed out at ten minutes. The conductor approved focused importer verification and preserved those tests unchanged. Full baseline output is `.flow/tmp/fn2442-baseline.log`.

Review retained one P3 module-map path finding. Commit 6ff3963bac corrects the documented Runner dependency path; the single re-review returned SHIP. Only Markdown and Flow records changed after the full gates, confirmed by the scoped source diff; the conductor explicitly directed reuse of those completed gates. The cumulative classifier reports FULL, so no docs-only or new-head green-receipt claim is made. Gate receipts remain attached to e104d5b199. The mechanical documentation fix required no memory capture.

stage: impl-review - ran [2026-10-24T02:43:18Z..2026-10-24T02:47:21Z] (codex:gpt-6.1-sol:high; same writer/reviewer family; NEEDS_WORK then SHIP)

The checkpoint includes the conductor's MILESTONES sequencing update. The conductor still owns marking this row complete and starting fn-244.1; this task leaves the Rust-layout work open.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e104d5b199e9673cd5faee5319591e3b39645731, 6ff3963bacdef8a1a71e1b6a2e5cf018b3e5529f
- Tests: go test -tags test_dep ./tools/umpire/cmd/umpire/... ./common/testing/testpilot/... ./tools/canary/..., CGO_ENABLED=0 go -C model/umpire/hosts/go test -tags test_dep -count=1 ./... (with caught-panic/overflow/native control environment), CGO_ENABLED=0 go -C model/umpire/reference/kettle test -tags test_dep -count=1 ./..., make umpire-guest, /bin/bash model/scripts/proto-go.sh --check, /bin/bash model/scripts/proto-kettle-go.sh --check, /bin/bash model/scripts/proto-temporal-go.sh --check, proto-go freshness negative controls: extra generated-tree file and symlink both rejected (exit 1), timeout 600 make model-check, make umpire-check-model-slice umpire-check-model-gaps, make lint-code-fast GOLANGCI_LINT_BASE_REV=f7043429349781a2c8d0269cd7d156c6f15fddb8 GOLANGCI_LINT_FIX=false, git diff --check; active retired-Go-import/path sweep; Kettle parity assertion equivalence, baseline: red (go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...; inherited legacy Scala/IR fixture errors and check/export 10m timeouts)
- PRs: