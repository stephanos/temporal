# Task 17 complete scratch validation

Full `make validate` passed with exit 0 on the exact task-17 candidate in
`/home/agent/.cache/codex-build/tmp/fn109-task17-checkpoint.0nly2wj9`.
This follow-up resolves the archive-input gap recorded in
`checkpoint-report.md`; the original report, source manifest and both RED
validation logs remain byte-identical.

Inspection of `manifestgen.Run`, `ListTests` and `listPlatformTests` showed
that qualification validation reads the generator spec, existing output and
only the enumerated package's top-level `_test.go` files. It applies build
headers for darwin/arm64 and linux/amd64 and parses test declarations.
It performs no root package compilation, dependency listing or test execution.

The follow-up archived exactly 113 committed `tests/*_test.go` files from
`add55cf6116fbed23e8c4051e120521b2f3f033c`. No additional root configuration
was required. It used no working-tree or task-19 source. The added input archive
has SHA-256 `539d0be661fac90ec9c0fe3cb644574707425b745a0e0949cf1cb2ae67bec05d`.
The new manifest lists every added path and full-file hash, verifies every blob
against `git show` at exact HEAD and retains the inspected generator identity.

The complete validation command ran once after adding these inputs, using
pinned stock Go 1.27.1, `GOWORK=off`, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`
and seed variables unset. All 1,050 scratch source files remained unchanged,
as did the 17 task-17 historical identities and its 13 actual changed paths.
Generator cache files remain isolated below scratch `.toolchain`.

The successful log is `checkpoint-final-validate-with-root-inputs.log`, SHA-256
`34f06af58f66580c54a6ec4c4caa8f1346bb2d4338b10e1be0d1e7f7b80b36c1`.
The follow-up manifest is `checkpoint-qualification-inputs-verification.json`,
SHA-256 `6ca5c43058d7587695d5bed1575b17d0d87c253aeeadb4860bf005ea21fa1242`.
The follow-up helper is `checkpoint-qualification-inputs.py`, SHA-256
`0c45db854799a74a9f05d7e3fe87f34b8cbdb3fd8732987a2c8af79f64cfc073`.

No commands remain live. No shared source/docs, Git/index or Flow write occurred.
Native/runtime/process acceptance remains open. This validation adds no root
runtime execution, native timer/IPC/isolation/replay evidence, formal SHIP verdict
or task completion.
