# Provisional candidate ownership and transfer plan

R6 remains incomplete. This source-only inventory records 81 original candidates, 1 new Temporal consumer branch, and 3 standalone refs: 85 repository/ref entries. All 81 original refs remain present and their recorded heads remain reachable. 36 original candidates require a framework/Temporal/shared/tracker split; 26 have no identified task. Zero original candidates are transferred; zero dispositions are complete.

[The JSON](provisional-transfer-plan.json) names current recovery refs/heads/worktrees, original ledger anchors, provisional per-path destinations, existing Flow IDs/status/dependencies, evidence links and explicit gaps for every candidate. Unknown owners stay unresolved; net deltas include inherited assemblies and never establish exclusive ownership or completion.

Standalone `work/fn2573-standalone` at `f7b42ce25558344ea2d38f1e05d002259427a8f6` and quick preparation `work/fn2577-quick` at `045b98897de7ca92a21c9e0d63c1190af765986f` are migration preparation refs. `umpire` at `cf8bec4d6231302df54b57fccdc5bdb9d11552fc` retains the export baseline. These refs do not constitute transfer of the original candidates. Consumer `work/fn2574-consumers` stays in Temporal at `91547f57ec192d576a13056ce570c8bf47e441bf`. Existing migration Flow records remain in Temporal; standalone receiving tracker records are not created.

Remaining actions:

- Resolve unknown task ownership, shared/experimental paths and historical mappings; select actual hunks and deduplicate inherited assemblies.
- Back up and verify exact dirty/untracked/unique ignored source and evidence before any retirement. Preserve the unresolved assembly index, fuzz source and frozen baseline control. Observing 37 Temporal and 2 standalone worktrees is not backup proof.
- Create and verify actual receiving candidate refs, transfer provenance and destination obligation records/links; retain existing dependency chains and deferred/live/performance obligations.
- After the validation hold ends, prove the exact standalone/consumer pair. Establish destination state and permissions, then complete separately authorized non-destructive publication and source removal, reference updates and closure.

No builds, tests, gates, replays, Git/task/remote mutations or source moves/removals were performed. Only this plan and its JSON were written.
