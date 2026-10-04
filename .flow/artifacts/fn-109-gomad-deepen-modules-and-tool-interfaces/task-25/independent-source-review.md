# Independent task25 source-progress review

Ready for SOURCE PROGRESS COMMIT: yes. No actionable introduced Critical, Important or Minor defect was found. Qualification remains incomplete.

## Scope and method

BASE_SHA and HEAD_SHA are both `ff7da7419bbc8321dd4004e58249bb149ffd31df`; the committed range is empty. I reviewed the actual working diff in `tools/gomad3sim/controller.go` and the complete untracked `controller_resolution_test.go`, surrounding resolution/identity code, full AGENTS.md and Gomad README, original fn-109 spec including R18/R19, task25, applicable MILESTONES scope, task24 source review and remaining findings, and every task25 receipt, source freeze and relevant log.

This is a fresh independent source-progress review using the requesting-code-review template and verification-before-completion skill. Repository prose rules also apply. The explicit dispatch limits the reviewer to two report files; no additional agents, bridge, worktree, source/index/HEAD/Flow mutation, download, build, native gate or broad gate rerun was used. Root's concurrent metadata remains separately owned.

Requested reviewer and writer are from the same family, `gpt-6.1-sol` at high. Actual execution model metadata is unavailable. Parent reported `Tier: session (jev-unavailable(no_key))`; I read the parent's judge state without repeating or rerouting it.

## Strengths

- `controller.go:159` changes exactly the redundant inner enum switch into restart-versus-other branching inside the unchanged three-kind outer case. Both original condition expressions and their evaluation order, target assignments, incarnation check, node lookup, prior/candidate selection, outer cases, cloning and identity construction remain intact. No public API, configuration, suppression, fabricated default/network case, generated source or native behavior was changed.
- `controller_resolution_test.go:11` supplies literal admission expectations for all three lifecycle kinds across the six declared states and both operation states, giving 36 cases. Successful results compare complete realizations against three literal canonical JSON identity inputs; failure results require the exact sentinel and zero realization. The oracle hashes literal bytes directly instead of calling production identity helpers.
- `controller_resolution_test.go:55` exercises node absence, Match.Node mismatch, current/next incarnation, deterministic candidates, first matching prior target, missing prior, prior override and existing explicit-node fallback. Candidate slices are detached. `controller_resolution_test.go:105` exercises all five network kinds, absent endpoints, unknown kinds and detached partition/heal groups.
- Source freezes independently bind the old controller to HEAD. Adding/correcting characterization changes only the test; the final freeze changes only the controller. Old/final corrected characterization logs are byte-identical and both exit 0. The actual package lint RED reports exactly the unreachable five network cases; the identical command becomes GREEN after the two-branch rewrite.

## Issues

### Critical

None in the reviewed source change.

### Important

None in the reviewed source change.

### Minor

None that warrants a source change.

Inherited qualification failures below retain their owners.

## Verification and evidence

[Independent checks](independent-source-review-checks.json) retain exact commands, cwd, environment, observed start/end bounds, terminal exits and complete fresh command output. Fresh ordinary gomad3sim tests with `-count=1 -tags test_dep` pass, with Go reporting 0.012 s. Actual unfiltered v2.13.0 package golangci with the existing config/tags and `--fix=false` exits 0 with 0 issues; its own reported execution time is 239.935 ms. Package errortype vet exits 0 with empty output. No command yielded or left a live handle.

The focused final source manifest and all three executable hashes pass before/after checks. Controller SHA-256 is `fa6e7a2bb46dadc0fb70fd51b18ebf0c2d9c282872b8eba95e540936a8b6d01a`; test SHA-256 is `9ff4317eb69567c64f73ab2638b798eda77eff22f91dd1b0e0218465cc77621d`. HEAD and the index digest remain unchanged; scoped whitespace and gofmt checks have no diagnostics. These freezes cover their declared inputs, not the whole repository.

The initial draft characterization failed two mistaken candidate expectations and is retained. Independent SHA-256 arithmetic confirms the selected candidate index is 1. Correction preceded production editing and passed on the old controller. This draft failure is not meaningful lint RED; `lint-red.receipt.json` supplies the actual regression evidence.

Every retained receipt records terminal exit, command, environment, start/end, freeze reference and zero before/after stability exits. Helper contracts include the real binary policy test without skips, package time 8.620 s and receipt elapsed 9 s. Root-fast records Make exit 2 and elapsed 75 s. Its 108-package ordinary root and tagged integration recipes finish with zero golangci issues and their errortype step, then automatic dispatch reaches the 55-package nested scope and fails. Nested errortype and later scopes remain unreached.

I independently parsed the retained raw root-fast log and compared all 419 path/line/column/message/linter/owner entries against task24's inventory; exact equality holds across 31 owners. The earlier `findings-comparison.json` is supported. I inspected/reused helper/root-fast evidence and did not rerun those gates.

Generator inspection supports omitting `make -C tools/gomad3 validate`. These files have no changed schema, template, generated layout, overlay, generator directive or listed implementation-identity input. Make VERSION_INPUTS, BOUNDARY_INPUTS and COMPATIBILITY_INPUTS exclude this bounded branch rewrite. Production generator/tool inputs remain unchanged.

Observed timestamp intervals bound command execution and tool round trips; tool wait limits are not actual durations. Retained zero-second values reflect whole-second clocks. Stock Go 1.27.1 on Linux aarch64 supplies developmental ordinary-package evidence and excludes `gomad3_toolchain` tests.

## Recommendations and limits

Root can commit this coherent source progress before assigning the next bounded implementation owner. Keep the 419 nested findings and task24's exact owner inventory, unchanged `^.git` reporting limitation that includes `.github`, task21, original R18/R19, workload/default proof, formal green-tree review and both native darwin/arm64 and linux/amd64 gates open. This review supplies no waiver, green baseline, formal SHIP, merge, task-completion or native-acceptance verdict.

## Assessment

Ready for SOURCE PROGRESS COMMIT: yes. The unchanged outer restriction makes the two inner branches behavior-equivalent, literal characterizations preserve admission and identity, and fresh affected-package checks pass on frozen source. Root owns fix decisions, lifecycle and commit.

Live command handles: none. Delegated agents: none. Reviewer writes are limited to this report and its checks receipt.
