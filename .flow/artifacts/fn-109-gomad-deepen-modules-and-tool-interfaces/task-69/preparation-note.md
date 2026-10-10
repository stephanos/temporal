# Task 69 gate preparation

Preparation only, requested Sol/high. No product edit, Go/test/build/lint/vet/generator/toolchain execution, Flow lifecycle or Git mutation was performed. Root retains worker admission, exclusive lane grants, integration, reviews, commits and completion. The supplied preparation HEAD is `176a269e651c10f7e88bb8dc11fc03ef78bf2e57`; it is not an execution receipt. Admission BASE remains `2af15fcbc0052764f9804fd005dfe298189ef8c7`.

Authority is [admission.md](admission.md) and the root-admitted task 69 JSON/Markdown. Source tracing and historical preparation RED are retained in [retained-success-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retained-success-next-slice.md), SHA-256 `f9b84fd1d78875610b9720f167932d2a4b9eef7d809f900e18400a2c1c248348`. AGENTS.md, MILESTONES.md and the complete Gomad README were read. This note adds no scope or outcome prediction.

## Exact edit and preservation check

Add exactly two syntactic instances of the existing statement:

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

The first goes inside `TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit`'s local `run` closure, after its final `SuccessBytesLimit` assignment and immediately before `return exploreWith(...)`. It supplies fresh final preparer/executor dependencies for each original invocation. The second goes in `TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts`, after final `SuccessBytesLimit` configuration and immediately before `exploreWith(...)`.

Only `tools/gomad3/runner/retention_test.go` is a product Touch. Preserve all original source-target mutation/hash/size, real copy and verification, shared-inode assertion, measured reduced-limit failure, two same-output artifact identities, seeds/output hashes, disk/journal/summary counts and full-file stored-byte totals. Helpers, imports, comments, assertions, fixture data, policies and metadata remain unchanged. A newly reached original failure returns to root for separate admission.

Bind and retain the complete admission BASE file before editing; its researched SHA-256 is `35a5599194d809a364751743318dc3c9e7304810bbeb8fd819e69dc8d3f14194`. The admitted worker must confirm that identity or report any root-approved baseline reconciliation. An in-memory checker must locate each named function, remove exactly one new assignment at the admitted location in each, reject any other insertion/deletion, and compare the reconstructed entire file with BASE byte-for-byte. Compare complete bytes, not selected body snippets or gofmt output. Record original, candidate and reconstructed hashes plus numeric comparison exit. Bind the checker executable and script bytes before and after this check. `gofmt -d` and `git diff --check` remain separate formatting checks; neither proves reconstruction. Check excluded sources against the frozen manifests.

## Focused selection

Selected-original regex, to run before either insertion and again after attachment:

```text
^(TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit|TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts)$
```

Exact focused regex for the unchanged controls plus both originals:

```text
^(TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit|TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts|TestRunFailsClosedWhenSuccessRetentionCountIsExhausted|TestRunRejectsSuccessfulRetentionWithoutReplayTranscript|TestPreparationDependenciesForwardRealFixtureInputs|TestPreparationDependenciesOperationErrorsRemainUnchanged|TestPreparationDependenciesFailuresStopAtOriginalStages|TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard|TestInjectionCharacterizationIsolatedPreparationDependencies|TestPortableProfilePublicGuardsRemainFirst)$
```

The two success-retention controls retain count-exhaustion and missing-transcript refusal. The four dependency controls retain real input forwarding, unchanged operation errors, original failure stages/attempt counts, public/executor-only defaults and the real bootstrap guard for a prepare-only attachment. Isolated preparation substitution remains rejected before coordination, with resume-preflight precedence. The portable public-profile control retains unsupported-host ordering. These exact names come from the retained research; their source is excluded from task 69 edits.

Before editing, retain actual preparation-stage RED for both unchanged originals at the admitted candidate. Historical combined66/67 RED is context only. Missing tools, compilation failure, early timeout or a result without both terminal test failures supplies no baseline RED. Retain baseline-control outcomes for comparison. After editing, observe the original real retention/disk/journal assertions and compare every actually emitted control outcome. No invented slash intermediates, subtest names, named-outcome totals or successor PASS counts are admissible.

## Proposed serial commands

These are proposals, not executed commands or lane authority. Root must first bind the actual worker checkout, candidate, tools and environment and grant one exclusive shared Go/build/lint/generator lane. Task 68 and siblings retain their own scopes. Freeze source throughout each command; all command handles must terminate before releasing or transferring the lane.

Use the retained task 68 capture recipe by reference: `../task-68/run-control-materialized.sh` at packet commit `524c092a3f6cbf5834895ae5ba7821d6fa610924`, SHA-256 `4d67887f4dbd41949e2f5b08280d0c7a9a08fb1bc1f28080059473cec64fb0b8`. Its read-only original location is `/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-fixtures/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68/run-control-materialized.sh`. It hardcodes task 68's cwd and packet: do not invoke it for task 69. The worker should prepare its own small task-bound adapter and receipt using this recipe, without copying task 68's histories or representing its results as task 69 executions.

Every launch must use `exec_command` with `login:false` and an outer `env -u BASH_ENV bash -c`, then an explicit `cd` to the exact admitted checkout. `BASH_ENV` otherwise resets cwd before a script's own unset can help. For the supplied PRIMARY, the exact launch prefix is:

