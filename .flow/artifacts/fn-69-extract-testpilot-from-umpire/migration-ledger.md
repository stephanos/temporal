# Testpilot migration ledger

This ledger freezes the pre-migration Case Runtime at commit
`5b53d57254993533218da99865b4c60f278cd056` (tree
`53daad263c7e207da1d5099d92c455ee1d8207ae`). It is the task-1 authorization
boundary for the owner and namespace moves in tasks 2 through 8. The older fn-64
ledger and the fn-66 audit artifacts remain historical records and are not inputs
to be rewritten during this migration.

Every selector below is closed over this baseline. A selector assigns all files
under it to one final owner and migration task; more-specific rows take precedence.
There are no unassigned or ambiguous rows in the active Case protocol/runtime
surface.

## Closed ownership inventory

| Current surface | Baseline contents | Final owner | Migration task |
| --- | ---: | --- | --- |
| `proto/internal/temporal/server/api/umpire/v1/*.proto` | 5 protocol sources | `proto/internal/temporal/server/api/testpilot/v1` | 2 creates the Testpilot sources; 8 removes the old sources |
| `api/umpire/v1/{case,contract,program,run,value}{.pb.go,.go-helpers.pb.go}` | 10 generated Go files | `api/testpilot/v1` | 2 generates the Testpilot package; 8 removes the old package |
| `tools/umpire/internal/ir/**` | 18 production/test Go files | `common/testing/testpilot/internal/ir` | 3 moves the complete corpus; 8 removes the old owner |
| `tools/umpire/internal/execution/**` | 22 production/test Go files plus README | `common/testing/testpilot/internal/execution` | 3 moves the complete corpus; 8 removes the old owner |
| `tools/umpire/verification/**` | 7 production/test Go files plus README | `common/testing/testpilot/internal/verification` | 3 moves the complete corpus; 8 removes the old owner |
| `tools/umpire/{prepare,prepared_case,profile,host}.go` and their tests, `conformance_test.go`, `carrier_test.go` | 4 production and 5 test files | `common/testing/testpilot` | 4 publishes the facade; 7 moves conformance; 8 removes the old facade |
| `tools/umpire/caseartifact/**` | 1 production Go file | top-level `common/testing/testpilot` Case ingestion | 4 introduces the public ingestion API; 7 migrates callers and removes this package |
| `tools/umpire/temporal/**` | 18 production Go files, 14 test files, 4 READMEs, 2 fixtures | `tests/testcore/testpilot` | 5 prepares destination fixtures; 6 moves the complete adapter and deletes this owner |
| `tools/umpire/cmd/umpire-gen-lean-api/**` | 9 production and 7 test Go files | Umpire Producer tooling, still under `tools/umpire/cmd` | 5 retargets generated identities; 7 migrates remaining Case consumers |
| `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**` | 3 production and 1 test Go files | Umpire Producer tooling, still under `tools/umpire/cmd` | 5 adds the functional mode; 7 retargets generic conformance ownership |
| `model/Temporal/{API.lean,API/**,CaseRuntime.lean,CaseRuntimeTests.lean,Tool/CaseRuntime.lean}` plus the aggregate imports in `Temporal.lean` and `TemporalModelTests.lean` | generated API, embedded protocol identity, Producer renderer, and build roots | Umpire authoring/Producer with Testpilot protocol identity | 5 performs the generated identity cutover; 7 reconciles remaining consumers |
| `tools/umpire/testdata/case-runtime-conformance/**` | 12 JSON files in 6 behavioral classes | `common/testing/testpilot/testdata/case-runtime-conformance` | 7 regenerates and moves the generic corpus |
| `tools/umpire/temporal/testdata/**` | 2 JSON files | `tests/testcore/testpilot/testdata` | 5 generates the Testpilot copies; 6 removes the old copies with the adapter |
| `tests/umpire4_async_nexus_case_test.go` | 1 live success test | active Testpilot live consumer in `tests` | 6 retargets and renames the consumer/selector |
| `Makefile` targets and variables named below | generation, conformance, model, focused, live, and aggregate commands | existing Makefile with Testpilot paths/selectors | 6 updates adapter/live selection; 7 updates generation/conformance; 8 closes all selectors |
| `.github/workflows/umpire.yml` | package-local and live proof selectors | existing workflow with Testpilot package/test selection | 6 updates moved package/test coverage; 8 closes naming and coverage |
| active documents named below | runtime, facade, adapter, and architecture ownership text | Testpilot docs plus retained Umpire Producer docs | 4 adds Testpilot facade docs; 6 moves adapter docs; 8 reconciles active architecture docs |
| `tools/umpire/internal/legacyvocabulary/check.go`, `tools/umpire/regression/**`, and `tools/umpire/vocabulary/legacy_gate_test.go` | active ownership, documentation, selector, and vocabulary enforcement | retained regression enforcement for the final Testpilot boundary | 6 updates live selection; 8 replaces temporary Umpire ownership checks |

The private-core selectors contain 24 production files and 21 test files. The
facade selector contains 4 production files and 5 test files. The adapter selector
contains 18 production files and 14 test files. The two generator selectors contain
12 production files and 8 test files. These counts are part of the closure check;
moving fewer files is a coverage loss.

## Protocol and generated-file manifest

All five protocol sources preserve their comments, declaration order, message and
enum shapes, field cardinalities, defaults, and numbers:

| Source | Messages | Enums | Fields | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| `proto/internal/temporal/server/api/umpire/v1/case.proto` | 4 | 2 | 17 | `e01dcc57268d21fc40067dfac0be5b2474878e0f95ac369b6d6bc5a3e72a4308` |
| `proto/internal/temporal/server/api/umpire/v1/contract.proto` | 9 | 3 | 33 | `fc5850da24264fc13747cdf68dd335c8a35892a96f9e6d6e3cf9ea3f64999a05` |
| `proto/internal/temporal/server/api/umpire/v1/program.proto` | 25 | 5 | 77 | `12fcc933a0410d9e9cacb673a2a7d1941b1f47faac1874298eaff69a7ff5aaf3` |
| `proto/internal/temporal/server/api/umpire/v1/run.proto` | 10 | 6 | 41 | `01e91b4e08fad850315b4b46ca99f15c844045c3656725f6b391d2f32da22168` |
| `proto/internal/temporal/server/api/umpire/v1/value.proto` | 40 | 5 | 48 | `711b36d101d7daecc184a78a20887922f166d652b7e9b80a8a6ca39938921e94` |

The structural baseline is therefore 88 messages, 21 enums, and 216 declared
fields. Task 2 must compare descriptor structures, rather than only these counts,
and must reject a changed number, kind, cardinality, default, oneof membership,
dependency edge, or missing descriptor. The complete pre-migration internal
descriptor image has SHA-256
`606efa66918490fc1aa77370cfb60fd5b800ece807017911de641306eb027636`.

| Generated file | SHA-256 |
| --- | --- |
| `api/umpire/v1/case.go-helpers.pb.go` | `27ee188b5ab05141bffad96993e179a93d72b9a41760f3e0f78ebf3ed352881e` |
| `api/umpire/v1/case.pb.go` | `613d7c019a87480a47980b7ff64097ecee1c10218ce6468153fb0c6ea2b14c18` |
| `api/umpire/v1/contract.go-helpers.pb.go` | `17d461ccdb2ebfc71eb397f38d8afd8d686830cb06f9d53d600c16a92f283eac` |
| `api/umpire/v1/contract.pb.go` | `152462fe9e862a9044680555efcc649236b967e03ffb320128142ffb3972ede4` |
| `api/umpire/v1/program.go-helpers.pb.go` | `4e9adb4b804b64f47dc01ed7bc296461c6ac0e694e821464634971ca010d91a2` |
| `api/umpire/v1/program.pb.go` | `4a60be8154d576af842ce3af348c35fdf082b0bda5585ff1e11c4dcd91fba69e` |
| `api/umpire/v1/run.go-helpers.pb.go` | `cfc4d145f2128531eda42d18c0d7e3bea4c50f888072046dcf29119af2cda00c` |
| `api/umpire/v1/run.pb.go` | `22a8c97cad3918349c9bc32503cbf16db896b46cf9696a9e45dbf92eba1a3818` |
| `api/umpire/v1/value.go-helpers.pb.go` | `1b4a8cbce6fabaac260d5ef0ad037db0cc216483ad1c27fa43b72445b3517134` |
| `api/umpire/v1/value.pb.go` | `7f19028f5b1bc31c2d751dad1b6da8867e7ca65c7fabfbf2affef903b7c02545` |

Task 2 added the five structurally equivalent Testpilot sources. Their hashes
after the approved path, protobuf namespace, and Go package substitutions are:

| Source | SHA-256 |
| --- | --- |
| `proto/internal/temporal/server/api/testpilot/v1/case.proto` | `84c1a9acf9eac70acac9f9c4feee83c7b3950e05111df2cb09ae2c4cd5c16004` |
| `proto/internal/temporal/server/api/testpilot/v1/contract.proto` | `87651873e8dea2c21f97660eeb7b91dafa1788cba95d7e7f9bb484dbbf8818da` |
| `proto/internal/temporal/server/api/testpilot/v1/program.proto` | `cc1469e09f5982772ee098d49f9056785e4ab6f0da11b3a13e9d74b999b5ed00` |
| `proto/internal/temporal/server/api/testpilot/v1/run.proto` | `1bab37c67ffba705e99beb552a0d277dd846deeadc884363f132b5bc96811891` |
| `proto/internal/temporal/server/api/testpilot/v1/value.proto` | `7e63a40a867f8b606f832d27996ab3229502759ecad8da3fb030d90cfb6f3297` |

The generated-diff allowlist for task 2 is exactly the ten files below plus
`proto/image.bin`. No existing generated file is allowed to change.

| Generated file | SHA-256 |
| --- | --- |
| `api/testpilot/v1/case.go-helpers.pb.go` | `409749e4cae3c9b569d2542a3e55f662a41d7695943cf12b3a184e9f2e95da58` |
| `api/testpilot/v1/case.pb.go` | `c05de853e72481445ecee305c483883ddeafd8e94e8e59f520e49b336a6d8171` |
| `api/testpilot/v1/contract.go-helpers.pb.go` | `d4264dc8f3b686fd079091851314adc65049064a0eb0a041866a21b68065a9e3` |
| `api/testpilot/v1/contract.pb.go` | `548eb6cea8e5eb9fbad594c136c0cebeeec0134b81eb0817f739d53549362b0b` |
| `api/testpilot/v1/program.go-helpers.pb.go` | `0c9d86568e7a0a4ab026dbb652cc6640228cad46537cc435b797f358221f1a09` |
| `api/testpilot/v1/program.pb.go` | `36abae810a66689567999d1d4e959e4f668c58c048441f96cc92314b115d76f0` |
| `api/testpilot/v1/run.go-helpers.pb.go` | `617901440f96a8fffd4af9ba7c5bcb5dfe86d2aeac1b9aeef69be67dd622c317` |
| `api/testpilot/v1/run.pb.go` | `c40e77a2c6dc659f9043ea5286459c3450bb29cc527e65ce3cbcab5556413b17` |
| `api/testpilot/v1/value.go-helpers.pb.go` | `51c24701bddcfbd4fd72df37fabc7bca8cd9745e33bc42f0fa5678a9f01124ac` |
| `api/testpilot/v1/value.pb.go` | `ddd544d968089a43ee2cb797d55f04f7b8d71df6bbdd8e691f497107b1bc8ac7` |
| `proto/image.bin` | `ac9e26117f1662b83926e70725c6c4f31bb6b3650bf1acfd0316f73067029cf9` |

The Umpire proto source and generated Go package remain temporary migration
inputs while consumers move in tasks 3 through 7. Task 7 owns the last consumer
reconciliation, and task 8 removes the old source and generated package. There
is no alias, converter, registry, fallback, or active consumer migration between
the coexisting packages.

## Compatibility classes

The migration compares these classes separately. A permitted identity change in
one class does not authorize normalization or drift in another.

