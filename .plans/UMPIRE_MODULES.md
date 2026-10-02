# Umpire module and migration map

This map fixes the fn-115 destinations and compatibility boundaries before source moves. The audit
covers the actual working tree at `8fff42d9a8f77a07e8f351f3871cc8030dfe1db6`, including the authorized,
uncommitted fn-107 implementation. It does not use that commit as the content baseline. The owner
has delegated the choices; the conductor obtains independent review of these artifacts before any
restructuring. This document records a map, not a review verdict or completed migration.

The [package audit](umpire-migration-audit.json) records each source package/module, its importers,
Make/CI evidence and disposition. The [migration manifest](umpire-migration-manifest.json) records
current files, the index, protected originals, source substitutions and every retained Case fixture.
The JSON files are evidence inventories; this map supplies their architectural decisions.

## Module ownership

Paths below are destinations. Public entry points retain their current signatures and behavior,
except that caller-visible checker and producer types acquire names in their owning public facade.
Returned data types, their used methods and error/status values are part of the same contract.
Standard libraries, existing support libraries and generated schemas are permitted where already
used; the dependency column constrains domain ownership, not those ordinary dependencies.

| Module and destination | One job | Public interface | Permitted domain dependencies |
| --- | --- | --- | --- |
| Umpire IR, `api/umpire/v1`, `proto/internal/temporal/server/api/umpire/v1` | Represent a lifted Model. | Existing protobuf messages under the new package names. | Protobuf support; no Testpilot schema dependency added. |
| DSL, `model/umpire` | Express finite Models in Scala. | Existing authoring declarations, prelude and realization types. | Scala standard library; no `temporal`, lifter or gate import. |
| Models, `model/temporal` | Declare Temporal behavior. | Existing machine, claim, Query and realization roots. | DSL and shared feature kit; generated Temporal API data when delivered by its later spec. |
| Lifter, `model/lifter` | Translate typed trees into Umpire IR. | `Lifter`, `LiftError`, existing CLI root/prefix arguments. | Generated IR, TASTy/Quotes and compiler libraries; Models read as TASTy, never source imports. |
| Gate, `model/gate` | Verify the authored model pipeline. | One Scala program, `--update`; internal schema-generation mode `--generate-ir [--if-stale]`. | Processes/files, lifter outputs, Go checks; no authoring dependency on the gate. |
| Reader, `tools/umpire/model` | Interpret the meaning of one admitted Model. | `Load`, `Validate`, `Check`, `Build`, `NewInterpreter`, `NewRealizer`, `TypeOf`, `PayloadFields`, `GuardProblem`, `Unknown`; existing value, receipt, scope, table and bound-claim data needed by consumers. | Umpire IR and its private checker; no Testpilot package or schema. |
| Checker, `tools/umpire/model/internal/checker` | Evaluate finite table claims. | Private to the reader; retained implementation declarations only. | No Testpilot or lower/conformance/export imports. |
| Lowering, `tools/umpire/lower` | Produce a Case from a Query's witness. | `NewProducer`, `Producer.Lower`, `Identity`, `IdentityFor`, `GenerateCases`, `DecodeManifest`, `SyncCases`; lowering standings, inventories and manifest data. | Reader, Testpilot facade/schema and private producer. |
| Producer, `tools/umpire/lower/internal/producer` | Assemble the executable Program and Contract. | Private lowering implementation copied from live `caseproducer` code. | Reader's table/claim types, Testpilot facade/schema; never the private checker directly. |
| Conformance, `tools/umpire/conformance` | Assess whether Run evidence is explained by a Model. | `Prepare`, returned `Factory.Binding`/`New`, existing limits and located errors; `Admits` becomes private because its callers are in the package. | Reader and Testpilot facade/schema. |
| Export, `tools/umpire/export` | Compare another backend's reading with the reader. | Current `Open`/`OpenWithin`, export/check/agreement methods and tool runners used by its opt-in tests. | Reader is its only model-domain dependency; generated IR and existing process/protobuf libraries allowed. |
| Exploration, `tools/umpire/explore` | Select model-declared executable candidates. | `New`, `Plan.Reduce`, `Plan.Proposal`, `ReadProposal`, `Serve`, `RenderTrace`, candidate/plan data. | Reader, lowering, Testpilot facade/schema, campaign, replay and recordedrun. |
| Testpilot, `common/testing/testpilot` | Execute an admitted Case. | Existing `Prepare` → `PreparedCase.Run`, assessment and Driver contracts unchanged. | Testpilot schema and existing internal runtime; nothing under `tools/umpire`, `model` or archives, including tests. |
| Recorded Runs, `common/testing/testpilot/recordedrun` | Preserve a closed Run's exact identities and evidence. | Existing `Encode`, `Decode`, `CaseIdentity`, `Digest`, support/agreement checks and data; exclusive writer becomes `Write`; canonical `Signature`, `RunSignature`, `ReportSignature`, `Line`/`Equal` and their data/prefix. | Testpilot facade/schema and Case-file encoding. |
| Case-file encoding, `common/testing/testpilot/casefile` | Admit the exact stored and canonical artifact byte forms. | `Compact`, `Persisted`, `Canonical`, `ErrNoncanonical`; keep `Indent` only if externally used. | JSON standard library only. |
| Evaluation, `common/testing/testpilot/evaluation` | Assess one closed Run under an Evaluation Profile. | Existing `Admit`, `Assess`, `LoadProfile`/`LoadProfileIn`, `ParseProfile`, receipt encode/decode/identity and caps APIs. | Recorded Runs, Case-file encoding, Testpilot facade/schema; no replay, campaign or Model dependency; receipt tests use publication directly. |
| Publication, `common/testing/testpilot/publish` | Publish review artifacts without replacing existing bytes. | `Publish`, `Check`, `Resolve`, `Within`, publication/conflict data; gains `Proposal`, `WriteProposals` and the existing exclusive proposal writer. | Filesystem/context only. |
| Campaign, `common/testing/testpilot/campaign` | Coordinate bounded Case execution over the producer-neutral bridge protocol. | Existing bridge/session, protocol frames, `Drive`/`DriveRecording`, Binder and cap APIs. | Testpilot facade/schema, Temporal binding adapter and Case-file helpers if used; no Model dependency. |
| Replay, `common/testing/testpilot/replay` | Decide whether a recorded violation reproduces through the bridge protocol. | Existing `Admit`, `Execute`, bridge/reducer, violation-key/classification, report and proposal APIs. | Testpilot, campaign, recordedrun, Case-file encoding, publication and Temporal binding for existing bounded teardown; no Umpire CLI dependency. |
| Temporal binding, `common/testing/testpilot/temporal/binding` | Bind a Case to deployment-owned Temporal resources. | Existing `Deployment`, `Open`, `Prepare`/`PrepareWith`, `Campaign.Bind`, bound Run/release and flag adapters. | Testpilot and Temporal Driver/provisioning libraries; no campaign import or Model dependency. |
| CLI edges, `tools/umpire/internal/cli` | Apply common command-edge policy. | `Interruptible`, `WriteLine`, `Flatten`, `OutsideModel`. | Publication; no runtime importing this package. |
| Commands, `tools/umpire/cmd/*` | Run the chosen pipeline operation. | The seven retained executables listed below and their existing argument contracts. | Relevant live modules only; no archived executable. |
| Functional wiring, `tests/testcore/testpilot` | Bind Cases to functional test clusters. | Existing fixture and cluster helper APIs with reviewed fixture identities. | Live Testpilot and Umpire modules; nothing imports this package from runtime/tooling. |
| Canary, `tools/canary` | Run the policy's pinned Case on a deployment schedule. | Existing canary entry point and policy contract. | Testpilot runtime/helpers and Temporal binding; no consumer imports from tooling/runtime. |

