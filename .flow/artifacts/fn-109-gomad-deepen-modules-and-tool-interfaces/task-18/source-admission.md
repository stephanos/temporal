# Filesystem-owner source admission

Task 18 claimed at 2026-10-04T02:53:01.265122Z under MILESTONES item 4,
after task 17's integrated frozen source review and conductor checks. Task 17
remains blocked on native acceptance, not done. This admission permits source
implementation only; it waives no R12, process, toolchain or platform gate.

Scheduling remains sequential: one writer owns overlay/descriptor/generated
surfaces in the shared checkout. Read-only scouts/reviews may run in parallel;
the conductor owns Flow state, review, verification and acceptance. User owns
commits, so no staging, commits, stashes, worktrees or publication.

The selected design is design-decision.md; source-scout.md records concrete
ownership, error precedence, mapping/capacity distinctions and tests. Use
task 17's final reviewed fixture shape, including actual Runner process cases
and canonical Makefile selection regression. Standalone/in-process share the
local filesystem representation; process owns remote IDs/cache. Three test
modes do not require three model implementations.

Judge was called once for this admission: Tier: session
(jev-unavailable(no_key)), implementer gpt-6.1-sol, spawn_model null.
Explicit Codex routing remains gpt-6.1-sol/high for implementation; judge
unavailability does not reroute the worker. Actual execution metadata is not
claimed. Task-aware review backend is codex, deferred until native gates are
available; separate fresh source audits do not establish formal SHIP.

Existing task 18 is the implementation plan. Design preparation used the
brainstorming/writing-for-agents skills under the user's standing autonomous
instruction. TDD requires meaningful old production ownership RED before
migration and independently derived real-operation preservation cases; compiler
errors, source-text checks and unavailable native tests are not substitutes.

Actual host is linux/arm64, patched executable absent, existing linter
incompatible. Canonical Quick baselines and final checks must retain these
limitations. Developmental scratch evidence must disclose exact provenance
and cannot qualify supported platforms. Tracker bridge was observed inactive.
