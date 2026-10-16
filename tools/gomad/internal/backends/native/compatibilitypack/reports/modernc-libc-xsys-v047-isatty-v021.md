# Compatibility Pack Review: modernc-libc-xsys-v047-isatty-v021

Review SHA-256: `sha256:f4a856b6adb46a488aec8eee559252505d980f4c650ced2ef56ebbeae3f5c0d3`

Owner: `temporal-server`

Reviewed at: `2026-08-15T00:00:00Z`

Justification: Preserves the exact go-isatty v0.0.21 rule from the former v0.47 pack under the same registered libc adapter activation without combining impossible module versions in one request.

Target: `go-test ./temporal`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestNewServerWithOTEL$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `temporal-representative`

## Activation

- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`
- `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`), replacement `adapter`
  - profile `gomad-deterministic/v1` / `sha256:b6a41f88d2f60e098713d75172d99b025e57ddb285ff54580e2a945bbce02b07`
  - adapter `modernc.org/libc@v1.72.3` / `h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`
  - source inventories `sha256:7455942bdfcf64ff4d46cd874f1f6e80a79e4ebe6f4d98a9c2d1ae6aaabb59da` → `sha256:94c69a969829324cfbcfe9382b5abe2ad73009fbbc94bd469f903d9ea0304681`
  - prepared source set `sha256:39d7fabdd955565c06edef231c35d384595799edd349d6ab8fc9278c46da8500`

## Reviewed packages

### `github.com/mattn/go-isatty`

Module: `github.com/mattn/go-isatty@v0.0.21` (`h1:xYae+lCNBP7QuW4PUnNG61ffM4hVIfm+zUzDuSzYLGs=`), replacement `none`

Source set: `sha256:febfbceb8a937efc86e6e8ed71c61433c44c5afba2e6921c43a97e919a980037`

Go sources:

- `doc.go`: `sha256:06182cb1a7113cae6fdef9be492893298610bfc63cf565a23f86203c3074a861`
- `isatty_bsd.go`: `sha256:b3df65aaddc2e985cc4b41be48e7a714eea17414cb8d09a04abcd2d35bf3f9e8`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