The reader combines admission, interpretation and checking because those are one caller operation:
reading a Model's meaning. Exploration combines reader, lowerer and runtime protocol data because
its result is a model-selected executable candidate. These are deliberate exceptions to splitting
an interface merely because its explanation can contain “and.” Campaign and replay remain separate:
one coordinates candidate execution, the other owns reproduction and reduction decisions.

The [parity claim inventory](umpire-migration-claims.json) maps retired oracle comparisons to
frozen evidence and surviving checks, and preserves their original commentary with source attribution.

## Public types and file boundaries

The reader owns the public names for every checker type its API exposes. Copy the checker into the
reader's `internal/checker`, then declare public aliases for the required immutable/data contracts
and narrowly expose the existing operations needed by the producer and export. This preserves Go
assignability and the existing algorithms without allowing sibling packages to import the checker.
`Scope.Compose`, `Scope.Progress`, receipt witnesses/limits/refinement/monitor results, `Machine.Table`
and the bound Property/Query surfaces must all use reader names in their declarations. `Table`,
`Query`, `Atom`, `Result`, `Trace`, `TraceStep`, `Limits`, `ComposeCeiling`, `ProgressKind`,
`RefinementFailure`, `MonitorVerdict` and the full dependency closure of those public signatures are
included. The audit's `reader_checker_public_aliases` names the additional direct external uses,
including canonical `Fingerprint`/`Quote`/`Group` operations used by the producer. Methods retain
behavior and ordering; this task does not replace the algorithms or add admission rules.
Existing reader `Found`, `NotFound` and `LimitReached` names remain their current `ReceiptKind`
constants. The required checker `Outcome` receives a public alias; outcome comparisons explicitly
convert the existing same-valued constants (for example `Outcome(Found)`). Verify equality with the
private checker constants; do not merge the receipt/outcome types or add a synonym constant family.

`Row` is public data in the `Table.Rows` signature closure. `NewTable`, `TableSpec`, `KeyProperty`,
`KeyScenario` and `KeyFind` remain private-checker operations, not public test-construction APIs.

Lowering exposes `Identity` and `IdentityFor` from its private producer so exploration and fixture
callers have no direct `caseproducer` import. Move producer-specific white-box tests with that
private implementation; higher-level tests use the lowerer's public API. Public declaration
inventory and actual selector callers are recorded in `surface_inventory`. R18's cleanup keeps
only the entry-point families above, the types required by their signatures and verified external
uses; test-only implementation helpers become package-private. Export's test-only entry points
become private when the same-package tests can use them. No compatibility forwarding package remains.

`recordedrun.Write` receives the actual exclusive creation logic from `replay.WriteRecordedRun`;
replay's aliases and pure forwarding functions disappear from live copies. CLI publication/path
forwarders likewise disappear. `Proposal`/`WriteProposals` belong to publication; replay keeps a
small local reporting function for its two `WriteLine` callers. This avoids making runtime replay
depend on a package that knows the Model output boundary. Preserve all existing write conflict,
symlink, partial-write, error and cancellation behavior during these moves.

The existing failure-signature encoder and tests move from `tests/testcore/testpilot/signature*.go`
to `common/testing/testpilot/recordedrun/signature*.go`: a canonical diagnostic projection of a
closed Run belongs with its recorded evidence. Preserve `SignatureLinePrefix`, `Signature`, its
rule/diagnostic data, `RunSignature`, `ReportSignature`, `Line` and `Equal` unchanged. Functional
assertions and the command's `TestSignatureOfReadsTheLineALiveTestLogs` use this same encoder;
the test still checks the actual encoder against the command parser. Task 4 copies the code/tests
and switches functional callers. Keep the old functional definitions only while the frozen tools
command still needs them; task 6 switches the copied command test and removes those definitions.
No tooling-to-functional-wiring exception or forwarding aliases remain after cutover.

Producer test claims have explicit transfers in the audit's `test_claim_transfers`. Task 2 captures
all seven original Nexus typed/keyed pairs and their full Cases, table/claim/witness facts and
identical realization/source inputs. Task 3 replaces the legacy parity oracle with equivalent
admitted IR fixture comparisons against that frozen evidence in the live lowerer; the original
`model/go/caseproducer` tests remain frozen. Task 5 omits their superseded oracle imports only in
the new producer copy. This is separate from R22 execution pin replacement. Task 5 carries all ten occurrence/projection/fingerprint tests into the private
producer copy, constructing the equivalent generic job IR with reader admission/Build and
`NewRealizer.Find`. Preserve domains, result/fact ordering within each row, nondeterministic alternatives, evidence,
IDs, paths, predicates and limits. The two remaining keyed-test claims move into same-package
lowerer tests using reader Queries and local realization conversion: retry/backoff-only node
placement, and exact Produce/Preflight evidence-source rejection agreement. Keep their assertions
and comments. No raw checker constructor is added to the reader for these tests.

