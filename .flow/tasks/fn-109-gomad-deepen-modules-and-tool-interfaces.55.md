---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.55 Check verify-only replay report delivery

## Description
Own only the unchecked verify-only replay stdout success write in runReplayWith: fmt.Fprintf(stdout, "gomad: verified %s\n", result.Artifact.Path). The preceding correction is independently reviewed and committed as a2b020a178bd8cae92016d2c8cd4cdca81023d23; task54's actual original-base RED96, affected RED2 and independent review are admission inputs. Root owns scope/lifecycle/review/commit; fresh worker owns implementation/tests/evidence. Task21 consumes this correction. Do not require the still-open CLI predecessor's Done status where its affected lint consumes this correction.

**Touches:** [tools/gomad3/cmd/gomad/internal/cli/cli.go, tools/gomad3/cmd/gomad/internal/cli/replay_output_test.go, tools/gomad3/cmd/gomad/internal/cli/characterization_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-55/**]

### Explicit correction and preservation

Correct the ignored success-report failure to return3, the CLI's existing output-publication failure classification; retain0 after a successful write. This single failure-status change is explicitly admitted. Preserve one attempt, literal bytes/path operand, empty stderr/no fallback, completed verification, request/callback timing, VerifyOnly/artifact/toolchain/observed-directory/supervisor fields and ordinary replay reports. Earlier installation failures3, preflight failures2 and other replay failures3 still make no success-output attempt. Add no retry/helper/seam/framework/library/flag/policy/pin/native guard. Preserve all old tests and comments except the explicitly admitted fixture reconciliation below.

### Writer-failure fixture reconciliation - 2026-10-09

The first frozen full ordinary command exposed the directly caused stale fixture in TestCharacterizeOutputWriterFailures/replay_verification. Its expected status0 and immediately owning comment "Verification-only replay does not check its output write." describe the corrected ignored-write bug. Root explicitly admits only that table datum changing from0 to3 and removal of that obsolete owning comment. Keep the fixture's invocation/arguments/failed writer and every other case/assertion/comment unchanged. Retain the original mismatch receipt as meaningful failure evidence, distinct from inherited ordinary failures. Reconstruct the complete original characterization file after reversing only this datum/comment edit. This is no permission to weaken an assertion, alter a gate or relabel prior failures.

Bind healthy before/after and meaningful genuine read-only-file EBADF RED/GREEN controls. The existing newFakeInstallation replayDependencies and TestCharacterizeReplayRequestOutputAndStatus give legitimate ordinary Linux/arm64 CLI-unit coverage through runReplayWith, not full exported-Run real artifact verification. Attempt a legitimate public-path stock-source closure artifact fixture if possible without changing production: actual stockGo test executable/build information/current profile/actual host, no World/choice/simulation state, real pinned stock tool identity and --verify-only returning before execution. Do not use arbitrary executable bytes, repository .toolchain assumptions, host spoofing or native-success claims. Retain exact limitations if the public fixture is unavailable; source-only verification is not patched-runtime replay/native qualification.

### Verification

Read project guides/current task/spec/CLI contract first. Use apply_patch, pinned stockGo1.27.1 and -tags test_dep -count=1, established local cache/file-proxy recipe and fresh /tmp directory. Serialize gates on frozen source. Focused before/after tests, full ordinary affected CLI tests with honest coverage exclusions, vet/standalone errortype, required architecture/public/purity/both supported static source sets, fresh check-only validate, format, actual unfiltered affected configured lint and actual FIX=false make lint-code-fast/admitted base plus make --trace lint-code-gomad3/original951c5516e9e7b3066e7e069adda9565cfd68844c are required. Measure exactly1removed/0added; bind commands/exits/elapsed/raw/source/tools. Reuse valid exact-source unchanged receipts, keep handover small, do not retry unchanged failures merely to reconfirm.

Measure affected RED2 to RED1 and original-base RED96 to RED95, with exactly one removed and none introduced. Formal acceptance stays open while required affected/integrated source gates are red; independent source-progress review may license only a separate progress commit. Original R18/R19/first-baseline/preservation and deferred fn149/fn128 obligations remain in force. No native revival, PR, push or CI authority.

### Retained source progress - 2026-10-09

Root retained fresh independent acceptance of the bounded progress in
task-55/independent-review.md. Current candidate evidence is reconciled-evidence.json
and reconciled-source-proof.json; initial evidence/proof and all mismatch/probe
receipts remain immutable. The final gate batch has 33 focused passes and 432
ordinary passes with three inherited failures. Actual lint falls from 2 to 1
and 96 to 95, removing one finding and introducing none. Required red source
gates and public/native coverage gaps keep acceptance in_progress.

## Acceptance

- [ ] The single verify-only success write checks its result and returns3 on failure/0 on success; exact bytes/attempts, completed verification, earlier statuses/no-output cases, request fields and ordinary reports remain unchanged. Only the stale replay-verification writer-failure datum changes from0 to3 and its immediately owning obsolete comment is removed; all other old tests/comments remain unchanged.
- [ ] Healthy before/after and genuine EBADF meaningful RED/GREEN controls, earlier-error/request controls and accurately labeled CLI-unit/public-path reachability evidence exist without new seams or native spoofing.
- [ ] Frozen-source required tests/vet/errortype/boundaries/both supported static sets/validate/format and actual unfiltered configured/fast/original-base lint evidence show exactly1removed/0added; required gate gaps remain open.
- [ ] Fresh independent source-progress review, root verification and separate commit feed task21. No Done/formal SHIP on red source gates or broader/native/publication acceptance is claimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
