# Task 23 source progress

The module-aware fast selector and scoped Gomad/mixedbrain targets are implemented. Real Git, Go package inventory and Make dispatch regressions pass, including the root integration harness's existing build tag. Qualification remains open because actual unchanged-policy lint reports existing source findings.

Task `fn-109-gomad-deepen-modules-and-tool-interfaces.23` remains `in_progress`. Root owns staging, commits, Flow and formal review under the explicit dispatch override. Base and HEAD are both `b43aeb5b15d438eebab65f2ce48eecb19e76e55c`; the worker commit range is empty. No worker command remains live.

## Source and policy

`cmd/tools/lintcode` reads the existing architecture literal classifications and runtime overlay descriptor. It rejects unknown modules, uncovered host files, missing live modules, invalid manifests and module/source symlinks. It retains exact fixture/evidence dispositions, selects changed root packages using the existing Git merge-base semantics, and checks all ordinary source packages in the two nested modules. Gomad's root harness and direct toolchain package are included; runtime overlays are not recursively linted. Future ordinary simulation Go source is inventoried rather than omitted by a fixed package list.

Root-owned hidden action tooling, simulation and integration source stay visible. Only the exact `tools/gomad3integration` package receives a separate batch with its existing `gomad3_integration` tag. Ordinary root and nested tags, tool pins, configuration, comparison rule and failure propagation remain unchanged. Both lint tools execute in the selected module cwd with the root config and pinned binaries. The full root target's `./...` behavior is preserved.

Existing CI now invokes mixedbrain lint in the linter job and Gomad lint in its host-tools job. Executable tests parse and run the actual new YAML commands through real Make. No remote CI execution is claimed. Gomad's default Make goal remains `generate`.

The latest executable freeze is [integration-tag-correction/verified/source-frozen.sha256](integration-tag-correction/verified/source-frozen.sha256). Corresponding source/tool after-checks passed. [Tool hashes](integration-tag-correction/verified/tools-frozen.sha256), [Go environment](integration-tag-correction/verified/go-environment.json) and each command receipt bind the exact argv, cwd, environment, HEAD, duration and exit code. Comparison remains main/merge-base `951c5516e9e7b3066e7e069adda9565cfd68844c`. No module manifests, config, pins, generator/runtime inputs or product source were edited.

## Verification

Fresh current-helper receipts under `integration-tag-correction/verified/` give these results.

| Check | Exit | Seconds |
| --- | ---: | ---: |
| Unfiltered helper golangci, no new-from filter | 0 | 0.910 |
| Helper errortype vet | 0 | 0.104 |
| Helper contracts with `test_dep` | 0 | 4.323 |
| Existing Gomad Make ownership test | 0 | 0.209 |
| Gomad `make validate` | 0 | 4.069 |
| Actual root fast lint | 2 | 150.532 |
| Actual tagged root integration batch | 0 | 52.729 |

The root fast gate now loads its 108 ordinary root packages and reports 57 revive findings, all in existing `tools/gomad3sim` source. Golangci exits 1, not the original loader exit 7. Its fail-fast behavior prevents vet and later integration/nested scopes from running. The separate [integration scope receipt](integration-tag-correction/verified/integration-scope.receipt.json) checks that previously unreached exact tagged batch using the same comparison and tools; golangci and errortype pass. It does not make the complete root gate green.

The original frozen nested captures remain immutable. [Gomad's actual gate](final-gomad-lint.receipt.json) exits Make 2/golangci 1 in 21.062 seconds after loading 55 ordinary host packages and reports 1,300 findings: 319 errcheck, 12 forbidigo, 2 gci, 14 goimports, 925 revive and 28 staticcheck. Its vet step is not reached. [Mixedbrain's actual gate](final-mixedbrain-lint.receipt.json) exits 0 in 15.198 seconds, including errortype. Its golangci result is under the existing comparison filter, not an unfiltered whole-module clean claim. These receipts bind the first freeze, before the exact root integration correction; their module source, config, manifests and ordinary tag policy are unchanged. Latest contracts reverify nested dispatch after that correction. No unrelated nested gate was rerun to replace those observations.

## RED to GREEN and baseline

Pre-edit ownership and validation passed. The unchanged-source 147.354-second real loader failure is reused from [task 21's diagnostic](../task-21/lint-tooling-diagnostic.md), not rerun. `selector-red.log` and `contracts-valid-fixture-base-red.log` reproduce the old selector's wrong module cwd and missing scoped targets against the actual base Makefiles in valid scratch repositories. The earlier `contracts-red.log` also contains fixture setup noise and is not relied on alone.

Latest contracts cover literal root/nested/hidden routes, tracked/untracked/staged/deleted/renamed Go source, exact exclusions, unknown modules and overlay near-misses, invalid comparison/policy inputs, empty/missing/symlink modules, both tool failures, actual CI commands and the preserved default goal. `integration-tag-correction/contracts-red.log` reproduces the missing integration batch before the correction; its focused GREEN and final full contracts prove the separate tags and failure propagation. `make-default-red.log` retains the introduced default-goal defect before its fix. Introduced helper lint findings, including corrective batching complexity, were fixed within Touches and their failed logs retained.

The known selector introduction is `109a38e8ca4827ae8c624fc1a9382290dcae0f69`, as grounded by the admitted source investigation. No broad historical bisect, external PR lookup, worktree or reproduction commit was performed; root explicitly owns commits. Similar-code investigation reused the existing architecture/descriptor classifications rather than adding a parallel module policy or generic plugin framework.

## Outstanding qualification

`SCOPE_EXCEEDED` applies to unchanged source lint findings outside this task's Touches. Root simulation lint owns the 57 revive findings, for example `tools/gomad3sim/cluster.go:80` and `controller.go:136`. Ordinary Gomad host lint owns the 1,300 findings, for example `artifact/manifest_copy.go:59`, `artifact/open.go:39`, and unchecked closes in `artifact/open.go:270`. Raw logs retain every finding. Root should admit a separate bounded source/policy owner before changing these files; no mass fix or policy relaxation was attempted here. Original R19 qualification and full-tree lint remain open.

The existing `^.git` config regex remains a visible reporting-policy limitation for `.github` source even though routing includes that live tooling. Native patched Go is absent, and Linux aarch64 is not either required darwin-arm64/linux-amd64 qualification host. D5, native and original R18/R19 acceptance are not closed by these stock-Go tooling checks. Task 20's original blocked acceptance is unchanged.

stage: impl-review - skipped(policy: conductor-deferred; root owns formal review and no passing whole-tree gate is claimed)

Tier session (jev-unavailable(no_key)). Requested implementer gpt-6.1-sol at high; actual model is unknown without host execution metadata. Two read-only scouts were delegated, with no concurrent source writer. No implementation bridge was used. The verification and code-style skills drove behavioral RED/GREEN evidence and removal of introduced lint findings; all lifecycle and commit steps were deferred under root's explicit override.
