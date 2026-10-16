# Compatibility Pack Review: temporal-leaf-xxhash-darwin-arm64

Review SHA-256: `sha256:deb0530c6b647ae42a708281a8c3b45643db2f9f12f9d9f52740a32861d68e03`

Owner: `temporal-server`

Reviewed at: `2026-09-27T00:00:00Z`

Justification: Admits the arm64 xxhash assembly of github.com/cespare/xxhash/v2 and klauspost/compress that the Prometheus client and zstd reach in the gomad build of the Temporal cache leaf test package on darwin/arm64; that closure lacks the snappy and x/crypto modules that activate temporal-functional-compute-darwin-arm64, which admits the same facts for the functional tests, and temporal-leaf-xsys-darwin-arm64 admits its golang.org/x/sys/unix facts.

Target: `go-test ./common/cache`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestSimpleCacheConcurrentAccess$ -test.count=1`

Build tags: `gomad`

Platform: `darwin/arm64`

Workload: `temporal-cache-concurrent`

## Activation

- `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`
- `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

## Reviewed packages

### `github.com/cespare/xxhash/v2`

Module: `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`

Source set: `sha256:287fab0ba25d4ed1e7cb6d7d87a0cbcbf78c6b212a6c44fc43987bf902f160ad`

Go sources:

- `xxhash.go`: `sha256:cc024316c7e49696f5705195951e49a8d24b612e2f95bec41ee4cd71990b78f9`
- `xxhash_asm.go`: `sha256:f5a64edc8b76317c95879329a0f3b358773fe3b529b8b88206012c9379145fc7`
- `xxhash_unsafe.go`: `sha256:b164ad04d24b0d1f5fbde666ae3806f4f33a23044359f63162aed343bcc97eb3`

Foreign sources:

- `assembly:xxhash_arm64.s`: `sha256:f878f122d4af5bf05d12d5cffb9ab841a42aebba32ef551afe153d9b3c2c3ad0`

Requested facts:

- `foreign:assembly:xxhash_arm64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/zstd/internal/xxhash`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:0f3195b6387cee5541136bbc3eca23f82ba0245f5d7fec7adaba331e6931ea8d`

Go sources:

- `xxhash.go`: `sha256:83344ca444865877a307d2980068f883716736e9a5b8fca36d13e5557ee319c1`
- `xxhash_asm.go`: `sha256:51742c9f72a6460f70d4a9dab6285074e7e59a874a40019f9af1821db34d3e23`
- `xxhash_safe.go`: `sha256:5a12c499074f3428854b32094344f11c8622d8e1548710d6c4e9f9ce365cd19a`

Foreign sources:

- `assembly:xxhash_arm64.s`: `sha256:0e2b30d48c0ab8035e201d06c5b74813e39da76c7dc7e3239f4dd4acba7fbb64`

Requested facts:

- `foreign:assembly:xxhash_arm64.s`: **allow** — **security-sensitive**

