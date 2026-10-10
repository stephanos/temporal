# Temporal server WASI binary size

Assessment date: 2026-10-09. Source candidate: `ac4c448f8626a5090a7be04f7cc249325d2198f9` on `wasm/wasi-build`. Toolchain: stock Go 1.27.0. This follows the [WASI compilation report](2026-10-09-wasi-compilation.md).

The measured provider exclusions reduce the stripped server from 217.47 MiB to 141.84 MiB while retaining SQLite, FTS5, Prometheus, and its core service graph. The exclusions must remove constructor or registration references from the compiled source. Choosing SQLite or disabling archives in YAML leaves the alternative constructors available to the linker.

## Measurements

Every artifact below compiled successfully for `wasip1/wasm`. Each provider exclusion uses `-ldflags='-s -w'` and differs from the stripped baseline by the named source overlay. The `release` row uses the existing build tag. MiB means 1,048,576 bytes. Savings compare against the stripped baseline.

| Artifact | Bytes | MiB | Saved MiB |
| --- | ---: | ---: | ---: |
| Current server, no stripping | 238,067,535 | 227.04 | - |
| Stripped baseline | 228,030,599 | 217.47 | 0.00 |
| Exclude auto-scaled-worker component | 186,832,280 | 178.18 | 39.29 |
| Exclude cloud archives | 211,271,065 | 201.48 | 15.98 |
| Exclude MySQL and PostgreSQL registrations | 217,855,270 | 207.76 | 9.70 |
| Exclude Cassandra provider and schema checks | 225,348,075 | 214.91 | 2.56 |
| Exclude ES AWS signing | 228,016,859 | 217.45 | 0.01 |
| Add upstream `release` tag | 218,954,224 | 208.81 | 8.66 |
| Combine the five provider exclusions | 148,735,231 | 141.84 | 75.62 |

The combined artifact saves 89,332,304 bytes, or 37.52%, against the current unstripped server. It saves 79,295,368 bytes, or 34.77%, against the stripped baseline. The individual cuts share dependencies and can jointly eliminate roots that either cut leaves alive. Their savings are nonadditive. ES signing alone saves 13,740 bytes because archives and WCI still retain AWS dependencies.

| WebAssembly section payload | Current server MiB | Stripped baseline MiB | Combined MiB |
| --- | ---: | ---: | ---: |
| Code | 125.80 | 125.80 | 78.07 |
| Data | 91.18 | 91.18 | 63.44 |
| Function names | 9.57 | 0.00 | 0.00 |

Function bodies fall from 129,386 to 91,964 in the combined artifact. Section payloads exclude section headers and other small sections. Data includes Go runtime and program data; the analyzer does not attribute it to individual dependencies.

The unstripped module's function names permit approximate package attribution of code body bytes. This excludes data, names, and section encoding overhead and gives no standalone removal estimate.

| Baseline package | Approximate code MiB |
| --- | ---: |
| AWS Bedrock AgentCore control client | 9.00 |
| Generated SQLite module | 7.09 |
| AWS S3 client | 6.73 |
| AWS Lambda client | 5.09 |
| AWS Bedrock AgentCore data client | 4.47 |
| Olivere Elasticsearch client | 3.25 |
| Temporal workflow service API | 2.86 |
| pgx PostgreSQL type codecs | 2.42 |
| Generated SQLite FTS5 module | 1.45 |

The compiled package closure falls from 1,414 packages to 1,000. The combined `go list` output contains no AWS SDK, `cloud.google.com/go`, `google.golang.org/api`, Kubernetes, MySQL driver, or pgx packages. It retains four `gocql` packages through configuration, three Olivere packages, eight Prometheus packages, sixteen ncruces packages, and all four default service package roots. This describes package closure; the code-size table describes linked function bodies.

## Experiment reproduction

The local overlay generator is [setup_variants.py](../../../.tmp/wasi-size/setup_variants.py). It removes MySQL/PostgreSQL blank imports, substitutes the existing cloud, WCI, and ES signing `gomad` implementations without enabling the wider tag, and replaces Cassandra factory and schema-check references with explicit unsupported paths. The combined overlay merges those five substitutions. These ignored experiment files are local evidence and are not part of the PR stack.

