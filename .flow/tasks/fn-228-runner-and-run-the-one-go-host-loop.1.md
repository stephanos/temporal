---
satisfies: [R1]
---
# fn-228-runner-and-run-the-one-go-host-loop.1 Renames alone: umpire-wasm to umpire-runtime, umpire-tests to umpire-fixtures, Realization in prose

## Description
Rename the crate umpire-wasm to umpire-runtime and umpire-tests to umpire-fixtures across Cargo manifests, Makefile targets, scripts, guest imports, README, DESIGN, MIGRATE and Go host comments; replace the word binding with realization in Umpire prose. Add a gate grep that fails on the old names. Byte-identical goldens and wire goldens; make model-check and UMPIRE_BACKENDS=require pass; model-time pasted. Runs alone, no other task in flight.

### Required integration contract

Rename active crate identifiers and current documentation only. Update macro expansion snapshots only for crate-name spelling; preserve behavioral/wire goldens. Exclude model/archived, historical Flow records and Git history from the no-old-name check. Preserve fn-230 embedded Temporal guest support and refresh its artifact only through the existing builder if the rename changes build bytes.

## Acceptance
- [ ] No remaining umpire-wasm/umpire_wasm or umpire-tests/umpire_tests; gate grep in place
- [ ] Prose says realization, not binding
- [ ] All goldens byte-identical; gate passes; model-time pasted

## Done summary
Renamed the active crates and directories to `umpire-runtime` and `umpire-fixtures`, repaired imports and fixture consumers, and updated Model-to-system Realization prose. The scoped retired-name gate joins `model-check`, rejects retired crate/path spellings, and preserves archived source and exact historical evidence files.

R1 is satisfied. All behavioral and wire goldens remain byte-identical across 785 protected files. Two exact reviewed substitutions are allowed: the realization macro snapshot changes only `umpire_wasm` to `umpire_runtime`; Kettle's `omitted_action.stderr` changes only the relocated fixture path. The existing exhaustiveness test reproduced the diagnostic mismatch with exit 101, then all four controls passed after the one-line correction. The new retired-name/path controls were also observed red before their fixes and green afterward. Source-constant Binding APIs and prose keep their distinct meaning. Existing backend changes only repair fixture include paths.

`make umpire-guest` canonically refreshed the Temporal guest from 3,408,653 to 3,408,655 bytes (SHA-256 `7983677b3dfa25322757b89f19b21bc680208b42812a1c0f6c52b2630d618d42`). Freshness, wasm builds and both Go-host parity suites pass on Linux/arm64. The live slice package passed in 6.242 s. Live gaps passed in 396.610 s, with both required parent PASS entries and all 2,336 final probe PASS entries. A monitoring mistake consumed 187 intermediate RUN lines; the complete final PASS section and exact 879,197-byte classification golden remain intact. `live-gaps-observation.txt` records the incomplete intermediate transcript.

Baseline was green through composite evidence. The pre-edit default full run reached its final Temporal Go-host stage before timeout 600 (exit 124); that exact missing stage then passed separately in 12.009 s. An initial relative-cwd wrapper failed before Go started; the corrected absolute-cwd command supplies the green observation. A redundant warm baseline attempt was stopped at the conductor's direction (exit 143). The first post-edit required timing run also timed out at 600 s (exit 124) during fixtures and is inconclusive. Its diagnostics were preserved. The subsequent timeout-3600 required run passed all 18 stages. The explicit post-edit default `env -u UMPIRE_BACKENDS make model-check` passed in 435.704 s, superseding the earlier proposal to rely on required-mode coverage plus focused default-skip evidence. Its initial external timer launcher failed before Make because `/usr/bin/time` is absent; the actual run used Bash's timer.

Both full gate receipt attempts returned `NO_RECEIPT` because conductor-owned `MILESTONES.md` documentation edits were awaiting the closure commit. The test results above are actual successful executions; no green cache receipt or skipped gate is claimed. Formatting, strict workspace clippy, warnings-denied docs, focused rename tests and changed-package Go lint also pass. Model time exceeds the 60 s gate soft target; the 8.588 s inner loop meets 10 s.

