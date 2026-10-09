# Task 10 diagnostic-write source admission

Continue task `fn-112-gomad-determinism-assurance-and-test.10` on the owner-selected
`gomad` branch at `4de2ba7892570a865c27a178b681e31f28fca79b`. Its prior worker and
independent reviewer are terminal, and their reviewed source progress is committed.
The existing runtime claim remains `in_progress` under this root's actor; no
other active task-10 worker is admitted. This is a continuation, not a new task.

The delivered soak command is semantically owned by this task's R6 contract but
its adapter was omitted from the original Touches declaration. Root now adds
exactly `tools/gomad3/cmd/gomadtool/soak.go` and `soak_test.go` for the retained
four unchecked diagnostic writes at current lines 37, 47, 55 and 62. The earlier
dispatch's out-of-Touches refusal remains historically correct; this is a new
bounded admission. No other maintainer command, runtime source, policy or pin
may change. The other 59 command-adapter lint findings remain outside this repair.

Retain the current unfiltered configured-lint failure from the preceding packet
as the meaningful failing control. Before editing production, exercise healthy
and genuine failed-writer controls through public `run`; preserve their exact
diagnostic bytes, attempted calls, primary classifications and statuses. Check
the four existing writes without changing their format strings, operands,
ordering, status 2/3 precedence, completed publications or healthy output.
Do not add stderr fallback, suppression wrapper, callback framework, production
injection seam, flag or dependency. A failure already classified by the command
keeps its original status even when reporting it also fails.

Add bounded tests in the admitted soak test file, reusing established test-only
writer controls where appropriate. An unexecuted infrastructure/report branch
must remain explicitly unproved; never spoof a native qualification pass to
reach it. Any portable helper evidence is solely command-adapter evidence.
Existing assertions and the original first-baseline/fixed-identity obligations
remain unchanged. No test is weakened to obtain green.

Use the proven private local overlayfs temp placement under `/tmp`, not the
prior FUSE temp root, and bind filesystem, source, Go/module/cache/proxy and tool
inputs. A full ordinary command-package baseline is admitted on this changed
temp input; retain both its outcome and the old FUSE Git-fixture failure without
assuming their causes are identical. Do not retry unchanged failures. Root owns
all Flow lifecycle writes, scope, reviews and commits; worker cannot certify Done.

The fresh task worker exclusively owns the shared Go/build/lint/generator/cache
lane. Retain focused controls, the full ordinary affected package, soak/set
coverage, architecture/public/purity boundaries, both supported static source
sets, affected vet, standalone errortype, check-only validate and actual
`make lint-code-fast` with fixes disabled against this base. Retain the complete
unfiltered adapter lint inventory; expect only these four findings to disappear
and disclose every residual. Green scoped deltas do not establish full lint.
Check generator inputs and prove unrelated tracked inputs remain byte-identical.

Formal implementation review is allowed only when all required task source gates
are green. Otherwise root may review and commit verified source progress while
acceptance stays open. Native fn-128/fn-149 qualification and bounds remain
deferred, with no patched Linux/arm64 execution, native revival, CI dispatch,
PR or push authorized. Protected user `.turbo` files remain untouched and unstaged.
