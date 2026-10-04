# Task 24 admission

Task 24 is a bounded R19 lint-configuration repair after source-progress commit
`ce80d2425cf34da103939b5aa23f90bde1c2092f`. Task 23 remains blocked on qualification;
task 21 gains task 24 as a dependency and retains task 23. No old task is marked
done or reset to disguise open acceptance.

The [source-bound task 23 observation](../task-23/lint-policy-path-observation.md)
traced pinned v2.13.0 matching to config-relative paths, but its doubled-escape
hypothesis is refuted by task 24's pre-edit parsed-regex controls and root byte
inspection. Current config SHA-256 is
`86d71dda338f89c748a7ecae99e989d03b71b8693280adddae3970eba04c930a`.
The worker must retain behavioral RED before repair and independently expected
positive/negative path cases. Preserve the working regexes unchanged and retain
the corrected diagnosis separately from historical reports. Root and nested
real-tool fixtures must also
prove that ordinary disallowed findings still report. Enabled rules, text and
forbid patterns, pins, comparison and fix policy stay unchanged.

Source admission is separate from task acceptance; task 24 has no artificial
dependency on the blocked acceptance it repairs. Root owns Flow, commits and
review. One checkout writer may delegate independent read-only scouts. No
concurrent writer, worktree, push, history rewrite or evidence normalization.
The configured review backend is codex; formal dispatch is deferred whenever
qualification stays red, with a fresh independent source-progress review rather
than a substituted SHIP. Both required native gates and original R18/R19 remain
open. Verify and commit this task's progress before another implementation task.
