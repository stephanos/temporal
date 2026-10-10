# fn-109.70 independent evidence review

The frozen packet preserves the recorded focused results and its sealed bytes. Its complete execution-binding claim needs qualification. Findings are critical 0, important 2 and minor 1. The findings concern capture coverage, with no evidence that a checker, tool or observed product execution changed. This independent evidence audit supplies no formal Flow implementation review, Done or SHIP verdict.

## Scope and identities

The audited worker is `/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence`. Its observed HEAD and BASE remain `fd9cfc4026db966a877597a9658a0af589f18aea`. The dispatch named primary HEAD `6e43763f1ab2de6745ee1ecebdf7791a7fc2441b`; my closing primary read observed `c4a3cc7e5b951003650af879943593e8f98c152c`. Root owns those later primary checkpoints. This audit binds the frozen worker, without rewriting its intentionally empty commit range or `commits[]`.

The candidate `choice_exploration_divergence_test.go` hashes to `5537ed38a236827a37b99a85e8584e886d6252d62e9d16311f514758034587c7`. The BASE file hashes to `93b138931736026645f0c4f05940a95ef8fe301de36bc3dbab3bea9c4c4b23df`. The authoritative primary owner spec remains `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`; the historical worker spec remains separately identified and grants no current authority.

I read primary AGENTS.md, the Gomad README, MILESTONES.md, the current primary owner spec, task admission and preparation note. I used read-only `flowctl usage`, `brief`, `show fn-109.70` and `cat fn-109.70`, plus the Flow-Next prose contract. Task 70 remains in progress. The existing independent source review hashes to `d104004c3fbf7fd028061766aa87cc9990169eb2154e82ca713b4cf637ad1828`; I did not duplicate its correctness review.

Requested reviewer routing is `gpt-6.1-sol` at high, in a fresh context and the same GPT model family as the writer. Actual execution-model telemetry is unavailable. Requested routing is therefore not independently verified runtime identity. I spawned no additional agents and ran no Go, environment probe, build, lint, vet, generator, installation or native gate. My only file mutation is this authorized primary report, outside the worker seal.

## Findings

### Important 1. Qualify the environment helper's capture window

**Location:** frozen `run-control.sh` lines 60-67 and `worker-summary.md` lines 29-31.

The wrapper records `post_source` and `post_tools`, then records the second environment capture, then invokes `/usr/bin/perl check-environment.pl`. Both source manifests contain the checker hash, and both tool manifests contain the Perl executable hash. Those observations bracket the principal command, but both precede the environment-proof helper's execution. After the helper returns, the wrapper hashes only itself and the proof output; it never captures the checker or Perl executable again. It uses the helper's exit to set `pre_post_match_exit=0`.

A consumer reading every checker as having complete pre/post execution coverage would therefore accept a stronger historical binding than the wrapper measured. The packet seal and later receipt auditor cannot extend that capture window. The checker has measured pre-execution identity, and its proof output is retained and hashed; post-helper checker/tool stability is unproved. There is no observed changed helper execution.

**Minimal correction:** retain the original packet unchanged and add a separate amendment stating this limitation. A successor read-only Perl proof can consume the original 38 immutable raw environment captures, bind the proof script, actual Perl executable and every input before and after execution, and independently prove equality for all 19 pairs using only the exact single ephemeral GOGCCFLAGS mapping. This needs no Go execution and must be labeled successor verification of retained bytes. It cannot repair the historical helper window. The successor adapter should capture checker/tool identities after its own helper finishes.

### Important 2. Bind the Make recipe shell for current standards evidence

**Location:** frozen `run-control.sh` lines 27-29, `final-fast-lint.json`, `final-validate.json`, root Makefile and `tools/gomad3/Makefile`.

The tool manifest measures `/usr/bin/bash` and `/usr/bin/make`, but contains no `/bin/sh` or resolved `dash` executable. The two Makefiles do not assign SHELL, and the retained argv supplies no SHELL override. GNU Make's ordinary Unix recipe route therefore uses `/bin/sh`; the inspected current filesystem resolves that path to `dash`. This route executes the validation recipes and the fast-lint recipe. Hashing Bash does not identify that separate interpreter, and the historical selected environment captures do not record Make's resolved recipe shell.

The raw Make logs still establish the recorded exits and displayed recipe observations. They do not establish the historical shell executable's SHA-256. A current hash or seal must not be presented as that missing execution-time identity. The packet's explicit exclusion of full tool installations and libc does not identify the recipe interpreter it directly used.

**Minimal correction:** disclose the original shell-binding limit. If current still-owned Make standards require complete significant executable routing, run only the current fast-lint and generated-validation commands again under a successor adapter that passes and records `SHELL=/bin/sh`, binds both the symlink resolution and resolved executable before/after, and retains actual argv, source/tool/environment identities and raw outputs. Existing original RED, focused outcomes, standalone errortype, direct host/source-set vets and boundary results need no rerun solely for this Make-shell gap. A successor success must not relabel the original records as fully bound, and no successor outcome is forecast here.

### Minor 1. Include auxiliary executables in the successor tool inventory

**Location:** frozen `run-control.sh` line 41 and root Makefile line 95.

`retain()` invokes `rm` when an identical content-addressed capture already exists. The identical pre/post source and tool manifests exercise that branch. Root Makefile also eagerly evaluates `MODULE_ROOT := $(lastword $(shell grep -e "^module " go.mod))`. Neither `rm` nor `grep` appears in the 26-entry tool manifest. These omissions narrow inventory completeness. The deletion target is a newly created capture scratch file, so this observation supplies no evidence that an unknown cache or user file was removed.

**Minimal correction:** include the resolved `rm` and `grep` executable identities in the successor adapter's pre/post inventory and state that the original packet did not measure them. Fold this into the successor capture correction; it does not independently require a broad product rerun.

