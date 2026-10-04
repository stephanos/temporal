# fn-110 task 5 source-size preparation

Literal R8's extracted-source reduction remains unsatisfied. The current final
`-U3` patch is 5,710 bytes and 105 lines larger than task 1's original `-U3`
baseline. Its canonical `-U1` representation is independently smaller than
that final `-U3`. Task 5 remains TODO, and this preparation supplies no native
or behavioral qualification.

## Fresh measurements

The run started at the host-observed `2026-10-04T04:52:47.977117+00:00` and
finished at `2026-10-04T04:52:49.168312+00:00`, exit 0, elapsed 1.191 seconds.
Host `uname` reports `Linux aarch64`, with Git 2.53.0 and GNU patch 2.8.
The observed clock date is retained literally even though the session's supplied
environment date is 2026-10-03. Requested routing was thinking scout
`gpt-6.1-sol` at high; actual model execution metadata was not host-observed.

| Patch | Bytes | Lines | Files | Hunks | Added / deleted |
| --- | ---: | ---: | ---: | ---: | --- |
| Original task 1 baseline `-U3` | 32,652 | 1,007 | 20 | 71 | 342 / 55 |
| Current final extracted source `-U3` | 38,362 | 1,112 | 20 | 81 | 318 / 103 |
| Current canonical `-U1` | 29,015 | 778 | 20 | 90 | 318 / 103 |

The first R8 comparison fails by +17.49% bytes. The representation comparison
saves 9,347 bytes (24.37%) and 334 lines. Canonical `-U1` is also 3,637 bytes
below original baseline `-U3`, but that combined comparison cannot substitute
for either independent requirement.

