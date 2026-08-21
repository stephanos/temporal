# Compatibility Pack Review: modernc-libc-xsys-v047-isatty-v021

Review SHA-256: `sha256:e1c430be2a51fa3a6d58dcace06c50948b86a59f230781773d7ec46bd57e4b56`

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
  - profile `gomadv3-deterministic/v1` / `sha256:61c4a61f898b88c5b84e5cf533aed31d471c777fa09da1bc8c29bb791ee2221f`
  - adapter `modernc.org/libc@v1.72.3` / `h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`
  - source inventories `sha256:6a2ed9798fa07019c328f0247548082ef51b21aad8829c5600168aac4f683429` → `sha256:8579228404e49a9df26f1a5f735cd530e17f6264ed1c231bf15051d20b2cc76c`
  - prepared source set `sha256:8e1663c90aa178a706929ae94f248051781e4278ca83991d9a5fc6fe05321833`

## Reviewed packages

### `github.com/mattn/go-isatty`

Module: `github.com/mattn/go-isatty@v0.0.21` (`h1:xYae+lCNBP7QuW4PUnNG61ffM4hVIfm+zUzDuSzYLGs=`), replacement `none`

Source set: `sha256:a5f81d700b1f9b93da0a2fe3637cac475f972d057b015ea7e6dc76fef7d4b309`

Go sources:

- `doc.go`: `sha256:06182cb1a7113cae6fdef9be492893298610bfc63cf565a23f86203c3074a861`
- `isatty_bsd.go`: `sha256:b3df65aaddc2e985cc4b41be48e7a714eea17414cb8d09a04abcd2d35bf3f9e8`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