| Class | Frozen behavior | Permitted substitution |
| --- | --- | --- |
| Descriptor shape and numbering | exact messages, enums, fields, numbers, cardinalities, oneofs, defaults, and dependency graph | file path `temporal/server/api/umpire/v1/*.proto` to `temporal/server/api/testpilot/v1/*.proto`; package/full-name prefix `temporal.server.api.umpire.v1` to `temporal.server.api.testpilot.v1`; Go package/import `api/umpire/v1;umpire` to `api/testpilot/v1;testpilot` |
| Ordinary protobuf wire values | exact tags, values, order, presence, and deterministic encoding for values that do not carry a descriptor name | none |
| Namespace-bearing protobuf values | every `Any` type URL, `NamedType.protobuf_type`, descriptor name, and identity derived from one | only the same Umpire-to-Testpilot prefix substitution; no blanket string normalization |
| Generated identities | five source paths, five Go raw descriptor variables, Go package/import identifiers, registration names, and protobuf full names | only substitutions implied by the preceding row and canonical generator output |
| Public API | `PrepareCase(case, Profile) -> *PreparedCase`, `PreparedCase.Run(ctx, Host)`, immutable snapshots, Profile/Catalog and prepared-plan views | `PrepareCase` becomes `Prepare`; `Host`/`HostIdentity` become `Driver`/`DriverIdentity`; `PreparedCase` and all message/result shapes remain |
| Public errors and failure precedence | exact categories, paths, detail text, nil/typed-nil timing, no target I/O during preparation/preflight, cleanup and committed-violation precedence | terminology substitutions from `Host` to `Driver` only where the renamed public seam requires them |
| Runtime behavior | authorized effects, reservations, events, diagnostics, cancellation, independent concurrent Runs, cleanup/quarantine, Run/Verdict values, and bounds | none |
| Corpus ownership | six generic conformance classes and two functional Cases with exact behavior | path/owner move plus the explicitly named descriptor-derived substitutions |

The exact current public facade documentation hash is
`cea09a81d1af411e2ba8c6e63b26980913a328d71a11f5720cd518878d64b444`.
Its directly authored public errors are:

- `Profile is required`
- `Profile catalog is required`
- `prepared Case, context, Host and Contract factory are required`
- `Host Profile or catalog identity changed`
- `Host returned no session`

All propagated IR/execution/verification error-category, path, and detail call
sites have sorted-manifest SHA-256
`53c5d5265cb090e034f61c39563534509a0dc6be2ea5fc180ab7da1cd2308422`.
Tasks 3 and 4 compare the extracted source to this manifest after only the allowed
package-path and Host-to-Driver terminology substitutions. In particular,
`execution.Run` retains open/preflight errors, fresh bounded cleanup after a
started Session, termination before cleanup, cleanup/close diagnostics, committed
violation precedence, and its existing returned-error behavior.

The complete runtime source/test/README manifest hash is
`47aa963be4311be1d1dfa359d83f88c31907e8e6907764e0b0af3cee0ede0594`;
the facade manifest hash is
`3a96834789b64fed6f756f0b04ae8c91c16e6977c13f63ae324fa8b689574c59`;
the functional adapter manifest hash is
`b40547173af04e29d1ac46a046af10901b599bc720aa866942232fe72dea68da`;
and the two generator Go-source manifests hash to
`815c57fa43e127c619dc3c9f13a7d308c6b1ee053d9598684fde5449220e9099`.

## Task 3 private-core extraction

Task 3 duplicated the complete 47-file private-core selector under
`common/testing/testpilot/internal` while retaining every old-owner file for
task 8. Relative paths map one-to-one from `tools/umpire/internal/{ir,execution}`
and `tools/umpire/verification` to the corresponding Testpilot package; no file,
test entry point, or README is omitted:

| Testpilot package | Production Go | Test Go | README | Test/Fuzz entry points | Destination manifest SHA-256 |
| --- | ---: | ---: | ---: | ---: | --- |
| `internal/ir` | 10 | 8 | 0 | 35 | `e6f2458ee04674b5dd18b29b18d934da3a9e71fd85e126f0513eb28b42d836a9` |
| `internal/execution` | 11 | 10 | 1 | 67 | `248dabc7bef861bce562576d6fa80232ce44aef83acb9cab4d20065487293186` |
| `internal/verification` | 3 | 3 | 1 | 22 | `d6b2556d71c4e3923df4c083fad983776492be95f2e5b9bd398ade08a1f1ff9e` |

The combined destination manifest SHA-256 is
`739100ff4524ba208f7747e117332bfeed66aa4447ad1e621b4c1dcfcd6bc29e`.
After the allowlisted Go import, protobuf alias, and
`temporal.server.api.testpilot.v1` full-name substitutions and `gofmt`, all 46
files outside `execution/dependencies_test.go` compare byte-for-byte with their
old-owner source. The boundary test is the sole intentional corpus extension:
it applies the transitive dependency check to IR, execution, and verification
and rejects Umpire tooling, server test adapters, and the Temporal SDK. Go's
`internal` compiler boundary additionally rejects cross-root internal imports.
The old and new cores have no imports between them, converters, aliases,
registries, fallbacks, or shared mutable state.

## Task 9 Testpilot descriptor refinement

Task 9 replaces the temporary task-2 structural clone before any public
Testpilot adoption. The refined schema intentionally breaks source and wire
compatibility with that clone while preserving the Case Runtime behavior
frozen above. The extracted core consumes the refined messages directly; no
alias, converter, compatibility registry, fallback, or translation layer was
introduced.

