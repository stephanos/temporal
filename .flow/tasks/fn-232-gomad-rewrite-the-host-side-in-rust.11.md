---
satisfies: [R10, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.11 install, doctor, discover, plan and recover in Rust; overlay invoked via the pinned Go toolchain

**Touches:** tools/gomad/rust/gomad/src/cli/{install.rs,doctor.rs,discover.rs,recover.rs}, tools/gomad/rust/gomad/src/install/{manifest.rs,toolchain.rs}

## Description
This task follows fn-232.10; the redundant transitive .6 edge does not permit starting before .10. Plan validation internals from .8 are reused when exposing the public plan verb here.
Port installation with a new manifest version: pin compiler, helper, runtime sources and configuration by digest, invoke the Go toolchain overlay generator through the pinned toolchain, never download; discover validates identities without opening build inputs; doctor, plan and recover as documented; preserve the WASM refusal for native-only analyze without porting native; Go-version manifests refused. Resolution order `--installation`, env, manifest beside binary, no fallback from an invalid one. Use the running binary's linked helper implementation by default. Preserve optional `--helper PATH` grammar, but authenticate an explicit path against the same linked Rust implementation identity and refuse any mismatch; never switch execution to that external path. Document the authorized required/default change from Go's mandatory external helper without changing flag grammar or refusal statuses.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Fresh install pins the same unchanged compiler/runtime/input digests and actual new Rust implementation identities; all owned public verbs have parity and golden fixtures; missing or mismatched tools refused; Go-version manifest refused; discover works after build inputs are removed; omitted --helper derives the linked implementation, a matching explicit path is admitted and a mismatched/legacy helper is refused, with CLI documentation and golden coverage for this explicit default exception.

## Done summary
R10/R13/R14 accepted for the exact declared installation corpus and immutable integrated fixture 63c43b4525d16dae1aa1432ddbff8424516c8595c31b3f67c37eaadcda133500 (source82ba157e16). Full15 regeneration and full16 independent strict checking each exited0 with316 direct public commands plus authenticated receipt/scoped-plan/recovery child lanes; four independent attempts agreed and all12 raw input digests remain unchanged. Golden654693efd83b2e9e49d0b420d623a5f364ca24c1ba58527be5b59f3ffaf4ff1b is committed in64f90e84c3. Fresh independent same-family source review has no remaining findings and sampled all8 new mutated/different-role helper refusals.

Install/discover/doctor/analyze refusal, exact compiler/runtime/configuration admission, linked default/identical helper copy and mismatches without external execution, removed-input discovery, explicit/env/sibling precedence with no fallback, zero-guest portable seed preparation, actual module-metadata tamper refusals, authenticated matching/empty scoped bases, stripped-debug refusal, and genuine Rust/Go seed+round recovery/no-op/lock/error cases passed. Retain literal Go-v3/Rust-v4 module/overlay identities and Go's actual lossy source reporting separately; do not normalize them or claim cross-lineage byte parity. The new typed Go-compatible configuration/descriptor decoder is covered by actual native Go/Rust cases and aliases/null/default/arity negatives.

Root current ordinary release frome8754d1cf8 built0, excludes fixture-preparation-open and reaches ordinary qualification input admission; its immutable81657d713cbb33ff3ca82bf8e92c224030097bb11d341fe59ab613e4556cb5a6 and raw receipt remain retained. Source integration94 library tests, shared oracle projection regression1 and driver4 tests passed. Single handover .flow/artifacts/fn232-11-installation-2026-10-22.md retains exact argv/statuses, earlier failures, source fixes and all raw paths. Task13/R12 still owns final CURRENT integrated-candidate identity refresh and complete frozen runtime/workload/replay qualification. No Go retirement, performance/support claim, CI dispatch, push or PR.

stage: impl-review - ran (fresh gpt-6.1-sol high; same-family independent context; no remaining findings)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Autonomous continuation design (2026-10-22)

Resume after .10 acceptance. Extract the existing `campaign/prepare.rs`
installation representation and resolution into `install`; separate manifest
admission from compiler/runtime/source admission so discovery survives removal
of build inputs. Preserve current Rust-manifest consumers while adding the
operator installation grammar, exclusive mode-0600 publication, pinned offline
overlay invocation and linked implementation admission. An explicit helper must
authenticate the same compiled implementation; it never selects execution.

Expose seed `plan --output FILE` through the existing preparation/retention
owners with zero guest executions and an authenticated portable module receipt.
This receives fn-222's prepare-only gate. Receive fn-230.1's scoped choice plan
through the existing scope validator and base-resolution implementation; record
its necessary base execution explicitly. Keep measured qualification separate.
Expose `recover` through the existing seed and round journals without executing
guests. Repeated recovery must become a no-op.

The Go WASM CLI refuses public `plan`, `analyze`, and installation-selected
`doctor`; preserve those actual refusal lanes. Rust's positive public plan is
the already-authorized receiving behavior, compared semantically with Go public
exploration rather than falsely claiming a successful Go public plan. Cover
installation flags/configuration, resolution precedence, missing or changed
inputs, linked-helper defaults and mismatches, removed build inputs, publication
collision, plan receipts/scope and repeated recovery through fresh public CLI
fixtures, raw authentication and repeated byte-identical golden regeneration.

## Evidence
- Commits: 942dadb6b2, 06e45c2cfb, 9c97d59ff7, 565ae1cbb3, cbdb0f5737, 21adc20beb, e5ea438c90, 46bfeb127c, a3a9222b51, f08f060e2d, 0ab3817c06, fd132cc66d, d43f137340, 42f7e93276, 76f565c9eb, 64f90e84c3
- Tests: full installation lane15 regeneration: exit0,316 direct commands plus authenticated child lanes; /home/agent/wt/evidence/fn232-install-integrated-15.log, full installation lane16 strict existing-golden check: exit0,316 direct commands plus authenticated child lanes; /home/agent/wt/evidence/fn232-install-integrated-16.log, root rehashed all12 actual input paths in both final roots: unchanged, focused plan02, recovery01 (Rust+Go seed/round), stock-plan01, resolution01, private-copy01 and helper-negatives: exit0, ordinary release build/no-feature fixture exclusion and ordinary role input admission: exit0 verifier; /home/agent/wt/evidence/fn232-integrated-ordinary-release/receipt.json, integrated cargo +1.99.1 test --locked --release -p gomad --lib:94passed; shared projection regression1passed; driver SQLite4passed, fresh independent source review and8 actual new helper refusal receipts: no remaining findings
- PRs: