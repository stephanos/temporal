"""Record each lane-local check and its exact source inputs."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[4]
ARTIFACTS = Path(__file__).resolve().parent
ENV = os.environ.copy()
ENV['CARGO_TARGET_DIR'] = '/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2485-target'
ENV['CARGO_BUILD_JOBS'] = '4'
commands = {
 'focused': [
  'cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked',
  'cargo test -p umpire-catalog -p umpire-dsl --lib --locked',
  'cargo test -p umpire-temporal-api --test codegen --test wire_schema --locked',
  'cargo test -p umpire-catalog --test generation --test probe --test black_box --locked',
  'cargo test -p umpire-dsl --test path_domains --test path_language_patterns --test outcome --test catalog --locked',
  'cargo test -p umpire-catalog-build --test neutral --locked',
  'cargo test -p umpire-temporal-annotate-syntax --test admission --test realize --locked',
  'cargo test -p umpire-reference-kettle --test realization --test annotations --test descriptor --locked',
  'cargo test -p umpire-temporal --tests --locked',
 ],
 'consumers': [
  'cargo test -p umpire-temporal-api --test descriptor --locked',
  'cargo test -p umpire-temporal-annotate-syntax --test admission --test realize --test expansion --locked',
  'cargo test -p umpire-reference-kettle --test realization --test annotations --test descriptor --locked',
  'cargo test -p umpire-temporal --tests --locked',
 ],
 'controls': [
  'cargo test -p umpire-temporal-api --test codegen --test wire_schema --test descriptor --locked',
  'cargo test -p umpire-reference-kettle --test realization --test annotations --test descriptor --locked',
  'cargo test -p umpire-runtime --test stateful --test proposal --test selected_start --locked',
  'cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked',
  'cargo test -p umpire-temporal --test admission --locked',
  'cargo test -p umpire-temporal-annotate-syntax --lib --locked',
 ],
 'callers': [
  'cargo test -p umpire-author-consumer --lib --locked',
  'cargo test -p umpire-temporal-annotate --lib --test derive_catalog --test response_checks --locked',
  'cargo clippy -p umpire-author-consumer -p umpire-temporal-annotate -p umpire-temporal-tests --all-targets --locked -- -D warnings',
 ],
 'wasm': [
  'cargo build --target wasm32-wasip1 -p umpire-runtime --locked',
  'cargo build --target wasm32-wasip1 -p umpire-stock-guest -p umpire-control-guest -p umpire-caught-panic-guest -p umpire-overflow-guest -p umpire-kettle-guest -p umpire-temporal-guest --locked',
 ],
 'clippy': [
  'cargo clippy -p umpire-runtime -p umpire-catalog -p umpire-dsl -p umpire-catalog-build -p umpire-catalog-syntax -p umpire-reference-kettle -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal --all-targets --locked -- -D warnings',
 ],
}
stage = sys.argv[1]
run_id = time.time_ns()
paths = subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z','model'],cwd=ROOT).decode().split('\0')
hashes = {p: hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in paths if p and (ROOT/p).is_file() and not '/archived/' in p}
(ARTIFACTS/f'{stage}-source-hashes.json').write_text(json.dumps(hashes,indent=2)+'\n')
results=[]
for i, command in enumerate(commands[stage]):
 log=ARTIFACTS/f'{stage}-{run_id}-{i}.log'; start=time.time()
 with log.open('w') as output:
  try:
   result=subprocess.run(['mise','exec','--']+command.split(),cwd=ROOT/'model',env=ENV,stdout=output,stderr=subprocess.STDOUT,timeout=600)
   code=result.returncode
  except subprocess.TimeoutExpired:
   code='INCONCLUSIVE: timeout 600s'
 result={'command':'cd model && mise exec -- '+command,'exit':code,'seconds':round(time.time()-start,3),'log':str(log.relative_to(ROOT))}
 results.append(result)
 (ARTIFACTS/f'{stage}-results.json').write_text(json.dumps(results,indent=2)+'\n')
 print(json.dumps(result),flush=True)
 if code != 0:break
sys.exit(0 if results and all(r['exit']==0 for r in results) else 1)
