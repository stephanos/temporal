---
satisfies: [R1, R2, R5, R6, R9]
---
# fn-229-gomad-symbolize-choice-sites-in-inspect.2 Inspect output: attach symbols to sites, decisions and divergences

**Touches:** tools/gomad/rust/gomad/src/cli/inspect.rs; tools/gomad/rust/gomad/src/evidence/inspection.rs (typed rendering wrapper only); tools/gomad/rust/gomad/src/evidence/symbolize/mod.rs (consumer visibility/plumbing only); tools/gomad/testdata/parity/**; tools/gomad/CLI.md; tools/gomad/internal/backends/wasm/CLI.md; tools/gomad/internal/backends/wasm/wasmhost/src/linked.rs; tools/gomad/internal/backends/wasm/wasmhost/src/linked/** (helper-owned validated read-only trace projection); tools/gomad/rust/gomad/src/fixture.rs; tools/gomad/rust/gomad/src/fixture_execute.rs (minimal feature-gated corpus producer extensions only).
**Semantic references only:** tools/gomad/internal/runner/inspect/inspect.go, inspect testdata, tools/gomad/CLI.md, tools/gomad/internal/backends/wasm/CLI.md

## Description
Integrate the task .1 Rust symbolizer after fn-232.5 and task .1, with no Go-host feature. Open the retained module once through the authenticated artifact handle, verify its digest before symbol reads, and share one bounded symbolizer and cumulative budgets across sites, listed decisions and both divergence decisions. Pre-resolve canonical unique requested sites in deterministic order into an immutable complete-result map, including unavailable results; every consumer reuses the stored object without budget resets. Resolve any whole-parse budget failure to symbol_table_too_large for all nonmissing sites before rendering, retaining site_missing for missing sites and discarding partial pre-failure results.

Obtain every site's original (kind, site_offset, site_missing) from the helper-owned validated raw-trace projection, extending its internal read-only linked API if needed. TraceSite's fingerprint/kind/count/maximum summary cannot recover the offset, and listed replay_decisions cover only branching decisions; neither is a substitute for the complete site input. Use those original keys to pre-resolve and join the immutable symbol map for every aggregate site, including single-alternative sites absent from replay_decisions. Do not invert fingerprints or add a duplicate Rust-host trace decoder. Keep internal site keys out of public inspect JSON except for the prescribed symbol addition, with no record or transport-wire schema change; pin a golden case whose site has no listed branching decision.

Populate the exact API Contracts JSON shape with fixed field order. Available symbols always carry inlined_from; physical fallback uses an empty list and omits unproved locations without claiming absence of compiler inlining. Render a proven full inline chain only under task .1's complete nonempty consensus; omit non-consensus file/line pairs independently. Preserve recorded runnable SiteMissing as unavailable: site_missing, with no invented creation/scheduling site. Use only the existing deterministic unavailable reasons from the spec; invalid metadata cannot leak parser failures or panic.

Extend the shared inspect golden/parity corpus with task .1's real pinned positive, inline and fallback fixtures and the actual stripped/absent-name/absent-pclntab distinctions. Keep symbols entirely in the inspect projection, never write the artifact or normalize record/failure/execution identities. Document field availability, physical fallback, limits and unchanged reasons in CLI.md.

If the existing foundation producer cannot freshly execute and retain the real Go module under its actual compiler/module identities, extend only that existing feature-gated corpus producer. Use the actual linked helper and authenticated publication/inspection path, preserving raw trace/tape and record integrity; do not relabel a Go-lineage artifact, rewrite trace identities, or substitute a fabricated site for the real R8 fixture. This adds no ordinary release entry point or public producer verb, and the fixture-only controls remain excluded from release/qualification.

Campaign receiving dependency: Go's structured divergence is campaign ExecutionInspection.divergence (runner/inspect/inspect.go), not artifact ChoiceProfile/ChoiceTrace. Preserve the helper-owned typed divergence's ordinal/reason/tape_records and optional expected/observed decision fields, including original kind/site_offset/site_missing; never recover these from standalone replay's textual path or invent an artifact/record/wire field. The artifact projection and reusable typed divergence enrichment may land first. fn-232.7/.8 must wire the actual authenticated campaign inspection/ledger consumer, resolving each execution against its own identity-matching retained module through the shared opener. Keep this task and R2 open until public inspect fixtures prove symbol attachment on both expected/observed decisions from validated campaign evidence, with repeated identity-preserving output. This is a receiving dependency under the existing batch policy, not a scope or acceptance waiver.

## Acceptance
- Golden inspect JSON covers every site, listed decision and divergence decision, preserves site fingerprints/kind/count/maximum alternatives, and is byte-equal across repeat runs/hosts for the same artifact.
- Real R8 inline attribution and the conflicting/unknown-pattern physical fallbacks appear consistently at every consumer; runnable missing sites retain site_missing.
- An actual retained -s module lacking the required name cross-check inspects successfully with no_symbols even if pclntab remains. Prove that metadata layout from the real fixture rather than assuming stripped pclntab; actual pclntab absence also yields no_symbols. Malformed/ambiguous metadata and name disagreement use symbol_mismatch, range/cap cases retain their named reasons, and a digest mismatch returns the existing open error before any symbol read.
- Before/after checks pin unchanged artifact bytes, record hash, failure signature and execution identity; no output evidence is written back. Missing required fixture modules fail, never skip.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
