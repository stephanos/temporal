---
satisfies: [R1]
---
# fn-229-realization-ergonomics-shared-temporal.1 Runtime constructors and Candidate outcomes in umpire-wasm; Realize::matches removed

## Description
Add `From<A> for Transition<A>` (choice zero), `Projection::step` and
`Projection::steps`, `Candidate { path, outcome: Option<Outcome> }` inside
`Projection::Candidates`, and `AnswerPlan::new`. Remove `Realize::matches`;
the Runtime compares each candidate's expected outcome with the replayed final
edge outcome (`None` matches any). Adapt every trait implementation, including
stock/control and test worlds, and minimally adapt the activity Realization so
it compiles and behaves identically. Wire goldens, Go host parity, the dedicated
slice and all Model goldens stay unchanged. Preserve the prior matches behavior
in negative controls. If removal changes any actual judgment, keep matches and
record the concrete reason in DESIGN.md rather than weakening judgment.

**Touches:** `model/umpire/wasm/**`, `model/temporal/features/activity/temporal.rs`, `model/temporal/realization/tests/**`, `model/DESIGN.md`
**Quick:** `cd model && mise exec -- cargo test -p umpire-wasm -p umpire-temporal`; `cd model && mise exec -- cargo clippy -p umpire-wasm -p umpire-temporal --all-targets -- -D warnings`
**Full gates:** `make model-check`; `make model-check UMPIRE_BACKENDS=require`; `make umpire-check-model-slice`.
DESIGN.md is writable only for the specified evidence-backed compatibility fallback.
## Acceptance
- [ ] Constructors and Candidate present; matches removed from Realize
- [ ] Expected-outcome mismatch is Unexplained as before
- [ ] Wire goldens, Go parity, slice and goldens unchanged

## Done summary
Runtime constructors and typed candidate outcomes are integrated into umpire. The independent Model, all activity goldens and protocol wire bytes are unchanged. Unit/custom outcomes are preserved through generic Candidate/Projection, with equality needed only at runtime judgment/dispatch.

Worker isolated focused tests, clippy, default/required-TLC gates and all six live scenarios passed. Conductor then ran the required-TLC model gate and all six live scenarios again against integrated umpire with its own Cargo target; both exited zero.

Fresh same-family gpt-6.1-sol high review fanout f53b962102a24ce5913c038ee7ad3113 passed correctness, contracts and integration with zero findings. Receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-229-realization-ergonomics-shared-temporal.1.json. Reviewer attempted builds were sandbox-blocked; executed worker/conductor gates supply test evidence.

Tier: session (jev-unavailable(no_key)).
stage: impl-review - ran (codex:gpt-6.1-sol:high; same-family; three-draw SHIP)
stage: wave-join - ran (original commit d20fd403f1 reachable in umpire)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d20fd403f1c97a3f2cb1cdd68fcb3e4164a323ae
- Tests: baseline: green; cd model && mise exec -- cargo test -p umpire-wasm -p umpire-temporal (exit 0, pre-edit), baseline: green; cd model && mise exec -- cargo clippy -p umpire-wasm -p umpire-temporal --all-targets -- -D warnings (exit 0, pre-edit), EXPECTED RED: cd model && mise exec -- cargo test -p umpire-wasm --test constructors (exit 101; absent Candidate, From, Projection constructors, AnswerPlan::new), INCONCLUSIVE: make model-check UMPIRE_BACKENDS=require (exit 2; shared Cargo cache reused an emission test binary with another worktree's CARGO_MANIFEST_DIR and absent TLC jar; corrected with isolated target and local pinned jar), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2291-runtime timeout 600 mise exec -- cargo test -p umpire-wasm -p umpire-temporal (exit 0), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2291-runtime timeout 600 mise exec -- cargo clippy -p umpire-wasm -p umpire-temporal --all-targets -- -D warnings (exit 0), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2291-runtime timeout 600 make model-check (exit 0), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2291-runtime timeout 600 make model-check UMPIRE_BACKENDS=require (exit 0; real TLC fixture agreement and semantic controls), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2291-runtime timeout 600 make umpire-check-model-slice (exit 0; Complete, Fail, HostRefusal, Cancellation, CancelAfterCompletion, CancelBeforeWorkerStart), git diff 32d63cceeaeba286d6bb843e1fb415b82ab82452 HEAD --name-only -- 'model/**/expected/**' 'model/**/model.rs' 'model/DESIGN.md' (empty; Model and golden bytes unchanged), git diff --check (exit 0), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check UMPIRE_BACKENDS=require (exit 0, integrated umpire source), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-slice (exit 0, all six scenarios, integrated umpire source)
- PRs: