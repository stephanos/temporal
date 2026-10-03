- [x] `go-interface-changes.md` lists every removed or changed exported declaration, its consumers, the replacement and the migration, written before the edits and updated to match the result.
- [x] No exported Runner signature or field mentions a `runner/internal/execution` type; `Executor`/`ReplayExecutor` injection exists only as private dependencies.
- [x] `Preparer` and `ArtifactReplayer` seams remain public and usable; the external-consumer fixture compiles from outside the Runner subtree.
- [x] Fake-executor failure, cancellation and watchdog tests pass through the private entry points with unchanged semantics; no global mutable test hook exists.
- [x] Default supervisor/coordinator behaviour and the injection-versus-isolation rejections are unchanged.
- [x] fn-105.3 is closed by reference to this task (one owner); its summary cites this task's evidence and does not claim separate work.

