# Prepared framework recovery snapshots

Result: **PASS, preparation only**. There are 48 verified proposed receiving targets and 50 new agent custody commits, all in isolated object stores. No object/ref import, HEAD/index/worktree change, feature acceptance or activation occurred in the real standalone repository. Canonical `05473210b48c3db862f22051eb7154ce98258f98` remains exact.

## Review entry points

- [Target summary](target-summary.json): exact48 source refs/OIDs, target OIDs, selected variant and file counts.
- [Complete manifests](prepared-targets.json): full per-path source/destination, modes, Git blobs, byte SHA-256/sizes and CAS paths; explicit deletion beforeimages and after-absence.
- [Independent verification](verification.json): raw Git-tree projection, complete object/byte comparisons, graph/author/parent checks,16 refusal controls and source/canonical preservation.
- [Proposed create-only transaction](proposed-create-only-transaction.txt): never executed against real standalone. [Replay control](transaction-replay-refusal.json) proves collision refusal without ref changes in the isolated test store.
- [049 ownership correction](candidate-049-ownership-disposition.json) and [full source evidence](candidate-049-consumer-prefix.json).

## What is preserved

The verifier checks 57,286 complete variant file records, 1,942 net projected deletions and 1,931 mapped source deletions. These are complete revision-qualified source projections, not copies of only the717 changed paths. Root LICENSE is retained. All modes/blob bytes, including symlinks, are exact. Historical source layouts are preserved without rebasing files into current crate roles; recovery snapshots are not claimed buildable or portable.

Normal selection reuses the approved task1 projector and is independently recomputed through raw Git tree objects with the same ancestry boundaries. Pre-archive Scala is never admitted. Mixed monorepo `.history-support` remains in source/canonical history rather than new receiver trees; no tracker/system/Scala source is imported.

The original projector refuses049 because a pre-cut legacy tool prefix collides with the new generic tool. Root independently approved one exact ownership correction at `86f812692797a0e3451133f0d59f382818584312`: the four pinned files under `model/dev/xtask` are the Temporal consumer stub, not generic tooling. They remain under original `refs/heads/work/fn25413-driver` and fn258.5 keeper/tracker disposition. The receiver has1163 generic files;1163 receiver +4 source-only records account for all1167 active qualified source records. All28 qualified mixed support blobs are additionally proven reachable in existing canonical export history. Global8983 ancestry rules and immutable R1/provisional maps are unchanged. OID/blob/extra-prefix exception drift refuses.

039 and075 each retain their complete committed projection plus a separately identified raw-dirty variant. The proposed48-ref targets select these dirty variants. Each dirty commit has exactly its new committed-custody snapshot as parent; this is an explicitly new custody sequence, not original branch history. Candidate039 contains unchanged unresolved working bytes and links all three original index stages in the verified custody archive. Candidate075 includes all40 untracked files plus five modified files. Neither snapshot resolves a conflict or grants build/feature credit.

All commits identify `Codex Recovery Custodian <codex-custody@localhost>` with a recorded new timestamp and custody-only message. Original authors, parents, dates and full source history remain under retained original Temporal refs/bundles; they are not falsely reconstructed in this snapshot graph. The recovered target graph contains exactly50 new custody commits and no imported monorepo history.

## Bundle and recovery

Prepared bundle: `/Users/stephan/Workspace/skunkworks/umpire/.recovery/fn258-receiving-snapshots/prepared-snapshots.bundle`

SHA-256: `efb9a7007b502c7d83f4b458fd5baa3e017ba1b10d9ca261a97557a8bf4d72cd`; size 1,756,635 bytes.

The bundle contains48 neutral `refs/prepared/fn258/candidate-NNN` refs. An independent empty bare repository unbundled it and verified every tree/blob/commit. The only executed48-line proposed receiving transaction was in a separate isolated test bare repository; replay fails because the refs already exist. All real `refs/recovery/fn258/candidate-NNN` names remain absent. Root admission and task4 acceptance precede any real import/ref transaction.

All current beforeimage payload blobs also occur in the bundle through another complete target snapshot. Keep the full manifests for beforeimage path/mode/absence semantics, the original source refs/bundles for authored history, and prior dirty/index custody archives. [Recovery primitives](recovery-primitives.json) identifies stores and required data; shared build caches were not touched.

## Reproduction and historical inputs

From the Temporal repository root:

```sh
python3 -B .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/verify-snapshots.py
```

The verifier is rerunnable, performs no real standalone writes, checks16 in-memory refusal controls and confirms the isolated create-only transaction refuses replay. The builder requires a new isolated store and refuses to overwrite prepared targets. All Python calls use `-B`; no generated `__pycache__` outputs exist in this lane.

Historical overlay, receiving-plan and custody manifest input bytes are frozen locally as `*.input.json`, with actual frozen-file path/hash locators and separately recorded historical_original_path provenance. Input-schema locator correction changed only evidence metadata; all50 tree/commit OIDs remain unchanged, as recorded in input-provenance-correction.json. Later current-only365→3f5 overlay/consumer observations do not silently rebind these48 original candidate identities or invalidate their historical inputs. Immutable R1, original plans and global projector were not modified. Current task4/pair validation and overall R6 remain separate open requirements.
