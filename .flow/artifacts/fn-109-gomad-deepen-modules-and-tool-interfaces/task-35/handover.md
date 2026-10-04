# Corpus cleanup source checkpoint

Corpus readers now report a genuine cleanup failure at each existing lifetime
boundary. A successful Close preserves the original result and concrete primary
error. A nonnil Close clears the result and returns either the raw sole cleanup
error or errors.Join(primary, cleanup), in that order. The inner Opened release
still completes before readSnapshot releases its outer file. Four existing
fixture defers now check the same single Close through nonfatal t.Errorf.

Status is SOURCE_PROGRESS_ONLY, with Flow still in_progress. Root owns the
independent source review, all staging/commits and lifecycle decisions.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Source and preservation

BASE is a80ad9b9d1a4195c4aeb2fe135557f71e6e6552a, retained in base_commit.
Only corpus.go and corpus_test.go product sources changed. Their final SHA-256
values are recorded in evidence.json and source-check.json.

source_audit.py reverses exactly two signatures/two defer closures, removes
only the three necessary test imports and appended tests, and reverses the
four original fixture wrappers. The reconstructed complete files match both
admitted BASE SHA-256 values. Every original production local statement,
comment, error/return path and test body/assertion therefore retains its bytes.
The unchanged merge owner validates before writeAtomic, updates memory after
publication and retains its existing true,error after cleanupCases failure.
All 1,043 protected inputs and the source-bound recommendation retain their
admitted SHA-256 values. The audit also checks source whitespace.

Similar code search found artifact/open.go's conditional retErr/closeErr
handling. This change follows that pattern at the existing corpus defers;
it adds no helper or injection seam. Fresh return names avoid changing any
original local statement.

## Real-file controls and results

Four bounded tests were appended before production changes. readSnapshot
controls cover direct success, mode before malformed input, concrete syntax
cause, schema before identity, identity mismatch and digest before identity.
Every failure has a zero Snapshot, exact deterministic text, preserved concrete
primary error shape, unchanged on-disk bytes and unchanged in-memory state.
validateEntry controls cover direct success and seven ordered failure pairs
through entry identity, coverage, features, case identity, payload size,
captured inputs and mount identity, with zero SharedTarget and unchanged memory.
The merge control uses a real published case with mismatched metadata and proves
failed validation cannot publish corpus.json or update the snapshot.

The canonical control captures one real admitted seed-7 entry against unchanged
BASE production in controls-focused.log. The retained literal covers complete
nonempty snapshot bytes, metadata and identities. Its byte hash is
sha256:e243d461e78c4a4a1a7f9832091709dc8bd2f8ade34e61d318914b8269fecf2f;
its snapshot identity is
sha256:b465cb3046d07998b31810e6a10167507ead9c68d33cade81d212552d9f57833.
preservation-focused.log proves the same literal assertions pass before the
production edit; final-focused.log proves them afterward. Expected bytes are
literal, never recomputed by the snapshot implementation.

| Serial check | BASE | Final |
| --- | --- | --- |
| Full corpus package | 20/20, exit 0 | 24/24, exit 0 |
| Focused old/new controls | 10/10, exit 0; appended and literal phases 14/14 | 14/14, exit 0 |
| Five actual nested-root boundaries | 5/5, exit 0 | 5/5, exit 0 |
| Unfiltered configured pinned corpus lint | Six errcheck diagnostics, exit 1 | Zero diagnostics, exit 0 |
| errortype | exit 0 | exit 0 |
| gofmt | Empty diff, exit 0 | Empty diff, exit 0 |

The boundaries execute TestPackageArchitecture,
TestPublicPackagesDoNotExportTypeAliases, TestArchitecturePublicSignatureFixtures,
TestRunnerRequestsCompileInExternalModule and TestRunnerExternalConsumerCompiles.
lint-delta.json attributes all six resolved baseline sites, two production and
four fixtures. No diagnostics remain or were introduced in this actual package
run. The historical whole-Gomad 419 count is unchanged historical evidence;
no fresh whole-scope count is inferred.

Every command's exact argv, cwd, effective Go controls, source/tool/config
hashes before and after, exit status, UTC start/end, elapsed time and raw-log
binding live in its *.receipt.json. All source freezes match and logs are
hash-checked. audit-environment.log confirms stock go1.27.1 on linux/arm64,
offline GOPROXY, local GOTOOLCHAIN and this nested module. Tests use test_dep
and count=1. All shared-cache commands ran serially. No command/delegate handle
remains running.

Generator input inspection covers Makefile descriptor/boundary/compatibility
inputs and protocol generation's explicit schemas/templates/implementation
lists, plus the version descriptor. Corpus sources are outside these inputs;
no generated consumer or canonical identity definition changed. validate was
not run because its inputs are unaffected. Unchanged rootfast/full/native and
missing patched-launcher environment failures were not retried.

## Remaining acceptance

The actual analyzer RED and GREEN prove the six ignored-return defects are
resolved. Normal real-file controls prove successful-close preservation and
ordinary validation/publication behavior only. Genuine first-Close failure
and simultaneous operation-and-Close failure execution remain unproved.
The new sole/combined cleanup branches have source evidence only. No second
Close, reflection/unsafe fault experiment, global hook, production seam or
generic cleanup framework was introduced.

Original R3/R13/R18/R19, shared fn108 assessment/retention, task12/relevant
predecessors/task21, matched first-baseline fixed identities, complete/full/
completion/formal/affected-consumer and native darwin/arm64 plus linux/amd64
qualification remain REQUIRED and OPEN. Developmental package checks close
none of those obligations. This candidate requires root's independent fresh
source review before its source-progress commit and before another writer.

Defect route:
- prior fixes: source-bound recommendation and retained corpus controls identify six admitted sites; external PR/tracker/history checks were not done because this source-only dispatch forbids network and Git/Flow operations.
- diagnosis: baseline configured errcheck reproduces exactly two production and four fixture ignored Close results; final configured analyzer resolves all six. Genuine cleanup-fault runtime diagnosis remains unexecuted.
- introduced by: not done; no known-good close-checking revision is supplied and history/bisect/worktrees are outside this dispatch.
- base: six actual errcheck findings; preservation controls pass with original production SHA. head: zero package findings and unchanged literal/control results on the frozen candidate.
- live: no live application surface; concrete filesystem/package behavior was exercised.
