---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.9 Framework/Temporal split: inventory gate steps and couplings, decide the framework entry point


## Description
Decide where the framework's gate lives and how it stands alone (spec R7), before .3, .5 and .6 build on the Makefile. A short design note; no code moves here.

**Size:** S
**Files:** `model/design/framework-gate.md` (new).
**Touches:** [model/design/gate-speed.md, model/design/framework-gate.md]

### Approach
- Classify every model step in `Makefile:1270-1315`, every script in `model/scripts/`, every `model/dev/xtask` module and the `model/build/` caches as framework or Temporal. Examples: `temporal-guest-freshness`, `source-boundary`, `temporal-protobuf-freshness` and `temporal-go-host` are Temporal; `retired-names` is mixed, since it excludes a Temporal path.
- List the couplings with file:line: the 45 `"../../../.."` repo-root paths in `model/umpire`, `Scratch::new_in(root, "model/build/tlc")` (`backends/process/src/support.rs:52`) and the Apalache equivalents, the framework include of `dev/xtask/src/apalache_pins.rs`, protoc generators from the root `.bin/` (`scripts/proto-go.sh:31,41`), and `model-check`'s `$(API_BINPB)` prerequisite.
- **Delegated decisions (record evidence and the selection in the spec's Decision Context; the owner directs following recommendations without another approval):**
  1. Entry point: an `umpire` xtask subcommand (`cargo xtask gate`, one-language rule from fn-247) or a `model/umpire/Makefile`. Recommend the xtask.
  2. Workspace: `model/umpire` as its own Cargo workspace with its own lockfile, with the Temporal workspace depending on it by path (strongest standalone proof), or one shared workspace with the framework gate selecting `umpire` packages. Recommend its own workspace.
- Name where shared repo files go: `rust-toolchain.toml`, `rustfmt.toml`, `clippy.toml`, `deny.toml`, `mise` pins, `.config/nextest.toml`.
- Adopt the recommended framework-owned Rust xtask and separate workspace unless the inventory proves a smaller directory-only solution. Classify generic structural agnosticism separately from Temporal's existing deny-vocabulary policy (`temporal/testing/boundary.rs:8-25`); keep Temporal vocabulary outside framework source and state how planted-name/dependency controls avoid rejecting their own literals. The generic scanner can accept externally supplied policy; do not introduce Temporal knowledge into it.
- Name .13's workspace/tools/pins/identity migration boundary separately from .10's orchestration/protobuf/guest split. Include filesystem-based hygiene without Git, lockfile closure, explicit test targets, guest source remapping and which worker owns refreshed Temporal guest bytes.

### Investigation targets
**Required:**
- `Makefile:1270-1315`, `model/scripts/`, `model/dev/xtask/src/`
- `model/Cargo.toml:1-60` — members and workspace dependencies
- `model/umpire/backends/process/src/support.rs:40-60` — cache root
- `.flow/specs/fn-247-one-language-for-the-framework-a-rust.md` — xtask conventions

## Acceptance
- [ ] Every gate step, script, xtask module and cache is classified framework or Temporal, with each coupling listed by file:line.
- [ ] Entry point/workspace selections and the no-Temporal policy ownership are recorded with evidence under delegated judgment.
- [ ] The note names the target layout that .10 implements, and what .3, .5 and .6 must place in it.

## Done summary
Selected the independent framework workspace, Rust gate entry point and exact standalone policy-data exception in `model/design/framework-gate.md`. The note includes the complete 19-stage/12-shell-script/one-ignore-file/nine-xtask-source inventory, declared cache/tool/generator ownership, all 45 exact repository-root joins, and concrete .13/.10/.3/.5/.6/.11/.12 placement and proof boundaries.

Tier: AGENTS implementer gpt-6.1-sol/medium; judge unavailable(no_key), explicit project routing retained.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

baseline: none (the fn-254 spec defines no Quick commands for this design task; owner waived fresh timing baselines and dispatch forbids production builds/tests/benchmarks). No production validation result is claimed.

Verification: final `verify-note.py` exited 0 with 19 timed stages, 13 files under model/scripts (12 shell scripts and one ignore file), nine xtask source files, all 45 exact root joins, 52 source citations and two Markdown links checked against the actual source. Early validation exposed the omitted scripts/tests/.gitignore and incorrectly qualified/out-of-range citations; corrected before the final pass. `git diff --cached --check` exited 0 before both commits. Clean tree and task-only changed-path check passed. Retained checker and result: `.flow/tmp/fn254-task9/verify-note.py` and `verification.log` in the conductor checkout.

Final `flowctl gate classify --base 3d54e19b1bfbce799ada92e4d5ffc7e8db636797` returned exit 1, `FULL: unmatched: model/design/framework-gate.md`. This is a conservative classifier observation, not a pass or tier-B classification. Per explicit dispatch no production suites were run for this reversible prose task. No invented GATE_SKIPPED lines or full-gate receipts were emitted.

Conductor actions: persist the exact Decision Context entry below through flowctl at integration; align .13's touch list with required framework scripts, DSL README control, Go target fallback and consumer documentation migrations. The design note makes this required admission explicit. Existing Temporal deny-vocabulary policy is unchanged; the exact .txt declarative policy exception is necessary to retain the standalone R7 name ban without causing the old Temporal .toml scan to reject its policy literals. .13 intermediate root coverage checks BOTH workspaces. No Model/conformance semantics or feature route changed.

Status remains in_progress. No impl-review, done, tracker/shared-spec write, integration or production build/test/benchmark was run. No attributable command is running; worktree retained for conductor review fixes.

Exact Decision Context text for integration:

2026-10-25, fn-254.9 delegated selection: adopt framework-owned Rust `cargo xtask gate`
at `model/umpire/dev/xtask` and independent `model/umpire/Cargo.toml`/`Cargo.lock`;
Temporal remains at `model/` with `exclude = ["umpire"]` and path dependencies inward.
The complete stage/script/module/cache inventory and all 45 root joins are recorded in
`model/design/framework-gate.md` at the inventory checkpoint named there. Copy tiny local
policy/tool configs; framework pins/tools/caches/nextest/budgets have no parent dependency.
Keep framework tool root distinct from caller guest workspace/lock/output; .13 owns remaps,
locked closure, current-input reproduction and refreshed Temporal guest publication. .13's
intermediate root gate checks both workspaces; .10 owns final orchestration and mixed-control
splits. Preserve R7's standalone planted-name/dependency rejection using exactly
`dev/gate/policy/framework-boundary.txt` as declarative gate-policy data exempted from its
own name scan, with no directory/source-code exemption. The existing Temporal vocabulary
policy stays Temporal-owned; generic libraries/scanners/compiler/runtime remain agnostic.
Filesystem hygiene and the standalone copy need no Git; use declared current inputs and
preserve path/symlink, raw-byte/status and dependency-containment checks. No Model or
conformance policy changes are authorized by this design.

Scratch-verifier correction after integration: the original checker FAILED in the main
checkout (exit 1) because raw filesystem enumeration included 10 ignored legacy Python
bytecode cache files. Preserved failure: `root-before-fix-failed.log`. Replaced raw
enumeration with `git ls-files --cached --others --exclude-standard -z` for source
inventories, retaining tracked files (even if ignored) and all nonignored current
candidates. This scratch checker uses Git; the designed standalone gate still uses
filesystem hygiene without Git. The same corrected checker exited 0 in both main and
the clean assigned worktree, with identical 13/9/19/45/52/2 inventory totals; see
`root-after-fix.log`, `worktree-after-fix.log` and `verifier-repair.json`. All 10 legacy
bytecode cache files remain present. No source/report/Flow state or commit changed.
Corrected verifier SHA-256: `3edc316562fae3e1d3ff169e836a5c687789f9cc4441636ee8002d054db7c4aa`.

Conductor integrated the note, persisted selected decisions and expanded .13 ownership through Flow. Same original plan-review session returned SHIP for the selected layout/policy. Three-axis implementation review finalized SHIP at633c2d2a91 with no findings. Joined native retired-name check passed. The initial main-checkout inventory probe included ignored legacy Python bytecode; corrected source-only candidate enumeration passed in main and clean worktree, with original failure/caches retained. These are documentary checks, not production gate/standalone-runtime proofs; .13/.10/.11 retain those obligations.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 83bf067850244e10b9bc56765bcab794a4ca5c1a, f1b485d9967b1eec15b7ff99408de9929bfc7bd0, 633c2d2a91
- Tests: python3 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task9/verify-note.py, git diff --cached --check, FAILED: original scratch verify-note.py in main checkout exited 1; raw rglob included ignored legacy __pycache__/*.pyc (root-before-fix-failed.log), PASS: corrected scratch verify-note.py in main checkout exited 0 (root-after-fix.log), PASS: same corrected scratch verify-note.py in clean assigned worktree exited 0 (worktree-after-fix.log), python3 .flow/tmp/fn254-task9/verify-note.py (corrected main and clean-worktree source inventory: pass), UMPIRE_XTASK=<root-target>/xtask-bootstrap/debug/xtask bash model/scripts/retired-names.sh (exit0, no compilation)
- PRs: