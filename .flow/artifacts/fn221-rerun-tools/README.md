# fn-221.2 resume checker handover

Task .2 remains in progress. The actual run at `/home/agent/wt/evidence/fn221/rerun-20261022T1705Z` has four completed rerun units and 340 remaining (three former timeouts, 337 infrastructure rows). Its missing `matrix/sweep.json` is incomplete evidence, never a passing matrix. No sweep was resumed by this repair. The four retained observations changed from timeout to `capacity-exceeded`, reason `WASI capacity: network-v2-evidence-bytes`, under the unchanged three-minute execution budget. Preserve these explicit flags in the eventual matrix.

`merge-check.py` is the maintained replacement for the external historical `/home/agent/wt/evidence/fn221/rerun-prep/merge-check.py`, SHA-256 `aed05c442756c348db80181298b63ff83a7c14fe18967a25c4350a8e9eb317dc`. That historical file was read and copied, never modified. Its immutable original journal digest remains `839654fcfc60bd65a5ea740b6ec01543336d2f0576798614220cb06be35dafb6`; candidate key remains `sha256:9bf9e9064761a8afcc075d6cba9aed57a57baa6a1f6de36f092233eb5c01040c`.

The replacement admits repeated matching invocation headers, including empty interrupted invocations. Each header must match the immutable original journal's candidate (including configuration and execution/preparation budgets), enumeration, statement, overall/max-test/retention budgets and journal schema. Recorded commands must match apart from the six relocated path arguments already authorized for this rerun: checkout, compiler, runtime root, output, work and artifact roots. The original byte prefix, selection, 344 replacements, 1144 unchanged rows, 1488 ordered units, module bindings and recomputed aggregate checks remain enforced. Duplicate/missing selected observations and pending not-run rows fail; multiple headers do not excuse duplicate outcomes. A missing sidecar writes an explicit incomplete report and exits 1.

## Checker tests only

From the checkout containing this directory:

```sh
python3 -B .flow/artifacts/fn221-rerun-tools/test_merge_check.py
```

Nine test methods passed. Fixtures copy the actual immutable original journal and selection so the pinned digest check is exercised. All appended outcomes and complete sidecars created by the tests are **synthetic checker fixtures**, never guest observations. Temporary fixture directories are created here and removed after each test. Tests read external evidence and write only temporary local fixtures.

Test-first reproduction against the unchanged historical checker:

```sh
FN221_CHECKER=/home/agent/wt/evidence/fn221/rerun-prep/merge-check.py \
  python3 -B .flow/artifacts/fn221-rerun-tools/test_merge_check.py \
  MergeCheckTests.test_repeated_same_candidate_headers
```

Expected historical failure: sole finding `rerun appended 2 headers, want 1`. The historical checker also incorrectly accepts nine altered single-header payload/budget/enumeration/schema/command cases exercised by `test_foreign_resume_header`; the replacement rejects them. Tests additionally cover a foreign later header, five legitimate headers, missing/duplicate/non-selected/not-run units, altered original bytes, duplicate invalidations, changed retained rows/modules/counts, explicit timeout-to-network-capacity flags, and the actual incomplete run. Only Python tests and source checks were performed; no Go/Cargo build or live guest execution occurred.

## Actual resume procedure (future quiet-host window)

Do not run this procedure during fn-232.10's measurement window. Wait until its owner releases the host, then use the existing quiet gate (load1 below 2.5, no competing Gomad/Cargo lanes). Do not run `rerun-344.sh` against the existing root: it seeds a new run and refuses an existing directory. Do not use the abandoned `rerun-20261022T1509Z` attempt.

```sh
bash /home/agent/wt/evidence/fn221/rerun-prep/quiet-gate.sh
cd /home/agent/wt/evidence/fn221/rerun-20261022T1705Z
sha256sum -c tools.sha256
stat -c '%n links=%h' /home/agent/wt/evidence/qualification-final/qual-3f532910be/helpers/ordinary/gomad-wasmhost
git -C /home/agent/wt/fn221-rerun-src rev-parse HEAD
git -C /home/agent/wt/fn221-rerun-src status --porcelain --untracked-files=all
```

Require tool hashes to pass, helper link count 1, a clean source checkout at `1d7b37f00a74a85399993f1edcad6fe742e778e3`, and no remaining processes from the previous run. Preserve existing logs and choose unused resume log names. The retained compiler copy is `/home/agent/wt/evidence/fn221/rerun-tools/go1.27.1`; its 2217 provenance-listed toolchain files were verified identical by the previous conductor.

Record `host-start-resume1.txt`, `host-samples-resume1.txt` every 30 seconds, source/tool state and start/end times as `rerun-344.sh` does. Keep the monitor alive during the command below and stop it afterward. The following executes the exact retained argv without shell evaluation and recreates the launcher's environment:

```sh
cd /home/agent/wt/evidence/fn221/rerun-20261022T1705Z
python3 -I - <<'PY'
import os
from pathlib import Path
import shlex
import subprocess

env = {k: v for k, v in os.environ.items()
       if not k.startswith('GOMAD_') and k not in
       ('GOROOT', 'GOFLAGS', 'GOEXPERIMENT', 'GOOS', 'GOARCH', 'BASH_ENV', 'SANDBOX_START_DIR')}
env.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0',
           GOMAXPROCS='2', CARGO_BUILD_JOBS='2', GOCACHE='/home/agent/.cache/go-build-local',
           GIT_DISCOVERY_ACROSS_FILESYSTEM='1')
argv = shlex.split(Path('command.txt').read_text())
with open('sweep-resume1.log', 'xb') as log:
    status = subprocess.call(argv, env=env, stdout=log, stderr=subprocess.STDOUT)
Path('sweep-resume1.exit').write_text(str(status) + '\n')
raise SystemExit(status)
PY
```

Supervise the new header as it appears after installation, before allowing execution to continue: its key must be the pinned key above and its payload must match the prior valid rerun header's candidate/configuration, budgets and enumeration. Stop the run if they differ. Resume must skip 1144 retained rows and the four completed units, executing the remaining 340. Preserve the same journal throughout interruptions; append another invocation header on the next resume, never reseed invalidations.

After a successful completion, run both independent checks. Set `FN221_TOOLS` to the integrated checkout's directory (the isolated path below is valid before integration):

```sh
FN221_TOOLS=/home/agent/wt/fn221-resume-tools/.flow/artifacts/fn221-rerun-tools
FN221_RUN=/home/agent/wt/evidence/fn221/rerun-20261022T1705Z
/home/agent/wt/evidence/fn221/full-1d7b37f00a/bin/wasmsweep \
  --verify "$FN221_RUN/matrix/sweep.json" > "$FN221_RUN/verify-resume1.log" 2>&1
python3 -I "$FN221_TOOLS/merge-check.py" \
  /home/agent/wt/evidence/fn221/full-1d7b37f00a "$FN221_RUN" \
  /home/agent/wt/evidence/fn221/rerun-prep/rerun-units.tsv \
  "$FN221_RUN/merge-check.json"
```

Require both exit codes 0 and zero checker findings before rendering or accepting a support matrix. Recheck tool hashes/helper links and clean pinned source at the end; retain host samples from every invocation. Use the existing `rerun-prep/render-matrix.py` on the actual final sidecar and strengthened report, with original and combined rerun host samples, then retain the observation-only matrix and sidecar on `fn221-matrix`. Task .2 is not completed by synthetic tests or this handover.

Kept worktrees/evidence: `fn221-resume-tools` holds this unintegrated repair; `fn221-rerun-src` holds the immutable guest source; `fn221-matrix` owns the eventual matrix; `evidence/fn221/full-1d7b37f00a`, `rerun-20261022T1705Z`, `rerun-prep` and `rerun-tools` remain required inputs of open fn-221. No unrelated worktrees or evidence were changed.
