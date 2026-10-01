---
satisfies: [R7, R8, R6]
---
# fn-110-gomad-minimize-the-runtime-patch.5 Qualify the final candidate and publish measurements and guidance

## Description
Qualify the integrated final candidate on this host, publish the final measurements and build identity, and update milestone status and maintainer regeneration guidance. No toolchain input changes here: a failure is reported against the task that introduced it, not patched over.

**Size:** M
**Files:** `docs/research/gomad/GOMAD_PATCH_SIZE.md`, `.plans/GOMAD_MILESTONES.md`, `tools/gomad3/README.md`, `tools/gomad3/CLI.md`
**Touches:** [docs/research/gomad/GOMAD_PATCH_SIZE.md, .plans/GOMAD_MILESTONES.md, tools/gomad3/README.md, tools/gomad3/CLI.md]

### Approach
- **Identity** — confirm `make -C tools/gomad3 generate validate` is clean, then record the final build key from `tools/gomad3/.toolchain/build-key` next to the baseline key from task 1. Explain that patch and overlay bytes changed the identity, that baseline artifacts keep their original toolchain binding, and that all evidence below was recorded fresh under the final key.
- **Darwin qualification**, in this order, against the per-workload "before" outcomes from task 1:
  1. `make gomad3` and `make -C tools/gomad3 test` (harness, toolchain, interception, host, overlay, world, builder, live-capability, runtime, upstream tiers)
  2. process simulation: `TestRootProcessSimulationUsesRunnerTransport` (`integration` tag) and the entropy test `TestProfileEntropyIsIndependentOfScheduleSeed`
  3. `make gomad3-integration-test`
  4. `make gomad3-smoke-qualification`
  5. `make gomad3-qualification` — `tools/gomad3integration/qualification/temporal.json` runs seeds 11 and 17 with `repeat: 2`; require same-seed repeatability and exact replay where the manifest requires them
  6. `make -C tools/gomad3 core-qualification`
- Compare outcome by outcome. An existing `intermittent` D14 expectation stays as written; a new divergence, watchdog, capacity failure, or weakened expectation leaves acceptance incomplete and is reported, not reclassified. The DTrace clock audit needs root and is CI-supplied; record it as not run.
- **Linux** — `linux/amd64` gates (`.github/workflows/gomad3.yml` jobs `host-tools-linux`, `core-linux`, and `gomad3-smoke.yml`) run only after the user pushes. Record R7's Linux gate and R4's Linux equivalence as incomplete, with the exact jobs that would close them.
- **Final measurements** in `GOMAD_PATCH_SIZE.md`: baseline `-U3`, final extracted `-U3`, final canonical `-U1` (bytes, lines, SHA-256), edited upstream files (20 → 19), added/deleted source lines, and overlay bytes/lines before and after, so moved code is visible. State both reductions independently and compare with the 27,845 / 20,352-byte prototypes as guidance, not a quota.
- **Milestones** — update the fn-110 row at `.plans/GOMAD_MILESTONES.md:33` and the Status line of "Runtime patch minimization (fn-110)" (`:320-356`) with what is done and what is incomplete. Leave the Open findings and the D12/D14 rows unchanged.
- **Regeneration guidance** — `tools/gomad3/README.md:571-578` and `tools/gomad3/CLI.md:466-474`: the canonical patch has one context line and comes only from `patch-regenerate`; overlay additions require the descriptor allowlist and `make generate`; the pinned check follows the descriptor. Edit only the sentences concerned. If the generated upgrade guide (`descriptor.go:287`) needs the same note, change the generator and run `make -C tools/gomad3 generate` rather than editing generated output.
- Superseded intermediate build directories from tasks 2-4 may be removed now if disk is tight, per the constraints below; list what was removed.

### Investigation targets
**Required:**
- `Makefile:165-225`, `tools/gomad3/Makefile:55-175` — native entrypoints
- `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml` — the Linux gates
- `.plans/GOMAD_MILESTONES.md:93-110,131-157,320-356`
- `tools/gomad3/README.md:540-580`, `tools/gomad3/CLI.md:450-475`
- `docs/research/gomad/GOMAD_PATCH_SIZE.md` — baseline and intermediate sections written by tasks 1-4

### Quick commands
```bash
make -C tools/gomad3 generate validate && cat tools/gomad3/.toolchain/build-key
make gomad3 && make -C tools/gomad3 test
(cd tools/gomad3 && GOWORK=off .toolchain/bin/go test -tags test_dep,integration -count=1 -run TestRootProcessSimulationUsesRunnerTransport ./runner/internal/execution)
make gomad3-integration-test
make gomad3-smoke-qualification
make gomad3-qualification
wc -c -l tools/gomad3/toolchain/runtime/go1.27.1.patch && git show HEAD:tools/gomad3/toolchain/runtime/go1.27.1.patch | wc -c -l
```

### Key context

**Working constraints (apply to every fn-110 task):**
- No `git commit`, `git add`, stash, or worktrees. The user owns commits; leave changes in the working tree and report them. Earlier fn-110 tasks may therefore be uncommitted working-tree changes — do not revert them.
- Host is `darwin/arm64` only. `linux/amd64` gates cannot run here: record every Linux-dependent check as **incomplete**, never as passing. Cross-compilation is not Linux evidence.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-105 D12 (Linux replay divergence) and D14 (Darwin `TestSignalWorkflowTestSuiteChasm`) keep their owners and dispositions. Do not edit qualification expectations to get a passing gate.
- Existing comments move with their code, unchanged. Add no allocations, host reads, dependencies, CLI flags, or capability grants.
- Always pass `-tags test_dep`. In testify code use `require`, not `assert`; plain `testing` files keep their existing style.

**Patch editing workflow (governed, no hand-edited hunks):**
1. Extract the verified archive `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` (SHA-256 in `toolchain/version/version.json`) into a scratch directory under the gitignored `tools/gomad3/.toolchain/fn-110/`.
2. `go -C tools/gomad3 run ./cmd/gomadtool patch-materialize --root="$PWD/tools/gomad3" --source-root=<scratch>/go`
3. Edit upstream files in `<scratch>/go/src`. Do not copy overlay files into the candidate: `changedFiles` in `toolchain/patch_regenerate.go` rejects added source paths.
4. `go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3" --candidate-root=<scratch>/go`
5. `make -C tools/gomad3 generate validate`, then `make -C tools/gomad3 toolchain` (the build runs the archive-based overlay collision check at `toolchain/build.go:176`).

The descriptor requires `patch_allowlist` and `overlay_allowlist` to equal the checked trees exactly (`toolchain/version/descriptor.go:114-160`), so a task that adds an overlay file or empties a patched file updates `version.json` and regenerates in the same task.

## Acceptance
- [ ] The full Gomad gate, process-simulation, entropy, integration, smoke, core, and seeds 11/17 Temporal qualification ran on darwin/arm64 under the final build key, with before/after outcomes listed per workload and no expectation or disposition edited
- [ ] Final `-U3` is smaller than baseline `-U3`, and canonical `-U1` is smaller than final `-U3`; patch, edited-file, added/deleted-line, and overlay figures are published separately with digests
- [ ] The new build identity is explained; baseline artifacts keep their original binding and no artifact was relabeled
- [ ] Milestone status and README/CLI regeneration guidance are updated; D12/D14 text is unchanged; every existing comment, negative test, and retained contract from the spec's Edge Cases is confirmed present
- [ ] `linux/amd64` gate (R7) and Linux `-U3`/`-U1` equivalence (R4) are stated as incomplete with the CI jobs that close them; nothing incomplete is reported as passing


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