| Input | SHA-256 |
| --- | --- |
| Original baseline `-U3` | `950063a87d63cb01dada2e6ef232fdeb4acc52f71a008e2158885666669f80bc` |
| Fresh final `-U3` | `86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c` |
| Current canonical `-U1` | `8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b` |
| Pinned Go archive | `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1` |
| Current descriptor | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` |

Baseline commit `38957053f1ce342a8797af1803f5f8f6bb53fcad` reproduces the
retained task 1 patch and descriptor hashes and all baseline overlay counts.
The observed current HEAD was `0dd05b313acd0986312da7fd3159520e6a21f1bf`;
working-tree patch, descriptor and overlay bytes were measured directly.
Their contents and the archive remained stable throughout the run.
The original baseline build binding remains
`8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc`.
The current `.toolchain/build-key` file is absent. No final qualified build
identity is claimed, and no retained artifact was relabeled.

## Overlay inventory

| Source tree | Files | Bytes | Lines |
| --- | ---: | ---: | ---: |
| Original baseline overlay | 57 | 601,516 | 17,105 |
| Current integrated overlay | 79 | 764,039 | 20,763 |
| Delta | +22 | +162,523 | +3,658 |

Current `src/runtime/gomad.go` is 65,607 bytes and 1,867 lines, versus
46,630 bytes and 1,375 lines originally. The three task 3 additive overlays
carry 1,520 bytes and 55 lines altogether. Current growth also includes
integrated diagnostics, time-wire, process-command and handle work from other
specs. The full per-file inventories and changes are in
[source-size-evidence.json](source-size-evidence.json), preserving that cost
without attributing every change to fn-110.

The descriptor's patch allowlist equals the current 20 modified paths, and its
overlay allowlist equals all 79 overlay paths. These are source-set comparisons;
this run did not execute governed `make validate` or an overlay collision gate.

## Gap attribution and approved extraction limits

The following exact section deltas account for all 5,710 additional `-U3` bytes.
Every other section has unchanged byte count, including `sizeof_test.go`, whose
assertion content changed between the baseline and integrated source.

| Upstream section | Original bytes | Current bytes | Delta |
| --- | ---: | ---: | ---: |
| `src/runtime/runtime2.go` | 1,352 | 6,439 | +5,087 |
| `src/runtime/time.go` | 2,798 | 3,803 | +1,005 |
| `src/runtime/os_linux.go` | 0 | 570 | +570 |
| `src/runtime/proc.go` | 15,145 | 15,423 | +278 |
| `src/runtime/select.go` | 1,583 | 1,696 | +113 |
| `src/crypto/rand/rand.go` | 747 | 0 | -747 |
| `src/syscall/env_unix.go` | 656 | 468 | -188 |
| `src/syscall/syscall_unix.go` | 1,155 | 747 | -408 |
| Total | | | +5,710 |

The `runtime2.go` section carries the new timer ordinal, `gomadHostDrawScope`,
`gomadHostTimed` and `gomadHostBatch` fields. Inserting `gomadHostDrawScope`
lengthens alignment across the existing `m` field block, so the patch carries
both old and reformatted lines. Fn-112 task 5's retained
[working-tree.patch](../../fn-112-gomad-determinism-assurance-and-test/task-5/working-tree.patch)
contains that field and alignment change, and its
[review-fixes.md](../../fn-112-gomad-determinism-assurance-and-test/task-5/review-fixes.md)
records the host-batch field. This grounds stream-isolation ownership of those
introduced inputs. The exact table is not a claim that all increases share
one owner or that their behavior can safely be removed.

The three approved task 2 bodies are already wholly in
`overlay/src/runtime/gomad.go` at this source snapshot:

- `gomadResumeSyscall`, line 1363, owns the idle-P resumption implementation.
- `gomadCheckDeadTime`, line 1536, owns the entire quiescence and time-selection
  body, with the external/retry/deadlock/advance branches and lock transitions.
- `gomadSimulationTimeQuiescenceChanged`, line 1591, owns the entire function.

[Task 2's Approach](../../../tasks/fn-110-gomad-minimize-the-runtime-patch.2.md)
explicitly requires `proc.go` to keep transport accounting, `forEachG`
transport/runnable handling, the stock `pidleget`/`mget` timer-wake path and
both simulation guards, plus the quiescing P-selection guard and the
arrival/global-queue choice. Current materialized `checkdead` at line 6479
and `exitsyscallNoP` at line 5152 retain those required seams. The moved
quiescence function no longer exists in `proc.go`.

No remaining implementation inside those three already-approved extraction
bodies provides the required additional reduction of at least 5,711 bytes.
Moving the required scheduler machinery would exceed the stated extraction
contract. Task 2 retains the extraction-size acceptance issue, coordinated
with the owners of the introduced runtime inputs, especially fn-112 task 5.
Preserve the original comparator and report the gap pending that reconciliation.
This audit recommends no embedding, compiler/linker consolidation, lifecycle
redesign, comment deletion, behavior removal or source edits.

## Reproduction and evidence limits

Run from the repository root with a new empty scratch directory:

```sh
mktemp -d /tmp/fn110-source-size-XXXXXXXX
python3 .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-5/source-size-verify.py --repo /Users/stephan/Workspace/skunkworks/gomad/temporal --scratch /tmp/fn110-source-size-CNRyRbvw
```

Substitute the newly returned directory for the recorded scratch path. The
script refuses an occupied scratch directory. The retained run used
`/tmp/fn110-source-size-CNRyRbvw` and leaves its bounded source snapshots there.
No scratch or build directory was deleted.

[source-size-commands.json](source-size-commands.json) records each exact argv,
working directory, exit, timestamps, duration, input and output hashes, and
application output. `git diff --no-index` returns expected status 1 for a
nonempty diff; every application and read-only Git query returned status 0.
The script validates all 17,327 archive entries (149,875,169 expanded bytes)
and extracts the union of all 21 baseline/current patched files plus `VERSION`.
It uses separate `a` and `b` directories to retain canonical headers without
creating a Git repository or index.

The patch commands match `MaterializePatch`'s
`patch --dry-run --batch -V none -p1 -F 0` and
`patch --batch -V none -p1 -F 0`. Additional checks reject offset/fuzz messages,
reversed application, `.orig` and `.rej` files. Original baseline `-U3` and
current `-U1` independently regenerate byte-identically using
`git diff --no-index --no-ext-diff --no-textconv --binary --no-prefix
--abbrev=7 --diff-algorithm=myers --unified=N -- a b`.
Repeated final `-U3` outputs match, and fresh `-U3` materialization is identical
to `-U1` for every patched member. The full member hashes are retained.

This developmental GNU patch text comparison leaves both supported native
hosts' R4 equivalence gates open. No package loading, generator, test, toolchain
build, native execution, qualification, Flow state mutation or Git index/history
mutation ran. R7 and all outstanding task/spec acceptance remain open.

Written shared files are this report, `source-size-verify.py`,
`source-size-commands.json`, `source-size-evidence.json`, and five patch files:
`source-size-original-baseline-U3.patch`, `source-size-reproduced-baseline-U3.patch`,
`source-size-current-U1.patch`, `source-size-reproduced-current-U1.patch`, and
`source-size-final-U3.patch`. All are under this task 5 artifact directory.
