---
satisfies: [R4]
---
# fn-113-gomad-reduce-version-pin-maintenance.3 Refresh invalidated packs in one command and remove unselected variants

## Description
One command that runs `discover`, `review`, and `generate` for every request a bump invalidates, stopping at approval, and removal of pack variants nothing selects (R4).

**Size:** M
**Files:** `tools/gomad3/cmd/gomadtool/compatibility_pack.go`, `tools/gomad3/internal/compatibilitypack/authoring/*.go`, `tools/gomad3/internal/compatibilitypack/{packs,requests,reports}/`, `generation.json`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/internal/compatibilitypack/**, tools/gomad3/Makefile, tools/gomad3/qualification/corpus/**]

### Approach
- Add a `refresh` subcommand beside the existing five. It runs on a checkout where the bump is already applied, so the candidate versions are the ones the working tree resolves. It takes the invalidated request set from the task 1 report run against that checkout.
- Each request is discovered in its own target module. `discover` needs `--working-dir`, and requests name a module and package, not a directory. Move the request-to-directory mapping the Makefile qualification list holds today (repository root, qualification corpus, fixtures) into one checked-in table that both the Makefile target and `refresh` read. A request with no mapping is invalid input.
- Discover into scratch first and compute the review digest of the fresh evidence. Write the request only when its evidence changed. `authoring.Discover` clears approval unconditionally today, so refresh must not call it in place.
- Skip predicate: a request is done when its stored approval matches the review digest of the freshly discovered evidence. An approval of older evidence does not count. Approval stays per request through the existing `generate --approve-review`.
- Depends on task 2: both edit `cmd/gomadtool` and the pack tree, and task 2 reports the libc-bound packs this command repairs.
- A request for another platform is reported as not evaluable on this host and left untouched.
- Stale variants: `modernc-libc-xsys-v041` is selected only by `internal/compatibilitypack/testdata/v041/go.mod`. Before removing a variant, show that no qualified module, corpus module, or test fixture selects it, and retain that evidence. Delete the pack, request, and report together and update the Makefile qualification list.
- fn-105 task 26 rebound stale packs by hand; reuse its audit.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/cmd/gomadtool/compatibility_pack.go:20-34` — subcommand dispatch
- `tools/gomad3/internal/compatibilitypack/authoring/discover.go`, `review.go`, `generate.go` — steps to chain
- `tools/gomad3/Makefile:103-106` — pack qualification list
- `tools/gomad3/internal/compatibilitypack/testdata/v041/go.mod` — the only selector of the v041 variant

**Optional** (reference as needed):
- `tools/gomad3/qualification/corpus/go.mod:13-16` — selector of the v047 variant
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.26.md` — earlier manual rebind

### Key context
- fn-109 task 8 edits `compatibility_pack.go`; check its state first.
- Pack validation rejects any pack admitting `os/exec`, `os/signal`, `os/user`, `plugin`, or `runtime/cgo`; refresh changes none of that.
## Acceptance
- [ ] `refresh` runs discover and review for every invalidated request in that request's mapped module and stops with one review digest per request
- [ ] A test with requests from two different modules shows discovery used each module's candidate versions; an unmapped request is invalid input
- [ ] Starting from two previously approved, now-invalidated requests: approving one refreshed request and rerunning leaves that one approved and reports only the other
- [ ] An approval that matches older evidence is never treated as current
- [ ] Other-platform requests are reported and untouched
- [ ] Each removed variant has retained evidence that nothing selects it; pack, request, report, and Makefile entry go together
- [ ] `make -C tools/gomad3 validate compatibility-pack-qualification` passes on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
