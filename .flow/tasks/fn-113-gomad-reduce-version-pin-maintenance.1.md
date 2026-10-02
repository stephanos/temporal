---
satisfies: [R1, R2]
---
# fn-113-gomad-reduce-version-pin-maintenance.1 Baseline the pins and add the pin impact report

## Description
Re-measure the pin baseline (R1) and add a `gomadtool` subcommand that reports every pin a candidate `go.mod` invalidates (R2). This is the spec's early proof point.

**Size:** M
**Files:** new subcommand file under `tools/gomad3/cmd/gomadtool/`, `tools/gomad3/cmd/gomadtool/main.go`, a new report package or an addition to `tools/gomad3/upgrade/`, `tools/gomad3/deterministicio/adapter_registry.go` (read access only), fixtures, `MILESTONES.md`
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/upgrade/**, tools/gomad3/deterministicio/**, tools/gomad3/internal/compatibilitypack/**, tools/gomad3/architecture_test.go, MILESTONES.md, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/**]

### Approach
- Baseline: count each pin class and list the commands and hand edits one bump of each needs today. Correct the counts in the milestones Maintenance cost section if they differ. fn-110 task 1 already baselined patch and overlay counts; reuse them.
- Report inputs: one candidate `go.mod` with its `go.sum`, default the repository root module.
- Read adapter identities and anchors through the adapter registry and pack rules through the pack loader, so the report uses the values the build checks.
- Output path-free canonical JSON through the existing canonical encoder plus a human rendering. Exit 0 for no invalidated pin, 1 for at least one, 2 for invalid input, 3 for infrastructure failure. A pin that cannot be evaluated is reported unknown and counts as invalidated.
- Resolve modules outside the target module with a private module cache; `go mod download` inside a target module rewrites its `go.sum`.
- Cover: version bump, same version with a changed sum, module removed or replaced, and an indirect-only bump.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/deterministicio/adapter_registry.go:75` — registry construction from version data
- `tools/gomad3/deterministicio/sentry_adapter.go:7-18`, `:59-65` — anchor constants and definition shape
- `tools/gomad3/toolchain/version/version.json` — adapter module identities
- `tools/gomad3/cmd/gomadtool/main.go:22` — subcommand list
- `tools/gomad3/upgrade/upgrade.go:189` — dossier, the nearest existing report

**Optional** (reference as needed):
- `tools/gomad3/internal/compatibilitypack/generation.json` — pack output digests
- `.flow/memory/bug/integration/go-mod-download-inside-a-target-module-2026-09-28.md`
- `tools/gomad3/internal/canonicaljson` — canonical encoder

### Key context
- fn-105 task 8 adds downstream adapters and changes the baseline count; record the commit the baseline was taken at.
- A new package must satisfy the import allowlist in `architecture_test.go`.

## Acceptance
- [ ] Baseline of pin classes, counts, and per-bump manual steps retained under the spec's artifacts directory; milestone counts corrected if different
- [ ] The report lists invalidated adapters, pack rules, interception fingerprints, and clock-inventory references for a candidate `go.mod`
- [ ] A fixture bump of one adapted and one packed module yields exactly the expected entries, and the build's fail-closed check rejects the same pins
- [ ] Changed sum at the same version, removed module, replaced module, and indirect-only bump each have a test
- [ ] The target module's `go.mod` and `go.sum` are unchanged after a run
- [ ] Unknown pins are reported unknown, never unaffected; exit statuses follow 0/1/2/3
- [ ] `go -C tools/gomad3 test -tags test_dep ./cmd/gomadtool ./upgrade/...` and `make -C tools/gomad3 validate` pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