## Independently verified packet observations

I verified the seal's exact member set and every member hash. The packet contains 115 sealed members and its seal, exactly 116 files. Seal SHA-256 is `83085c16386950d0fa3d7ddf27b18a60da7c5fb4213166c619545539a7016ead`. Summary SHA-256 is `99518e5efa1be6eb5e7e6035e11286bc361a0c2db130bf2b091f6280cd1ea707`; evidence SHA-256 is `c968752ad33fcd3e6a1d65989d382bd91400f3ae291374fd595e27af5959ad9f`.

All 19 actual JSON receipts match the handover fields and their receipt, raw-log, environment-proof, wrapper and content-addressed manifest hashes. Sixteen principal commands record exit 0 and three record exit 1. All record numerical elapsed times, explicit cwd, worker HEAD, timeout 900 seconds and stability 0. UTC end-minus-start matches each elapsed value. Every principal command ended before the handover's `2026-10-10T03:32:43Z` lane release. Start/end fields cover the principal child command, not the later capture/proof/receipt-writing work. The worker's final receipt auditor checks 18 earlier receipts; it is not a self-audit of all 19.

The nine source manifests contain 11,537 entries matching current bytes, 648 explicit ABSENT entries still absent, and two historical entries for the BASE divergence file. Those two discrepancies exactly match the admitted BASE/candidate file hashes. All 26 measured executable hashes still match. Every principal Perl checker and its consumed file argv is present with its actual hash in the relevant source capture. Wrapper SHA-256 `92b6ce6c28726556dccc33bd4c2a3c7dff4087a250a126bb3fb461a9adebff53` is the same in every receipt. These successful checks retain the capture-window and interpreter limits above.

I independently recomputed equality for all 19 raw environment pairs. Each raw record remains retained under its distinct content hash. Normalization replaces exactly one `/go-build[0-9]+=/tmp/go-build` fragment within the GOGCCFLAGS JSON value; every other captured byte is equal within each pair. Explicit supported-platform/CGO overrides remain in the principal argv, while the raw captures describe the outer environment. Selected environment plus full outer `go env` is not a complete inherited-process environment or a hermetic execution claim.

The first validation manifest contains all 132 materialized `tests/` files, including 113 top-level `_test.go` files. Their current hashes match. The Make validation log shows version/protocol/boundary checks, compiler-test checks, patch/overlay and script validation, compatibility checks, the tagged host-pack test and qualification-manifest check. Its principal exit is 0 and captured source inputs remain unchanged. The shell identity limitation remains applicable.

I reran the read-only whole-file preservation, outcome comparison and complete Runner diagnostic-block checkers. Preservation reconstructs the entire BASE file after removing exactly three admitted assignments and confirms three insertions, zero deletions and no excluded product-path changes. The observations are:

| Retained observation | Principal exit | Verified result |
| --- | --- | --- |
| Unchanged original three | 1 | 27 FAIL, three actual parents and 24 leaves; every leaf contains the developmental linux/arm64 preparation refusal |
| Unchanged controls | 0 | 67 actual named PASS outcomes |
| Final original three | 0 | The same 27 names PASS |
| Final focused selection | 0 | Exactly the same original/control union, 94 PASS and zero FAIL/SKIP |
| Configured Runner lint before/after | 1 / 1 | Six complete header/source/caret blocks unchanged, zero introduced/removed |
| Formatting/diff/preservation | 0 | Empty gofmt diff, clean recorded diff check and whole-file reconstruction |
| Host vet, standalone errortype, supported-source vets | 0 | Affected commands explicitly recorded; darwin/arm64 and linux/amd64 use explicit CGO0 argv |
| Five boundary tests | 0 | All five actual selected test names PASS |
| Generated validation / fast lint | 0 / 0 | Recorded validation completion and diff-filtered 55-package fast lint, subject to the shell-binding finding |

The complete Runner block digest is `0343dd7822baa18fb03d36e7917bd5c69ff5bd5b6e483f7a0f3c592ae52cb31b`. Baseline Runner lint retains 1,559 warning lines, final Runner lint 1,555, and fast lint 2,995 plus six missing sparse-directory find warnings. These logs are not warning-free. Standalone affected errortype was reached; fast Make output does not independently expose its internal errortype argv. Fast lint's `diff: 50/0` filtering and exit 0 establish no aggregate lint GREEN or complete original-base RED50 comparison.

Early environment captures record the generator-cache path absent; later ones record the literal task69 retained-success cache destination. The authorized setup time and no-overwrite checks were supplied by root and the handover; the 115-member packet contains no separate setup command receipt. I verified the environment transition and preserved its raw bytes without inferring qualification of cache contents. Likewise, principal receipt times support completion before release, while the process check and exclusive-lane release are handover observations rather than an independently retained process snapshot in this packet. Cache contents, full tool installations, C headers/libc and nonselected inherited environment remain unqualified.

## Evidence disposition

Retain the complete original packet and this original report. Correct the strong all-checkers/tool wording through a separate dated amendment, with successor evidence addressing the measured gaps. The immutable raw focused observations, exact source reconstruction and inherited Runner RED6 blocks remain usable within their stated bounds. The supplemental raw-environment proof can establish current independently bound verification of all retained pairs without rerunning Go. Fresh narrowly scoped Make receipts can supply current shell-bound standards evidence if root authorizes the serial lane.

Root's combined70 ordinary comparison, complete original-base lint-block comparison and broader static batch remain pending for this audit. I infer no future PASS total, aggregate GREEN or full acceptance from the worker's selection. Still-owned source requirements remain open wherever red or unproved. Native fn-128/fn-149 qualification stays deferred and unverified; no native full-host pass, replay qualification, soak bound, CI, PR or push authority follows.
