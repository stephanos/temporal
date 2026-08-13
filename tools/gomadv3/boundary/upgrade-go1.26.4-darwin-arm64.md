# Gomad v3 upgrade qualification: go1.26.4-darwin-arm64-v1

Generated from [`../version.json`](../version.json). Do not edit this guide directly.

## Pinned inputs

- Go release: `go1.26.4`
- source archive SHA-256: `4f668a32fbfc1132e6a881fb968c2f1dada631492a339211735fbb255a42602d`
- supported platforms: `darwin/arm64`
- boundary manifest: `go1.26.4-darwin-arm64-v1`
- patch: [`../go1.26.4.patch`](../go1.26.4.patch)
- adapter: `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`)

## Qualification command

Run from the Gomad source module root after updating `version.json`, the boundary manifest, patch, and overlays:

```sh
make generate
make upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>
```

The command publishes `.toolchain/upgrade-dossier.json`, even when a behavioral gate or boundary approval fails. The dossier contains the complete upstream patch diff, semantic boundary-manifest diff, expected and applied interception evidence, archive-based overlay collision results, disabled-mode upstream results, mandatory-probe gates, host-clock escape audit, retained core-corpus report, and platform qualification. If the dossier reports boundary changes, rerun only after reviewing and approving the complete diff:

```sh
make upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit> GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256=<boundary_manifest_diff.sha256>
```

CI uploads the dossier on every run.
