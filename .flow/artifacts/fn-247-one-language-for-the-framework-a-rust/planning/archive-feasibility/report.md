# fn247 archive feasibility experiment

The old installer accepts the last exact named member when duplicates exist. It has no duplicate-refusal policy. A plain stdout extraction of that name concatenates duplicate payloads under GNU tar, so that command alone is not a drop-in implementation. The installer also accepts exactly one SHA-pinned production archive, rather than caller-supplied arbitrary tar input. These are separate facts with different planning consequences.

## Experiment boundary and evidence

Root authorized this disposable experiment after the draft-only handoff. All generated archives, mocked installations and outputs stayed under `.flow/tmp/fn247-archive-feasibility`. The harness loaded the actual helper at source `94e47d7878` with `git show` and executed its `install` function against temporary roots. It mocked download by writing fixture bytes and made no network download. Each case ran once with the original production archive checksum and once with its archive checksum patched as the existing installer tests do. The jar checksum in research was the local test jar checksum.

Actual environment was Linux, CPython3.14.4, GNU tar1.35 and gzip1.14. `bsdtar` was absent. No native macOS/BSD execution is claimed. The harness sets `LC_ALL=C`, `TZ=UTC` and empty `TAR_OPTIONS`. Complete argv, exit code, stdout/stderr hex and readable byte representations are in [results.json](results.json); the case summary is [summary.txt](summary.txt), and the reproducible research harness is [harness.py](harness.py). These are research artifacts, not a proposed production implementation or new runtime prerequisite.

Twenty-eight fixture archives exercised regular/contiguous/unknown types, symlink/hardlink/directory/FIFO, missing members, duplicate order, PAX name overrides and control characters, GNU format, name prefixes/trailing slash, unrelated traversal, wrong jar bytes, tar header checksum corruption, gzip CRC corruption, truncated gzip and trailing garbage.

Both tar operations filtered to the fixed ASCII member before listing/extraction. The listing command was:

```sh
tar --list --verbose --numeric-owner --no-recursion --no-wildcards --anchored --file ARCHIVE -- apalache-0.62.4/lib/apalache.jar
```

The extraction command substituted `--extract --to-stdout` for `--list --verbose`. Additional runs selected `--occurrence=1`, `=2` and `=3`. Every extraction was stdout-only. The dedicated extraction cwd remained empty and the traversal sentinel retained bytes `protected` after all commands. The harness itself created only its known fixture/private-install outputs.

## Observed old helper semantics

