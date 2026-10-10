# Combined70 independent evidence review

The retained combined70 comparison is valid within its measured input domain. Findings are critical 0, important 0 and minor 1. The remaining minor finding concerns an omitted executable used by Make. Ordinary Runner and original-base lint acceptance remain OPEN/RED. This evidence review supplies no formal implementation-review, SHIP or Done verdict.

## Scope and identities

PRIMARY is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, observed at HEAD `e2695a790c2770da4f3b971bd2c0057d3e5276d2`. Execution ROOT is `/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence`, observed at frozen HEAD `b4047ac5451693e01b0fc220b58650b600fddcca`, with parent `fd9cfc4026db966a877597a9658a0af589f18aea`. The actual combined69 baseline execution HEAD is `c668243e0ecab6e4080aa7dad0810ccc2cedb08f`. Later primary documentation checkpoints do not replace that execution reference.

The authoritative primary owner spec remains SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. ROOT's historical owner spec remains separately bound to `0866b495ef6150bd0341aa904358f5de49ab4977e88da25c67e9df1531106f8a`. Candidate `choice_exploration_divergence_test.go` remains `5537ed38a236827a37b99a85e8584e886d6252d62e9d16311f514758034587c7`; removing exactly three inserted assignments reconstructs the entire baseline file `93b138931736026645f0c4f05940a95ef8fe301de36bc3dbab3bea9c4c4b23df`, with zero deletions.

I read AGENTS.md, the complete Gomad README, MILESTONES.md, the owner spec, task70, original and supplemental admissions, preparation notes, source-backed research, original and supplemental evidence reviews, the combined69 evidence review, the complete combined70 wrapper and relevant Make/lint routing source. I applied review-code and verification-before-completion within the assigned single-reviewer evidence scope and read the Flow-Next prose contract. Read-only `flowctl usage` preceded any consideration of Flow operations. Root retains lifecycle and commit ownership.

Requested reviewer routing is `gpt-6.1-sol` at high, in a fresh context and the same GPT family as the writer. Actual execution-model and effort telemetry is unavailable. I spawned no agents. My checks used read-only Python in-memory parsing/hashing, filesystem reads and Git-object reads through `login:false`, `env -u BASH_ENV bash -c`, an explicit PRIMARY `cd` and `PYTHONDONTWRITEBYTECODE=1`. I ran no Go, go-env, Make, build, test, lint, vet, generator, retained wrapper/checker, preflight, version probe, cache operation, cleanup, native gate, CI, PR, push, Git write or Flow mutation. My only file mutation is this report, outside every explicit seal.

## Finding

### Minor 1. Disclose the unmeasured Make find executable

**Location:** combined70 `run-gates.py` tool inventory and routing helper; ROOT `Makefile` lines 115 and 117; combined70 `tools.json` and `integrated-lint.log`.

**Failure scenario:** a consumer treats the strengthened current Make inventory as covering every directly invoked auxiliary executable, although Make executes `find` without measuring its selected route or bytes. That consumer would infer execution-time input stability beyond the observations retained in the packet.

The root Makefile eagerly evaluates `ALL_SRC := $(shell find ...)` and `ALL_SCRIPTS := $(shell find ...)`. The actual lint log also retains six sparse-directory `find` warnings, proving additional find expansions occurred in the outer and nested Make invocations. The 22-entry current inventory binds `/bin/sh`, its resolved `dash`, and selected/resolved `grep`, but contains no `find` entry or route. The baseline's 19-entry inventory also omits it. My present read-only selection resolves `/usr/bin/find`, SHA-256 `f42ef717cc43b84ffed01fb1e2b0fe2a01ab3425e18b976b189017bfb15f791f`; this current observation cannot identify its bytes or ignored-prefix selection during the earlier execution. There is no observed changed executable or affected test/lint result.

Retain all sealed evidence unchanged and disclose this additional selected-tool limit in an additive root reconciliation. Future captures should bind the actual selected/resolved find route and executable before and after Make. Current hashes, older worker tool manifests and a postcapture seal cannot repair this execution-time omission. This finding authorizes no rerun, gate or broad tool change.

## Immutable domains and independent byte checks

I rejected duplicate JSON keys, checked the exact fixed member sets and recomputed every member hash for both combined70 and combined69. Each seal contains exactly the prescribed 20 members and excludes itself and later review files. Metadata, per-command receipts, raw logs and derived comparisons match those hashes. Both seals' actual execution HEADs, input-binding hashes, pre-gate metadata hashes, source equality and terminal fields agree.

