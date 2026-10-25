#!/usr/bin/env python3
"""Extract retained evidence only; never starts a build, test or backend."""
import json
from pathlib import Path
import re
import shutil

from observer import digest, write

main = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
root = main / '.flow/tmp/fn254-baseline/historical'
root.mkdir(parents=True, exist_ok=True)
fn247 = main / '.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3'
validation = json.loads((fn247 / 'validation.json').read_text())
assert validation['exit_code'] == 0
summary = dict(fn247=validation, binary_runs=[], inputs=[], limitations=[
    'Historical harness intervals exclude initial Cargo build but can include nested trybuild compilation.',
    'No per-binary compiler/link attribution or stable per-test timestamps are available.',
    'Historical serial Cargo output is not a current quiet-host baseline or exact migration inventory.',
])
for name in ['fn2511-dependents.log', 'fn2511-r2-dependents.log']:
    source = main / '.flow/tmp' / name
    destination = root / name
    shutil.copy2(source, destination)
    summary['inputs'].append(dict(path=str(source.relative_to(main)), preserved=str(destination.relative_to(main)), sha256=digest(source)))
    current = None
    lines = re.sub(r'\x1b\[[0-9;]*m', '', source.read_text()).splitlines()
    for index, line in enumerate(lines):
        running = re.search(r'Running (.+) \(([^)]+)\)', line)
        if running:
            current = dict(target=running[1], executable=running[2])
        result = re.search(r'test result: (\w+)\. (\d+) passed; (\d+) failed; (\d+) ignored;.*finished in ([0-9.]+)s', line)
        if result and current:
            summary['binary_runs'].append(dict(current, log=name, line=index+1, status=result[1], passed=int(result[2]),
                                               failed=int(result[3]), ignored=int(result[4]), harness_seconds=float(result[5]),
                                               compile_seconds=None, link_seconds=None))
            current = None
for name in ['fn2373-final-model-time.log', 'fn2373-final-gate-results.jsonl']:
    source = main/'.flow/tmp'/name
    destination = root/name
    shutil.copy2(source, destination)
    summary['inputs'].append(dict(path=str(source.relative_to(main)), preserved=str(destination.relative_to(main)), sha256=digest(source)))
for relative in [
    '.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json',
    '.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/workspace-tests-observed.log',
    '.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/workspace-results.json',
    '.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/tools-platform.log',
    '.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-implementation-evidence.json',
    '.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task8/joined-program-correction/required-validation-attempt1.json',
]:
    summary['inputs'].append(dict(path=relative, sha256=digest(main/relative)))
pending = main/'.flow/tmp/milestones-resume-20261024/fn246-joined-required-validation-attempt2.json'
pending_copy = root/'fn246-attempt2-observed.json'
if not pending_copy.exists():
    shutil.copy2(pending, pending_copy)
summary['fn246_attempt2_snapshot'] = json.loads(pending_copy.read_text())
write(root/'summary.json', summary)
print(f'historical evidence extracted: {len(summary["binary_runs"])} exact Running/result pairs; no commands executed')