`getmember` selects the last exact named entry, then `isfile` decides its type, then `extractfile` supplies its data for the pinned jar hash. The Python documentation explicitly describes last-occurrence selection. It does not impose duplicate rejection. [Python tarfile reference](https://docs.python.org/3/library/tarfile.html#tarfile.TarFile.getmember)

The actual helper reproduced that behavior. An earlier wrong regular entry followed by the correct jar succeeded. A correct jar followed by a wrong regular entry failed its jar hash. A symlink followed by the regular jar succeeded; the reverse order failed regular-member admission. The helper accepted contiguous-file type7 as a file. Symlink, hardlink, directory, FIFO and unknown type were refused.

Its archive SHA check precedes tar parsing. All28 synthetic archives failed the original production hash before member inspection. Only deliberately mock-pinning an alternate archive exposed the generalized reader behavior. The public installer takes no archive argument and pins one release version/archive hash/jar hash. Unlike Cargo.lock/TOML inputs supplied to helper commands, it exposes no arbitrary-archive input grammar. The release archive itself was not available to this experiment and was not downloaded, so the known production archive's actual member count/type has not been newly observed here.

## Exact tool results and counterexamples

| Case | Actual old helper with fixture pin | GNU exact-name listing / stdout extraction |
| --- | --- | --- |
| Regular jar | Success,24bytes | Both exit0; same24bytes |
| Wrong jar | Jar integrity failure | Both exit0; stdout contains wrong bytes, so jar hash remains essential |
| Symlink/hardlink/directory/FIFO | Regular-member refusal | Exit0 with corresponding metadata; extraction emits no bytes. Exit success alone proves no regular-file admission |
| Missing | KeyError | Both exit2, empty stdout |
| Wrong then good duplicate | Success with final24bytes | Full extraction exit0 emits29bytes, `wrong` followed by jar. Occurrence2 emits final jar; occurrence3 exits2 not found |
| Good then symlink | Regular-member refusal | Full extraction still emits the earlier good jar. A hash-only stdout approach would wrongly accept this mock-pinned case |
| Symlink then good | Success with final jar | Listing contains both records; blanket duplicate/nonregular-record rejection would wrongly refuse this mock-pinned case |
| Contiguous type7 | Success | Listing begins `C`, not `-`; extraction emits correct bytes |
| PAX path override to exact member | Success | Both exit0, exact selected path/data |
| PAX override away / leading dot / leading slash / child-only | Missing exact member | Both exit2, no selected bytes |
| Other name containing newline plus regular member | Success | Only fixed exact name appears in selected listing/data |
| Selected link target with fake newline listing text | Regular-member refusal | Link target newline is escaped in GNU output; first record remains a link. No forged additional listing row appeared |
| Unrelated traversal plus regular member | Success | Only selected member emitted; extraction cwd and sentinel unchanged |
| Bad tar header checksum | ReadError | Both exit2, despite usable-looking selected stdout. Ignoring exit would lose this failure |
| Bad gzip CRC / truncated gzip trailer / appended garbage | Success after lazy tar data read | Both GNU operations exit2, while emitting the expected listing/jar. Strict gzip/tar exit checking differs in this mock-pinned universe |

The two most direct counterexamples to a naive port are `duplicate_last_good.tgz` and `duplicate_regular_then_link.tgz`. Their exact bytes and command output are retained in the results. The gzip cases establish another reader difference; adding a Rust gzip library would not automatically reproduce Python's lazy trailer behavior either.

## What exact filtering settles

GNU exact-name/no-recursion/no-wildcards filtering successfully excluded unrelated attacker-controlled names before verbose listing in these controls. Numeric ownership removed the intentionally malicious user/group strings from records. This is materially narrower than parsing an unrestricted archive listing. It still does not make one type byte a complete semantic interface. Contiguous type7, final-occurrence selection, link metadata, parser exit status and exact member spelling matter independently.

GNU occurrence selection provided last-entry data for the duplicate controls without filesystem extraction. A complete implementation would have to establish occurrence count and distinguish missing-occurrence termination from archive errors; this experiment has not declared such an algorithm complete. The GNU manual documents literal member selection and pattern controls. [GNU tar manual](https://www.gnu.org/software/tar/manual/tar.html), [GNU matching controls](https://www.gnu.org/software/tar/manual/html_node/controlling-pattern_002dmatching.html)

BSD evidence is source/documentation only. Its manual says stdout mode emits file data, no-recursion disables recursive processing, and fast-read selects the first matching entry. Its source sends every selected entry's data to stdout. Verbose listing prints a mode plus separate hardlink/symlink suffixes, and escapes control characters in path/link text. Thus a regular-looking first mode character alone cannot exclude a hardlink. The listing format is explicitly described as unspecified in its source. This evidence supports careful fixed-name filtering, but it supplies no native macOS proof or portable last-occurrence implementation. [libarchive bsdtar manual](https://raw.githubusercontent.com/libarchive/libarchive/master/tar/bsdtar.1), [bsdtar reader](https://raw.githubusercontent.com/libarchive/libarchive/master/tar/read.c), [bsdtar listing implementation](https://raw.githubusercontent.com/libarchive/libarchive/master/tar/util.c)

## Planning recommendation

Preserve the no-new-resolved-package plan. State archive parity against the one fixed production hash and the existing installer control suite. Retain archive-before-parse and jar-after-extraction hash authority, explicit regular-member admission, private staging, locks, symlink/containment checks and publication controls. Do not add duplicate rejection or full arbitrary-tar acceptance as a new requirement based on mock-pinned research.

The implementation's first archive proof should retain the actual pinned release from the old path, establish its selected member metadata and compare old/new bytes and outcomes using the fixed-name-filtered, stdout-only system-tool route on Linux and macOS. Any metadata parser must validate its complete selected-record grammar and reject ambiguous output; it cannot infer file type from a single leading byte. Existing positive/error/concurrency installer tests still require counterparts. Native BSD results remain an explicit evidence obligation rather than a reason to presume a new crate.

No portable drop-in for the broader, mock-pinned arbitrary-archive universe was demonstrated. If the conductor deliberately requires that additional universe, the precise counterexamples above need a parser-level solution. The smallest conventional Rust alternative is one tar reader plus one gzip decoder (`tar` and `flate2`, with a minimal Rust backend), separately pinned/admitted under R18. Neither is currently resolved/admitted, their transitive package changes require accounting, and matching Python's lazy malformed-gzip behavior would still need an explicit compatibility decision. A system libarchive FFI route adds an external ABI/library prerequisite instead. This report requests neither alternative; no evidence yet establishes that the fixed no-argument production CLI needs one.

No source, Cargo, canonical Flow state, index/history, shared build cache or framework prerequisite changed. No Cargo test/probe or official release download ran.
