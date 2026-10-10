# Raw help whitespace staging observation

Root authorized preserving the exact raw Go flag.PrintDefaults stderr bytes. The whole cached whitespace check returned 2 with exactly five diagnostics in `refresh-help.stderr`. The raw stream retains SHA256 42c826d83aea0bcd282e9cab26f5026bdd076f2c551cba950280536e9a2e5443.

Command `git diff --cached --check` produced these diagnostics. Each displayed output line below preserves its four spaces followed by a literal tab.

```text
.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr:3: space before tab in indent.
+    	Git revision holding each working directory's go.mod and go.sum before the bump (default "HEAD")
.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr:5: space before tab in indent.
+    	absolute pack authoring root owned by another module (default: internal/compatibilitypack)
.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr:7: space before tab in indent.
+    	go command that resolves module graphs
.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr:9: space before tab in indent.
+    	existing pin-impact JSON report for the bumped checkout
.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr:11: space before tab in indent.
+    	Gomad v3 module root
```

The scoped command `git diff --cached --check -- . ':(exclude).flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/current-doc-reconciliation/refresh-help/refresh-help.stderr'` returned 0 over the other staged files. This records a raw-evidence formatting exception for one exact path. It does not report the whole cached check as green, change lint policy or establish any source acceptance gate. No raw capture, pinned packet, source-only seal or independent review was edited.
