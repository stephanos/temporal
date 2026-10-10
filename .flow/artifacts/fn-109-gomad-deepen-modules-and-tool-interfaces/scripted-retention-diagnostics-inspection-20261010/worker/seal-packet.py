import json
import subprocess
import sys
sys.dont_write_bytecode = True
import capture_v2 as r

files = sorted(r.PACKET.glob('*.py')) + sorted(r.PACKET.glob('*.md')) + sorted(r.PACKET.glob('*.json'))
evidence = json.loads((r.PACKET/'evidence.json').read_text())
assert r.git('rev-parse', 'HEAD').strip() == evidence['commits'][-1]
for name, record in evidence['raw_receipts'].items():
    assert r.sha(r.OUTPUT/(name+'.json')) == record['receipt_sha256']
    assert r.sha(r.OUTPUT/(name+'-binding.json')) == record['binding_sha256']
    assert r.sha(r.OUTPUT/(name+'.log')) == record['log_sha256']
groups = {}
for path in sorted(r.OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    value = json.loads(path.read_text())
    if 'process_group' in value:
        groups[path.stem] = {'process_group': value['process_group'], 'remaining_members': r.group_members(value['process_group'])}
observer = subprocess.run(['/usr/bin/python3', str(r.PACKET/'inspect-lane-v2.py')], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)
lane = json.loads(observer.stdout)
owned_terminal = all(not record['remaining_members'] for record in groups.values())
print(json.dumps({'packet_files': {str(path): r.sha(path) for path in files}, 'owned_groups': groups, 'all_owned_commands_terminal': owned_terminal, 'current_gate_lane': lane, 'observer_exit_code': observer.returncode, 'observer_stdout': observer.stdout, 'observer_stderr': observer.stderr, 'go_settings_probe': 'NOT_EXECUTED: final packet observer'}, sort_keys=True, indent=2))
raise SystemExit(0 if owned_terminal and observer.returncode == 0 and lane['gate_lane_empty'] else 1)
