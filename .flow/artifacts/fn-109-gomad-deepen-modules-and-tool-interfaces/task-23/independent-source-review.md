# Task 23 independent source review

Task 23 is ready for a source progress commit. I found no actionable defect in the frozen six-file correction. Existing lint failures, inherited path-policy defects, remote CI execution and both native qualification gates remain open.

## Scope and reviewer

This fresh-context review assesses the current uncommitted product diff against `b43aeb5b15d438eebab65f2ce48eecb19e76e55c`, which remains HEAD. The commit range is empty. I read the four changed Make/workflow files and both untracked `cmd/tools/lintcode` files. Spec, task 21 JSON, milestone and task 23 lifecycle records belong to the conductor and are outside the product-change verdict.

I read the full project AGENTS.md and Gomad README, applicable milestone delivery instructions, the fn-109 spec and original R19 criteria, task 23, source admission, handover, evidence and root observations. I examined actual Make/CI/config inputs, the architecture literal classifications, overlay descriptor, affected ownership test and the new inherited path-policy observation. I applied the review-code criteria within the conductor's single-reviewer, read-only scope and used the prose and verification contracts for this report. No additional reviewer or backend was dispatched.

The requested reviewer and writer are both `gpt-6.1-sol at high`, the same requested model family. Dispatch reported `Tier: session(jev-unavailable(no_key))`. Actual execution model identity is unknown because host metadata is unavailable. This report is an independent source-progress assessment, with no formal SHIP or native/full-goal acceptance claim.

## Strengths

- `Makefile:495` replaces the filesystem-directory selector with module-aware dispatch while retaining the fast target's Git merge-base comparison. `cmd/tools/lintcode/main.go:135` keeps `--no-renames`, tracked working-tree changes and untracked Go files. Root scopes include only changed directories; a selected nested owner checks its complete ordinary host source inventory. Deleted empty root packages disappear, and remaining source in changed directories still receives analysis.
- `cmd/tools/lintcode/main.go:252` reads the existing literal fixture/module policy and exact overlay descriptor. Unknown modules, missing required live modules, malformed/nonliteral policy and unlisted runtime overlay source produce errors. `main.go:419` checks source membership against real Go package metadata, validates selected source paths against symlinks and rejects unsafe package-directory characters before shell dispatch. The exact root fixtures and `.flow` evidence receive visible dispositions. The policy introduces no generic module registry or dependency.
- `cmd/tools/lintcode/main.go:166` and `main.go:194` retain root simulation, hidden action tooling and integration ownership. The exact root integration package receives its existing `gomad3_integration` tag in a separate batch. Ordinary root and nested tags retain their existing values. Gomad's root test harness, direct `toolchain` package, developer packages and ordinary simulation source enter the nested inventory; runtime overlays and declared qualification fixtures do not enter the ordinary host scope.
- `Makefile:504` executes both pinned tools in the selected module directory with the unchanged root configuration. The helper returns each Make failure, preserving fail-fast behavior. The root full target retains its `./...` default. `tools/gomad3/Makefile:35` exposes the nested gate after `generate`, preserving the existing default goal.
- `cmd/tools/lintcode/main_test.go:22` uses independent literal ownership expectations and real scratch Git repositories, Go inventory and Make recipes. Tool doubles record cwd/argv and inject failures at both lint steps; they do not establish real linter results. The retained actual frozen runs establish those separate boundary observations. `main_test.go:277` parses and executes the actual new CI commands. `.github/workflows/gomad3.yml:44` fetches complete history for the existing `HEAD~` comparison, and `gomad3.yml:57` plus `linters.yml:177` place the nested gates in their existing jobs without a waiver.

## Issues

### Critical

None identified in the reviewed source correction.

### Important

None identified in the reviewed source correction.

### Minor

None identified in the reviewed source correction.

The recorded qualification gaps are material, but they are not introduced source-review findings. Root fast lint ends with Make 2/golangci 1 after loading 108 ordinary root packages and reporting 57 existing simulation revive findings. It stops before root vet, integration and nested scopes. The separately retained exact integration batch returns 0 for golangci and errortype. The original Gomad nested receipt reports Make 2/golangci 1, 55 loaded host packages and 1,300 existing findings; its vet step is not reached. Mixedbrain's original receipt returns 0, including vet, under the existing comparison filter. It is not an unfiltered whole-module cleanliness result.

Those two original nested receipts bind the first freeze. They do not bind later helper bytes. Nested Go source, module manifests, config and ordinary tag policy remain unchanged, and the current independently run contracts exercise the final nested dispatch. The handover and evidence preserve this limitation correctly.

The inherited config-relative path base and doubled plain-YAML regex escapes are recorded in [lint-policy-path-observation.md](lint-policy-path-observation.md). The current config SHA matches BASE. Neither mechanism was introduced by task 23. The `.github` reporting limitation also remains explicit. Those observations require a separate policy owner, rather than reclassifying current failures as green.

I checked the parent's patch-version reproducibility question against the local pinned golangci v2.13.0 source. `pkg/goutil/version.go:19` reduces the compiler runtime version with `go/version.Lang`; `version.go:26` compares it with `TrimGoVersion`, whose implementation at lines 44-57 drops the target patch version. A binary built with Go 1.27.0 therefore does not fail that check merely because Gomad declares 1.27.1. Actual Go commands must still satisfy the nested module's Go directive. This source inspection establishes no new binary-installation or native-execution result.

## Independent verification

All 15 entries in [the latest freeze](integration-tag-correction/verified/source-frozen.sha256) matched before and after the independent checks. All three tool hashes matched before and after. The helper source SHA is `3808f1512e6e05f2a03a6bdd305619c18b298b3ec97370a1364db8df6e9759b9`; its test SHA is `a15cbf6d3e27acd8decb14071fa6e99405316424f270ad72ffc6c386b0b891ae`. No tracked Go product source differs from BASE. HEAD and the clean index stayed unchanged.

[The independent receipt](independent-source-review-checks.json) retains exact commands, raw terminal output, exit codes, tool-call elapsed times, scope and source identities. Commands ran from `/Users/stephan/Workspace/skunkworks/gomad/temporal` with pinned stock Go 1.27.1 on PATH, `GOENV=off`, empty GOFLAGS, `GOWORK=off`, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, and both Gomad seeds plus `LINT_TEST_BASE_REV` unset.

| Independent check | Exit | Wall seconds |
| --- | ---: | ---: |
| `go test -count=1 -tags test_dep -v ./cmd/tools/lintcode` | 0 | 3.385 |
| Unfiltered pinned helper golangci, `--fix=false`, unchanged config | 0 | 0.105 |
| Pinned helper errortype vet | 0 | 0.048 |
| `git diff --check` | 0 | <0.001 |

The helper package reports `ok` in 3.203 seconds. I inspected the retained source-bound ownership, validation, broad root and nested receipts without rerunning broad lint or native gates. No independent mutation test or worktree was created under the read-only/no-worktree scope. All tool calls terminated; no live process/session handle remains. I added only this report and its new independent receipt.

## Recommendations

Root should commit the reviewed source progress and evidence together before starting another implementation task, recording the original task/R19 qualification as open. Keep the old failed receipts immutable. Give inherited lint path policy and remaining source findings explicit bounded ownership, preserve hidden tooling's reporting limitation, and retain actual remote CI and both native platform results before closing acceptance.

## Assessment

Ready for SOURCE PROGRESS COMMIT: **yes**. No source fix is required by this review. Formal SHIP, full-tree lint, D5, original R18/R19, darwin/arm64 and linux/amd64 qualification remain unaccepted by this report.
