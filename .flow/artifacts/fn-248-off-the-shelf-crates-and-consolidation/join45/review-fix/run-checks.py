"""Focused source-bound verification for the isolated task4/task5 join."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[5]
ARTIFACTS = Path(__file__).resolve().parent
ENV = os.environ.copy()
ENV['CARGO_TARGET_DIR'] = '/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2485-target'
ENV['CARGO_BUILD_JOBS'] = '4'
commands = {
 'combined': [
  'cargo test -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --locked',
  'cargo test -p umpire-temporal-api --test codegen --test wire_schema --test descriptor --test services --locked',
  'cargo test -p umpire-catalog --test ui --locked',
  'cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked',
  'cargo test -p umpire-catalog --test generation --test probe --test black_box --locked',
  'cargo test -p umpire-reference-kettle --test annotations --test realization --test descriptor --locked',
  'cargo test -p umpire-temporal --test admission --test support --test declarative --locked',
 ],
 'clippy': [
  'cargo clippy -p umpire-runtime -p umpire-catalog -p umpire-dsl -p umpire-catalog-build -p umpire-catalog-syntax -p umpire-reference-kettle -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal --all-targets --locked -- -D warnings',
 ],
 'ui': [
  'cargo test -p umpire-temporal-annotate --test ui --locked',
 ],
}
stage = sys.argv[1]
if len(sys.argv) > 2:
 commands[stage] = commands[stage][int(sys.argv[2]):]
run_id = time.time_ns()
paths = subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z','model'],cwd=ROOT).decode().split('\0')
hashes = {p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in paths if p and (ROOT/p).is_file() and not '/archived/' in p}
(ARTIFACTS/f'{stage}-{run_id}-source-hashes.json').write_text(json.dumps(hashes,indent=2)+'\n')
results=[]
for i, command in enumerate(commands[stage]):
 log=ARTIFACTS/f'{stage}-{run_id}-{i}.log';start=time.time()
 with log.open('w') as output:
  try:
   result=subprocess.run(['mise','exec','--']+command.split(),cwd=ROOT/'model',env=ENV,stdout=output,stderr=subprocess.STDOUT,timeout=600)
   code=result.returncode
  except subprocess.TimeoutExpired:
   code='INCONCLUSIVE: timeout 600s'
 entry={'command':'cd model && mise exec -- '+command,'exit':code,'seconds':round(time.time()-start,3),'log':str(log.relative_to(ROOT)),'source_hashes':f'{stage}-{run_id}-source-hashes.json'}
 results.append(entry)
 (ARTIFACTS/f'{stage}-{run_id}-results.json').write_text(json.dumps(results,indent=2)+'\n')
 print(json.dumps(entry),flush=True)
 if code != 0:break
sys.exit(0 if results and all(r['exit']==0 for r in results) else 1)
