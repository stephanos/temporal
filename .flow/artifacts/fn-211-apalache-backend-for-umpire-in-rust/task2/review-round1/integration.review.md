I’ll read the task spec first, then review the diff and its integration with unchanged callers, focusing verification on the named tests.
The main integration change is the shared process runner: TLC and Apalache now launch through a Python supervisor. I’m checking that it preserves existing command behavior and process cleanup, alongside the new bound and result parsing.
I found a containment gap: after the tool exits, the runner reopens its log paths for writing. If the tool replaced a log with a symlink, that write follows it even when the run has already detected unsafe output.

The quick Cargo test is blocked by the read-only build cache. I’m checking which focused tests can run without filesystem writes.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/backends/tlc/src/process.rs:209`
- **R-IDs**: [R2]
- **Problem**: The runner reopens tool-accessible log paths after execution: `fs::write(stdout, &output)?;`. A tool can replace `check.stdout` or `check.stderr` with a symlink, causing this write to truncate a file outside the build root. Replacing either with a FIFO can block indefinitely. These writes occur before `result?`, so detecting unsafe output or exceeding the timeout does not prevent the write or hang. Both Apalache and unchanged TLC callers use this path.
- **Suggestion**: Retain and write through the file handles opened before launch. Ensure subsequent diagnostic reads also reject symlinks and remain bounded. Add focused symlink and FIFO replacement controls.

Verification: four parser/bound tests passed using the retained test binary; the Python Darwin-policy test and a Linux supervisor cleanup probe passed. Fresh Cargo testing was blocked by the read-only build cache.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R2 | partial | Pinned admission, native-depth bound, explicit outcomes and shared supervision implemented; log handling still violates containment and bounded lifecycle requirements. |

Unaddressed R-IDs: [R2]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R2"]}
```

<verdict>NEEDS_WORK</verdict> - Fix unsafe log reopening before merging.