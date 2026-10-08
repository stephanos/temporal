# Independent cache-control verification

The conductor reran precisely the four admitted unchanged cache-cleanup tests
after the worker released every Go command. The foreground command terminated
normally at 2026-10-08T18:23:27.415Z, exit 0. All 17 test events passed; no test
failed or skipped. The raw log shows five UID-1000 permission-denial probes and
two explicitly logged RemoveAll failures; the remaining fault classifications
are checked by the unchanged registry assertions. This replay supplies no new
unique coverage and is not a review verdict or native qualification.

[Raw receipt](../source-acceptance-20261008/conductor-independent-cache-controls-receipt.json)
and [log](../source-acceptance-20261008/conductor-independent-cache-controls.log)
bind unchanged source
`1ea646fc66ac63aa0d2dde38e434f8b5529c28ac9040b55b7a04c56c4ebc0f3f`
and unchanged pinned tools. The log hash was independently verified as
`641228a5db8d818bf8f3341033bbe378f4657450da8e0666daac123c8b0ca089`.

The actual command overrides TMPDIR inline with
`/dev/shm/gomad-fn1132-cleanup-3pOD0l3j`. The receipt's outer harness environment
still names the workspace TMPDIR; it is not the test's effective value.
GOTMPDIR remains `/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX`,
and build/module caches remain on the assigned workspace. The admitted tmpfs is
rw,nosuid,nodev,noexec; these callback fixtures execute no child programs there.
The Go lane was released again after this verification. Source, assertions,
tool identities and global filesystem/telemetry settings were not changed.