The synthetic job fixture alone requires one closed row permutation: the original order
`idle-submit, queued-take, queued-drop, running-fail, waiting-settle, running-finish,
running-check, running-close` becomes the reader’s state-major order
`idle-submit, queued-take, queued-drop, running-fail, running-finish, running-check,
running-close, waiting-settle` (original indexes `0,1,2,3,5,6,7,4`). Task 2 retains the
original baseline unchanged. Task 5 requires these exact sequences and compares complete rows by
key for this fixture transfer only. Result/fact order, domains, evidence, witnesses, Query answers,
full Case bytes and fingerprints must stay unchanged. The only additional serialization adaptation
is the two empty fact lists described below; any other difference stops the transfer.
Production Model goldens remain strictly order-sensitive. The comparator may adapt only
`oracles/job/{once,retried,closed}/table.json`: first require lossless decode/re-encode of each
original table, exact original and approved actual row-key vectors, and complete per-key row
equality including result/fact/evidence order, subject only to the following empty-list projection.
The original `waiting-settle.results[0].facts` and `running-check.results[0].facts` are JSON
`null`; reader construction initializes each empty fact list and therefore emits `[]`. Require
exactly one result in each named row, original facts nil and actual facts non-nil with length zero,
then project only those two expected fields to empty arrays in memory. Never alter nonempty facts,
other nil lists, result order, or the interpreter. Only then permute the expected Rows array in the
in-memory copy before the existing whole-inventory byte comparison. The immutable files and all
other entries stay byte-strict. Negative checks reject other orders, missing/duplicate/unknown
keys, changed row contents, unexpected null/empty conversions and nonempty or reordered facts.
This exception adapts a test fixture constructor; it does not relax
the interpreter or its production golden comparisons.

The original checker `model/go/umpire/replay_test.go` also imports handwritten Models. Leave it
frozen and omit it from the private checker copy. Transfer its comment and exact witness replay /
rebound-Action-ID rejection assertions into the reader-owned replay tests over admitted activity
and Nexus IR, with a nonvacuous expected Query inventory. This keeps checker tests independently
scoped; `Query.Replay` already belongs to the public Query method closure.

The reader's `load.go` retains file decoding; `validate.go` owns whole-IR admission; operand typing
and protobuf paths live in separate concern files, with focused tests. The lifter first gains one
typed context carrying `Quotes`, definition/symbol indexes, source-prefix mapping and declaration
accumulators. Extract types, expressions, declarations, realizations, compositions and claims into
concern files around that context; the entry point only arranges inspection and output. This is
structure work, not a second interpretation of the Model.

## Immutable migration goldens

The test-only helper `model/scalav2/goir/internal/golden` moves to
`tools/umpire/internal/golden`. It admits the fixed twelve-input inventory and the eighteen
path/twelve source-label substitutions above, with only standard-library, protobuf and IR imports.
No production package imports it. Reader tests and their data move to `tools/umpire/model`;
lowerer tests and their data move to `tools/umpire/lower`. Task 3 removes the existing legacy
oracle edges that currently bring Testpilot into the reader's test graph. The new reader golden
support introduces no such edge or production test-construction API.

Only the lowerer's external test package may import exploration, conformance, Testpilot runtime
and recordedrun to verify the complete artifact identity chain. This integration-test exception
does not permit those edges in lowerer production code. A narrow bridge defined only in a lowerer
`*_test.go` file may expose the existing local Query/realization conversion to that external test
package. The original Nexus oracle transfer reuses frozen IR with the five existing comparative
history `Exhaustive` adjustments and the captured original Source/Identity. Admission also requires
removing the corresponding `asyncNexus` / `controller` / `history` closing-read declarations.
Both ordered vectors must equal `temporal.nexus.caller.evidence.` followed by `started`, `completed`,
`failed`, `canceled`, `timedOut`; no other declaration is removed. Full oracle equality must prove
that every other value is unchanged. This bridge is absent from production builds.
Task 12 checks production and test imports separately. When removing parity tests, retain `tableSide`/`sideOf` as dedicated reader test support;
replace the temporary refined-row oracle comparison while retaining its frozen golden values.

