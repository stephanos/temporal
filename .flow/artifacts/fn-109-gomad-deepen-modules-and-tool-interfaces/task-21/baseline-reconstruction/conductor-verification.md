# Independent baseline verification

On 2026-10-04 the conductor independently checked the reconstructed fn-109
first-task nested-module baseline. All 670 paths, complete Git blob identities
and executable modes also match the subsequently committed nested-module tree
at `38957053f1ce342a8797af1803f5f8f6bb53fcad`. There are no missing paths, extra
paths, byte mismatches or executable-mode mismatches. This supplies full-file
Git identity corroboration for all seven recovered additions, not just their
historical diff prefixes or physical line counts.

The baseline remains the original `6782b55f49a0317b230e827ea2a63a37d116d502`
plus the dirty fn-108 changes described in [reconstruction.md](reconstruction.md).
The later commit is an independently verified equivalent **nested-module**
tree, not a replacement baseline or proof of equivalence outside `tools/gomad3`.
The historical abbreviated gate fingerprint is still not reproduced. Task 21
remains unclaimed; no campaign measurements, acceptance or native qualification
follow from this input verification.

## Fresh manifest checks

From the repository root, `sha256sum --check --quiet` on `input.sha256` exited 0
with no output. From `/tmp/fn109-baseline-reconstruction.lDSSw8Gx`, the same
command on the artifact's absolute `source.sha256` path exited 0 with no output.
Tool-reported durations were 0.00000725 and 0.000007 seconds respectively; these
are reported tool timings, not independently measured checksum timings.

The fresh manifest/script hashes were:

| Artifact | SHA-256 |
| --- | --- |
| source.sha256 | d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845 |
| input.sha256 | 5b1904a15a5e11dd93926465d97c5467ab2a008f48b20610d9b0ffcfa4e8d0e6 |
| reconstruct.py | f19edebc9708e3d6716682e58d91b09aa9cc30dcd9d7d6d666f7cfbba6e2796c |

The conductor read the complete reconstruction script, including its exact
old-position/new-position and hunk-content checks, input blob-prefix checks,
inventory reconciliation and exclusion of overlapping intermediate hunks.

After these successful checks, the conductor added the reconstruction pointer
to task 21 through Flow. A subsequent full input-manifest check therefore reports
exactly one mismatch: `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md`
changed from historical hash
`4aba7e49a41f5567dceed8c81c038351af45ddb258cc577e753c341eaa30176b`
to `ab5b964fde268884338678cd88da78d9acc6cd78e8e22027c837cba6719aab72`.
All other recorded inputs still verify, and the source manifest is unaffected.
Preserve the historical manifest unchanged; record this explicitly identified
task-description metadata drift when checking inputs for future measurements.
It does not authorize drift in any reconstruction source input.

## Independent committed-tree comparison

A read-only Python command used `git ls-tree -rz
38957053f1ce342a8797af1803f5f8f6bb53fcad -- tools/gomad3`, independently enumerated
the scratch files and computed each Git blob as
`SHA1("blob " + decimal_byte_length + NUL + complete_file_bytes)`. It required
regular blob modes `100644`/`100755`, compared both complete path sets and every
blob identity, and checked executable permission presence against each Git mode.
An assertion required every mismatch list to be empty. No archive extraction,
index operation, checkout, source write or Git mutation occurred in this check.

The command exited 0. Its own measured interval was
2026-10-04T04:48:36.153506+00:00 through
2026-10-04T04:48:36.183563+00:00, elapsed 0.029993625001225155 seconds. Complete
comparison output:

```json
{
  "comparison_revision": "38957053f1ce342a8797af1803f5f8f6bb53fcad",
  "reconstructed_files": 670,
  "commit_files": 670,
  "missing": [],
  "extra": [],
  "byte_mismatches": [],
  "executable_mode_mismatches": []
}
```

No Go loading, tests, builds, generation, profiles or bridges ran. The conductor
wrote only this verification record and the task-description pointer. Commits
remain user-owned (`commits: []`).
