# Released standalone child target cleanup

Removed only `/home/agent/.cache/codex-build/tmp/umpire-standalone-Bnt8GY/target` after task2 closure `c258547ed5` and explicit conductor release. The target occupied 6,662,246,400 allocated bytes; its path is now absent.

Independent recovery verification passed before removal and again afterward: all 158 objects and 7,979 archive members remain valid without accessing original caches. The 23,524-entry target inventory was sealed before cleanup. Live cwd, path arguments, selected environment, open descriptors and descriptor-lock checks found no owner; system PIDs 10, 542 and 557 were unreadable and remain reported.

Cargo refused the orchestration root because its real `CACHEDIR.TAG` is absent. Five genuinely tagged child targets were cleaned successfully with exact `cargo clean --target-dir` commands. No tag was fabricated. The remaining unchanged subset of the sealed task-created reproducible output inventory was then removed using a literal path, device/inode/realpath guards, renewed ownership checks and protected-source checks.

The child root, source and receipts are exactly preserved. The parent target metadata inventory and every protected named payload hash are unchanged. The umbrella recovery store remains valid, and no source, worktree, Git ref, Flow state, shared/global/Scala cache or parent target was removed. Root HEAD remains at the reviewed closure commit.

[Cleanup receipt](cleanup-receipt.json) contains before/after bytes and both filesystem measurements; raw Cargo outputs, original and remaining inventories, protected seals and immediate recovery verification logs are adjacent. Private free space increased from 50,035,486,720 to 55,985,885,184 bytes; the filesystem delta includes concurrent unrelated activity and is distinct from the removed target allocation.
