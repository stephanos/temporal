---
satisfies: [R2, R5]
---
# fn-97-gomad-f3-qualify-the-frontend.3 Record F3 results including the linux status

## Description
Update F3 status; record linux as not re-measured if it cannot be run here.

## Acceptance
- F3 status updated

## Done summary
F3's Status in MILESTONES.md now carries a 2026-09-27 darwin/arm64 paragraph. It covers the PIE ASLR channel and the re-exec fix (cf1bc982c2, 85c71462b8), the static_addresses fixture, seeds 11 and 17 qualifying with replay_match and choice_replay_exact, and the per-seed transcript and choice-tape numbers. It also records the manifest `qualified`/guarded expectation, the 6/11/1/0 corpus result with user-timers still intermittent, and the CI assertion (403b1d36ec). Linux is recorded as not re-measured; the unrepeatable expectation stays. The work-tracking row now reads done on darwin/arm64 with linux/amd64 not re-measured. The README already states the ASLR re-exec contract (added in cf1bc982c2, including "linux/amd64 targets are not position independent"), so README and ARCHITECTURE.md are unchanged.

stage: impl-review - ran (codex requested; deterministic triage_skip SHIP, docs-only)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5e758fe5979e26a3068f4823ec2bd832e79492f1
- Tests: baseline: none (spec defines no Quick commands)
- PRs: