Qualify the integrated final candidate on both native qualified platforms, publish the final measurements and build identity, and update milestone status and maintainer regeneration guidance. The current linux/arm64 development host supplies neither native gate; source-only input preparation does not start this task or close its dependencies. No toolchain input changes here: a failure is reported against the task that introduced it, not patched over.

**Size:** M
**Files:** `docs/research/gomad/GOMAD_PATCH_SIZE.md`, `MILESTONES.md`, `tools/gomad3/README.md`, `tools/gomad3/CLI.md`
**Touches:** [docs/research/gomad/GOMAD_PATCH_SIZE.md, MILESTONES.md, tools/gomad3/README.md, tools/gomad3/CLI.md]

### Approach
- **Identity** — confirm `make -C tools/gomad3 generate validate` is clean, then record the final build key from `tools/gomad3/.toolchain/build-key` next to the baseline key from task 1. Explain that patch and overlay bytes changed the identity, that baseline artifacts keep their original toolchain binding, and that all evidence below was recorded fresh under the final key.
- **Darwin qualification**, when a native darwin/arm64 host or CI is available, in this order, against the per-workload "before" outcomes from task 1:
  1. `make gomad3` and `make -C tools/gomad3 test` (harness, toolchain, interception, host, overlay, world, builder, live-capability, runtime, upstream tiers)
  2. process simulation: `TestRootProcessSimulationUsesRunnerTransport` (`integration` tag) and the entropy test `TestProfileEntropyIsIndependentOfScheduleSeed`
  3. `make gomad3-integration-test`
  4. `make gomad3-smoke-qualification`
  5. `make gomad3-qualification` — `tools/gomad3integration/qualification/temporal.json` runs seeds 11 and 17 with `repeat: 2`; require same-seed repeatability and exact replay where the manifest requires them
  6. `make -C tools/gomad3 core-qualification`
- Compare outcome by outcome and preserve the actual current D12/D14 dispositions, including D14's resolved Darwin expectation. A new divergence, watchdog, capacity failure, or weakened expectation leaves acceptance incomplete and is reported, not reclassified. The DTrace clock audit needs root and is CI-supplied; record it as not run until its source-bound native evidence exists.
- **Linux** — native `linux/amd64` gates (`.github/workflows/gomad3.yml` jobs `host-tools-linux`, `core-linux`, and `gomad3-smoke.yml`) need a native host or source-bound CI after the user pushes. Record R7's Linux gate and R4's Linux equivalence as incomplete until that evidence exists. The current unsupported host cannot close the corresponding Darwin gates either. Do not dispatch, push, or relabel historical CI results without authorization.
- **Final measurements** in `GOMAD_PATCH_SIZE.md`: baseline `-U3`, final extracted `-U3`, final canonical `-U1` (bytes, lines, SHA-256), actual edited upstream file counts, added/deleted source lines, and overlay bytes/lines before and after, so moved code is visible. State both reductions independently and compare with the 27,845 / 20,352-byte prototypes as guidance, not a quota.
- **Prepared source inputs and confirmed gap** — read `task-5/source-size-verification.md`, `source-size-evidence.json` and `conductor-source-size-verification.md`. The verified current candidate is 38,362 bytes / 1,112 lines at `-U3`, versus original task1 baseline 32,652 bytes / 1,007 lines. Literal R8 extraction reduction is unmet by 5,710 bytes; canonical `-U1` at 29,015 bytes / 778 lines only demonstrates the separate context reduction. Return this source gap to task2, coordinated with owners of introduced runtime inputs; do not change the comparator or source here. Developmental zero-fuzz textual equivalence and complete overlay counts do not close either native R4/R7 gate. Task5 remains unclaimed until its predecessors qualify.
- **Milestones** — update the fn-110 row at `MILESTONES.md:33` and the Status line of "Runtime patch minimization (fn-110)" (`:320-356`) with what is done and what is incomplete. Leave the Open findings and the D12/D14 rows unchanged.
- **Regeneration guidance** — `tools/gomad3/README.md:571-578` and `tools/gomad3/CLI.md:466-474`: the canonical patch has one context line and comes only from `patch-regenerate`; overlay additions require the descriptor allowlist and `make generate`; the pinned check follows the descriptor. Edit only the sentences concerned. If the generated upgrade guide (`descriptor.go:287`) needs the same note, change the generator and run `make -C tools/gomad3 generate` rather than editing generated output.
- Superseded intermediate build directories from tasks 2-4 may be removed now if disk is tight, per the constraints below; list what was removed.

### Investigation targets
**Required:**
- `Makefile:165-225`, `tools/gomad3/Makefile:55-175` — native entrypoints
- `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml` — the Linux gates
- `MILESTONES.md:93-110,131-157,320-356`
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
- Recheck the actual host before gates. This development session is `linux/arm64`, with the patched toolchain absent; neither native `darwin/arm64` nor native `linux/amd64` qualification is available here. Record every missing native check as incomplete. Stock-host tests, textual patch equivalence, cross-compilation and source/type checks do not substitute for either platform's required native execution.
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