There are 48 reader snapshots (inputs, semantics, declarations and refined Properties) and 1,363
artifact snapshots (original, mapped and original producer-oracle evidence). Each complete Case
has independently inspectable Program and Contract files. Read-only verification strictly matches
the current IR inputs, then regenerates both complete original and mapped artifact variants plus
the producer-oracle inventory from frozen inputs. It compares every entry, including the inactive
variant, and rejects missing or additional entries. Capture is an explicit test entry point that
rejects any existing destination. Run captures sequentially,
using separate new absolute directories whose parents exist:

```sh
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir -run '^TestCaptureMigrationGoldens$' -count=1 -args -capture-goldens=/absolute/new-reader-capture
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^TestCaptureMigrationGoldens$' -count=1 -args -capture-goldens=/absolute/new-artifact-capture
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/...
```

After relocation, use `./tools/umpire/model` and `./tools/umpire/lower` for their respective
capture and verification commands. The captured original evidence stays immutable when the
legacy oracles retire; subsequent capture generators must preserve those original inputs and
relationships through admitted IR when each old constructor retires. Task 3 replaces the Nexus
handwritten Model oracle; generic job-only checker builders remain until task 5 copies the producer
and performs their admitted-IR transfer. Capture output is never automatically installed over a baseline.
Two independent captures must match byte-for-byte before any intentional baseline addition.
Read one Contract without rewriting or expanding the whole baseline:

```sh
python3 - <<'INSPECT'
import gzip, json
from pathlib import Path
root = Path('model/scalav2/goir/testpilot/testdata/migration')
path = next(root.glob('original/ir/nexus-caller.json/queries/*/contract.json.gz'))
print(path)
print(json.dumps(json.loads(gzip.decompress(path.read_bytes())), indent=2))
INSPECT
```

After relocation the inspection root is `tools/umpire/lower/testdata/migration`. Keep the narrow
Git ignore exceptions with both fixture trees and reject missing or additional baseline entries.

## Scala layout and commands

`model/project.scala` is the shared directive file copied from `scala/project.scala`. Compile the
DSL with exactly that file and `model/umpire`; compile/test/package Models with that file plus
`model/umpire` and `model/temporal`. Tests stay under their existing `umpire/test` and `temporal/test`
subtrees. The lifter and gate have independent `project.scala` files. Their generated jars, fixtures
and stamps live under ignored `model/gen`; the gate names source roots explicitly, so a broad
recursive Scala compile never merges the authoring, lifter and gate projects. Lint/format commands
visit these explicit roots. The native Scala evaluator (`Search`, `Table`, `Lower`, etc.) stays in
the DSL until fn-113; this refactor does not retire it.

Checked IR, generated Cases, specimens, specs, README and SEMANTICS move directly under `model/`.
Reserve `model/examples` for fn-119 authoring examples and `tools/umpire/explore` for fn-120 explorer
work; no example or UI is implemented here. Root lists keep their present meaning until fn-114.

| Command after migration | Existing source / behavior |
| --- | --- |
| `make umpire-check-model` | `umpire-check-scala`; runs `mise exec -- scala-cli run model/gate`. |
| `make umpire-gen-model` | `umpire-gen-scala`; same program with `-- --update`. |
| `make lint-scala`, `fmt-scala`, `fix-scala` | Same language checks, explicit new roots; IR jar prerequisite uses the gate's schema-generation mode. |
| `make umpire-check-cases`, `umpire-gen-cases` | Run `umpire-gen-cases` without/with `--update`; default managed tree is `model/cases`. |
| `make umpire-check-fixtures`, `umpire-gen-fixtures` | Same generator with `--kind functional` without/with `--update`; only the explicitly supported R22 replacement subset is managed. |
| `make canary-check-case`, `canary-gen-case` | Same generator with `--kind canary` without/with `--update`; one pinned Case and reviewed policy identity. |
| `make umpire-ir-bridge` | Build the existing `cmd/umpire-ir-bridge` to `.build/umpire-ir-bridge`; default IR root becomes `model/ir`. |
| `make umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`, `umpire-assess` | Existing command build interfaces. |
| `make umpire-fuzz-run`, `umpire-replay-run` | Existing deployment flags; use the live IR bridge and model-declared set/target names. |
| `make umpire-repeat-run`, `umpire-assess-run` | Existing caller-supplied Run/Case interfaces. |
| `make umpire-check-exploration-bridge`, `umpire-check-replay-bridge` | Existing live protocol claims exercised against the IR bridge; no Lake invocation or success-by-skip. |
| `make umpire-check-live-tests` | Existing live functional gate, preserving a nonzero executed-test floor. |
| `make umpire-check-backends` | `UMPIRE_BACKENDS=require` opt-in Go tests in `tools/umpire/export`, using the copied pinned `quint.sh`; preserve existing tool versions/output environment controls. |
| `make umpire-check-testpilot-protocol` | Preserve the live protobuf comment/schema check; remove its retired frontend generation/build tail. |
| `make canary-build` and other live canary build/run targets | Preserve production canary build and existing operator interfaces. |
| `make umpire-rerecord-pinned-runs` | Remains an explicit operator action targeting current live fixture identities; never runs automatically as part of this migration or ordinary tests. |

