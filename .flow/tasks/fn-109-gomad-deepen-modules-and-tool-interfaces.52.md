---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.52 Check remaining maintainer diagnostics while preserving primary outcomes

## Description
Finish the remaining 59 unchecked stderr writes in the maintainer command adapters at source base `eb82ea59a4`. The retained unfiltered inventory is task10's `diagnostic-writes-20261009/final-configured-lint.stdout`: compatibility_pack.go 24, diagnostic.go 5, main.go 27, upgrade.go 3. Existing task46 owns successful stdout reporting; task50 owns five already repaired generator diagnostics; task51 owns the corrected usage fixture. Preserve those corrections and all original acceptance and dependencies.

This correction is admitted to unblock fn112.10's retained source lint and fn109's source delivery, on the owner-selected gomad branch. Root owns scope, review, lifecycle and commits. One fresh task worker owns implementation, tests and the serialized Go/build/lint/generator/cache lane. No native revival, publication, PR, push or CI is authorized. Existing native qualification remains deferred under fn149/fn128 and supplies no current-candidate pass.

**Touches:** [tools/gomad3/cmd/gomadtool/main.go, tools/gomad3/cmd/gomadtool/compatibility_pack.go, tools/gomad3/cmd/gomadtool/diagnostic.go, tools/gomad3/cmd/gomadtool/upgrade.go, tools/gomad3/cmd/gomadtool/terminal_diagnostics_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-52/**]

### Approach

- Check exactly the inventoried fmt calls without changing their format strings, arguments, attempted bytes or order. A diagnostic-write error preserves the original already selected primary status. Do not add fallback, retry, output framework, callbacks, suppression, flag, dependency or policy/pin changes.
- Terminal calls immediately followed by a fixed return may return the same status on write failure. Calls before classified errors (build-key and injected toolchain failure) must resolve the original classification regardless of writer failure. Keep injected failure status 86 and invalid-input/operation precedence unchanged.
- Preserve compatibilityPackRefreshStatus after its error diagnostic and the request parser's original Request{} / status-2 pair. diagnostic-diff's DiffDiagnostics error is unreachable through validated public trace inputs; check that statement and retain its runtime coverage gap without adding a seam.
- checked-run's mismatch reports and conformance's failed-case output form ordered sequences. Attempt every originally eligible diagnostic even after a preceding write fails; preserve report files, payload eligibility, primary status and completed operation order. Retain the first write error separately if needed, using local variables and existing style. Do not short-circuit these sequences.
- Retain source proof for every one of the 59 statements and all unaffected originals. Exercise real public run paths with healthy/counting writers and genuine read-only-file EBADF controls. Bind exact healthy diagnostic literals and call order before editing production. Include usage, invalid input, missing inputs, build-key classification, checked-run child mismatch with stdout/stderr payloads and completed report files, and diagnostic-diff failed summary reporting. Use established fixture patterns, genuine tiny valid inputs and real processes; no production injection seam or fake native success.
- Test both success and failure behavior; test repeated writer failures and later successful diagnostic attempts where sequences exist. Identify unavailable branches explicitly; source inspection is not native execution proof. All transferred native obligations retain their original owners.
- Reuse current source-identical receipts where valid, and freeze source throughout checks. Check generator inputs before edits. Do not rewrite historical evidence or touch protected user .turbo files. Keep handover small, with source/tool identity, commands, exits, elapsed times, meaningful failing lint control, passing preservation controls and review pointer.

### Quick commands

Use the established pinned stock Go1.27.1 and existing lint2.13.0/errortype with the task10 packet's explicit local cache/module/proxy setup and a fresh private overlayfs TMPDIR under /tmp; every Go test includes -tags test_dep -count=1. Use actual supported-platform static checks, with portable execution truthfully labeled developmental linux/arm64.

Run focused TestTerminalDiagnostics preservation controls before and after production edits; full ordinary ./cmd/gomadtool plus ./qualification/soak ./qualification/set; affected vet and standalone errortype; root architecture/public/purity tests including both supported static source sets; check-only make -C tools/gomad3 validate; formatting; unfiltered configured lint of the affected packages; make lint-code-fast with fixes disabled against the task source base; and actual make --trace lint-code-gomad3 with fixes disabled against original integrated base 951c5516e9e7b3066e7e069adda9565cfd68844c. No lint exclusions or gate changes are admitted. The remaining integrated findings continue under their existing correction owners; this task cannot waive them or claim the interrupted integrated errortype stage passed.

## Acceptance
- [ ] Exactly 59 admitted unchecked stderr results are checked. Original bytes, operands, error/status precedence, conditional attempts, report ordering, completed publications and cleanup remain unchanged; unaffected source statements and tests retain their behavior.
- [ ] Meaningful configured lint RED59 is retained; independent healthy and genuine EBADF controls through public run pass on the baseline and corrected candidate. Sequence controls prove later diagnostic attempts and completed checked-run files survive earlier reporting failures. Classified failure statuses remain original.
- [ ] Full ordinary affected command/soak/set tests, focused regressions, affected vet/errortype, architecture/purity/public boundary checks, both supported static source sets, check-only generated validation, format and unfiltered affected configured lint pass on frozen candidate with retained command/source/tool identities. Native unavailable paths remain explicit and under the existing native owners.
- [ ] Actual changed-base fast lint and original-base integrated lint are retained, with exact remaining findings and errortype reachability. Removing this task's 59 findings does not waive unrelated correction acceptance or fn112.10's integrated source gate.
- [ ] Fresh independent review finds the bounded correction and its evidence acceptable; root verifies the retained gates before flowctl done and a separate task commit. Task21 consumes the correction. Original R18/R19 acceptance and native deferrals are unchanged.


## Done summary
Checked all59 remaining maintainer stderr results while preserving original
statuses, diagnostic operands/bytes/order, subsequent reports, completed files
and cleanup precedence. All83 fmt expressions and normalized original production
bytes are preserved. Original tests and generator inputs are unchanged.

Focused preservation controls passed before/after; ordinary cmd/soak/set passed
112 top-level and411 total tests with no failure/skip. Affected vet/errortype,
architecture/public/purity, both supported static source-set vet, fresh generated
validation, format, affected configured lint and actual fast lint passed.
Original-base integrated lint remains RED145, exactly59 fewer than prior204,
with zero added/changed residuals and integrated errortype unreached. That gate
remains open under its other owners and fn112.10; this correction grants no waiver.

Fresh independent same-family review accepted the bounded correction with no
findings. Root verified raw receipt/tool/source hashes and test events. Evidence,
individual runtime gaps, review and handover are in task-52/. Native qualification
stays deferred/unverified under fn149/fn128; no native pass or bound is claimed.

stage: impl-review - skipped(policy: broader original source gate remains red; bounded independent task source review accepted, independent-review.md)
stage: code-review - ran (requested reviewer gpt-6.1-sol/high; actual execution telemetry unavailable)
stage: plan-sync - skipped(config: disabled)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: go test -tags test_dep -count=1 -json -run "^TestTerminalDiagnostics" ./cmd/gomadtool, go test -tags test_dep -count=1 -json -run "^TestTerminalDiagnostics" ./cmd/gomadtool, go test -tags test_dep -count=1 -json ./cmd/gomadtool ./qualification/soak ./qualification/set, go vet -tags test_dep ./cmd/gomadtool ./qualification/soak ./qualification/set, go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./cmd/gomadtool ./qualification/soak ./qualification/set, /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --verbose --build-tags test_dep --timeout 10m --fix=false --config=/Users/stephan/Workspace/skunkworks/gomad/temporal/.github/.golangci.yml ./cmd/gomadtool ./qualification/soak ./qualification/set, go test -tags test_dep -count=1 -json -run "^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestDomainModulesDoNotExportWireFraming|TestPublicPackagesDoNotExportForwardingAliases|TestHostPackageVet)$" ., make -C tools/gomad3 validate, make lint-code-fast GOLANGCI_LINT_BASE_REV=eb82ea59a4db14a19ce07c9ced0ee3f8e4bcddd6 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype, make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype, test -z "$(gofmt -l tools/gomad3/cmd/gomadtool/main.go tools/gomad3/cmd/gomadtool/compatibility_pack.go tools/gomad3/cmd/gomadtool/diagnostic.go tools/gomad3/cmd/gomadtool/upgrade.go tools/gomad3/cmd/gomadtool/terminal_diagnostics_test.go)" && git diff --check
- PRs: