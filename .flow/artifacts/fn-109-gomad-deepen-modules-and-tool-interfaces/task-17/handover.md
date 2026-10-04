# Task 17 final source handover

SOURCE FROZEN. NO LIVE COMMANDS. Complete requested source scope is delivered. Flow remains `in_progress`; `native_acceptance: false`, `commits: []`. Parent frozen-source/delta review and conductor admission remain pending.

The exact 17-file frozen boundary is `final-source-post-correction.sha256`, SHA256 `1692362f697d85d42aeabf6b96e878e77245c03848280e0ac7b72cf6bd487095`. Original 15-file `final-source.sha256` and `pre-correction-{handover.md,evidence.json}` retain implementation, test, preimage, developmental provenance and preservation detail. All original 15 hashes remain unchanged after the correction. `evidence.json` indexes every changed file/hash, exact consequential command, log, exit, timing and native gap.

The final review found that Makefile's canonical `test-simulation` integration filter excluded all 13 newly implemented real-process cases. Corrected only Makefile plus `simulation_gate_selection_test.go`: the test expands the actual Make target, discovers actual process test functions via AST, and evaluates its emitted Go test filters. Old-filter RED: exit 1, 0.410s, all 13 exclusions observed. Corrected GREEN with architecture/typed-command tests: exit 0, 1.733s. The separate forward-delay command remains selected; the known strict-delay watchdog remains excluded. Makefile preimage is retained in `pre-correction-Makefile`.

After correction, `make validate` passed (0, 2.257s), stock package vet passed (0, 0.048s), and `git diff --check` passed. Original implementation gates remain valid unchanged: generation/validation, AST ownership old RED/final GREEN, behavioral characterization on actual dirty old task-14 source, developmental overlay/model suites, 20 repeats, scoped race/vet checks, root link-only and Runner compile-only. These are not native process execution.

Native acceptance remains open: pinned executable absent; actual linux/arm64 host is unsupported for complete mode. Final actual-Go1.27.1 native builder exit 2 is retained in `native-toolchain.log`. Canonical overlay/Runner/root process execution, full host/test-host, and darwin/arm64 plus linux/amd64 qualification remain unavailable. Existing incompatible Mach-O linter measured task-16 evidence is reused, not replaced/retried.

The retained external scratch runtime shim is DEVELOPMENTAL only (profile/control disabled, global domain token, stock nanotime, inert arrivals, unavailable blocking/trace transport), never IPC/timer/scheduler/replay/isolation qualification. See `pre-correction-handover.md` and `evidence.json` for exact path/SHA256. No native waiver, staging/commit, Flow mutation, dependencies or unrelated source changes.

