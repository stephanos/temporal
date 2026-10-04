# Blank-line grouping source experiment

Four blank lines plus pinned `gofmt` reduce the scratch candidate's `-U3`
patch by 3,766 bytes. The candidate still exceeds the original task1 `-U3`
baseline by 1,944 bytes, so this bounded hypothesis does not satisfy literal
R8. A further 1,945-byte reduction would be needed to make it strictly smaller.
The candidate remains in scratch and is not adopted or qualified.

| Representation | Bytes | Physical lines | Files | Hunks | Added / deleted |
| --- | ---: | ---: | ---: | ---: | --- |
| Original task1 baseline `-U3` | 32,652 | 1,007 | 20 | 71 | 342 / 55 |
| Current integrated `-U3` | 38,362 | 1,112 | 20 | 81 | 318 / 103 |
| Grouping candidate `-U3` | 34,596 | 1,056 | 20 | 81 | 292 / 73 |
| Current canonical `-U1` | 29,015 | 778 | 20 | 90 | 318 / 103 |
| Grouping candidate `-U1` | 25,330 | 722 | 20 | 90 | 292 / 73 |

Candidate `-U1` saves 3,685 bytes against current `-U1` and 9,266 bytes against
candidate `-U3`. The independent extraction comparison still fails. These
counts retain the original comparator without attributing integrated diagnostic
growth wholly to fn-110.

## Exact change and preservation evidence

The experiment used a new scratch directory,
`/tmp/fn110-alignment-1TWNKL9O`. It copied both the pristine and current source
trees only after verifying every retained member identity. The previous scratch
directory remained unchanged.

The only manual source edit inserted two blank lines around the five existing
Gomad fields in `g`, and two around the existing `m.gomadHostDrawScope` field.
The pinned Go 1.27.1 `gofmt` then aligned each resulting group. It left the
already separated `gomadHostTimed` and `gomadHostBatch` fields and their comments
unchanged. Every field retains its name, type and order.

The exact current-to-candidate change is retained in
[formatting-only.patch](formatting-only.patch). Only
`src/runtime/runtime2.go` differs between the complete current and candidate
source inventories. Its source changes from 57,829 bytes / 1,533 lines to
57,585 bytes / 1,537 lines. Its `-U3` patch section changes from 6,439 bytes /
124 lines to 2,673 bytes / 68 lines. The other 19 patch sections match the
current sections byte-for-byte.

The standalone [token-compare.go](token-compare.go) helper uses
`go/scanner` with `scanner.ScanComments`. It compares the complete ordered
`(token kind, literal)` sequences, including comments, automatically inserted
semicolons and EOF, while ignoring token positions. It rejects CR bytes to
avoid scanner normalization obscuring raw comment differences. No file has
CR bytes. All 21 Go members have equal token and comment sequences. The
remaining member, `VERSION`, stays byte-identical.

For `runtime2.go`, all 3,762 token entries and 856 comment literals match.
The ordered token-sequence SHA-256 is
`ed4dbd0c77366ab92ff8b75dff342a17af1cf424fb02815da9a9500dd9b90f2e`;
the ordered comment-sequence SHA-256 is
`6cf8075a9fbd963f9c2a8d32af59e57e836b036f589f77e7ae7771a2b7530921`.
The helper records these identities for the verified equal sequences in
[tokens.json](tokens.json). This establishes source-token and comment-literal
equality, including field order. It does not establish compiled ABI or native
runtime behavior.

## Identities and commands

| Input | SHA-256 |
| --- | --- |
| Original baseline `-U3` | `950063a87d63cb01dada2e6ef232fdeb4acc52f71a008e2158885666669f80bc` |
| Current integrated `-U3` | `86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c` |
| Current canonical `-U1` | `8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b` |
| Grouping candidate `-U3` | `760cd2b9a5324108de27eed5e98673c16b2bd7f2dc2b6dbe4da1ffc7efd0ab98` |
| Grouping candidate `-U1` | `b5294ed0f6d45ad562a19ecb85d62c7d50f34eee8ade936ab62b665ed35e25c9` |
| Candidate `runtime2.go` | `cdf896d1acd4c1e06746aa76a6e369e01443a3667856f0c670688f6e6b4826f2` |
| Shared descriptor | `94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779` |
| Pinned source archive | `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1` |

Preparation verified all 22 pristine/current members and all 79 current overlay
files against [the retained source-size evidence](../source-size-evidence.json).
The shared patch, descriptor, archive, complete overlay and both previous
scratch trees remained stable before and after the experiment.

The preparation command exited 0 in 0.115754 seconds. The verification command
exited 0 in 2.569228 seconds, from host-observed
`2026-10-04T05:15:03.733371+00:00` to
`2026-10-04T05:15:06.302626+00:00`. The supplied environment date remains
2026-10-03; these observed timestamps are retained literally.

The retained invocations are:

```sh
python3 .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-5/alignment-experiment/verify.py --scratch /tmp/fn110-alignment-1TWNKL9O --prepare
python3 .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-5/alignment-experiment/verify.py --scratch /tmp/fn110-alignment-1TWNKL9O
```

They describe this completed experiment. Preparation refuses an occupied
scratch directory; a repeat needs a fresh directory and the same four manual
blank-line insertions before verification. Preserve this evidence separately
if repeating, since the verifier writes its named output files.

Both representations use the original measurement command exactly:

```sh
git diff --no-index --no-ext-diff --no-textconv --binary --no-prefix --abbrev=7 --diff-algorithm=myers --unified=N -- a b
```

Current `-U3` and `-U1` reproduce the retained current patches byte-identically.
Repeated candidate `-U3` and `-U1` outputs match. Each candidate representation
passes GNU patch dry-run and real application using
`--batch -V none -p1 -F 0`, with no fuzz, offsets, rejects or backups. All 22
materialized members match the candidate bytes after each application.
All nonempty index-free diffs returned expected status 1; helper, formatting,
application and version commands returned status 0 with empty stderr.
[commands.json](commands.json) retains exact argv, working directories,
environment overrides, exits, durations and input/output identities.

The only Go compilation built the standalone scanner helper in its own scratch
directory and cache. The exact Go/gofmt paths, both helper source identities,
helper binary identity, full member inventories and verification-script identity
are retained in [evidence.json](evidence.json). The observed tool is stock
`go1.27.1 linux/arm64`; host `uname` is `Linux aarch64`. Requested routing was
thinking scout `gpt-6.1-sol` at high. Actual model execution metadata was not
host-observed.

## Owner handoff and limits

Blank-line grouping recovers alignment noise while preserving every diagnostic
field and comment, but leaves a measured 1,944-byte excess. Return that gap to
fn-110 task2 and the introduced-input owners, especially fn-112 task5. This
experiment supplies no authority to extend extraction scope, move protected
scheduler hooks, remove features or replace the original baseline.

No shared source or descriptor edit, generator, Gomad package loading, Gomad
test/build, Flow-state mutation or Git index/history operation ran. The candidate
has no qualified build identity and no retained artifact was relabeled.
Native R4 equivalence and R7 qualification on `darwin/arm64` and `linux/amd64`
remain open. No task, spec, milestone or SHIP acceptance is claimed.