The retained executable set is `umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`,
`umpire-assess`, `umpire-gen-cases`, `umpire-ir-bridge`. Their old source files are copied out of the
frozen tools tree. Delete targets/variables whose only purpose is archived generation, inventory,
views, model-module indexes or retired frontend checks. `commands` in the audit records the exact
old bodies and disposition; no old target becomes a stub. Preserve current profile JSON as live
configuration data while archiving its old renderer. The existing Umpire CI jobs use the root mise
configuration and live Go/fixture checks, retain the canary harness check and production build, and
receive no new JVM job. Default unit tests neither run Scala nor regenerate fixture bytes.

## Archive and intermediate-state rules

1. Check the manifest's original hashes and index before any extraction. Task 2 captures the golden
   while the current paths and original oracles still exist. Task 3 transfers each oracle claim
   before removing its import; original model/go source remains untouched for the archive.
2. Promote neutral helpers by copying into their final Testpilot destinations. Update live consumers
   outside old `tools/umpire`; the original tools tree stays byte-identical. Its old commands can
   continue using their old helpers until the namespace swap. Copied helper tests read copied
   fixtures, never fixtures through an archive path. Leave Model-specific `campaign/bridge_live_test.go` and `replay/bridge_live_test.go`
   in the frozen original tooling during task 4 rather than copying its Lean executable dependency
   into Testpilot; task 6 transfers those claims to live IR bridge integration tests before archiving.
3. Task 5 copies the checker to `model/scalav2/goir/internal/checker` and the producer to
   `model/scalav2/goir/testpilot/internal/producer` **before** switching reader Query ownership.
   In that same tested task, the lowerer uses its local producer copy, and that producer uses public
   `model/scalav2/goir` types/operations. Never pass the copied checker's Query to the original
   `model/go/caseproducer`. Retarget external identity callers through public lowering. Task 6 moves
   both private subtrees with their owners. Original checker/producer and old tools stay unchanged.
4. In one tested relocation task, guard that archive destinations do not exist, rename old tools
   to `tools/umpire0`, place the selected live command/CLI copies into the newly free tools namespace,
   move the reader/lowerer/conformance/export/explore there, and move authoring to `model/`. Add
   nested archive `go.mod` files and required README metadata. Update every live import and essential
   Make/gate/test path within this task. Never finish a task with a live archive import.
5. Move each retired model tree into its matching `model0/<tree>` directory. The original Quint
   launcher also has a live copy under export. Move the four `.plans/lean` documents individually
   into `.plans/archive/lean`: all four destination names are absent; the existing sixteen documents
   remain byte-identical. Only the archive index is edited.
6. Keep path/import-only relocation evidence separate from subsequent cleanup. Use guarded
   filesystem operations and content manifests; do not stage, commit, create worktrees or recursively
   delete. Preserve every original byte recorded in `protected_originals`, except the explicit
   archive metadata additions. Cache exclusions are listed, not guessed from a broad `testdata` rule.

Original legacy source-dependent tests outside the archive trees also require explicit disposition:
archive the renderer-only portion of `protobuf_lean_authoring_test.go`, preserving its useful schema
assertions in live Testpilot tests; transfer bridge protocol claims from the old binary-dependent
functional helper, `campaign/bridge_live_test.go` and `replay/bridge_live_test.go` to the existing IR bridge tests before removing
old executable paths. Preserve all runtime/fixture/negative-test claims. The audit does not authorize
archiving a functional test merely because its Case still needs an R22 exception.