I independently verified the original task70 domain's 115 members and seal, and the supplemental domain's 44 members and seal. ROOT retains exactly 116 and 45 direct files respectively. Every member and seal matches PRIMARY byte-for-byte; PRIMARY's additional independent reviews remain outside those explicit domains. The captured ROOT task70 inventory contains 161 files across both domains, distinct from checkpoint added-path counts.

The original seal remains `83085c16386950d0fa3d7ddf27b18a60da7c5fb4213166c619545539a7016ead`; the supplemental seal remains `83bb1814654a71a323ab3cf7efa67082ad5a132579d850323f2971a23d31c0b6`. The original audit remains `9c693bb209af0eb2da4b579f29ac40f8471e44bc8fc332f42da2d63ca7d18034`, retaining its two important and one minor historical capture gaps. The accepted supplemental audit remains `70e1a64941889e3b6f74094f79a145005bd892dcad2975f88a63a00b50bd5150`, with 0/0/0 findings for current verification of 19 retained environment pairs and fresh shell-bound Make observations. I used that accepted review within its stated bounds, rather than repeating its seven-command audit. Neither successor evidence nor this review retroactively qualifies the original unmeasured helper or auxiliary executables.

Combined70 before/after source manifests are byte-identical. All 1,326 materialized entries match their current absolute/ROOT-relative bytes, with no required-absence or missing-tracked paths. All 1,126 product, build, module and lint-routing inputs independently match PRIMARY, ROOT and ROOT's execution-HEAD Git objects. All 161 captured ROOT task70 files match PRIMARY. The wider source inventory also binds the baseline packet, worker packet, task/admission/research and primary versus historical owner inputs.

All 22 recorded current tools match actual current bytes. All 19 historical listed tools remain equal to their current counterparts. Current additions are exactly `/bin/sh`, `/usr/bin/dash` and `/usr/bin/grep`; the find limit above remains separate. `/bin/sh` retains literal link `dash`, resolves to `/usr/bin/dash` and hashes to `87630eb41654f7888e28fa5ef3ed0a351682e939d382b9239a24a8aefe84aeb9`. The bound Make bytes identify version 4.4.1 without execution and hash to `b12eeb672d64e798b84f297c116651ccbb3ca726a108c74ffe9a60d85547315d`. The actual ROOT/.bin-prefixed grep search path selects `/usr/bin/grep`; literal and resolved routes agree with the retained current binding.

## Actual receipts and environment

Every summary receipt equals its separate receipt byte-derived JSON value. I checked source/tool/route pre/post hashes, source/tool/route equality, run-binding hash, raw-log hash, exact argv, numeric exit, numerical elapsed, UTC interval and terminal status. The four intervals are ordered, nonoverlapping and agree with elapsed times within one millisecond. The wrapper implements an external TERM900/kill15 bound. Its actual retained invocation names the frozen candidate, actual combined69 execution HEAD and exact baseline seal.

| Actual command | Exit | Elapsed seconds | UTC interval on 2026-10-10 |
| --- | --- | --- | --- |
| `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner` | 1 | 21.755353302 | 04:12:43.587593–04:13:05.342916 |
| Original-base `make lint-code-gomad3`, explicit `/bin/sh`, fixes disabled | 2 | 8.450382629 | 04:13:05.664203–04:13:14.114568 |
| Darwin/arm64 CGO0 static vet | 0 | 1.262632667 | 04:13:14.480495–04:13:15.743112 |
| Linux/amd64 CGO0 static vet | 0 | 2.005615667 | 04:13:16.077186–04:13:18.082568 |

The lint argv differs from combined69 only by `SHELL=/bin/sh`, retaining original base `951c5516e9e7b3066e7e069adda9565cfd68844c`, `GOLANGCI_LINT_FIX=false`, recorded golangci-lint 2.13.0/errortype paths and `ALL_TEST_TAGS=test_dep`. Both static commands cover `./runner`, `./internal/gomadtool/conformance` and `./runner/internal/execution`, with explicit supported GOOS/GOARCH and `CGO_ENABLED=0`. Both logs are empty with the SHA-256 empty-byte digest.

Actual-Go metadata records exit 0, terminal status, unchanged source and capture before the four commands; its hash matches the pre-gate binding. Effective settings are equal to combined69, stock Go1.27.1 on linux/arm64, default CGO1, `CC=gcc` and `CXX=g++`. Selected exported settings differ only in `SANDBOX_START_DIR`, reflecting the different isolated cwd. I recomputed every recorded settings/tool difference and actual argv comparison from the underlying manifests. Combined69 has no retained shell/grep route or explicit shell argv; current additions cannot identify those historical executables. Selected environment, reused caches, incomplete C headers/libc and tool-installation inventories supply no hermetic execution claim.

## Raw outcomes and complete diagnostic blocks

