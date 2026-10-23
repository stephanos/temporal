# Namespace image conformance corpus

`TestNamespaceImageConformanceCorpus` generates these named codec vectors with
the Go production namespace image writer. Malformed vectors change the named
field or truncate the encoded image at the named byte. Valid vectors pin the
SHA-256 of the complete decoded public image, including entry order, paths,
readonly flags, data, configured mount order, stdin, limits and epoch. The Rust
fixture loader also constructs a namespace from each accepted image.

`repeat` inserts `count` copies of `byte` at the encoded input's `offset` before
decoding. Only the large-path metadata vector uses this compact representation;
the declared binary path length and expected decoded-image hash include the
expanded bytes. It has no runtime protocol meaning.

From `tools/gomad`, regenerate with:

```sh
go test -tags test_dep -count=1 -run '^TestNamespaceImageConformanceCorpus$' ./internal/model/wasi -args -update-conformance
```

Repeat the command and require a byte-identical corpus before accepting changes.
