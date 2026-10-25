# fn-256 dependency survey

The six Scala-era Go subtrees can be removed without breaking the observed dependency closure of the retained command, Temporal Driver, Testpilot, canary or integration tests. The plan's repository-wide zero-reference acceptance conflicts with protected archived sources and historical records. Correct that acceptance before implementation.

Surveyed the working tree at HEAD `e00dc1b89b14fe33fb292450381d7449a762d4ed` on 2026-10-25. Read AGENTS.md, fn-256 spec and tasks 1/2, Umpire specification/architecture/module-map material and the relevant source/build files. Only this scratch report was written. No build, vet, test, cleanup, tracked-file edit or Flow mutation was performed. The report follows the flow-next-prose contract read from `/home/agent/.codex/skills/flow-next-prose/SKILL.md` and its prose document.

## Dependency evidence

Every shell explicitly unset BASH_ENV, changed into `/Users/stephan/Workspace/skunkworks/umpire/temporal`, and used login=false.

1. `rg -n 'go.temporal.io/server/tools/umpire/(ir|interp|check|realization|export|internal)' --glob '*.go'` with exclusions for those six subtrees returned no remaining literal full-path matches. Follow-up shorter-path searches found one synthetic test string, described below.
2. `mise exec -- go list -mod=readonly -deps -tags test_dep ./tools/umpire/cmd/umpire ./tools/umpire/temporaldriver ./common/testing/testpilot/... ./tools/canary/...` returned only `tools/umpire/cmd/umpire` and `tools/umpire/temporaldriver` within `tools/umpire`. Repeated with `-test` and obtained the same package set plus their test binaries.
3. With `set -o pipefail`, ran `mise exec -- go list -mod=readonly -deps -test -tags 'test_dep integration canary_harness' ./tools/umpire/cmd/umpire ./tools/umpire/temporaldriver ./common/testing/testpilot/... ./tools/canary/... ./tests | rg 'tools/umpire/'`. Exit 0. Again only retained command/driver and their test variants appeared. This includes the canary harness and repository integration test dependency graphs. It establishes import resolution, not compilation or passing tests.

| Retained caller | Actual domain dependencies and evidence |
| --- | --- |
| `tools/umpire/cmd/umpire/main.go:21` | Testpilot `temporal/act`, `tools/umpire/temporaldriver`, `umpire.local/model/runner`. |
| `tools/umpire/temporaldriver/driver.go:9` | Temporal SDK client, Testpilot `temporal/act`, Temporal reference `worker`, `worker/raw`, `worker/sdk`. |
| `common/testing/testpilot/temporal/act` | Runner, generated Model/realization/Run/Temporal protocol packages; its tests also import `umpire.local/temporal`. No deleted tools package. |
| `tools/canary/controller/run.go:20` | Retained temporaldriver, Model host/runner and protocol packages. `controller.go` also imports committed Temporal guest package. |

The root module has local replacements at `go.mod:236,238` for `model/umpire/hosts/go` and `model/temporal/hosts/go`. Those modules and generated/embedded guest inputs remain required. Deleting the six tools trees does not authorize deleting generated Umpire API schemas, Rust/Go host code or their assets. Root `go build ./...` is not a substitute for nested-module tests, though it builds imported nested packages.

`common/testing/testpilot/dependency_boundary_test.go:119-120` constructs a forbidden `tools/umpire/ir` import as test data using `serverModule + ...`. It is not an import. Preserve this negative control; optionally update its synthetic forbidden package to retained `tools/umpire/temporaldriver` if cleaning obsolete examples is desired. That optional source edit must be named in task scope.

## Exact build and CI surfaces

- `Makefile:1-8` and matching `endif` at 1268 own the special `check-framework` wrapper. The normal default goal is `all` at line 17.
- `Makefile:440` lists live and retired `.PHONY` targets together. Remove only retired entries.
- `Makefile:530-539` owns the Quint backend comment, installer and check targets. `export/quint.sh:3` pins npm Quint 0.33.0; `export/tools_test.go:28` repeats that version. Deleting export removes both pins/helper. There is no separate Quint pin in mise.toml.
- `Makefile:541-817` owns the archived Scala block. Delete its variables, helper functions and targets together, retaining the named-scratch half of `umpire-clean-scratch` at 808-816. Remove its broad `find` cleanup of Scala state. The scratch target is explicitly documented as used by agents.
- Preserve `canary-build` at 442, Testpilot protocol at 447, `umpire` at 460, guest and live-test targets at 465-527. Live checks include slice and gaps prerequisites and execute integration-tag tests.
- `.github/workflows/umpire.yml:34` runs `go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...`; line 36 runs live tests. There is no explicit Quint/backend installation or execution step to delete. The canary job retains its production build, package tests and `canary_harness` tests. Its exact checks are guarded by `tools/canary/workflow_test.go:22-43`.
- `mise.toml:4` is the Scala CLI pin. Keep Java and Rust tools.
- The Rust Make section begins at the stable marker `##### model: Umpire in Rust (fn-210) #####`, currently line 1270. Compare this suffix against the lane's admitted baseline, not hard-coded post-edit line numbers or a stale pre-fn254 tip.

