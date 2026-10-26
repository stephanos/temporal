---
satisfies: [R3, R4]
---
# fn-257-extract-the-standalone-umpire.7 Preserve quick consumer selection across sibling Git repositories

## Description
Preserve the quick command's affected-consumer behavior after its workspaces move to different Git repositories.

**Size:** M
**Files:** exported dev/xtask/src/quick/{mod,selection,tests}.rs and native command documentation; the retained Temporal quick caller if an explicit caller-root argument is required.
**Touches:** [../umpire/dev/xtask/src/quick/**, ../umpire/dev/xtask/README.md, model/dev/xtask/src/quick/mod.rs]

### Approach
- Replace the current identical-Git-root refusal with change collection once per distinct repository; merge absolute changed paths before the existing cross-workspace dependency graph selects consumers. Keep full native/interpreter/borrowed-source coverage, untracked/deleted/renamed handling and conservative global-input fallback.
- Preserve single-repository BASE behavior. For multiple repositories, apply explicit BASE to the caller's repository and use HEAD for the others, reporting each effective base. Use existing admitted caller context and fail clearly if ownership is ambiguous; do not silently reinterpret an invalid requested base or infer a shared commit identity across unrelated histories.
- Preserve the existing selection JSON contract where possible, target/profile/cache policy, full-gate cache refusal and complete consumer selection. No nextest/source-assembly preparation is imported as part of this fix.
- Add isolated two-repository controls, reverse-consumer and invalid-base cases in the existing quick test module. Actual Rust execution remains deferred under the owner hold; source preparation alone does not complete this task.

### Investigation targets
**Required:** exported dev/xtask/src/quick/mod.rs (current same-repository guard); exported dev/xtask/src/quick/selection.rs; exported dev/guest-driver/src/context.rs (admitted caller context); Temporal model/dev/xtask/src/quick/mod.rs; task4's provisional caller wiring.

### Quick commands
When validation resumes, run the focused quick unit/bridge controls and an actual reversible content edit in each sibling checkout. Assert every direct/reverse consumer remains selected. Shared full gates remain task5's responsibility.

## Acceptance
- [ ] Correct independent sibling Git roots are accepted and changed inputs from either repository select all direct and reverse consumers.
- [ ] Single-repository BASE behavior stays intact; the multi-repository effective bases are explicit and invalid caller BASE fails.
- [ ] Two-repository source/selection controls and the actual sibling command pass after validation resumes; no consumer is dropped to achieve acceptance.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
