# Current pre-cutover archive and Scala-cache baseline

This new baseline records actual main-Temporal archive/cache bytes before source cutover. It is not a reconstruction of historical ignored-cache state and does not claim a post-cutover comparison. Earlier fn-257/source-history preservation explicitly excluded ignored caches.

The frozen and current Git archive tree both equal `cb5223b9a4839233876c98cf6158a46dc9d97a19`. All 1,627 tracked archive files match their Git blob bytes and executable/symlink modes. The actual bounded disk inventory includes 2,087 entries and 1,713 regular files totaling 34,007,555 logical bytes, with full POSIX modes retained; no actual symlinks were observed. A second pre-cutover capture found zero additions, removals, byte/mode/link changes or mtime changes.

All 86 ignored on-disk files were hashed. Forty are historical source-like sample `.test` files and must not be mistaken for disposable binaries merely because Git ignores that extension. The other 46 recognized cache/IDE/semantic-database files total 1,384,700 bytes. Coverage includes every hidden/ignored entry under `model/archived`, nested `.bsp` directories, sixteen `.semanticdb` files, and repository-root `.scala-build`/`.bsp` state.

The manifest explicitly records eighteen absent locations, including model-root `.scala-build`/`.bsp`/`.bloop`/`.metals`, archive-root Scala IDE caches/build/gen directories, `model/gen` and the named former `model/build` Scala jars/sample/history outputs. Root `.bloop`/`.metals` are also absent. Existing `model/build` TLC/Apalache directories belong to current Rust checker execution and were not scanned as Scala caches. External Scala CLI/global Coursier/toolchain locations referenced by IDE metadata were not followed. No worktree, global/shared cache or active release target was scanned.

Capture/compare is reproducible from the Temporal root:

```sh
python3 -B .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/archive-cache-identity/archive-cache-baseline.py capture --output <new-manifest.json>
python3 -B .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/archive-cache-identity/archive-cache-baseline.py compare --baseline .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/archive-cache-identity/pre-cutover.json --output <comparison.json> --phase <actual-phase>
```

Use `post-cutover` only after cutover actually occurs. The existing [comparison](pre-cutover-stability.json) is explicitly `pre-cutover-stability-check`; no post claim exists. Six in-memory controls confirm refusal of changed file bytes, permission modes, symlink targets, missing files appearance of a formerly absent cache, and disappearance of the entire archive. No real source/cache mutation was used for those controls.

Modes, content hashes and raw link-target bytes form identity; observed mtime-only changes are reported separately. Atime/ctime, ACLs and xattrs are not covered. Per-file stat checks detect concurrent changes during hashing, but this is a bounded snapshot rather than a filesystem transaction. No historical cache equality or unobserved external cache preservation is implied. Nothing was cleaned, regenerated, permission-changed, committed or gated.
