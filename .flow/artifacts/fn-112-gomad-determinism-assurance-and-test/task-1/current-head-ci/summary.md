Current-revision smoke qualification
====================================

GitHub Actions [run 37027023410](https://github.com/stephanos/temporal/actions/runs/37027023410) tests commit `67dbe02666afd68c064c6c3cb9197b03d4664687`. Both native platform smoke jobs completed successfully. Each report confirms `qualified`, matching replay, and exact choice replay for activity, child-workflow, update, and user-timer workloads at seed 11. The Linux manifest permits intermittent results, but these four actual observations all qualified.

The adjacent summaries retain platform, commit, report SHA-256, and per-workload replay fields. Download the source reports from run artifacts `gomad3-functional-smoke-37027023410` (Linux) and `gomad3-functional-smoke-darwin-arm64-37027023410` (macOS); local report paths in the summaries identify this session's downloaded copies.

The overall run and core qualification jobs were still running when this evidence was recorded. This does not close D12, prove other seeds, or qualify subsequent implementation changes. `dispatch.json` is the launch-time status snapshot.
