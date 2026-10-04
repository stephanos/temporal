# Conductor verification of the alignment experiment

Independent checks on 2026-10-04, after reading the complete experiment verifier
and scanner helper, confirm a source-only candidate rather than an adopted fix.

The retained scanner binary freshly compared all 21 Go files in
`/tmp/fn110-alignment-1TWNKL9O/current/b` and `candidate/b` and returned
`all_equal: true`. It includes ordered comment literals and inserted semicolons;
the `runtime2.go` sequence contains 3,762 tokens and 856 comments. Its token
SHA-256 is `ed4dbd0c77366ab92ff8b75dff342a17af1cf424fb02815da9a9500dd9b90f2e`.
The helper rejects CR bytes, avoiding normalization of comment text.

Fresh recursive comparisons of `candidate/b` with both retained `applied-U3`
and `applied-U1` returned no differences. A fresh index-free Git diff using
the original command (`--diff-algorithm=myers --unified=3 --no-prefix
--abbrev=7 --binary --no-ext-diff --no-textconv`) matched the retained
candidate `-U3` byte-for-byte. These read-only checks exited zero.

Fresh SHA-256 and physical-byte checks returned:

| Representation | Bytes | SHA-256 |
| --- | ---: | --- |
| Scratch candidate `-U3` | 34,596 | `760cd2b9a5324108de27eed5e98673c16b2bd7f2dc2b6dbe4da1ffc7efd0ab98` |
| Scratch candidate `-U1` | 25,330 | `b5294ed0f6d45ad562a19ecb85d62c7d50f34eee8ade936ab62b665ed35e25c9` |
| Original baseline `-U3` | 32,652 | See [experiment report](alignment-experiment/report.md) |

The candidate saves 3,766 bytes against the current 38,362-byte `-U3`, but
still exceeds the original baseline by 1,944 bytes. R8's strict extraction
reduction therefore remains unmet. The 9,266-byte context-only saving between
candidate `-U3` and `-U1` cannot substitute for it.

The shared canonical patch SHA-256 remains
`8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b`,
and the descriptor remains
`94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779`.
No shared runtime source, patch, descriptor, generator, Git index, or historical
artifact changed during these checks. They do not establish a compiled ABI,
native runtime behavior, or native R4/R7 qualification. No source fix or
acceptance status is inferred.
