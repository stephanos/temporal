import json
import subprocess
from pathlib import Path


base = "b32dad53fc544ab75d56f6b9c41fba9b99a75858"
changes = {
    "tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go": [
        ('\t"sync"\n', '\t"sync"\n\t"sync/atomic"\n', 1),
        ('\tstop := make(chan struct{})\n', '\tvar stopped atomic.Bool\n', 1),
        (
            '\t\t\tfor {\n\t\t\t\tselect {\n\t\t\t\tcase <-stop:\n\t\t\t\t\treturn\n\t\t\t\tdefault:\n\t\t\t\t}\n\t\t\t}\n',
            '\t\t\tfor !stopped.Load() {\n\t\t\t}\n', 1,
        ),
        ('\t\t\tclose(stop)\n', '\t\t\tstopped.Store(true)\n', 2),
    ],
    "tools/gomad3/runner/internal/execution/process_test.go": [
        ('\t"sync"\n', '\t"sync"\n\t"sync/atomic"\n', 1),
        (
            'func TestUnresponsiveSupervisorHelper(t *testing.T) {\n\tif os.Getenv("GOMAD3_PROCESS_SUPERVISOR") != "1" {\n\t\tt.Skip("supervisor subprocess only")\n\t}\n\tfor {\n\t}\n}\n',
            'func TestUnresponsiveSupervisorHelper(t *testing.T) {\n\tif os.Getenv("GOMAD3_PROCESS_SUPERVISOR") != "1" {\n\t\tt.Skip("supervisor subprocess only")\n\t}\n\tvar activity atomic.Uint64\n\tfor {\n\t\tactivity.Add(1)\n\t}\n}\n', 1,
        ),
    ],
}
for path, replacements in changes.items():
    expected = subprocess.check_output(["git", "show", f"{base}:{path}"], text=True)
    for before, after, count in replacements:
        if expected.count(before) != count:
            raise ValueError(f"unexpected baseline replacement count for {path}")
        expected = expected.replace(before, after)
    if expected != Path(path).read_text():
        raise ValueError(f"unadmitted source change in {path}")
tracked = subprocess.check_output(["git", "diff", "--name-only", base], text=True).splitlines()
untracked = subprocess.check_output(["git", "ls-files", "--others", "--exclude-standard"], text=True).splitlines()
allowed = set(changes) | {
    "tools/gomad3/internal/gomadtool/conformance/cpu_load_lifecycle_test.go",
    "tools/gomad3/runner/internal/execution/unresponsive_supervisor_lifecycle_test.go",
}
prefix = ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-67/"
unexpected = [path for path in tracked + untracked if path not in allowed and not path.startswith(prefix)]
if unexpected:
    raise ValueError(f"unadmitted changed paths: {unexpected}")
print(json.dumps({"base": base, "original_files_exact_except_admitted_edits": list(changes), "product_scope": sorted(allowed), "unexpected_paths": unexpected}, indent=2))
