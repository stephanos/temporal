import json
import subprocess
import sys
sys.dont_write_bytecode = True
import run as r

groups = {}
for path in sorted(r.OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    value = json.loads(path.read_text())
    if 'process_group' in value:
        groups[path.stem] = {'process_group':value['process_group'],'remaining_members':r.group_members(value['process_group'])}
observer = subprocess.run(['/usr/bin/python3',str(r.PACKET/'inspect-lane.py')],stdout=subprocess.PIPE,text=True,check=False)
lane = json.loads(observer.stdout)
owned_terminal = all(not value['remaining_members'] for value in groups.values())
print(json.dumps({'owned_groups':groups,'current_gate_lane':lane,'observer_exit_code':observer.returncode,'all_owned_commands_terminal':owned_terminal},sort_keys=True,indent=2))
raise SystemExit(0 if owned_terminal and lane['gate_lane_empty'] else 1)
