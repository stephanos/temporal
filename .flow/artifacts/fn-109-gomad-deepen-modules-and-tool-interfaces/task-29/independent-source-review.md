# Task 29 independent source review

Root can commit the frozen private-payload cleanup progress. Verdict is
`SOURCE_PROGRESS_COMMIT_ONLY`. No actionable introduced Critical, Important or
Minor defect was found. This review authorizes no formal SHIP, task completion,
merge, push or qualification closure.

The reviewed change is the working diff in `tools/gomad3/artifact/store.go` and
`store_test.go` against `e98c7a3e2ca845b92edcdef185f5c9ae67be4a3a` on `gomad`.
BASE equals HEAD, so the committed range is empty. The two source hashes are
`a41297665317751bef4d62264eb8c68bdf0ea5cc3d79f234424ff8946978a7d1` and
`fad48f38fe5344b4f6767321580bdf3af79c3245dd68cbc355fecfaf5e535d6d`.

The requested reviewer route was `gpt-6.1-sol` at high, the same configured
family as the writer. Root supplied the single routing result
`Tier: session (jev-unavailable(no_key))`. Actual executing-model metadata is
unavailable. The reviewer made no judge, bridge, rerouting or delegation call.
The flow-next-prose skill governed the report's evidence wording.

## Source findings and strengths

Critical findings are zero. Important findings are zero. Minor findings are zero.
No source correction is required before root commits this bounded progress.

At `store.go:334`, `:353` and `:416`, each acquired handle receives one observing
defer. Every defer changes `retErr` only when its Close returns nonnil. A nil
Close therefore preserves the exact returned primary error object, text,
single or multiple unwrap shape, classification and associated zero
`record.File`. The original operation-error return expressions are unchanged.
The deferred output Close runs before input Close. If both fail after an
operation failure, the nested joins traverse operation, output, then input,
and retain that order in error text and `errors.As` selection.

At `store.go:371`, `:375` and `:432`, ownership becomes false before the checked
Close attempt. A failed explicit Close cannot be retried by the defer. An
output-Close failure still leaves input cleanup owned by its defer. Successful
copy remains copy/hash/count, context-aware Sync, output Close, input Close,
then literal metadata. Inline success remains write, context-aware Sync, file
Close, then metadata. All error branches return zero metadata. The old ignored
second input Close no longer occurs or becomes a successful-publication error.

Source/data routing, ordinary `os.Open` and Stat acceptance, nonregular rejection,
destination `O_WRONLY|O_CREATE|O_EXCL`, Chmod, hash/count and context checks are
unchanged. The helpers retain partial destinations. Store owns staging removal;
manifest-last publication, canonical finalization, no-replace rename, reuse,
capacity accounting and the independent target-pool transaction are unchanged.
There is no new public API, helper framework, host-I/O grant, waiver, suppression,
dependency, pin, runtime or generated-source edit.

Five added tests at `store_test.go:16`, `:54`, `:89`, `:119` and `:130` use real
temporary files and standard contexts. Successful source and inline writes check
whole literal metadata, independent SHA256 literals, bytes, file modes and
private parent mode. Cancellation occurs after handle acquisition in the actual
helper and checks the exact copy/write wrapper, direct unwrap to
`context.Canceled`, classification, zero metadata and retained empty destination.
Collision checks preserve the sentinel destination and the single wrapper around
the original open PathError. Directory input retains its literal error and never
creates a destination. Store failure removes staging and publishes nothing.
The single-unwrap controls reject gratuitous `errors.Join(primary, nil)`; source
success rejects observation of the old second Close. These controls establish
neither genuine first-Close failure nor simultaneous cleanup-error execution.

## Evidence audit and fresh checks

[independent-source-review-checks.json](independent-source-review-checks.json)
retains the exact read-only audit command and compact receipt bindings. The audit
checks all 12 selected worker gates and six fresh review gates against exact
commands, cwd, controlled environment, current tool/config hashes, raw log
hashes, start/end/elapsed times, authoritative child exits and all 23 artifact
Go source hashes. Every gate's before/after source hashes agree. Baseline source
hashes match BASE; characterization uses BASE production with the final tests;
all final and review source hashes match the frozen candidate. The five
characterizations passed before production changed. The policy RED is the actual
pinned baseline analyzer failure, not a claimed failing runtime regression.

The reviewer independently recomputed all 1,042 protected file hashes, compared
every protected byte with BASE through Git blobs, and reproduced aggregate
`40adf6923b35cc17eb5f76a189308bb506f41dfc4ab235a5d01736f2aed7f520` with sorted-key
JSON and ordinary separator spaces. Only `store.go` and `store_test.go` are
excluded. Source-admission and both protected receipts agree. Every byte outside
the two private helpers, every original Store test body, task 12, original
documents, and the parent API/criteria/boundaries prefix remains unchanged.
Existing comments and golden expectations retain their original bytes.