| Temporary task-2 surface | Refined Testpilot surface | Intentional incompatibility |
| --- | --- | --- |
| five broad source files | eight cohesive `case`, `value`, `expression`, `instruction`, `program`, `outcome`, `run`, and `contract` files | descriptor file identities and dependency edges change |
| `CaseMetadata`, `CaseDefinitionKind`, bindings, known gaps, and source locations | `CaseProvenance` with producer identity, version, and opaque producer bytes | producer-only authoring taxonomy is removed from the runtime contract |
| shared `ValueExpression` | separate `ProgramExpression` and `ContractExpression` trees | contracts cannot reference program slots or instruction outcomes; programs cannot reference observations, captures, or run events |
| `SlotSchema.kind` beside independent type fields | `SlotDefinition.content` oneof containing a value type or opaque capability | contradictory slot classifications are structurally unrepresentable |
| `EntrypointContext` plus an activation binding | direct activation oneof on `EntrypointDefinition`; cleanup has no activation | contradictory context/activation pairs and non-controller cleanup contexts are structurally unrepresentable |
| authored nodes named `*Schema`, runtime values named `*Disposition`, references named `*Reference` | authored `*Definition`, runtime `*Outcome`/`*Result`, references `*Ref` | public source names change to encode lifecycle and ownership |
| categorical enums mixed with `*Type` and `*Kinds` names | categorical enums use `*Kind`; `RunEventFilter` names the set-valued filter | public enum source names change without changing their runtime meaning |
| lifecycle/result enums named for dispositions and outcomes | `RunStatus`, `CleanupStatus`, `RuleVerdictStatus`, and `VerdictStatus` | public enum source names change; existing numeric values remain fixed |
| wrapper messages used only to model scalar presence | scalar oneofs on diagnostic support IDs and evaluation-failure instruction IDs | presence remains explicit with a smaller runtime schema |
| Host terminology in Testpilot diagnostics and private-core contracts | Driver terminology | public diagnostic vocabulary changes from the Producer-era seam name |

`protocol_compatibility_test.go` records these incompatibilities as descriptor
invariants. It also compares the refined runtime enum numbers to the frozen
Umpire descriptors, verifies the separated expression scopes, proves the slot
and activation oneofs, checks scalar presence, and rejects the retired broad
files and public names.

The final refined protocol source hashes are:

| Source | SHA-256 |
| --- | --- |
| `case.proto` | `f7a88f9c903c942f456e489b22a42f28077988d81ba7bfc411d5f4ab21ae3eeb` |
| `contract.proto` | `58f8fdfaf829f712c5ceb72d9adf479b584f0b8c84c6f8da0cc91f9737adf600` |
| `expression.proto` | `587a596e3b7c0e544f8746387e9b96e9990ff69390738fbf73579a78fde1a421` |
| `instruction.proto` | `ded60429f60c5cfb299792df88679af08044249763a3d823da8fa3d3507ca54a` |
| `outcome.proto` | `4c906f64f642dd82e9dcc2ebea7f3aa60efced635569b2179a268beb38b986d2` |
| `program.proto` | `e5eff9c74df83344088ab55f11276742eea6318199a3d71a402c27f443e7187a` |
| `run.proto` | `ae6303443d011579a9bb7494c1db1c4784932ceeecec2a40d220d2f20ffd8a83` |
| `value.proto` | `8b100f439dd8577a3ffa8591b58018c806595e3d6cf6a0b55849a799c7a028b4` |

The sorted protocol-source manifest SHA-256 is
`d0d6f94213fd7c5643ff37a639e6de2be2c36b8906c1776fabc03274a2fd4f38`.
The 16 canonical generated Go files have sorted-manifest SHA-256
`4490d626789dd064ebacc49ef8705e7f085a751c42d7280898f5b52c6cbe57c3`.

## Task 4 public facade

Task 4 publishes the root `common/testing/testpilot` boundary through
`Prepare`, immutable `PreparedCase`, `PreparedCase.Run`, `Profile`/`Catalog`,
`Driver`/`Session`, and the prepared Program views required by environment
adapters. Public value references and expression views wrap the private IR;
the API exposes no internal IR, scheduler, recorder, monitor factory, or
evaluator type. The public Driver adapter converts owned effects,
reservations, capability bridges, and coordinates without adding an effect,
retry, observation, translation, or fallback path.

Strict `DecodeCaseProtoJSON` and deterministic `PackCaseProtoJSON` now belong
to the root Testpilot package. An external-package non-functional Driver uses
only Testpilot and the Testpilot protobuf API to prove bounded success, Driver
failure, cleanup failure, and package dependency direction. The task-7
conformance corpus remains at its temporary Umpire path until its Producer and
fixture identities migrate in task 5.

The ten top-level files have sorted manifest SHA-256
`4fdf73ac28884246c04cec5acc536f693417a494359c0078e1abb460980edab0`.
The top-level package owns five production Go files, three task-9 descriptor
tests, and nine facade/ingestion tests. The old facade and caseartifact package
remain temporary task-8 owners for existing Umpire consumers; there is no
import, alias, converter, registry, or fallback between the two facades.

## Active consumers

The exact-import/full-name search at the baseline assigns every active consumer
outside the source/generated protocol and the owner groups above:

| Consumer | Final disposition | Task |
| --- | --- | --- |
| `model/Temporal/CaseRuntime.lean` | retain Umpire Producer semantics; substitute the embedded Testpilot full name | 5 |
| `model/Temporal/API.lean`, `model/Temporal/API/**` | regenerate the API projection with Testpilot descriptors | 5 |
| `model/Temporal.lean`, `model/Temporal/CaseRuntimeTests.lean`, `model/Temporal/Tool/CaseRuntime.lean`, `model/TemporalModelTests.lean` | retain as Umpire authoring/Producer and aggregate build roots; retarget only protocol identities/imports | 5 and 7 |
| `tests/umpire4_async_nexus_case_test.go` | use the testcore Testpilot Driver and Testpilot protocol/facade | 6 |
| `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go` | compare/generate Testpilot descriptor identities without weakening structural checks | 2, 5, and 7 |
| `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate_test.go` | retain Umpire-owned generator tests; publish final Testpilot fixture roots | 5 and 7 |
| `tools/umpire/conformance_test.go` | move generic public conformance to Testpilot | 7 |
| `tools/umpire/carrier_test.go` | move core assertions or rewrite through supported Testpilot public views, then remove old internal reach-through | 3, 4, and 8 |
| `tools/umpire/host_external_test.go` | become the external-package Driver proof using only Testpilot public APIs | 4 |
| `tools/umpire/caseartifact/case.go` | fold decode/pack behavior into the Testpilot facade and remove after callers migrate | 4 and 7 |

The exact protocol reference search additionally finds only files inside the
classified core, facade, adapter, generator, proto, generated, and fixture groups.
Historical `.flow` records and the immutable fn-64/fn-66 artifacts are evidence,
not active consumers, and remain unchanged.

## Fixture hashes and namespace occurrences

The combined sorted SHA-256 manifest for all 14 JSON files below is
`5befaa8e65018873d56404587da7945be63001009329a89671574958a5e6ca72`.
The six conformance classes remain exactly: `satisfied`, `violated`,
`inconclusive`, `static-preparation-rejection`, `cleanup-failure-after-proved-violation`,
and `cross-run-isolation`.

