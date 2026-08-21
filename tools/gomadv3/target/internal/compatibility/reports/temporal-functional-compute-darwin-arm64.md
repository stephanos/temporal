# Compatibility Pack Review: temporal-functional-compute-darwin-arm64

Review SHA-256: `sha256:02de41d59b69070b4be0030c7cdd132280ecc2832d492bee75a8023268a84815`

Owner: `temporal-server`

Reviewed at: `2026-08-20T00:00:00Z`

Justification: Admits exact source-bound deterministic hashing, compression, and cryptographic assembly used by the local Temporal frontend functional workload.

Target: `go-test ./tests/gomadfunctional`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestFrontendSystemInfo$ -test.count=1`

Build tags: `disable_grpc_modules,test_dep`

Platform: `darwin/arm64`

Workload: `frontend-system-info`

## Activation

- `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`
- `github.com/golang/snappy@v1.0.0` (`h1:Oy607GVXHs7RtbggtPBnr2RmDArIsAefDwvrdWvRhGs=`), replacement `none`
- `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`
- `golang.org/x/crypto@v0.54.0` (`h1:YLIA59K4fiNzHzjnZt2tUJQjQtUWfWbeHBqKtk3eScw=`), replacement `none`

## Reviewed packages

### `github.com/cespare/xxhash/v2`

Module: `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`

Source set: `sha256:16884aeccbafc942c58cf1cfae6ce796ffc210f4cbd8a679243bb7da05e000e0`

Go sources:

- `xxhash.go`: `sha256:cc024316c7e49696f5705195951e49a8d24b612e2f95bec41ee4cd71990b78f9`
- `xxhash_asm.go`: `sha256:f5a64edc8b76317c95879329a0f3b358773fe3b529b8b88206012c9379145fc7`
- `xxhash_unsafe.go`: `sha256:b164ad04d24b0d1f5fbde666ae3806f4f33a23044359f63162aed343bcc97eb3`

Foreign sources:

- `assembly:xxhash_arm64.s`: `sha256:f878f122d4af5bf05d12d5cffb9ab841a42aebba32ef551afe153d9b3c2c3ad0`

Requested facts:

- `foreign:assembly:xxhash_arm64.s`: **allow** — **security-sensitive**

### `github.com/golang/snappy`

Module: `github.com/golang/snappy@v1.0.0` (`h1:Oy607GVXHs7RtbggtPBnr2RmDArIsAefDwvrdWvRhGs=`), replacement `none`

Source set: `sha256:26e5bf1a63d834390db92e6cfc6760c4eed832097386ce4c5fd101fbe070c031`

Go sources:

- `decode.go`: `sha256:eebff83e4ab463713bb79b4d9f35c0212e72b9e1e02b18fb1632b412d4c8c192`
- `decode_asm.go`: `sha256:37ffc5a5ac8a0c376dc497891901010529b6bb510a8b9b5512f94eda3497274b`
- `encode.go`: `sha256:b4e357ac92d94c339a523d5c81834eb171f44047a5350376475e7424e1fbbe6e`
- `encode_asm.go`: `sha256:992413050d507073a011886d44c2d157d56e0dc949bda4704995a907edabca7b`
- `snappy.go`: `sha256:6fb3bc0c2c735aa29e587a64139267fb9cb3e1f947c88c2182f2998ebb2d3e5e`

Foreign sources:

- `assembly:decode_arm64.s`: `sha256:bbb0e55057c75a4f982cd49b46692af9c227d9fad802aa85b7b119d24fc899ba`
- `assembly:encode_arm64.s`: `sha256:ec7c32640eb4a29f02f471e44c5a610f2d8cae135ad64d7f8abc41f1a34be853`

Requested facts:

- `foreign:assembly:decode_arm64.s`: **allow** — **security-sensitive**
- `foreign:assembly:encode_arm64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/zstd/internal/xxhash`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:816382ab980b5f8264b4a1c06aa96e75a00854c5698d94852991098f16094f86`

Go sources:

- `xxhash.go`: `sha256:83344ca444865877a307d2980068f883716736e9a5b8fca36d13e5557ee319c1`
- `xxhash_asm.go`: `sha256:51742c9f72a6460f70d4a9dab6285074e7e59a874a40019f9af1821db34d3e23`
- `xxhash_safe.go`: `sha256:5a12c499074f3428854b32094344f11c8622d8e1548710d6c4e9f9ce365cd19a`

Foreign sources:

- `assembly:xxhash_arm64.s`: `sha256:0e2b30d48c0ab8035e201d06c5b74813e39da76c7dc7e3239f4dd4acba7fbb64`

Requested facts:

- `foreign:assembly:xxhash_arm64.s`: **allow** — **security-sensitive**

### `golang.org/x/crypto/chacha20`

Module: `golang.org/x/crypto@v0.54.0` (`h1:YLIA59K4fiNzHzjnZt2tUJQjQtUWfWbeHBqKtk3eScw=`), replacement `none`

Source set: `sha256:246ec5fac15e3d199bb0ec684eb0586d4a19fec0d2a5a95035f7702f1ce2af71`

Go sources:

- `chacha_arm64.go`: `sha256:50974d7653c355d8356aaa33318f1df5d9127b5dc96a38b5d88e91aeb188feb4`
- `chacha_generic.go`: `sha256:34403e82b1387b4402b00ce30c1364508c333f4bdbe671321690c6ebaa8d3180`
- `xor.go`: `sha256:c3adc8555766e36629b4dd3fcd579969d2d2737c65a379db997253a2b7b18072`

Foreign sources:

- `assembly:chacha_arm64.s`: `sha256:73036d54d1961a9e5cb8ce57298bd7803443dbe0de7cae4a9143b62d3a7d84b5`

Requested facts:

- `foreign:assembly:chacha_arm64.s`: **allow** — **security-sensitive**

