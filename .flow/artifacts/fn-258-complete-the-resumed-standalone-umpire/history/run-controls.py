"""Run adversarial checks only against isolated export clones; never mutate source."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--source', type=Path, required=True)
p.add_argument('--export', type=Path, required=True)
p.add_argument('--synthetic', type=Path, required=True)
a = p.parse_args()
here = Path(__file__).resolve().parent
source, export, synthetic = a.source.resolve(), a.export.resolve(), a.synthetic.resolve()
def git(repo, *args, data=None):
    return subprocess.check_output(['git', '--no-replace-objects', '-C', str(repo), *args], input=data)
assert source != export and source != synthetic and not synthetic.exists()
assert (export/'.git').is_dir()
checkpoint = git(export, 'rev-parse', 'HEAD').decode().strip()
base = [sys.executable, str(here/'verify-export.py'), '--source-repo', str(source), '--export-repo', str(export), '--checkpoint', checkpoint, '--commit-map', str(here/'export/commit-map'), '--manifest', str(here.parent/'source/source-manifest.json'), '--history-map', str(here/'history-map.json')]
results = []
def check(label, command):
    r = subprocess.run(command, capture_output=True)
    assert r.returncode == 1, (label, r.returncode, r.stdout, r.stderr)
    report = json.loads((here/(label+'.json')).read_text()) if (here/(label+'.json')).exists() else None
    results.append({'control':label, 'exit':r.returncode, 'stdout':r.stdout.decode(), 'stderr':r.stderr.decode(), 'failures':report.get('failures') if report else None})
readme=export/'README.md'
raw=readme.read_bytes()
try:
    readme.write_bytes(raw+b'\nsynthetic corruption\n')
    check('file-negative', base+['--output',str(here/'file-negative.json')])
finally:
    readme.write_bytes(raw)
assert readme.read_bytes()==raw
mapping=(here/'export/commit-map').read_text()
# The fixed source itself is legitimately pruned. Delete its row to corrupt coverage.
bad='\n'.join(line for line in mapping.splitlines() if not line.startswith('ac60daaf47d2513b41eeff612f0e2758d33e186d '))+'\n'
(here/'negative-commit-map').write_text(bad)
command=base.copy();command[command.index('--commit-map')+1]=str(here/'negative-commit-map')
check('map-negative',command+['--output',str(here/'map-negative.json')])
# Inject a sibling archive parent while keeping all raw metadata and trees equal.
subprocess.run(['git','clone','--bare','--no-local',str(export),str(synthetic)],check=True,capture_output=True)
rows=dict(line.split() for line in mapping.splitlines()[1:])
canonical=rows['6264f117607d756d41a97342273ebff189ef77f1'];sibling=rows['0fc76f7805f1ef68f828b03ed78d7f86fcb03cfc']
rewrites={}
for oid in git(synthetic,'rev-list','--reverse','--topo-order',checkpoint).decode().splitlines():
    payload=git(synthetic,'cat-file','commit',oid);header,message=payload.split(b'\n\n',1)
    lines=[]
    for line in header.split(b'\n'):
        if line.startswith(b'parent '): line=b'parent '+rewrites.get(line[7:].decode(),line[7:].decode()).encode()
        lines.append(line)
    if oid==canonical: lines.insert(2,b'parent '+sibling.encode())
    mutated=b'\n'.join(lines)+b'\n\n'+message
    rewrites[oid]=git(synthetic,'hash-object','-t','commit','-w','--stdin',data=mutated).decode().strip()
mutated=rewrites[checkpoint]
git(synthetic,'update-ref','refs/heads/synthetic',mutated)
(here/'parent-negative-commit-map').write_text('old                                      new\n'+''.join(old+' '+rewrites.get(new,new)+'\n' for old,new in rows.items()))
command=base.copy();command[command.index('--export-repo')+1]=str(synthetic);command[command.index('--checkpoint')+1]=mutated;command[command.index('--commit-map')+1]=str(here/'parent-negative-commit-map')
assert git(synthetic,'rev-parse',mutated+'^{tree}')==git(export,'rev-parse',checkpoint+'^{tree}')
check('parent-negative',command+['--output',str(here/'parent-negative.json')])
(here/'negative-controls.json').write_text(json.dumps({'controls':results,'readme_restored_exactly':True,'original_commit_map_unchanged':(here/'export/commit-map').read_text()==mapping,'parent_control_active_tree_unchanged':True,'parent_control_checkpoint':mutated},indent=2)+'\n')