| Fixture | SHA-256 |
| --- | --- |
| `tools/umpire/temporal/testdata/async-nexus-case.json` | `63514cf0139ab503f467511d45f875b3164541c0d71d48079a6949aa4f2cdaf6` |
| `tools/umpire/temporal/testdata/get-system-info-case.json` | `a5c6ec475fe69d07843573e5e12c1b5109ec13dca8d72c18943d6ea4e1e3d633` |
| `tools/umpire/testdata/case-runtime-conformance/cleanup-failure-after-proved-violation/case.json` | `95696956bb0409bb84a6c9687592d52b62b5bb64e41cf5f3ba1e087a4feb72b1` |
| `tools/umpire/testdata/case-runtime-conformance/cleanup-failure-after-proved-violation/expected.json` | `ff9eb63cb63b2624adde241704f066d1eb71d4c2e881235bff6c1e8853f8d03d` |
| `tools/umpire/testdata/case-runtime-conformance/cross-run-isolation/case.json` | `ae3b6b3d015cb784e9fd938ba7eb56567d58082dbc229d89def9256e20b88e86` |
| `tools/umpire/testdata/case-runtime-conformance/cross-run-isolation/expected.json` | `d36e171e907672fa3f067443f387a469e0c93a6395c9e6d621651b3e49562463` |
| `tools/umpire/testdata/case-runtime-conformance/inconclusive/case.json` | `32173679143d4cba920c448b4ae865b31e96973513954aacf045c77452162693` |
| `tools/umpire/testdata/case-runtime-conformance/inconclusive/expected.json` | `75a10bdc8c73fd22df4be259617fb5cbe0068f5aae354f37b239ac670ba0e5af` |
| `tools/umpire/testdata/case-runtime-conformance/satisfied/case.json` | `8c8be38a483440e8958ca0a85282544aa0da61241843c1888f9bce3454ded8ea` |
| `tools/umpire/testdata/case-runtime-conformance/satisfied/expected.json` | `d114c1df2d69ba56bad27b78b7aa0a305bb73c73af4747026e1bbe6084c77101` |
| `tools/umpire/testdata/case-runtime-conformance/static-preparation-rejection/case.json` | `5bb8f03f12fc081b84d4b4de91ee36c97283757657b1361fd9b210b0c2a27d7f` |
| `tools/umpire/testdata/case-runtime-conformance/static-preparation-rejection/expected.json` | `210fe392c9766f7c042e60964e14651c34a2251d79cae39ff95a6cbfbe31bc0d` |
| `tools/umpire/testdata/case-runtime-conformance/violated/case.json` | `5a39f7ccc1f083e80342efddd18a0a8910e0a90ced1f6b73f36fa726de01413b` |
| `tools/umpire/testdata/case-runtime-conformance/violated/expected.json` | `26eeee9aebca0f702af9b9b7f4e3de7147320314a9d51c8c520b76319a4c4d31` |

There are exactly 16 embedded
`temporal.server.api.umpire.v1.InstructionOutcomeStatus` `NamedType` values:
8 in `async-nexus-case.json`, 1 in `get-system-info-case.json`, and 7 across the
six conformance Case files. There are zero Umpire `type.googleapis.com` URLs in
this corpus at baseline. Tasks 5 through 7 may change those 16 values only by
the exact prefix substitution to
`temporal.server.api.testpilot.v1.InstructionOutcomeStatus`; all other JSON bytes
and all expected Run/Verdict files remain exact unless canonical JSON rendering
changes only the same named value.

## Tests, packages, commands, and selectors

The baseline has 19 Go packages under `tools/umpire/...`. The private core owns
124 top-level Go Test/Fuzz entry points, the facade owns 11, the functional
adapter owns 83, and the two Case generators own 37. The broader active
`tools/umpire` plus live-test scan has 335 Test/Fuzz entry points. Final counts
must account for moves and intentional renames without reducing behavioral
coverage.

| Command at the pre-edit baseline | Exit | Result |
| --- | ---: | --- |
| `go test -count=1 -tags test_dep ./common/testing/testpilot/...` | 1 | inherited/pre-migration: destination does not exist |
| `go test -count=1 -tags test_dep ./tests/testcore/testpilot/...` | 1 | inherited/pre-migration: destination does not exist |
| `make proto` | 0 | generated successfully with no new working-tree drift |
| `make umpire-gen-lean-api` | 0 | generated successfully |
| `make umpire-check-case-runtime-conformance` | 0 | 45 Lean jobs, generator package and public facade conformance passed |
| `cd model && mise exec -- lake build Temporal.CaseRuntimeTests` | 0 | 22 jobs built successfully |
| `TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tools/umpire/cmd/umpire-gen-lean-api` | 0 | both focused generator packages passed |
| `TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/...` | 1 | inherited local tooling failure: `runtime/cgo` cannot find `stddef.h`; delivery and worker packages pass |
| `make umpire-check-regression` | 0 | aggregate passed; 324 Lean jobs built and all Go/regression prerequisites completed |

The aggregate live selector observed exactly these inherited raw failures and
then exited 0 because they match the repository policy:

- `TestUmpire2TestSuite`
- `TestUmpire2TestSuite/TestPlanAndDriveKitchenSinkNexusOperation`
- `TestUmpire2TestSuite/TestPlanAndDriveNexusOperationCHASM`
- `TestUmpire2TestSuite/TestProbeNexusDegraded`
- `TestUmpire2TestSuite/TestProbeNexusExploration`
- `TestUmpire2TestSuite/TestProbeNexusFlagged`
- `TestUmpire2TestSuite/TestProbeNexusRandomized`
- `TestUmpire2TestSuite/TestProbeNexusResilience`
- `TestUmpire3ParticipantProcessCrashAndRestartResumesRealSDKProgram`

The command/selector inventory is:

- `UMPIRE_GEN_LEAN_API_COMMAND`, `umpire-gen-lean-api`, and its fixture target:
  retained Umpire Producer commands; tasks 5 and 7 retarget descriptor identities.