These commands reproduce the experiment in this candidate checkout with the retained local scripts:

```sh
make temporal-server-wasi
mkdir -p .tmp/wasi-size
python3 .tmp/wasi-size/setup_variants.py
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -mod=readonly \
  -tags disable_grpc_modules,sqlite3_dotlk -ldflags='-s -w' \
  -o .tmp/wasi-size/baseline-stripped.wasm ./cmd/server
for variant in no-sql no-cloud no-wci no-cassandra no-aws-signing combined; do
  GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -mod=readonly \
    -tags disable_grpc_modules,sqlite3_dotlk -ldflags='-s -w' \
    -overlay ".tmp/wasi-size/$variant/overlay.json" \
    -o ".tmp/wasi-size/$variant.wasm" ./cmd/server
done
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -mod=readonly \
  -tags disable_grpc_modules,sqlite3_dotlk,release -ldflags='-s -w' \
  -o .tmp/wasi-size/release.wasm ./cmd/server
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go list -mod=readonly -deps -json \
  -tags disable_grpc_modules,sqlite3_dotlk \
  -overlay .tmp/wasi-size/combined/overlay.json ./cmd/server \
  > .tmp/wasi-size/deps-combined.json
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go list -mod=readonly -deps -json \
  -tags disable_grpc_modules,sqlite3_dotlk ./cmd/server \
  > .tmp/wasi-size/packages-baseline.json
python3 .tmp/wasi-size/analyze_wasm.py temporal-server.wasm .tmp/wasi-size/*.wasm \
  > .tmp/wasi-size/analysis.txt
wasm-tools validate .tmp/wasi-size/combined.wasm
```

`wasm-tools validate` passed for the combined artifact. [analysis.json](../../../.tmp/wasi-size/analysis.json) retains exact byte counts, section payloads, function counts, SHA-256 values, and approximate code attribution. [analysis.txt](../../../.tmp/wasi-size/analysis.txt) contains the text rendering; [analyze_wasm.py](../../../.tmp/wasi-size/analyze_wasm.py) reads the WebAssembly sections. The analyzer uses [packages-baseline.json](../../../.tmp/wasi-size/packages-baseline.json), collected with the baseline `go list -deps -json`, to map function names to package paths. [deps-combined.json](../../../.tmp/wasi-size/deps-combined.json) retains the combined closure.

The current server's SHA-256 is `d406fb8f8a6cefccbff7523e751e28cf5cd8ccb676ba7582bab791d8098ecde6`. The combined artifact's SHA-256 is `6675f2375ce46b36e031618acbf8ddd37ca87b4192df8f7604880449a5fec465`.

## Dependency wiring

