---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.54 Check doctor reports and classify output failures

## Description
Own exactly the three unchecked doctor stdout writes retained in the CLI: JSON report, text headline and per-check row. Task53's separately reviewed source-progress commit is8177ddec76f30c1db027f282358a213176855ef8; its task-53/independent-review.md and actual original-base RED99 are the admission inputs. No task53 Done dependency is required because its affected source lint consumes this correction. Task21 consumes the new owner's source proof. Root owns admission, lifecycle, independent review and commit; the fresh worker owns implementation/tests/evidence in the serial lane.

**Touches:** [tools/gomad3/cmd/gomad/internal/cli/cli.go, tools/gomad3/cmd/gomad/internal/cli/doctor_output_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-54/**]

### Explicit correction and preservation

The CLI contract already assigns doctor report failures status3. Correct ignored output errors to return3 immediately at each of the three existing writes. This explicitly changes failure behavior: an error at the headline or a check row stops later rows and the footer, rather than ignoring the failure and continuing toward availability status0/1. A JSON report failure overrides availability status0/1. Preserve exact healthy bytes, write boundaries/operands, check order, newline/spacing, report construction, probing, installation resolution and completed side effects. Leave the existing checked footer and every earlier validation/inspection status and zero-output outcome unchanged. No fallback stderr write, retry, helper, callback, framework, seam, library, policy, pin, native guard, API, old-test or unrelated comment changes. Replay stdout and application error casing remain excluded.

Read AGENTS.md, README.md, MILESTONES.md, CLI.md and nearby output tests/conventions. Before production edits bind complete healthy text/JSON bytes and independent expectations through exported Run with explicit absolute toolchain/artifact paths. Use a real read-only os.File for syscall.EBADF, not a sentinel. A test-only selected-attempt writer permits other writes to succeed so headline/row failures provide meaningful RED: BASE ignores these failures and returns1 on Linux/arm64; the corrected path returns3 and stops after its failing attempt. An always-failing writer is insufficient RED because the old checked footer already returns3. Test JSON, headline, first/interior/final rows, checked footer and earlier invalid flag/argument/root errors. Keep retained healthy successful prefixes, attempt counts, empty stderr and availability outcomes exact. Expectations must not call the production Check implementation.

All three source paths are reachable on Linux/arm64: Check honestly reports unsupported host and still renders its report. Healthy status0 remains a supported-host coverage gap, never spoof host or native success. Existing writeDoctorFixture is planning input, but old writeDoctorCommandFixture pins go1.26.4 and its available-status assertion cannot establish current-host coverage. Retain genuine tool identity and disclosed gaps. No native revival, PR, push or CI authority.

### Verification

Reuse exact-source unchanged prerequisites; run focused meaningful RED before and focused GREEN after, full ordinary affected CLI tests with honest native-only coverage accounting, affected vet/standalone errortype, required architecture/public/purity and both supported source-set static checks, fresh check-only generated validation, format, actual unfiltered affected configured lint, actual make lint-code-fast against the admitted base and original-base make --trace lint-code-gomad3 against951c5516e9e7b3066e7e069adda9565cfd68844c, both FIX=false. Serialize shared gates, freeze source throughout, retain raw commands/exits/elapsed/source/tool hashes, measure exactly3 findings removed and0added. No suppression/filtering/changed baseline or weakening old assertions. Use pinned stockGo1.27.1, -tags test_dep -count=1, existing local cache/file-proxy recipe and a fresh private /tmp directory. Keep handover compact and reference old evidence.

Formal acceptance remains open while required affected/integrated source gates are red. A fresh independent source-progress review may license a separate progress commit only; no formal SHIP or Done follows from a partial lint delta. Original R18/R19, first-baseline/preservation and deferred fn149/fn128 gates remain unchanged.

### Retained source progress - 2026-10-09

Root retained the three-check correction and its fresh independent review in
task-54/handover.md, evidence.json, source-proof.json and independent-review.md.
The meaningful before RED becomes 13 focused passes. Actual affected lint
falls from 5 to 2 and integrated lint from 99 to 96, removing three findings
and introducing none. Required red source gates and coverage gaps remain open;
the task stays in_progress for a separate source-progress commit.

## Acceptance
- [ ] Exactly three doctor stdout results are checked, reporting failures immediately return3 under the explicitly admitted stop behavior, and healthy bytes/statuses, completed operations, earlier errors and existing footer handling remain unchanged. No old tests or unrelated source changes.
- [ ] Independent exported-Run healthy controls and genuine selected-attempt EBADF controls establish meaningful before RED/after GREEN for JSON/headline/rows, retain footer/earlier-error controls, and disclose status0/native gaps without spoofing.
- [ ] Frozen-source focused/ordinary tests, vet/errortype, architecture/public/purity, both supported static source sets, fresh generated validation, format and actual unfiltered configured/fast/original-base lint evidence establish exactly3 removed and0introduced. Required source-gate gaps stay open.
- [ ] Fresh independent review, root verification and separate task commit feed task21. Formal acceptance stays open on red required source gates; original preservation/first-baseline/native transfers remain in force, with no publication authority.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
