# Retained World gzip codec

This private Rust codec preserves the byte stream produced by the pinned stock
Go 1.27.1 `gzip.NewWriter` (default compression). It ports the level-6 fast matcher,
length-limited Huffman generator, token entropy estimate, and block/table reuse
choices. It does not call Go or a system compressor in production. No Cargo
dependency was added. The gzip framing and DEFLATE tree validity rules follow
that same Go source. `GO-LICENSE` retains its BSD terms.

The caller retains the GMWORLD2 envelope and canonical GMWSNAP1 plaintext. Encoding
caps expanded input at 384 MiB and gzip output at 64 MiB. Output stops at the
compressed cap; matcher history remains bounded to 327,675 bytes. Decoding accepts
stored, fixed and dynamic blocks, checks gzip header CRC/data CRC/size, and stops
at the first member. It returns consumed bytes so the World envelope can reject
trailing data. Output allocation capacity never exceeds the caller's declared
expanded limit. Header strings use Go's 512-byte bound; extra data uses its 16-bit
length. Noncanonical but valid gzip headers and compression levels remain valid
inputs, as in Go. The canonical encoder always writes zero MTIME, XFL=0, OS=255.

The Go algorithm uses signed history offsets and intentional low-bit truncation.
The Rust module keeps those operations and f32 entropy arithmetic explicit; its
384 MiB entry bound keeps the history offset below Go's 2 GiB reset threshold.
This codec must be requalified when the pinned Go version changes.

## Tests

The ordinary Rust suite runs dependency-free fixed regressions and malformed-input
checks. Differential tests are opt-in because they execute the independent Go
oracle, and the full-profile test additionally needs a coordinated memory window.
These are codec tests, separate from live guest/qualification acceptance.

From the repository root, build the tiny test-only oracle using the stock pinned
compiler (`GOMAD_STOCK_GO` must name Go 1.27.1):

```sh
mkdir -p .flow/tmp
(cd tools/gomad && GOENV=off GOTOOLCHAIN=local "$GOMAD_STOCK_GO" build -o ../../.flow/tmp/world-gzip-oracle ./rust/wasmhost/src/world/deflate/testdata/gzip-oracle)
export GOMAD_GO_GZIP_ORACLE="$PWD/.flow/tmp/world-gzip-oracle"
cargo +1.99.1 test --locked --manifest-path tools/gomad/rust/wasmhost/Cargo.toml --lib world:: -- --include-ignored --test-threads=1
```

The oracle imports the unchanged Go World model to generate real snapshots and
uses only the pinned Go standard library to encode/decode. Tests compare complete
raw gzip bytes, not normalized or decompressed-only representations. The corpus
covers empty/small/repeated/random/lazy-match inputs, Huffman distribution changes,
65,535-byte boundaries, history rollover, actual World snapshots, all DEFLATE
block kinds, corruption and truncation, optional headers, and original 384/64 MiB
bounds. The full-profile test emits the actual expanded/compressed byte counts.

A small standalone harness is available at `testdata/standalone.rs`; it exercises
the same production modules and tests without building Wasmtime. Supply the
existing project's crc32fast 1.5.2 rlib to rustc and its dependency search path.

## Pinned Go source SHA-256

Paths are relative to the pinned Go distribution root. The implementation was
read from `.build-cache/fn232-stock-go1.27.1` in the original checkout; no frozen
reference source or binary was modified.

```text
50d6c83cdff9ccd5cce90ce1dc9d827de0648dd01a78032d3ad4a8932b642d04  src/compress/flate/deflate.go
ca01c6426266a5e2b5154cf1123a357be37f016247f14409c97a8d744bdc6518  src/compress/flate/deflatefast.go
0bbd86f2ec1b856d293c5e2cf2f09bd7e82cbb4d99f5aa7a2959b719b330b26c  src/compress/flate/level6.go
ce22b308bb43d26661d79dd68b15dff0526212b3f2ff4ecaffb12f7bbed45fcc  src/compress/flate/huffman_bit_writer.go
1098a9d935c86863d49b8fc9e81481a6ac3bca41686c4350be120ef10c68fb85  src/compress/flate/huffman_code.go
19c2dabd2e911839dc74c9f63131eb092cdc053c4bddf6654310d6b5da70e3be  src/compress/flate/token.go
28b241ba6b66cc14ac93cc17a49756cd78daaade60afb47d7cd6aed2489dc5cb  src/compress/flate/inflate.go
15564f213c583a22e79da0611c173c28ead54fa6651ad9bc7797938048e77ba0  src/compress/gzip/gzip.go
5886344b7fd61358d430872ab735653b16a314c87dab1f0bde4d8d9db0381ec4  src/compress/gzip/gunzip.go
911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad  LICENSE
```
