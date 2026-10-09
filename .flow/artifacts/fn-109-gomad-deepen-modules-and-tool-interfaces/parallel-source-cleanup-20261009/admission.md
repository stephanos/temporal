# Parallel source-cleanup admission

Root admits three disjoint source corrections to unblock fn-112.10's required integrated lint. The builder source-progress commit is 87733ae65f; committed milestone/planning base is f830467fcb712a83ba504c7dd43e2beb7f2cf223. Task56's corrected packet measures RED95 to RED80, with 15 findings removed and zero introduced. Fn-112.10 remains the close-out target, with all retained source acceptance and native fn149/fn128 deferrals unchanged.

Root selected these bounded task-ID corrections from the available source owners. All three depend only on Done task52, whose dependency closure is empty. Product Touches are disjoint (12, 4, 2 paths), with no lockfile/generated/migration/Flow product surface. Task57 has 11 existing files plus one new test file and 16 cleanup results. Task58 has two production files plus two new tests. Task59 changes only two existing no-op test switches. Handovers are task-unique lifecycle outputs. Root holds other owners outside this critical-path correction wave.

Scheduling: wave (bounded task-ID corrections)
Selected wave: [fn-109.57, fn-109.58, fn-109.59]
Selection rule: independent dependencies, disjoint product scopes, current fn-112.10 lint blockers
Isolation: linked worktrees
Dispatch count: 3

| Task | Worker | Workspace | Branch |
| --- | --- | --- | --- |
| fn-109.57 | /root/fn10957_test_cleanup | /tmp/gomad-fn109-parallel.pmgezCtg/task-57 | gomad-fn10957-source-cleanup-20261009 |
| fn-109.58 | /root/fn10958_production_cleanup | /tmp/gomad-fn109-parallel.pmgezCtg/task-58 | gomad-fn10958-source-cleanup-20261009 |
| fn-109.59 | /root/fn10959_exhaustive_cases | /tmp/gomad-fn109-parallel.pmgezCtg/task-59 | gomad-fn10959-source-cleanup-20261009 |

Each workspace has task-unique handover.md/evidence.json under .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-N/. Workers use committed authoritative tasks, explicit workspace assertions and the pinned source-test environment. Root owns selection, shared lifecycle, integration, independent review and target commits. No green baseline handoff is asserted. Judge returned no_key; requested implementer routing is gpt-6.1-sol/high, with actual-model telemetry unobserved. Reviewer routing uses the same GPT family.

Editing may run concurrently in isolated worktrees. Shared Go/build/test/vet/lint/generator/format commands require root's explicit serial gate grant and frozen checked source. Initial grant belongs to task59 for focused baseline only. Tasks57/58 prepare additive controls and request baseline grants before production/test cleanup edits. Root grants subsequent slots after terminal receipt/handle verification, never on silence or narration alone. Workers retain immutable source/tool/raw/command/exit/elapsed bindings, exact source scopes and all required red gates. Root verifies the combined target before any completion claim. Formal implementation review/Done remains unavailable while required source gates are red; bounded independent source-progress reviews may support separate progress commits.

Task59 diagnosed inherited BASH_ENV=/etc/sandbox-persistent.sh changing nested Bash cwd to primary via SANDBOX_START_DIR. Its verified isolated anchor uses env -u BASH_ENV after explicit cd/assert. Every worker removes BASH_ENV from gate descendants, binds ROOT/cwd to its own workspace and records that adjustment. No persistent environment/config change is admitted.

Parallel-wave workers have no shared lifecycle/review/integration authority. Root's project commit-after-review contract holds local checkpoints until root grants them. No PR, push, CI, native-host spoofing or native revival follows from this admission. Both protected user Turbo files remain unchanged and unstaged. Tracker bridge is inactive and plan-sync is false.

Task59's focused baseline is terminal exit 1, with one pass and 88 failures across eight top-level tests (one passes and seven fail), zero skips, and unchanged checked source. Genuine Linux/arm64 adapter rejection prevents campaign controls from reaching their assertions. Root verified raw receipt hashes/counts and amended task59's erroneous ordinary-reachability claim. Root admits only the same two no-op cases with every failed source requirement retained. No new native waiver or guard/fixture rewrite follows.

stage: parallel-dispatch - ran (three isolated workers; shared gates serialized)
stage: tracker-sync - skipped(config: bridge inactive)
