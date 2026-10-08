# Read-only formatter replay correction

The first conductor orchestrator was stopped with SIGTERM (PID 1287101; terminal exit 143) before either frozen formatter writer entrypoint was dispatched. Three completed test receipts were retained; the already spawned minimizer capture completed normally with exit 0 and 48 passing tests. Its capture and Go child were confirmed absent before resumption.

The corrected orchestrator invokes the underlying read-only gofmt argv directly, never the worker's receipt-writing formatter helpers. It validates existing terminal receipts before proceeding. All 150 worker-sealed files remained byte-identical before corrected dispatch; the independent audit rechecks the complete seal. No Go assertion failed because of this orchestration stop, no test was killed, and no worker artifact was rewritten.

Raw whole-tree `gofmt -l` exits 0 but prints the inherited runtime overlay path. That is not a green whole-tree formatting gate. The original task6 source-path selection exits 0 with empty output.

The first conductor audit exited 1 at its own failure-count assertion: it incorrectly expected 41 total terminal failures in the operation observations. The retained raw observations contain 41 top-level failures plus 12 failed children (53 terminal failures). The corrected audit checks both quantities, alongside the four separate choice-control failures. This was an audit-authoring error, not an additional product test failure or a changed worker result; no audit.json was produced by that attempt.

After staging previously untracked evidence, the raw staged diff check exits 2 on literal godoc/anchor output and the retained unified inventory diff: three stdout files and one diff file. Their original trailing spaces/EOF blank lines remain byte-exact; frozen receipts are not sanitized. A second staged check excludes only those four exact evidence paths and still checks every source, task, inventory, script and remaining artifact. This is a raw-log whitespace distinction, not a product formatting or lint exception. The earlier unstaged diff check did not examine these then-untracked files.