- `UMPIRE_GEN_CASE_RUNTIME_CONFORMANCE_COMMAND`,
  `umpire-gen-case-runtime-conformance`, and
  `umpire-check-case-runtime-conformance`: retained Umpire Producer commands;
  tasks 5 and 7 add the functional mode and move output/check roots.
- `umpire-build-model`: retained Umpire model command; tasks 5, 7, and 8 keep it
  selected after generated identity changes.
- `umpire-check-live-tests` and `-run '^TestUmpire'`: task 6 adds the renamed
  Testpilot success identity without dropping the exact Umpire2/Umpire3 inherited set.
- `umpire-check-regression`, its `./tools/umpire/...` package selection, and
  `.github/workflows/umpire.yml` package/live steps: tasks 6 through 8 replace
  moved paths and retain all core, adapter, generator, regression, and live coverage.
- `umpire-check-legacy-vocabulary`: task 8 replaces old-owner allowances with
  Testpilot ownership and zero-active-reference enforcement.

## Active documentation

These are the active documentation references found by the Case Runtime,
facade/path, and namespace search. Each has one final owner:

| Document | Final disposition | Task |
| --- | --- | --- |
| `tools/umpire/internal/execution/README.md` | move with the private execution core to Testpilot | 3 |
| `tools/umpire/verification/README.md` | move with private verification to Testpilot | 3 |
| `tools/umpire/temporal/README.md`, `tools/umpire/temporal/server/README.md`, `tools/umpire/temporal/worker/README.md` | move and rewrite ownership with the functional testcore Driver | 6 |
| `tools/umpire/CONTEXT.md`, `tools/umpire/CLEANUP_INVENTORY.md` | retain Umpire authoring/Producer scope and remove former runtime ownership claims | 8 |
| `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md` | retain Umpire authoring/Producer semantics and point execution/protocol ownership to Testpilot | 8 |
| `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md` and active `.plans/UMPIRE4_*.md` | preserve historical rationale/comments while updating active owner/path statements | 8 |
| downstream active fn-22, fn-26, fn-29, fn-33, and fn-68 specs/tasks | retain their separate replay, qualification, canary, exploration, and Nexus3 ownership; update only the Testpilot dependency/interface | 8 |

Historical Flow specs, task receipts, memory, and the fn-64/fn-66 artifacts are
excluded from active-name deletion checks and remain immutable evidence.

## Closure gates

- [x] Five proto sources and ten generated files are individually hashed and assigned.
- [x] Every private core, facade, adapter, generator, importer, fixture, test, command, selector, and active-document surface has one destination and task.
- [x] Descriptor structure, ordinary wire bytes, namespace-bearing values, generated identities, public API/error behavior, runtime behavior, package/test counts, corpus hashes, and live identities are frozen as separate compatibility classes.
- [x] Existing focused model/generator/conformance results and the aggregate result are recorded; inherited missing-destination, C-header, and exact live-suite failures are distinguished from code failures.
- [x] The fn-64/fn-66 artifacts and retained Umpire authoring/generator owners are outside implementation changes.
- [x] No runtime, protocol, scanner, drift gate, alias, converter, registry, or fallback is introduced by task 1.

Task 2 may begin only against this zero-ambiguity inventory. Every later task
updates this ledger with actual source/destination hashes and reconciles its
assigned rows before task 8 deletes the old owners.

## Task 5 Lean/API identity and functional fixture cutover

Task 5 regenerated the Lean descriptor projection after task 9 refined the Testpilot schema. The
generated projection remains structural: recursive protobuf fields are represented by opaque
`Temporal.API.Proto.MessageRef` values and therefore cannot carry executable Case values. The
dedicated `Temporal.CaseRuntime.TestpilotProtoJSON` producer path instead lowers the typed Umpire
Case model directly into the refined Testpilot ProtoJSON shape. Only the existing
`get-system-info` and `async-nexus` renderer entries use this path; all six conformance entries
retain the old Umpire serializer until task 7 migrates their owner and corpus.

The functional lowering preserves Program and Contract meaning while applying task 9's recorded
representation changes: metadata becomes opaque provenance bytes, Program and Contract expression
trees use their separate legal vocabularies, slots and entrypoint activations use structural
oneofs, and authored/runtime fields use the refined Definition, Ref, Kind, Status, and Result
names. Illegal cross-scope expressions fail rendering. This is Producer-side Testpilot generation,
not an old-protobuf decoder, runtime converter, alias, registry, or fallback.

The generator's explicit `--mode functional` transaction owns exactly
`tests/testcore/testpilot/testdata`, validates both rendered artifacts through
`testpilot.DecodeCaseProtoJSON` and `testpilot.PackCaseProtoJSON`, and replaces stale files as one
complete set. Its default conformance mode, exactly-six manifest, and old fixture root are
unchanged.

| Task-5 output | SHA-256 | Embedded outcome identity count |
| --- | --- | ---: |
| `tests/testcore/testpilot/testdata/async-nexus-case.json` | `c6a14f899c7ff7d34f352b8a3f4d1eeb2bc40262f462a3fa8d9358d0ce10371e` | 8 |
| `tests/testcore/testpilot/testdata/get-system-info-case.json` | `c2eb7a01c8623455cda0db10a4b2ed95e7db9f4c5e73a14ec337f6ae532e1339` | 1 |

All nine functional identities are exactly
`temporal.server.api.testpilot.v1.InstructionOutcomeStatus`; no Umpire outcome identity remains in
the new root. The still-active adapter fixtures remain byte-identical at their task-1 hashes:

| Retained Umpire adapter fixture | SHA-256 |
| --- | --- |
| `tools/umpire/temporal/testdata/async-nexus-case.json` | `63514cf0139ab503f467511d45f875b3164541c0d71d48079a6949aa4f2cdaf6` |
| `tools/umpire/temporal/testdata/get-system-info-case.json` | `a5c6ec475fe69d07843573e5e12c1b5109ec13dca8d72c18943d6ea4e1e3d633` |

The task-5 generated and authored output hashes are:

| Output | SHA-256 |
| --- | --- |
| `model/Temporal/API.lean` | `a9ba87d3b42d85a53a811d7581f381de128625b77ff5c8ee521814ca9828b812` |
| `model/Temporal/API/Types.lean` | `003501b39b6e83f84097dd32af46ebe81edffb317a570c7c54312eb396e210f2` |
| `model/Temporal/CaseRuntime.lean` | `d4c40aa50af3322842e117927e509c1c926bf2e2a30579fbcb450ba7d5d0e852` |
| `model/Temporal/CaseRuntime/TestpilotProtoJSON.lean` | `088c046507318a5c13f526e6d44558fed6f78b029b8fab311520e020a3eac11a` |
| `model/Temporal/Tool/CaseRuntime.lean` | `303409440a5c51f5ec3ad37f5de84f948f57eb0084bd9ffdb6c07d18c2374f6e` |

## Task 6 functional Driver move

Task 6 moved the complete functional Temporal Driver from `tools/umpire/temporal` to
`tests/testcore/testpilot`: 18 production Go files, 14 Go test files, and the three package
READMEs now live under the functional test-core owner. The former adapter directory and both old
fixture copies are absent. The task-5 Testpilot fixtures remain byte-exact at their recorded hashes:

| Retained Testpilot fixture | SHA-256 |
| --- | --- |
| `tests/testcore/testpilot/testdata/async-nexus-case.json` | `c6a14f899c7ff7d34f352b8a3f4d1eeb2bc40262f462a3fa8d9358d0ce10371e` |
| `tests/testcore/testpilot/testdata/get-system-info-case.json` | `c2eb7a01c8623455cda0db10a4b2ed95e7db9f4c5e73a14ec337f6ae532e1339` |

The composite, controller/server, SDK-worker, and delivery-ledger boundaries moved together. The
public facade supplies only `PreparedProgram` snapshots, entrypoint plans, and reservation-carrier
plans to the Driver. Worker fixture preparation now captures that public view through a deliberately
failing Driver `Open`, and adapter artifact tests use public `testpilot.Prepare`, `PreparedCase.Run`,
and strict Testpilot ProtoJSON decoding. The async Nexus correlation mutation matrix and live/offline
evaluator parity now live in the common Testpilot verification corpus, where the private evaluator
is already owned, rather than widening common Testpilot internals.

The remaining Umpire public-facade conformance test keeps its task-7 protocol ingestion and fixture
root, but obtains the WorkflowService descriptor closure through the moved testcore adapter so the
coexistence state remains buildable. Task 7 still owns moving that test and regenerating its corpus.

The task-6 baseline generation checks exposed the task-7 staging boundary exactly: all six checked
conformance Case files regenerate with seven total `NamedType.protobufType` substitutions from
`temporal.server.api.umpire.v1.InstructionOutcomeStatus` to
`temporal.server.api.testpilot.v1.InstructionOutcomeStatus` (two occurrences in
`cleanup-failure-after-proved-violation`, one in each other class). No other generated difference
appeared. This follows from task 5 changing the shared `statusType` producer identity in
`model/Temporal/CaseRuntime.lean`; task 7 owns regenerating and moving the six-class corpus, so task
6 leaves the checked corpus unchanged.

The live consumer is now `TestTestpilotAsyncNexusCase` and uses the testcore Driver and Testpilot
protocol/facade. Its physical task queue and Nexus endpoint values remain the ordinary Umpire wire
values embedded in the task-5 fixture. Make and workflow package selectors cover
`./tools/umpire/...`, `./common/testing/testpilot/...`, and `./tests/testcore/testpilot/...`; the live selector includes the exact
`TestTestpilotAsyncNexusCase` success identity alongside the unchanged inherited Umpire2/Umpire3
failure set.

| Task-6 verification | Result |
| --- | --- |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tests/testcore/testpilot/...` | pass: all four moved packages |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./common/testing/testpilot/...` | pass: public facade and all private core packages, including moved Nexus correlation mutations and live/offline parity |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` | pass: exact Make/workflow package selector across all three owners |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire -run '^TestCaseRuntimePublicFacadeConformance$'` | pass: coexistence descriptor bridge and six classes |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/regression -run '^(TestUmpireCIWorkflowRunsSeparatedUnitAndLiveProofs|TestUmpireDocumentationStatesAttachedOwnershipAndBoundedClaim)$'` | pass: exact package/live selectors and moved README ownership |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase$'` | pass: real Testpilot async Nexus success path |
| `GOFLAGS='-p=1' CGO_ENABLED=0 TMPDIR=/private/tmp make umpire-check-live-tests` | pass: Testpilot success plus exact inherited Umpire2/Umpire3 failure identities |
| `make umpire-check-case-runtime-conformance` | inherited staging red: only the seven task-7-owned `NamedType` substitutions above |

## Task 7 remaining Producer and conformance cutover

The Umpire-owned `temporal-case-runtime` executable now sends all eight established renderer
arguments through the task-5 direct Testpilot ProtoJSON lowering. The command, typed Umpire Case
model, and compiler remain under Umpire; the generic generator validates and deterministically
packs all rendered Cases through the public Testpilot ingestion API. The former `caseartifact`
package has no callers and is removed.

The generic transaction now owns exactly
`common/testing/testpilot/testdata/case-runtime-conformance`. The established
`make umpire-gen-case-runtime-conformance` command regenerated twelve files in the same six
behavioral classes. All six expected-result files retain their task-1 SHA-256 values byte for byte,
so preparation acceptance, Run count, event projection, cleanup result, diagnostics, Verdict, rule
support, and cross-Run isolation expectations are unchanged.