During task 4's copy-before-archive state, frozen
`tools/umpire/regression/TestTestpilotOwnsCaseProtocolAndRuntime` rejects the six newly permitted
Temporal Driver edges: production `campaign/run.go` and `replay/report.go` to `temporal/binding`,
and fixture tests `campaign/integration_test.go`, `evaluation/admission_test.go`,
`replay/driver_test.go`, `replay/key_test.go`. Keep its original bytes. Transfer its useful generic
runtime dependency restrictions and negative checks into live Testpilot ownership tests with only
these map-approved allowances. The transitional broad Go command excludes exactly that obsolete
test with `-skip '^TestTestpilotOwnsCaseProtocolAndRuntime$'`; every other old regression test still
runs. Record the original failure and revised green command. Task 6's archive exclusion removes
this temporary command exception; final main-module checks have no such skip.

## Exact artifacts and fixture compatibility

Only the eighteen observed source paths and twelve exact Model source labels in the manifest may
change in their typed fields. Preserve every line and column; historical-attribution removal leaves
blank lines where necessary. New, unexpected source strings fail migration verification rather than
being accepted by a generic prefix normalizer. The six checked Models and six expected fixture IR
files form the initial IR inventory; all their Queries and unsupported standings enter task 2's
baseline. Confinement remains the existing source-position test, not a new runtime loader rule.

Recompute hashes in the manifest's dependency order. Ordinary generated Cases retain the same
`IdentityFor` inputs and IDs while mapped provenance changes their canonical checksum. Exploration
hashes the entire deterministic candidate IR, so the candidate digest, digest-derived Case/Program/
Contract IDs and local references, Case checksum, proposal payload/hash/name and rendered source
links change together. Their semantic data and candidate ordering stay fixed. Model assessment's
source-stripped identity remains identical, including its persisted `goir.model/v1` domain. Stable
`scala.`, `scala.explore.`, `scala-model` and `goir.model/v1` strings are compatibility data, not
historical package names to rewrite. The proto rename freezes descriptor structure and wire meaning.

R22 replacement is a separate artifact decision. The inventory contains sixteen already-generated
Scala Cases, eight legacy functional Nexus/control Cases with actual replacement Queries, one
pinned canary replacement, five functional/synthetic exceptions and seventeen runtime-conformance
exceptions. Each row records original bytes, Program/Contract inspection hashes, sources, consumer
evidence, the exact replacement Query where one exists, or the absent declaration/missing primitive.
The missing functional declarations concern two-operation Nexus correlation, System Info, workflow
start, workflow-worker outage/recovery and the synthetic payload admission fixture. No fictional
Scala Query is used as a replacement.

The canary uses the existing `nexus-caller.json` `nexusProtocol/syncCompletion` Query. Its pinned
bytes, explicit policy Case identity, fixture constants and Profile-dependent expectations change
as one reviewed replacement. Require admission under the existing literal role/opcode/command and
ceiling authority; never expand production credentials to make migration pass. Historical Runs, evaluation receipts and saved proposals are never rewritten to impersonate new
executions. Recorded Runs store a Case checksum rather than embedded Case bytes. Before replacing
the execution pins, copy their exact old Cases into
`common/testing/testpilot/replay/testdata/nexusCallerControl-forgedCompletion-case.json` and
`tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-case.json`. Historical record tests
use these companions with the recorded original Profile/binding identities. These are historical
compatibility fixtures, not execution pins or absent-primitive exceptions. Existing crossed-identity
rejection remains the compatibility decision.

The runtime-conformance fixtures keep their exact bytes and expected outcomes under R22's exception.
There is no Scala synthetic Model/Query/Realization for those scripted Driver cases; admission-
negative Cases additionally cannot be emitted by legitimate lowering because their invalidity is
the test. Their current hashes, adjacent expectations and archived renderer arguments preserve
inspectable mutation provenance. Task 11 does not invent a negative-fixture authoring language or
replace them with unrelated successful Queries. Ordinary tests keep using these retained files.

All subsequent tasks run the task-2 golden and relevant consumer checks. Closing checks compare gate
outputs against the dirty authorized baseline, include existing production/test import directions,
verify future generated fixtures are not ignored, and enforce R25 from a file outside `model/`.
An independent reviewer evaluates this map before implementation moves; the conductor owns the
review receipt and Flow lifecycle.
