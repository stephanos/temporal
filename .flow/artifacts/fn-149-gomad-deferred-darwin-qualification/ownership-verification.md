# Native ownership amendment verification

The owner approved Darwin deferral and kept Linux qualification/CI deferred, with no PR, push or CI run. [fn-149](../../specs/fn-149-gomad-deferred-darwin-qualification.md) records four blocked Darwin tasks. fn-128 retains all seven blocked Linux tasks. Both specs remain not ready and have zero ready tasks. The [transfer manifest](../native-scope-transfer-2026-10-07.md) owns the mapping; this receipt verifies the administrative change only.

## Checks

- Original bodies of seven amended specs and 69 mapped open tasks were compared byte-for-byte after removing the exact inserted authority notes. All 76 matched their pre-amendment bodies, including the already authorized narrow fn-110 current 642-byte U3 waiver. Original criteria, Done summaries and histories are preserved.
- All 76 existing JSON records changed only their updated_at timestamps. Existing source dependencies and lifecycle snapshots are unchanged; no source task was completed by the transfer.
- `flowctl validate --all --json` returned exit 0, 22 valid specs, 198 tasks and zero errors. Two warnings concern uncovered requirements in untouched fn-104 and fn-107. fn-149 coverage validation returned exit 0 with four tasks, no errors and no warnings.
- Document links in the new owner/task records, transfer manifest, AGENTS, MILESTONES and Gomad README resolve. MILESTONES status rows match live Flow state.
- `git diff --check` returned exit 0. No Go source, module dependencies, toolchain definitions, qualification manifests or workflow files changed. Documentation-only verification follows MILESTONES; no tests, lint or native commands were rerun as green evidence.

## Review

[Round 1](plan-review-1.md) returned SHIP and a P2 write-scope finding. The corrected task 2 declares canonical pack authoring outputs; task 3 declares actual downstream pack/guidance surfaces and requires binding real checkout paths before writing. [Fresh round 2](plan-review-2.md) reports the prior finding fixed and returns SHIP with no new blocker.

The project requested gpt-6.1-sol/high for both fresh host reviews, from the same GPT family as the writer. The host exposes no actual-model metadata or tool-enforced read-only dispatch flag; receipts retain those limits rather than claiming verified execution metadata. Reviewers reported no mutations or qualification runs.

## Remaining acceptance

The full source lint gate's retained result is still 265 findings at source d28d67c40ce74dd8886cf11b36fe7d2ddaf23675. The ownership change fixes none of them. Source implementation, ordinary host coverage, preservation, first-baseline and non-native measurements, both-source-set static checks, generated validation, source reviews and actual-checkout prerequisites remain with the donor tasks wherever unproved. Native aggregate test-host execution and required scheduled/dispatched soak proof remain deferred under their native owners; partial portable runs establish neither a full native pass nor a measured bound.

No PR, push, workflow dispatch or external tracker write was performed. Revival requires the owner's explicit request and the corresponding supported native execution; it grants no publication or CI authority.
