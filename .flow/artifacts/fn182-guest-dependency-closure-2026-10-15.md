# fn-182.1 guest dependency closures and selected exclusions

Source: `gomad` `01c419c40a83553302312e532934b77dc857fa70` (pre-fn-178; provisional
under the batch-first strategy). The seam was measured on branch `fn182`. Re-verify on the
joined fn-178 candidate before completion. Host: real Linux/arm64, stock Go 1.27.1.
This report covers R1 and the module-size part of R4. The timed R4 baseline is still open (see the end).

## Inputs

Guest build environment (the provider's): `GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 GOFLAGS=
GOWORK=off GOTOOLCHAIN=local GOEXPERIMENT=nogreenteagc GOENV=off`, `-mod=readonly -pgo=off`.

| Guest | Package | Qualified tags |
| --- | --- | --- |
| One-box (`TestActualOneboxQualification`) | `./tests` (go test) | `disable_grpc_modules,sqlite3_dotlk,test_dep` |
| Multi-instance (`TestActualTemporalClusterQualification`) | staged `testdata/temporal_storage/recovery_test.go` + `testdata/temporal_broker/replication_test.go` (staged here as `.tmp/fn182-cluster-fixture`) | same |

Closure: `go list -mod=readonly -pgo=off -deps -test -json -tags <tags> <package>`.
Analysis scripts are in `fn182-guest-dependency-closure-2026-10-15/`. `candidates.py` bans whole
package prefixes. `edgecut.py` removes only named import edges. `diff.py` compares two actual
listings. `inits.py` lists `init`/`Register*` hooks in removed packages.

## Closures

| Guest | Packages (deduplicated) | Largest optional families (source bytes, packages) |
| --- | --- | --- |
| One-box | 1466 | AWS SDK 16.0 MB/117 (s3, bedrock, ecs, lambda); auto-scaled-workers 15.1 MB/129; GCP 6.1 MB/134; k8s 3.4 MB/92; olivere/elastic 1.9 MB/3; pgx 1.0 MB/12; gocql 0.5 MB/4 |
| Multi-instance | 1446 | same families |

Package counts and source bytes are not linked bytes. Linked module bytes come from the disposable
builds below.

## Candidates

| Candidate | Seam (non-candidate importers) | Packages removed | Decision |
| --- | --- | --- | --- |
| Cloud archivers (gcloud, s3store) | one: `common/archiver/provider` → `archiver/gcloud`, `archiver/s3store`; existing native `provider_cloud{,_gomad}.go` seam | 47 | **Selected.** Configuration-selected by `gs`/`s3` URI schemes; guests use none |
| MySQL/PostgreSQL test drivers | one: `tests/testcore` → `sqlplugin/mysql`, `sqlplugin/postgresql` (`flag_sql.go`) | 39 | **Selected.** Flag-selected (`-persistenceDriver`); qualified guests use SQLite |
| Ringpop membership | one: `temporal` → `membership/ringpop`; existing `membership{,_gomad}.go` seam | 40 | **Selected.** Used only without `StaticServiceHosts`; `testcore` always passes `temporal.WithStaticHosts` |
| Elasticsearch visibility | 11 edges: `common/config` (`client.Config` YAML type), `visibility` factory, `service/history` (`ProcessorConfig`), `service/frontend` (store name), `service/worker/scheduler` (`NewNameInterceptor`, `NewQueryConverterLegacy`), `store/query` → `olivere/elastic` (query/sorter types), testcore/tests/temporal/resourcetest | 0 reachable by any config-only seam | **Rejected.** The scheduler's query converter needs `olivere/elastic` and the ES store package with any visibility store. Gating `elasticsearch.NewVisibilityStore` or `esclient.NewClient` removes no package |
| ES AWS request signing | `elasticsearch/client` → `aws/*` (native `aws{,_gomad}.go`) | 0 | **Rejected.** `./tests` imports auto-scaled-workers compute providers directly, which keep `aws`, `config`, `credentials` and `sts` |
| WCI worker component | `service/worker` → `auto-scaled-workers/wci/workercomponent` | 4 (one-box), 128 (multi-instance) | **Rejected.** Not configuration-selected: an always-enabled per-namespace worker component that registers WCI workflows/activities. Removing it changes runtime behavior; `./tests` also imports WCI directly |
| Cassandra | `persistence/client`, `temporal`, `persistence-tests` → `persistence/cassandra`; `common/config` → `gocql` | 4 (0.25 MB source) | **Not shipped.** Needs three seams for little gain, and `gocql` stays through `config` |
| Prometheus, OTel, gRPC, k8s via WCI | core metrics/RPC or the direct WCI import | — | Required |

Removing the three selected seams (`archivers+sql-drivers+ringpop`) eliminates 126 packages and
11,013,066 source bytes from **both** guests. Nothing is added. The actual slim listing equals the
default listing minus those five import edges. The predicted and actual removed sets are identical
(`removed-packages.txt`). `TestSlimGuestClosureCutsOnlyOptionalProviderSeams` re-checks this
equality for `./tests` on every run. It also checks that the ES, Cassandra, WCI, filestore, static
membership and SQLite packages stay included.

Removed modules (fully): cloud.google.com/go{,/monitoring,/storage}, GoogleCloudPlatform
opentelemetry-operations-go (detectors, exporter, resourcemapping), contrib/detectors/gcp,
aws-sdk-go-v2/service/{s3,internal/checksum,internal/s3shared}, go-sql-driver/mysql,
jackc/{pgx/v5,puddle/v2,pgpassfile,pgservicefile}, lib/pq, filippo.io/edwards25519,
temporalio/{ringpop-go,tchannel-go}, apache/thrift, uber-common/bark, sirupsen/logrus,
rcrowley/go-metrics, opentracing-go, cactus/go-statsd-client, benbjohnson/clock.

## Registration review (removed packages)

`inits.py` lists every `init` and `Register*` call in the 126 removed packages.

- **Protobuf registrations:** genproto `api/{distribution,label,metric,monitoredres}` and
  `type/{calendarperiod,date,timeofday}`, plus `monitoringpb` and `storagepb`. These register only
  Google Cloud message types into the global registry, which Temporal payloads do not use.
- **`database/sql` drivers:** `mysql`, `postgres`, `pgx` and Temporal `sql.RegisterPlugin` for
  MySQL/PostgreSQL. These register only selectable drivers; selecting a removed one now fails
  (below).
- **`grpc/stats/opentelemetry`:** its `init` sets only `otelinternal.SetPluginOption`, which is
  used by OTel-plugin consumers (GCS gRPC).
- **Other packages with `init` code:** `membership/ringpop` (ringpop-local service-type map),
  `rcrowley/go-metrics` (allocates `gcStats.Pause`) and logrus/tchannel/thrift (package-local
  state). None of them registers into a package the guest keeps.

## Disposable same-selection builds (stock `go test -c -p 4`, no runtime overlay)

All bytes are linked `.wasm` sizes. The workload, arguments and selection are unchanged.

| Build | One-box `./tests` | Δ |
| --- | ---: | ---: |
| default | 337,877,939 (`347bd65a…`) | |
| archivers only | 320,358,564 | −17,519,375 |
| ringpop only | 334,790,566 | −3,087,373 |
| SQL drivers only | 327,052,828 | −10,825,111 |
| all three (`gomad_wasm_slim`) | 306,274,379 (`ffdda843…`) | **−31,603,560 (−9.35%)** |

| Build | Multi-instance fixture | Δ |
| --- | ---: | ---: |
| default | 240,188,710 (`e7baf681…`) | |
| `gomad_wasm_slim` | 208,975,675 (`d0bc8dd4…`) | **−31,213,035 (−13.0%)** |

The per-seam rows used temporary single-seam tags in the scratch tree, reverted before commit.
The cooperative provider modules (runtime overlay) are measured by .3 from the prepared receipts.

## Preserved-provider seam

One guest-only tag, `gomad_wasm_slim` (`backend.SlimGuestBuildTag`), extends the existing native
`gomad` seams. The native `gomad` tag is not applied to WASM guests, because it also changes
local-IP, interrupt, password and errno behavior and drops test files. Default production,
stock and native builds select the same files as before: `!gomad && !gomad_wasm_slim` equals
`!gomad` when the new tag is unset. Selecting an excluded provider fails explicitly:

- **Cloud archivers:** configured `gs`/`s3` schemes return `errCloudArchiverUnavailable`.
- **Ringpop:** the default membership module is `fx.Error`, so a cluster must supply
  `StaticServiceHosts`.
- **MySQL/PostgreSQL:** the slim `flag_sql_wasm_slim.go` keeps every SQL driver name. MySQL and
  PostgreSQL therefore still select SQL visibility (no fallback to Elasticsearch) and fail at
  plugin lookup with `sql.ErrPluginNotSupported`: `supported plugins: [sqlite]`.

## Open: timed pre-change R4 baseline

The timed baseline (compiled-cache bytes, cold compile, warm load, fresh instantiation,
execution wall, helper peak RSS) needs a quiet host and the post-fn-178 baseline source. It has not
been measured. The prepared rerun procedure is in the fn-182.3 evidence.