Independent raw JSON parsing found exactly 673 unique terminal names in each ordinary log, with one matching run event per name, no duplicate terminal events, no non-JSON records, no foreign package records, no added/missing names and an actual package-level failure in both logs. Counts change from 394 PASS, 267 FAIL and 12 SKIP to 421 PASS, 240 FAIL and 12 SKIP.

Exactly the 27 admitted actual emitted names change FAIL to PASS. They are the three parents and their 3 policy, 14 host-error and 7 mismatch-reason children, matching the complete name list in task70/preparation-note.md and every entry in outcome-comparison.json. Every baseline leaf retains the unsupported linux/arm64 preparation diagnostic. All other 646 outcomes remain unchanged; no intermediate slash name or new control is synthesized. This independently establishes the actual comparison, rather than deriving a pass total from the three source assignments.

I parsed every complete three-line lint header/source/caret block, retained multiplicity, and compared full text. All 50 blocks remain unchanged, with zero introduced/removed blocks, zero mapping gaps and no line relocation. For all 21 diagnostic-source paths, actual combined69 Git-reference bytes match its retained source-before hash. Each printed excerpt matches that bound source line. Exact matching-line projection into current ROOT reproduces every current block, and the independently reconstructed block lists match lint-comparison.json. Residual codes remain eight forbidigo and 42 staticcheck findings.

The current lint log records 55 host packages, `diff: 50/50`, 50 issues, 2,995 cache warning lines reporting no space left on device, and six sparse-directory find warnings. It is not warning-free or aggregate GREEN. Make stops at the golangci recipe at line505, before the subsequent errortype recipe, then reports the outer line498 failure. Integrated errortype remains unreached. Separate worker standalone errortype and diff-filtered fast-lint observations retain their independent reachability and comparison bounds.

## Key retained hashes and disposition

| Evidence | SHA-256 |
| --- | --- |
| Combined70 fixed20 seal | `6adbc92107cc4ec2df71171b12da5ca02e780d975d4bc3652c4603999da5998b` |
| Combined69 fixed20 seal | `71b566f5f17961cb370c3791a8008116241503b0cc31f4b00f9fa406c06fa0b5` |
| Wrapper | `4e6b60947ed52b9ee090823a78669be987ba4b8dddc23afe35422202085a4105` |
| Equal before/after source manifests | `d73b8d89ebd7a9435cf1b2885d6ac832bd0ed559af4a8eebb27ab65bea4304d6` |
| Current22 tool manifest | `d3c8742fc25dd64aa96ca9ed15581d2951b06504cc3458ee4f632f47ddf8a588` |
| Run binding | `de8508bb43da88ed2521b62ec64527d8c5d48ef6602e695ec98947c3b4ed2e58` |
| Ordinary raw log | `5c51d0d093fa218cb42d26b30bcb5c039c30c63272144f0a3ca59db2f3f28089` |
| Integrated-lint raw log | `58f7d66a25f2f336a8aa2defe1f199e029a4013e8f53688a3a6cf4485024b095` |
| Outcome comparison | `e1140dc6421243014b1cc5bd401fc5ce09611def7a98627d2ff9a77770c85d91` |
| Lint comparison | `388e7766653cf34170fef9c52dff372c5807dcffc608de04fa3c24d9564d3b20` |
| Actual-Go settings | `f2e8e378fa54635f4284f654abc883d4d1f5b44cb5a17acb51309d43d6852910` |
| Environment comparison | `83aa92212614bc4e009833062bf98e1c80a0d6061fec54eb272663db7edd12e4` |
| Summary | `cbeda687a42dc56b3dc5ae8a73ea0b80a4e4c5660f6609b6f037019c0674284b` |

Root reports sole-lane release at 04:14:15 UTC, after terminal wrapper session25550 and an attributed non-Go process check with exit0, actual46-row count and foreign bungee-lang Make PID1390727 excluded by measured cwd. The fixed20 packet retains the four terminal receipts; it does not retain that later process-check output or the host session completion itself. I treat the release, attributed snapshot and no-Go-after-release as explicit root handover observations, rather than retrospective continuous-process evidence.

The postcapture seals bind retained bytes after capture and create no retroactive pre-execution binding of outputs. Accept the recomputed ordinary/full-block comparison and listed source/tool/route bindings with the minor find limit disclosed. This review does not duplicate the separate source-correctness review. Root retains reconciliation, integration, commits and Flow lifecycle. Task70, fn-109.63 and fn-112.10 aggregate source acceptance remain OPEN/RED wherever requirements are unproved. Native fn-128/fn-149 qualification stays deferred and unverified. No supported-native full-host pass, runtime/bootstrap qualification, exact native replay, determinism/soak bound, CI, PR or push authority follows.