```sh
env -u BASH_ENV bash -c 'cd /Users/stephan/Workspace/skunkworks/gomad/temporal && export SANDBOX_START_DIR="$PWD" && exec env GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS= TZ=UTC PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH" timeout --signal=TERM --kill-after=15s 600s COMMAND'
```

`COMMAND` below is an explicit argv proposal, not a shell variable to execute unchanged. For an isolated worker, root must replace the literal PRIMARY with that worker's exact absolute cwd before execution and bind the resulting launch bytes. The task-bound receipt wrapper must perform the task 68 recipe's ambient Go/CGO clearing, fixed cache/TMPDIR/offline-proxy setup, actual effective Go environment capture and source/tool hashing before launching the child. It must retain TERM/KILL timeout status, not turn that status into test RED. No downloaded, newly built or substituted tools are implied; verify the already available executable/version/hash identities first. Pin both `go` and `gofmt` to the existing Go 1.27.1 installation above and the existing lint/errortype tools below.

| Purpose | Child argv under that external bound |
| --- | --- |
| Original preparation RED | `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner -run '^(TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit\|TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts)$'` |
| Baseline controls / final focused | The same pinned `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner ./deterministicio -run` with the exact focused regex above; baseline controls may use that regex with only the two selected-original alternatives removed. |
| Formatting | `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt -d tools/gomad3/runner/retention_test.go`; require empty output, then byte reconstruction and `git diff --check`. |
| Affected ordinary host vet | `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 vet -tags test_dep ./runner` |
| Affected errortype | `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./runner` |
| Supported Darwin source-set vet | `env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 vet -tags test_dep ./runner` |
| Supported Linux source-set vet | `env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 vet -tags test_dep ./runner` |
| Baseline / final unfiltered Runner lint | `env -u BASH_ENV bash -c 'cd /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3 && exec /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --verbose --build-tags test_dep --timeout 10m --fix=false --config ../../.github/.golangci.yml ./runner'` |
| Required repository fast lint | `make lint-code-fast GOLANGCI_LINT_BASE_REV=2af15fcbc0052764f9804fd005dfe298189ef8c7 GOLANGCI_LINT_FIX=false ALL_TEST_TAGS=test_dep LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype` |

The escaped pipe in the Markdown table renders the actual regex's plain `|`; pass the exact text regex above as one argv element. All test commands carry `test_dep` and `-count=1`; integration tags are not applicable. The fast-lint BASE is the task 69 admission BASE, subject only to explicit root reconciliation of the worker's actual admitted baseline. Resolve/materialize required source cones, including `tests/mixedbrain/go.mod` and `go.sum`, before a frozen fast-lint gate; a sparse-checkout failure before analysis is inconclusive. No lint fixes or policy changes are authorized. Standalone errortype success does not prove configured aggregate analyzer reachability or an aggregate lint pass.

## Compact receipt and remaining ownership

Use one deduplicated source manifest and tool/environment manifests with immutable per-command raw log and JSON receipt, following the task 68 recipe. Bind task 69 admission/task/research bytes, authoritative PRIMARY owner spec, actual checkout HEAD and product diff, complete relevant consumed source/module/config inputs, BASE retention bytes, launch/wrapper and every checker script. Include Go/gofmt/lint/errortype, shell/timeout/Git/Make/JQ, Perl if used, and other actually invoked checker executables in the execution-time tool manifest. Capture exported settings plus effective `go env`, OS/architecture, exact cwd and explicit static overrides. Record exact argv array, numeric exit/signal/timeout status, UTC start/end, elapsed time, pre/post source/tool/checker identities and immutable raw-output hash. Pre/post checks must agree; publish receipts without overwriting earlier failures. A later hash or seal cannot retroactively bind a missing checker input.

Retain original and final unfiltered complete lint blocks and compare their header, source and caret bytes with exact line mapping. Root owns the future frozen ordinary comparison against combined66/67's actual 673 names and complete original-base RED50 blocks. Enumerate terminal outcomes only and permit changes only within actual root-admitted fixture scopes, including actually emitted sibling table children. Require zero unauthorized changes/missing originals and zero introduced/removed complete lint findings. Historical totals remain 389 PASS, 272 FAIL, 12 SKIP; no future total is forecast. Preserve aggregate nonzero exits and configured errortype reachability explicitly.

Applicable generated validation, architecture/private/public boundaries and broader supported source-set checks need current evidence or exact reconciliation of their actual consumed inputs, using retained task 68/task 67/task 65 receipts by reference. Equality of a broad fingerprint or a focused PASS alone supplies no inherited-gate pass. The two-assignment fixture scope does not itself authorize generation or changes to generated/runtime/toolchain inputs. Fresh independent integrated review remains root-owned.

Ordinary stock Go execution on the actual linux/arm64 development host is developmental host-source coverage. Darwin/arm64 and Linux/amd64 `vet` selections are supported source-set static checks, not execution on those native hosts. Deferred fn-128/fn-149 retain full native test-host, runtime, replay and soak requirements and remain unverified; no native revival, CI, PR or push authority follows. Reused caches, incomplete installation/header/libc identities and selective environment capture remain receipt limits; do not claim hermetic execution.

Unresolved before implementation: root must supply the exact worker checkout/candidate and serialized lane grant, verify available pinned tools and needed source cones, reconcile the admission BASE to actual unchanged retention bytes, and approve the task-bound capture/checker inputs. Original RED, final behavior, preservation, affected standards, inherited boundary reconciliation, root ordinary/full-block comparisons and independent integrated review remain unexecuted by this preparation agent.