Fresh review results are bound to the same frozen candidate.

| Gate | Authoritative exit | Result |
| --- | --- | --- |
| Whole ordinary artifact package | 0 | 42 top-level tests and 27 subtests pass |
| Private/Store/publication/pool/mode/retained-cost focus | 0 | 32 top-level tests and 20 subtests pass |
| Architecture/public alias/signature/external Runner boundaries | 0 | Five named tests and ten signature subtests pass |
| Pinned errortype | 0 | Empty diagnostic output |
| Actual unfiltered pinned artifact lint | 1 | Same 21 residual findings |
| Scoped diff/gofmt | 0 | Empty output |

The baseline lint has 28 findings. Exactly seven errcheck findings at baseline
lines 333, 346, 352, 356, 402, 406 and 410 disappear. The remaining diagnostic
multiset, preserving multiplicity by path/message/linter, matches baseline after
those seven removals. Fresh lint findings also match the worker final findings
including locations. The unchanged directory Close shifts 470 to 490. Residual
counts are 17 errcheck, two exhaustive, one forbidigo and one staticcheck. There
are zero introduced diagnostics. The historical whole-Gomad 419 count was not
rerun or inferred from this package delta.

`TestRecordAndArtifactHaveSeparateOwners` is absent. The actual executed
`TestPackageArchitecture` requires both owners, assigns each package to its
owner, permits artifact to import record, and prevents record importing artifact.
`TestPublicPackagesDoNotExportTypeAliases`,
`TestArchitecturePublicSignatureFixtures`,
`TestRunnerRequestsCompileInExternalModule` and
`TestRunnerExternalConsumerCompiles` each actually ran and passed.

The reviewer inspected Makefile VERSION_INPUTS, BOUNDARY_INPUTS,
COMPATIBILITY_INPUTS and validation recipes. Neither helper is a generator
input. The existing `final-validation.json` has child exit zero, unchanged
candidate source, pinned tool/config/log bindings and a complete `make validate`
log. Protected generator inputs and outputs match BASE and current source, so
the reviewer reused that receipt without regenerating or rerunning it.

## Reproduction and limits

Run the commands below from
`/Users/stephan/Workspace/skunkworks/gomad/temporal`. The existing capture script
uses the pinned stock Go 1.27.1 linux/arm64 executable and analyzers, unchanged
root lint config, `GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS=''`, cleared
seed variables and test `-count=1 -tags test_dep`. The JSON child `exit` is
authoritative because the capture wrapper itself returns zero for lint failure.
These commands overwrite only the corresponding review receipts and logs.

```sh
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-package test -v ./artifact
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-focused test -v ./artifact -run 'Test(PrivatePayload|Publish|RetainedBytes|PoolTarget|PruneTargetPool|DamagedSharedTarget|CopiedArtifact)'
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-boundary test -v . -run 'Test(PackageArchitecture|PublicPackagesDoNotExportTypeAliases|RunnerRequestsCompileInExternalModule|RunnerExternalConsumerCompiles|ArchitecturePublicSignatureFixtures)'
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-errortype errortype
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-lint lint
python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/run-gate.py review-static static
```

The cached pinned `os/file_posix.go:20`, `os/file_unix.go:307` and `errors/join.go`
confirm that repeated File.Close becomes a close PathError wrapping
`os.ErrClosed`, and a nil-secondary Join would still replace the primary wrapper.
There is no legitimate deterministic first-Close fault reproduction in this
review. Conditional error ordering and non-retry behavior on a failed explicit
Close are source-reviewed only. No fake seam or descriptor race was introduced.

Original task 12/predecessors, task 21, R13/R18/R19, matched original first-baseline
fixed identities, complete/full/formal qualification, actual patched-runtime
native darwin/arm64 and linux/amd64, affected integration/qualification and the
remaining lint owners stay open. The unchanged successful record vector passes;
that does not supply the missing historical first-baseline canonical-byte proof.
Stock linux/arm64 evidence remains developmental.

Root should commit this verified source progress and retain acceptance-open.md.
Public CopyPayload, directory sync and shared verification retain their separate
cleanup owners. The reviewer changed only this report, its checks JSON and the
six review command receipt/log pairs. No source, index, HEAD, branch, Flow
lifecycle, parent/task or MILESTONES write occurred. All owned commands are
terminal. Pending handles, commands and delegates are zero.
