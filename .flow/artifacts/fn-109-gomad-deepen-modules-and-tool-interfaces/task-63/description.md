Added by the 2026-10-09 amendment (see the spec's "Format-compatibility removal and storage handoff" section).

The Runner's local campaign orchestration is one function of about 780 lines. Remaining Runner tasks (.15, .26) and the fn-152 storage rewrite all change it. Decompose it first so those changes land on small, separately testable steps instead of colliding in one function.

Split it along its existing phases (request validation and preparation, campaign open/resume, the seed scheduling loop, per-execution assessment and publication, and completion/summary) into named functions or a small orchestration type. Keep scheduling, failure policy and counters in the campaign controller. Preserve existing comments with their code. Behavior is unchanged; formats may change under the amendment but this task has no reason to change them.

**Touches:** [tools/gomad3/runner/runner.go, tools/gomad3/runner/runner_local.go, tools/gomad3/runner/runner_local_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-63/**]

Root admits this source-only decomposition under [the isolated admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-63/admission.md). Original acceptance remains unchanged. Existing Runner/CLI and lint failures remain source-owned; verified reviewed progress does not complete the task. Native qualification remains deferred and unverified under fn149/fn128.
