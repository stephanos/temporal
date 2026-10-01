# fn-105.13 (D13) implementation review

Reviewer: gpt-5.6-sol at high reasoning effort, through `codex exec -s read-only` on the working-tree delta (commits forbidden). The reviewed delta is `fn105-d13-reviewed-delta.patch`; the regenerated `tests.json` was reviewed in the repository.

## Round 1: NEEDS_WORK

- **Blocking — replay taxonomy excludes a valid CLI mode.** [README.md:249](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/README.md:249) says an untraced qualification claims nothing about replay. However, [`qualify.go:108`](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/cmd/gomad/internal/cli/qualify.go:108) permits `--replay-successes` without `--choices`, and [`workload.go:73`](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/qualification/workload/workload.go:73) retains and replays those successes. Such a run claims successful artifact replay, but not choice-tape replay. Describe these as independent dimensions and limit “nothing was replayed” to runs with both options disabled.

- **Blocking — traced manifests are overstated as exact-replay gates.** [README.md:394](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/README.md:394), [integration README.md:108](/Users/stephan/Workspace/temporal/gomad/tools/gomad3integration/README.md:108), and [GOMAD_MILESTONES.md:62](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:62) claim verified exact replay is held by `temporal.json`, `smoke.json`, and Chasm. Their `intermittent` expectations accept `replay_divergence` via [`execution.go:121`](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/qualification/set/execution.go:121), so the sets can pass without `choice_replay_exact`. Call them tracing-enabled replay/conformance gates; only an individual qualified seed with exact replay evidence verifies choice-tape replay.

- **Non-blocking — milestone status contradicts itself.** [GOMAD_MILESTONES.md:210](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:210) says D13 is implemented in the working tree, while [line 216](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:216) still says “Implementation has not started.”

VERDICT: NEEDS_WORK
Resolution: the guides now describe `--choices` and `--replay-successes` as independent flags and name Artifact replay without a choice tape; the traced manifests are called tracing-enabled replay gates, with choice-tape replay verified per seed by `choice_replay_exact`; the Status-paragraph edit was reverted, and the stale "Implementation has not started." sentence is reported to the conductor as outside this task's wording scope.

## Round 2: SHIP

No blocking or non-blocking findings. The prior review issues are resolved, generator defaults and overrides satisfy R13, and D12/D14 traced gates remain intact.

VERDICT: SHIP