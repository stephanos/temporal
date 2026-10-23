The receiver preserves `parallel/run-validation.sh`'s original admission: the
scratch parent's filesystem must not be `fuseblk`, contain `fuse`, or be
`virtiofs`; both the evidence root and scratch parent need at least 12 GiB
available to the invoking user. The bound `scratch` is a fresh canonical path
outside source and evidence. Its existing parent is inspected before creating
scratch. `temporary_paths` must explicitly bind distinct direct scratch children
for `TMPDIR` and `GOTMPDIR`; these override inherited temporary directories for
all driver commands. Other configured tool/cache paths remain independently
bound and do not waive either original volume check.

`host_admission.py` uses the original fixed `/usr/bin/stat -f -c %T PATH` probe on the
actual serving path and `statvfs` available fragments. The probe binary is a
frozen input, has a five-second bound, and any probe error refuses. It retains actual
volume facts and a PASS or REFUSED `host-admission.json` before source guards,
product admission, builds, guests, or scheduler dispatch. A refusal also reaches
the ordinary incomplete outcome receipt. Direct stage execution requires the
matching initial admission and rechecks both volumes before its adapter runs;
its own facts are retained beneath the stage. The module itself must be frozen
in the source/input manifest. No send epoch or deadline is reset.

Portable unit tests inject bounded facts at the threshold and below it, exercise
all original forbidden filesystem patterns, reject a controlled stacked-mount filesystem result, inspect
real kernel facts without dispatch, and prove admission refusal precedes source
and product commands. These tests do not qualify any candidate. Low actual free
space must be resolved before a future qualification attempt; there is no bypass
or adjustable threshold.
