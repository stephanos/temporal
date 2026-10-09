import collections
import datetime
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BASE = '15f56644664f3d3749bab2387aa97936a1cac6dd'
owned = ['.github/workflows/gomad3.yml', 'tools/gomad3/CLI.md', 'tools/gomad3/README.md',
         'tools/gomad3/TUTORIAL.md', 'tools/gomad3integration/README.md', 'tools/gomad3integration/qualification/soak.json']
gates = {}
for path in sorted(OUT.glob('*.json')):
    data = json.loads(path.read_text())
    if 'command' in data and 'exit_code' in data:
        gates[path.stem] = data
counts = {}
for name in gates:
    log = OUT / (name + '.log')
    events = []
    for line in log.read_text().splitlines():
        try:
            event = json.loads(line)
            if 'Action' in event:
                events.append(event)
        except json.JSONDecodeError:
            pass
    if events:
        counts[name] = {'top_level_pass': sum(event.get('Action') == 'pass' and bool(event.get('Test')) and '/' not in event['Test'] for event in events),
                        'all_test_pass': sum(event.get('Action') == 'pass' and bool(event.get('Test')) for event in events),
                        'failed_tests': [event['Test'] for event in events if event.get('Action') == 'fail' and event.get('Test')],
                        'skipped_tests': [event['Test'] for event in events if event.get('Action') == 'skip' and event.get('Test')]}
bindings = {}
for path in owned:
    before = subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)
    bindings[path] = {'before_sha256': hashlib.sha256(before).hexdigest(), 'after_sha256': hashlib.sha256((ROOT / path).read_bytes()).hexdigest()}
lint = (OUT / 'configured-lint.log').read_text()
issues = re.findall(r'^([^\n]+\.go):\d+:\d+: .+ \(errcheck\)$', lint, re.M)
inputs = {}
for path in ['run_gate.py', 'link-count-control.go', 'current-source-audit.py', 'seal-evidence.py']:
    inputs[path] = hashlib.sha256((OUT / path).read_bytes()).hexdigest()
for path in ['/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype',
             '/home/agent/.codex/scripts/flowctl', str(ROOT / '.github/.golangci.yml')]:
    inputs[path] = hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
result = {
    'task': 'fn-112-gomad-determinism-assurance-and-test.10', 'status': 'in_progress', 'acceptance_verdict': None,
    'base_commit': BASE, 'head_commit': head, 'source_range': BASE + '..' + head,
    'commits': subprocess.check_output(['git', 'rev-list', '--reverse', BASE + '..HEAD'], cwd=ROOT, text=True).splitlines(),
    'tests': [data['command'] for data in gates.values()], 'prs': [], 'gates': gates, 'test_counts': counts,
    'source_changes': bindings, 'input_sha256': inputs,
    'all_tracked_changed_paths': subprocess.check_output(['git', 'diff', '--name-only'], cwd=ROOT, text=True).splitlines(),
    'go_paths_changed_from_admission': subprocess.check_output(['git', 'diff', '--name-only', BASE, '--', '*.go'], cwd=ROOT, text=True).splitlines(),
    'inherited_root_changes': ['.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.json', '.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md', 'MILESTONES.md'],
    'lint_red': {'diagnostics': len(issues), 'by_file': dict(collections.Counter(issues)),
                 'task_semantic_surface_outside_touches': 'tools/gomad3/cmd/gomadtool/soak.go:36,46,55,62',
                 'authority': 'root declined expansion; prerequisite owner routing remains with root'},
    'baseline': 'red on original FUSE TMPDIR, two named set failures; original wrong-cwd attempt inconclusive',
    'temp_filesystems': {'original': {'path': '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', 'type': 'fuseblk', 'fs_id': '0'},
                         'admitted_changed_input': {'path': '/tmp/fn11210-portable.6ZZCZ0Ix', 'type': 'overlayfs', 'fs_id': 'e6312165da52bad5'}},
    'native_qualification_claim': False, 'native_bound': None, 'live_command_handles': [],
    'tier': 'session (jev-unavailable(no_key))', 'requested_implementer': 'gpt-6.1-sol at high',
    'review': 'not dispatched; root owns review/lifecycle; source acceptance remains open on lint RED',
    'sealed_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
}
with (OUT / 'evidence.json').open('x') as output:
    json.dump(result, output, indent=2)
    output.write('\n')
print(json.dumps({'gates': len(gates), 'lint_diagnostics': len(issues), 'head': head, 'go_changed': result['go_paths_changed_from_admission']}))