| Family | Current linker roots | Exclusion cost and boundary |
| --- | --- | --- |
| MySQL and PostgreSQL | [Server entrypoint](../../../cmd/server/main.go) blank-imports both plugins. Their `init` functions register implementations. PostgreSQL registers both `lib/pq` and `pgx` drivers in [plugin.go](../../../common/persistence/sql/sqlplugin/postgresql/plugin.go). | Split optional blank imports into files selected by a dedicated build tag. The slim binary loses those plugin names. SQLite registration and the shared SQL persistence implementation remain. |
| Google Cloud Storage and S3 archives | [provider_cloud.go](../../../common/archiver/provider/provider_cloud.go) imports both implementations. [provider.go](../../../common/archiver/provider/provider.go) calls their constructors for `gs` and `s3`. | An existing [gomad implementation](../../../common/archiver/provider/provider_cloud_gomad.go) preserves scheme names and returns an unavailable error after checking configuration. Reuse that boundary for an explicitly scoped profile. Filestore and injected custom archivers can remain. |
| Auto-scaled worker controller | [wci.go](../../../service/worker/wci.go) imports the external worker component module. Its [worker component](https://github.com/temporalio/temporal-auto-scaled-workers/blob/719679297cbe0199f174543487c9c14408da2823/wci/workercomponent/component.go) registers controller workflows and activities. The compute providers register constructors during initialization, including [Kubernetes](https://github.com/temporalio/temporal-auto-scaled-workers/blob/719679297cbe0199f174543487c9c14408da2823/wci/workflow/compute_provider/k8s.go), Google Cloud Run, and AWS implementations. | The existing [wci_gomad.go](../../../service/worker/wci_gomad.go) replaces this component with an empty FX option. A dedicated exclusion retains the worker service and its other components while dropping auto-scaled-worker orchestration. Cloud archive and WCI cuts share some Google/AWS dependencies, so their byte savings overlap. |
| Cassandra persistence | [DataStoreFactoryProvider](../../../common/persistence/client/fx.go) constructs Cassandra stores. [ServerOptionsProvider](../../../temporal/fx.go) invokes Cassandra schema compatibility checks. | Split both provider construction and schema verification behind a provider boundary. Reject Cassandra configuration before constructing services. [Persistence configuration](../../../common/config/persistence.go) also uses `gocql` consistency types and parsers, so removing the store alone does not prove the whole driver closure disappears. |
| Elasticsearch AWS signing | [aws.go](../../../common/persistence/visibility/store/elasticsearch/client/aws.go) links the AWS signer and default credential chain. ES client creation calls this helper. | The existing [gomad helper](../../../common/persistence/visibility/store/elasticsearch/client/aws_gomad.go) rejects enabled signing. A separate exclusion can preserve unsigned Elasticsearch access. Removing S3 archives alone leaves this AWS root. |
| Elasticsearch visibility | [Visibility factory](../../../common/persistence/visibility/factory.go) constructs ES stores. [Temporal wiring](../../../temporal/fx.go) constructs the ES client. [Frontend operator handler](../../../service/frontend/operator_handler.go) contains ES search-attribute operations. | Removing the client and visibility provider also removes ES/OpenSearch access and related operator behavior. The [schedule query evaluator](../../../service/worker/scheduler/query.go) uses ES name interception and query conversion for in-memory schedule filtering. Preserve that logic, or extract its query evaluation dependency, before excluding the entire ES implementation. |
| Metrics exporters | [MetricsHandlerFromConfig](../../../common/metrics/config.go) retains Prometheus and StatsD implementations for both Tally and OpenTelemetry. [OpenTelemetry provider](../../../common/metrics/opentelemetry_provider.go) constructs the registry, exporter, and HTTP listener. | Preserve Prometheus for the current profile. A separate profile could remove StatsD and one metrics framework while retaining the selected Prometheus implementation. Selecting a no-op handler at runtime does not remove these factory references. |
| OTLP tracing | [TraceExportModule](../../../temporal/fx.go) constructs configured and environment-selected exporters. [env.go](../../../common/telemetry/env.go) and [config.go](../../../common/telemetry/config.go) reference OTLP gRPC implementations. | A compile-time no-export profile would lose OTLP export. Keep no-op tracing APIs used by services. Shared gRPC and protobuf dependencies remain required by the server. |
| JWT authorization | [Claim mapper factory](../../../common/authorization/claim_mapper.go) selects the default JWT mapper. [Token key provider](../../../common/authorization/default_token_key_provider.go) imports JOSE and JWT support. | A custom-only authorization profile can remove these constructors while preserving interceptor interfaces. A no-auth profile changes access-control behavior and needs an explicit owner decision. TLS and other cryptographic users limit shared crypto savings. |
| SQLite and FTS5 | [Server entrypoint](../../../cmd/server/main.go) registers SQLite. [Driver adapter](../../../common/persistence/sql/sqlplugin/sqlite/driver.go) calls `fts5.Register` for every connection. | Keep both for a self-contained SQL server. The [visibility schema](../../../schema/sqlite/v3/visibility/schema.sql) creates FTS5 virtual tables and maintenance triggers. Dropping FTS5 breaks schema setup and visibility behavior. |

The `gomad` tag already changes cloud archives and ES signing, alongside [membership](../../../temporal/membership_gomad.go), [interrupt handling](../../../temporal/interrupt_gomad.go), [password commands](../../../common/config/persistence_password_gomad.go), and [worker wiring](../../../service/worker/wci_gomad.go). A binary-size profile should name its own provider exclusions so its behavior remains reviewable.

WCI exclusion leaves the lighter controller client and interface types used by [worker deployment workflows](../../../service/worker/workerdeployment/version_workflow.go). A shipped profile must also reject or disable compute configuration operations that require the omitted controller. The measured `release` variant changes the selected Go source files in only the upstream compute-provider package, removing `k8s.go` through its `!release` constraint. Its package closure falls to 1,314 packages, with cloud compute providers retained.

## Linker and SQLite limits

Go's linker starts with entrypoints and initialization tasks, follows symbol references, and prunes unreachable code. Runtime branches keep both alternatives reachable. Package registration in `init` keeps registered implementations available. A dependency remaining in `go.mod` says little about its contribution to the final executable. The linker also keeps matching interface methods conservatively. A reachable nonconstant reflective method lookup causes retention of all exported methods on reachable types. These rules appear in the [Go 1.27.0 linker source](https://github.com/golang/go/blob/go1.27.0/src/cmd/link/internal/ld/deadcode.go).

The server's `validate-dynamic-config` command uses `text/template`; its [Go 1.27.0 evaluator](https://github.com/golang/go/blob/go1.27.0/src/text/template/exec.go) performs a nonconstant `MethodByName` lookup. The [config loader](../../../common/config/loader.go), [Chasm Nexus operation handlers](../../../chasm/lib/nexusoperation/task_handler_base.go), and [history Nexus operation handlers](../../../service/history/hsm/nexusoperations/executors.go) also use templates. Removing the entrypoint's formatting command alone cannot eliminate all these roots. Broader reflection changes would require evidence for preserved template behavior and their actual size effect.

The selected [ncruces driver release](https://github.com/ncruces/go-sqlite3/blob/v0.35.6/README.md) translates SQLite WebAssembly to Go with wasm2go. Its [generated module](https://github.com/ncruces/go-sqlite3-wasm/blob/v6.3.35304/sqlite3.go) initializes an indirect-call table with SQLite callbacks. Those references retain code beyond the calls visible in Temporal's adapter. Removing unused wrapper methods or individual SQL features requires a regenerated SQLite build with explicit compile options and compatibility testing. It is a larger change than removing optional server providers.

## Candidate implementation order

1. Add an optional stripped artifact using `-ldflags='-s -w'`. The [Go linker flags](https://pkg.go.dev/cmd/link) remove symbol and debug information. This candidate has no DWARF sections; stripping removes its 10,038,076-byte WebAssembly function-name section while retaining the same code section and function count. Retain an unstripped artifact for diagnostics.
2. Define a slim provider profile that retains SQLite, FTS5, Prometheus, and frontend/history/matching/worker services. Split MySQL and PostgreSQL registration, cloud archive construction, the auto-scaled-worker controller component, Cassandra construction and verification, and ES signing at their existing boundaries. Keep the default native and WASI builds unchanged. Return clear configuration errors for omitted providers.
3. Measure a complete ES client/provider exclusion while preserving schedule query evaluation. Measure exporter and authorization exclusions separately so behavior loss can be assessed against their actual byte savings.
4. Investigate generated SQLite compile options only after the provider cuts. Preserve the SQL features used by Temporal's schemas and query converters. Check FTS5, JSON extraction, timestamp compatibility, named in-memory databases, cancellation, and WAL/locking behavior after any driver change.

Service selection via `--service` is runtime configuration. [TopLevelModule](../../../temporal/fx.go) still references constructors for every service, so a history-only CLI invocation cannot shrink the artifact. Compiling out the worker, history, matching, or frontend graph changes server capabilities and requires a separate target with explicit coverage.

## Acceptance boundary

This research changes a report and its index. The measurement experiment uses ignored overlay files and compiled artifacts; it changes no production source. Compilation demonstrates linker reachability and artifact size. It establishes no WASI startup, socket availability, database execution, functional-test pass, or determinism bound.

A production slim profile needs native coverage for retained providers and configuration rejection, compilation for WASI, and the repository's changed-package lint gate. Guest execution must subsequently verify SQLite schema setup and transactions, visibility, metrics behavior, and server networking. The earlier native ncruces tests recorded in the [compilation report](2026-10-09-wasi-compilation.md) do not qualify a newly excluded or regenerated provider graph.
