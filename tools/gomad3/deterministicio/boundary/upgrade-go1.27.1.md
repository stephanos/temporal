# Gomad v3 upgrade qualification: go1.27.1-v1

Generated from [`../../toolchain/version/version.json`](../../toolchain/version/version.json). Do not edit this guide directly.

## Pinned inputs

- Go release: `go1.27.1`
- source archive SHA-256: `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`
- supported platforms: `darwin/arm64`, `linux/amd64`
- boundary manifest: `go1.27.1-v1`
- patch: [`../../toolchain/runtime/go1.27.1.patch`](../../toolchain/runtime/go1.27.1.patch)
- adapter: `golang.org/x/net@v0.57.0` (`h1:K5+3DljvIuDG9/Jv9rvyMywYNFCQ9RSUY6OOTTkT+tE=`)
- adapter: `google.golang.org/grpc@v1.80.0` (`h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM=`)
- adapter: `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`)
- adapter: `modernc.org/memory@v1.11.0` (`h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=`)

## Qualification command

Run from the Gomad source module root after updating `toolchain/version/version.json`, the boundary manifest, patch, and overlays:

```sh
make generate
make upgrade-dossier GOMAD3_BASELINE_REF=<previous-commit>
```

The command publishes `.toolchain/upgrade-dossier.json`, even when a behavioral gate or boundary approval fails. The dossier contains the complete upstream patch diff, semantic boundary-manifest diff, expected and applied interception evidence, archive-based overlay collision results, disabled-mode upstream results, mandatory-probe gates, host-clock escape audit, retained core-corpus report, and platform qualification. If the dossier reports boundary changes, rerun only after reviewing and approving the complete diff:

```sh
make upgrade-dossier GOMAD3_BASELINE_REF=<previous-commit> GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256=<boundary_manifest_diff.sha256>
```

CI uploads the dossier on every run.