## Current documentation versus preserved history

Allowed current-doc edits are the files already named by the plan. `AGENTS.md:7` says Scala targets remain unmaintained; line 58 still describes the old reader/lowering/conformance. `model/README.md:8-10` likewise says the Scala targets remain. `tools/umpire/README.md` introduction, package table, ownership description and Checks section describe the six obsolete packages and Quint target; retain the live CLI and driver configuration.

`.plans/UMPIRE_MODULES.md` is mixed current architecture and history. Current Rust ownership begins at line 27. The historical Scala/Go ownership section begins at line 72 and contains obsolete package rows at 109-115 alongside retained live host/Runner/Testpilot/driver/canary rows around 117-128. Later migration sections contain historical reader paths and old targets, including lines 454-456 and 512-525. Add an explicit retirement boundary, accurately present surviving live ownership, and preserve historical statements as history. R3 should require accurate current guidance, not erase historical records. Do not globally rewrite `.plans/*.md`, migration audit/manifest JSON, `.plans/archive`, `.turbo/plans`, or `docs/superpowers/specs` to satisfy a textual absence check.

The archive contains protected references outside README, including `framework/dsl/Domain.scala:4`, `framework/dsl/Action.scala:302`, `framework/check/Samples.scala:56-57`, framework export fixtures, disabled Temporal gate code and generated legacy lint JSON. These files must remain byte-identical. The archive README itself contains historical descriptions and should receive a retirement note without rewriting its historical body. Name the actual deletion commit in a subsequent documentation commit; a commit cannot embed its own final hash.

## Minimal plan corrections and controls

1. Replace task1's global `rg` absence acceptance with absence of the six directories, no live Go imports/dependency edges to them, and no executable Make/CI backend hooks. Allow historical documentation/archive references and the synthetic boundary-test string. A global `rg` depends on ignore/hidden settings and can include generated output or omit hidden `.plans`; widening it can sweep caches or worktrees. Use explicit tracked paths.
2. A concrete bounded import scan is `git grep -nE '^[[:space:]]*([[:alnum:]_.]+[[:space:]]+)?"go.temporal.io/server/tools/umpire/(ir|interp|check|realization|export|internal)(/|\")' -- '*.go' ':!model/archived/**'`, after deletion. Pair it with the successful `go list` forms above; import scans alone do not prove compilation. Check hooks separately with `git grep -nE 'umpire-(install|check)-backends|tools/umpire/export' -- Makefile .github/workflows/umpire.yml`.
3. Explicitly add `make umpire-check-live-tests` to task1's acceptance, matching spec R1. Task2's dry-run does not execute it. Preserve canary unit/harness checks when changing CI; run the existing workflow test plus retained package tests. Use `-tags test_dep` for tests as AGENTS requires. For vet, prefer `mise exec -- go vet -tags test_dep ./tools/... ./common/testing/testpilot/...`; if retaining the original untagged vet requirement, record it separately rather than silently treating it as equivalent. Untagged builds remain useful production checks. `test_dep` selects actual fault/testhook implementations, so the modes are materially different.
4. Task2 asks for `time make -n help`, but `make -n help` currently fails with `No rule to make target 'help'`. Replace it with `time make -n canary-build >/dev/null` before and after, a real target with no Scala prerequisite. Keep `make -n model-check` as parse coverage; do not call dry-run evidence equivalent to R4's behavioral gate.
5. Record archive preservation using a before/after tracked tree or diff check excluding only README; capture and compare the Rust Make suffix at the stable marker. Serialize integration against fn254 so legitimate predecessor changes are not mistaken for fn256 edits. For R4, reuse valid integrated fn254 gate evidence where admissible and identify its baseline, or schedule the real gate once. Do not claim a dry run proves behavior.

These corrections keep the intended deletion scope and the owner's reading-material constraint. No retained caller requires a replacement implementation.
