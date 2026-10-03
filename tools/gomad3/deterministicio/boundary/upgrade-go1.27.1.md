# Gomad v3 upgrade qualification: go1.27.1-v1

Generated from [`../../toolchain/version/version.json`](../../toolchain/version/version.json). Do not edit this guide directly.

## Pinned inputs

- Go release: `go1.27.1`
- source archive SHA-256: `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`
- supported platforms: `darwin/arm64`, `linux/amd64`
- boundary manifest: `go1.27.1-v1`
- patch: [`../../toolchain/runtime/go1.27.1.patch`](../../toolchain/runtime/go1.27.1.patch)
- adapter: `github.com/Masterminds/sprig/v3@v3.3.0` (`h1:mQh0Yrg1XPo6vjYXgtf5OtijNAKJRNcTdOOGZe3tPhs=`)
- adapter: `github.com/cactus/go-statsd-client/v5@v5.1.0` (`h1:sbbdfIl9PgisjEoXzvXI1lwUKWElngsjJKaZeC021P4=`)
- adapter: `github.com/cockroachdb/pebble@v0.0.0-20260703021901-41f35d3cb7df` (`h1:p7vkumDcPw0de7t8pYA95HPC4cYQZGDG6b57d4Om5cA=`)
- adapter: `github.com/getsentry/sentry-go@v0.46.0` (`h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0=`)
- adapter: `github.com/go-playground/validator/v10@v10.30.1` (`h1:f3zDSN/zOma+w6+1Wswgd9fLkdwy06ntQJp0BBvFG0w=`)
- adapter: `github.com/hashicorp/go-metrics@v0.5.4` (`h1:8mmPiIJkTPPEbAiV97IxdAGNdRdaWwVap1BU6elejKY=`)
- adapter: `github.com/hashicorp/go-sockaddr@v1.0.7` (`h1:G+pTkSO01HpR5qCxg7lxfsFEZaG+C0VssTy/9dbT+Fw=`)
- adapter: `github.com/hashicorp/memberlist@v0.5.4` (`h1:40YY+3qq2tAUhZIMEK8kqusKZBBjdwJ3NUjvYkcxh74=`)
- adapter: `go.opentelemetry.io/otel/sdk@v1.44.0` (`h1:nHYwb9lK+fJPU/dnT6s7W7Z8itMWyqrnVfbheVYrZ58=`)
- adapter: `go.temporal.io/sdk@v1.48.0` (`h1:WDctKDVuh0Z8Nf7euAyqs/EwcPg1JTIIq1Fut8Tq118=`)
- adapter: `go.uber.org/fx@v1.24.0` (`h1:wE8mruvpg2kiiL1Vqd0CC+tr0/24XIB10Iwp2lLWzkg=`)
- adapter: `golang.org/x/net@v0.58.0` (`h1:ynWG7rqYi4ccpTEuPZ2QGWHktVEM9DMCj9yzDE0Q7To=`)
- adapter: `google.golang.org/grpc@v1.83.2` (`h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU=`)
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

## Dependency bumps

A bump of an adapted or compatibility-packed module keeps every pin exact. With the bump applied to the target module and not yet committed, run from the Gomad source module root:

```sh
go run ./cmd/gomadtool pin-impact
go run ./cmd/gomadtool adapter-regenerate --module=<adapted-module> --version=<new-version>
go run ./cmd/gomadtool adapter-regenerate --module=<adapted-module> --version=<new-version> --approve-review=<reviewed-digest>
go run ./cmd/gomadtool compatibility-pack refresh --root=.
```

`pin-impact` lists the invalidated pins. Regenerate each adapter it names, reviewing the dry run's upstream diff before applying its digest; an anchor that no longer matches exactly once stops the command and needs a person. `compatibility-pack refresh` re-reviews every invalidated request and prints the `compatibility-pack generate --approve-review` command for each; each platform's host approves and qualifies its own requests with `make validate compatibility-pack-qualification`. After committing the bump, pass the revision before it with `--baseline-ref`. A changed adapter changes the target identity, so rebuild `.bin/gomad` and requalify its workloads.