| Testpilot conformance output | SHA-256 |
| --- | --- |
| `cleanup-failure-after-proved-violation/case.json` | `192e303d3017d48a28fb47f224d5d289c6afb6be58875337e2667d822ea8767f` |
| `cleanup-failure-after-proved-violation/expected.json` | `ff9eb63cb63b2624adde241704f066d1eb71d4c2e881235bff6c1e8853f8d03d` |
| `cross-run-isolation/case.json` | `81f71631096e3d290628508e174b791405b59f888d7fe1d30aa8fcd302658f29` |
| `cross-run-isolation/expected.json` | `d36e171e907672fa3f067443f387a469e0c93a6395c9e6d621651b3e49562463` |
| `inconclusive/case.json` | `83c4725c56eeab574964e2c81e667a79f636672bc8c978100bc5e2282acf0385` |
| `inconclusive/expected.json` | `75a10bdc8c73fd22df4be259617fb5cbe0068f5aae354f37b239ac670ba0e5af` |
| `satisfied/case.json` | `5842021c12ca44f91031d9f214e63f6d3be8eeb9e293ef5a07ec3d200fa14aff` |
| `satisfied/expected.json` | `d114c1df2d69ba56bad27b78b7aa0a305bb73c73af4747026e1bbe6084c77101` |
| `static-preparation-rejection/case.json` | `a7c6411b51197fc7c2c5d67b900fbe86603b64c6e645cc163ad51facc27da296` |
| `static-preparation-rejection/expected.json` | `210fe392c9766f7c042e60964e14651c34a2251d79cae39ff95a6cbfbe31bc0d` |
| `violated/case.json` | `6515b7c531421c64b02d85a65b13e1bae2da5d741c7fbfd8bcf89815d0841db1` |
| `violated/expected.json` | `26eeee9aebca0f702af9b9b7f4e3de7147320314a9d51c8c520b76319a4c4d31` |

Every Case-file difference is explained by the task-9 refined Testpilot schema table above:
metadata becomes opaque provenance; expression trees, slots, activations, definitions, references,
and lifecycle status fields use their refined structural forms and names. Case IDs, Program IDs,
rule IDs, methods, bounds, scalar values, event filters, and all other ordinary values are
unchanged. The only namespace-bearing values are exactly seven `NamedType.protobufType` entries:
two in `cleanup-failure-after-proved-violation` and one in every other class. Each is the task-1
allowlisted prefix substitution from
`temporal.server.api.umpire.v1.InstructionOutcomeStatus` to
`temporal.server.api.testpilot.v1.InstructionOutcomeStatus`; no Umpire protobuf name or `Any` type
URL remains. Testpilot descriptor/catalog identity and generated Run ID prefix are consequently
Testpilot-owned, while admission compares the prepared identity supplied to the Driver and the
conformance test continues to prove fresh IDs without treating either derived identity as behavior.

Generic conformance ownership and its exact six-class execution test moved to
`common/testing/testpilot`; Make generation, diff, focused-test, and retired-vocabulary paths follow
that owner. Its WorkflowService catalog is built from a local public descriptor closure, so the
common Testpilot test owner has no dependency on the functional testcore Driver. Umpire's Lean API
schema tests now exercise only the refined Testpilot contract. The
only remaining Go imports of the old Umpire proto are the frozen compatibility comparison and the
temporary old runtime implementation that task 8 removes.

## Task 8 final ownership closure

Task 8 began from synthetic dependency snapshot
`2d9b1cea04f73635e877ccefc4a74ffb2de77dac` (tree
`b4a8248665e7ab23fa8b4841d604afeb5c7d0cce`, parent
`5b53d57254993533218da99865b4c60f278cd056`). The snapshot was constructed in a bare repository
with the checkout object store as an alternate and a private temporary index, so the shared index
and the tasks 1-7 and 9 working state were unchanged.

Zero-consumer searches and `go list -tags test_dep` proved the remaining old imports were confined
to the former owners and their tests. Task 8 then removed all five old proto sources, ten generated
files, the old public facade and carrier test, the complete old IR/execution/verification trees, and
the temporary protocol comparison. The already-removed adapter, `caseartifact`, fixtures, and
conformance copies remain absent. No former directory is left empty or present. Umpire retains its
Lean authoring, Producer, generator, regression, and vocabulary packages.

The final package closure is 10 Umpire Producer/regression packages, four common Testpilot packages,
and four functional testcore Driver packages. Common Testpilot owns 137 Test/Fuzz entry points,
covering the 124 private-core and 11 facade baselines plus the final protocol assertions. The
functional Driver retains all 83 adapter Test/Fuzz entry points. The two retained Producer generators
now have 41 Test/Fuzz entry points. Testpilot imports no Umpire, functional testcore, SDK, or canary
package; private core packages have the same one-way boundary, and server, worker, and internal
adapter packages cannot cross their authority boundaries.

The vocabulary implementation, command, Make target, aggregate prerequisite, and test file are now
`retiredvocabulary`, `umpire-check-retired-vocabulary`, and `retired_vocabulary_test.go`. The guard
scans the common Testpilot protocol/runtime, functional Driver, generated Testpilot API, and Testpilot
proto source roots in addition to Umpire authoring and active downstream documents. No active guard
path or declaration retains the former naming.

All twelve generic conformance files and both functional fixtures retain the task-7/task-6 SHA-256
values recorded above. There are zero old Umpire protobuf names or imports in the final protocol,
runtime, Driver, Producer, or live-test roots. The only namespace-bearing Case values are the seven
generic and nine functional Testpilot `InstructionOutcomeStatus` identities already reconciled in
tasks 5 and 7. An initial timestamp-preserving `make proto` exposed a stale deleted Umpire descriptor
in `proto/image.bin`; forcing that descriptor target once and regenerating the Lean API removed the
old namespace. The final generated `model/Temporal/API/Types.lean` has SHA-256
`3680e83ad104d63c7b1630f9c441ccd33a9993080e82d523785f82c2a6f8aeac`, and subsequent ordinary
`make proto` and `make umpire-gen-lean-api` runs do not restore an old owner.

Final serial verification passed the complete 18-package selector, proto generation, Lean API
generation, the six-class conformance check, the renamed vocabulary gate, the 330-job model build,
261-target model lint, proto lint, API lint, the exact live selector, and the 325-target aggregate
regression gate. A scoped golangci v2.13.1 run over the exact synthetic task-8 patch used
`--fix=false` and reported zero issues; the known whole-repository lint path was not used because its
prior transient peak exceeded 7.4 GiB. The first post-edit aggregate attempt observed the unrelated
intermittent `TestUmpire3SparseRegressionCompletionBeforeStartResponse`; its focused rerun passed,
and the final aggregate rerun matched the exact inherited failure set and exited zero. `git diff
--check`, fixture hashes, generated identities, import closure, and empty-directory scans pass.

The task removes duplicate code and performs no new execution work. Cancellation, crash, cleanup,
security, authority, and bounded 10x behavior remain owned and exercised by the moved Testpilot and
Driver tests; no retry, allocation, persistence, credential, network, or concurrency path was added.