The external report `model/design/native-symbolic-backend.md` appeared during this task. Required `git add -A` captured it, and the conductor directed its byte-for-byte preservation. It is separate design evidence, outside this task's implementation and acceptance proof. Root audit and verification reports are retained under `.flow/artifacts/fn-228.1/`. The conductor owns subsequent milestone progression.

Tier session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-23T06:20:39Z..2026-10-23T06:28:36Z] (codex gpt-6.1-sol high; same family; NEEDS_WORK -> SHIP)

The review correction was a mechanical diagnostic path update, so memory capture was skipped.

`UMPIRE_BACKENDS=require make model-time` passed. Pasted measurements:

```text
temporal-guest-freshness         40.800s
retired-names                     0.705s
timing-controls                   0.039s
guest-build-controls              4.021s
capability-controls               0.133s
source-boundary                   4.893s
protobuf-freshness                0.206s
kettle-protobuf-freshness         0.515s
temporal-protobuf-freshness       0.224s
fmt                               1.825s
clippy                            7.377s
docs                              6.799s
tests                            790.710s
wasm32-wasip1                     1.009s
core-wasm-clippy                  2.176s
wasm-guest                       72.622s
go-host                          43.231s
temporal-go-host                 13.218s
inner loop (edit+test)            8.588s
gate total                       991.181s  (soft target 60s)
inner loop                        8.588s  (soft target 10s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 74ec3bbe1c0e6417a9e16366a50ce5b3850cd9f5, aa0c9d9970300e5f568eceac238c793bafe03081
- Tests: INCONCLUSIVE baseline: timeout 600 make model-check (exit 124 at final Temporal Go host; earlier stages passed), INCONCLUSIVE baseline wrapper: bash: line 1: cd: model: No such file or directory (exit 1; Go did not start), INTENTIONAL_STOP: warm timeout 600 make model-check baseline attempt (exit 143 at conductor direction), cd model && CGO_ENABLED=0 mise exec -- go -C /Users/stephan/Workspace/skunkworks/umpire/temporal/model/temporal/hosts/go test -tags test_dep -count=1 ./... (baseline final stage exit 0), python3 -m unittest discover -s model/scripts/tests -p test_retired_names.py, /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/retired-names.sh, cd model && mise exec -- cargo fmt --all --check, cd model && mise exec -- cargo test -p umpire-runtime -p umpire-fixtures -p umpire-temporal-realization-macros --quiet, cd model && mise exec -- cargo test -p umpire-runtime --quiet, cd model && mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings, cd model && RUSTDOCFLAGS="-D warnings" mise exec -- cargo doc --workspace --no-deps --quiet, cd model && mise exec -- cargo clippy -p umpire-reference-kettle -p umpire-kettle-guest --all-targets --all-features -- -D warnings, cd model && RUSTDOCFLAGS="-D warnings" mise exec -- cargo doc -p umpire-reference-kettle -p umpire-kettle-guest --no-deps --quiet, cd model && mise exec -- cargo test -p umpire-reference-kettle --test exhaustiveness --locked (exit 101 before exact path correction, exit 0 afterward; all four controls), make umpire-guest, cd model && env -u UMPIRE_BACKENDS mise exec -- cargo test -p umpire-fixtures --test fixtures -- --nocapture, make umpire-check-model-slice (exit 0; package 6.242s), make umpire-check-model-gaps (exit 0; package 396.610s; 2336 final probe PASS entries; 187 intermediate RUN entries lost), make lint-code-fast GOLANGCI_LINT_BASE_REV=f76f622a7954434e91d0277dfad6e0fda97c9635, INCONCLUSIVE: timeout 600 env UMPIRE_BACKENDS=require make model-time (exit 124 during fixtures), timeout 3600 env UMPIRE_BACKENDS=require make model-time (exit 0; gate 991.181s; inner loop 8.588s), INCONCLUSIVE timing launcher: /usr/bin/time: No such file or directory (exit 127 before Make started), timeout 3600 env -u UMPIRE_BACKENDS make model-check (exit 0; 435.704s)
- PRs: