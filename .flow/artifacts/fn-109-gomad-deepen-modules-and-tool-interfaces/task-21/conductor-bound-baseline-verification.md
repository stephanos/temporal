# Bound baseline checkpoint verification

On 2026-10-04 the conductor independently verified the frozen developmental
baseline measurements before checkpointing task 21's preparation. The complete
116-line report was read, including its environment and qualification limits.
Task 21 remains `todo` behind task 20; neither R19 nor the milestones are complete.

## Fresh checks

From the repository root, the existing command
`python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/bound-baseline-measurement/verify_bound_evidence.py`
exited 0. It checked the complete 675-file current scratch path/hash/mode set,
all ten retained source inventories, all 98 successful child commands and their
14 explicit environment bindings, all 171 completion-manifest output hashes,
profile metric reconciliation and publication denominators. It also verified
the unchanged original 670-file reconstruction, 337 historical artifacts and
675 historical scratch files. Each case's completed runtime settings are
GOGC 100, disabled soft memory limit and GOMAXPROCS 2.

The verifier serializes its deterministic result again. A subsequent
`sha256sum -c handoff-output.sha256 --quiet`, from the bound measurement
directory, exited 0 against all 201 final handoff entries. Thus the verifier's
result bytes and every other frozen artifact remain unchanged. The handoff
manifest SHA-256 remains
`d1381842f5eb1b8a1a4b1b5b24d0d54544f61ab30cfea311647e65225abc1133`.

The reconstruction's complete `source.sha256` also passed a fresh quiet check
from `/tmp/fn109-baseline-reconstruction.lDSSw8Gx`. From the repository root,
the historical input manifest passed after excluding exactly the known mutable
task-description path, with pipeline failure checking enabled. All other
historical inputs still match; the historical manifests were not rewritten.

## Historical metadata and current policy

The bound report correctly records task 21's description at measurement time
as `ab5b964fde268884338678cd88da78d9acc6cd78e8e22027c837cba6719aab72`,
versus historical reconstruction-input hash
`4aba7e49a41f5567dceed8c81c038351af45ddb258cc577e753c341eaa30176b`.
After measurements and integrity checks, the conductor used Flow's description
setter to add the bound-evidence pointer and current commit policy. Its new hash
is `19c1b6843a0aa3603e6508096b1984c89939a7d2987a641e785955b76f1e3c56`.
This additional description-only drift is not a baseline source change.

Older reconstruction reports saying commits were user-owned describe the policy
at their execution time. The user's subsequent instruction and `MILESTONES.md`
supersede that restriction. Root now commits verified per-task progress and its
Flow records; native gates remain incomplete. Pushes are not authorized.

## Checkpoint scope and remaining acceptance

Retain lightweight reconstruction inputs, fixture/driver/verifier scripts,
bound reports, structured command and measurement evidence, and manifests in
Git. Keep compiled binaries, bulk payloads, raw profiles, raw/top profile logs
and duplicated inventory snapshots local, identified by the retained manifests.
Verification against these local outputs remains possible on this host; a new
checkout must reproduce them rather than assume the omitted outputs exist.
Whitespace validation excludes only literal archived patch files and the four
raw binary-build metadata logs: their context prefixes and captured trailing
tabs are original evidence bytes, checked by the manifests rather than edited
for formatting. Production and hand-written report checks retain normal rules.

The baseline's four completed campaigns, two-role logical policy size and
allocation-site attribution are preparation for a matched frozen current-tree
comparison. They do not prove absence of every transient copy or measure all
private novelty storage. The compilation-only transport guards were never
called; process transport and patched-runtime behavior were not qualified.
Neither native `darwin/arm64` nor native `linux/amd64` gates ran here. Preserve
their commands and incomplete acceptance until source-bound native results
exist; do not mark the task done because this checkpoint can be committed.
