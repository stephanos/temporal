---
satisfies: [R9]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.20 Reconcile architectural guidance with the delivered owners and interfaces (fulfils fn-105.5 D5)

## Description
Stage 6, R9 (F8). Most of the original F8 premise is already fixed: verify and cite that, then document what this spec changed. This task is the single owner of fn-105.5 (D5, origin brief `.flow/tasks/fn-102-gomad-architecture-consolidate.6.md`); close fn-105.5 by reference afterwards.

**Size:** S/M
**Files:** `tools/gomad3/ARCHITECTURE.md`, `tools/gomad3/SPEC.md`, `tools/gomad3/README.md`, `tools/gomad3/CLI.md`, `tools/gomad3/TUTORIAL.md`, `.plans/GOMAD_MILESTONES.md` (status lines only), `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`.
**Touches:** [tools/gomad3/*.md, .plans/GOMAD_MILESTONES.md, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md]

### Approach
- Already fixed since the assessment; needs evidence, not rewriting: `SPEC.md:176` names both `darwin/arm64` and `linux/amd64`; `ARCHITECTURE.md:17-23` and `:659-662` state both platforms; the closing paragraph (`:687-691`) no longer classifies choice tracing as research; `GLOSSARY.md` was deleted by fn-111 and its terms live in SPEC. Reuse fn-111's retained evidence (`flowctl show fn-111-gomad-consolidate-vocabulary-and-update`, its tasks .1/.2 and `.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/`) rather than re-deriving it. If fn-111 is still open, cite its state and do not edit the same passages concurrently.
- New documentation owed by this spec: the options owner and coordinator envelope; the preparation owner with its two operations; the Go-command seam and its two output contracts; the installation description; private executor injection and the Artifact reference/handle split, as intentional Go interface changes sourced from `go-interface-changes.md`; the generated simulation-time protocol under "Binary protocol ownership"; the simulation progress lifecycle owner under "Process arbitration and model evidence"; backend-specific handles; the architecture checks under "Maintenance gates".
- Keep four claims separate everywhere: capability support, same-seed repeatability, exact replay, and CI expectation matching. Use existing SPEC requirement IDs; add none.
- Residual findings stay visible and unchanged in meaning: Linux and Darwin replay divergence (D12/D14), suites without exact replay, host-clock escapes such as `MemStats.LastGC`. A structural refactor resolves none of them; say so.
- No performance or support claim without a measurement retained under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`. Describe each change in the present tense as current behaviour; delivery history belongs in the milestones file.
- Update only this spec's status rows in `.plans/GOMAD_MILESTONES.md` (work-tracking row and the fn-109 section status); leave other specs' text alone.

### Investigation targets
**Required:**
- `tools/gomad3/ARCHITECTURE.md` (whole), `tools/gomad3/SPEC.md:140-240`
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`
- `.flow/tasks/fn-102-gomad-architecture-consolidate.6.md`
- `.plans/GOMAD_MILESTONES.md` sections "Open findings", "Deep modules and tool interfaces (fn-109)", "Vocabulary and documentation (fn-111)"
- `tools/gomad3/toolchain/version/version.json` (supported platforms as generated fact)

### Quick commands
```bash
cd tools/gomad3
grep -n 'darwin/arm64\|linux/amd64' SPEC.md ARCHITECTURE.md README.md
GOWORK=off go test -count=1 -tags test_dep . -run 'TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership'
make validate
cd ../.. && flowctl show fn-105-gomad-follow-ups-deferred-scope.5 && flowctl tasks --spec fn-111-gomad-consolidate-vocabulary-and-update
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] `documentation-evidence.md` cites, with file and line, the already-correct platform, choice replay/exploration and backend statements, and links fn-111's evidence instead of duplicating it.
- [ ] Architecture guidance describes the delivered owners (options, preparation, command seam, installation description, generated simulation-time protocol, progress lifecycle, backend handles, architecture checks) using existing requirement IDs.
- [ ] The intentional Go interface changes (executor injection, Artifact reference/handle) are documented with their replacements.
- [ ] Capability support, repeatability, exact replay and expectation matching are stated as separate claims; residual D12/D14 findings and known clock escapes remain documented as open.
- [ ] No unmeasured support or performance claim, obsolete delivery-state claim, or failure described as qualification success is present; local links and code fences resolve.
- [ ] fn-105.5 is closed by reference to this task (one owner).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
