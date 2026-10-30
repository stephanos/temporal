import pathlib,json,hashlib,re,subprocess,sys
from importlib.machinery import SourceFileLoader
m=SourceFileLoader('proof','/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task6/run-proof.py').load_module()
root=m.candidate;mode=sys.argv[1]
if mode in ['prewarm','warm']:
 path=root/'model/temporal/feature/activity/model.rs';old=path.read_bytes();new=old.replace(b'//! Bounded activity retries, optional deadlines and cooperative cancellation.',f'//! Bounded activity retries, optional deadlines and cooperative cancellation ({mode} consumers).'.encode(),1)
else:
 assert mode=='phase-mutant';path=root/'model/temporal/feature/activity/phase.rs';old=path.read_bytes();before=b'Refusal::unsupported("unsupported activity execution status")';after=b'Refusal::malformed("unsupported activity execution status")';assert old.count(before)==1;new=old.replace(before,after,1)
assert old!=new
try:
 path.write_bytes(new)
 if mode=='phase-mutant':
  assert m.run('consumer-fix/phase-selection',['mise','exec','--','cargo','run','--locked','--manifest-path','model/Cargo.toml','-p','temporal-xtask','--bin','temporal-xtask','--','quick','--select-only'])==0
 code=m.run('consumer-fix/'+mode,['make','model-quick'])
 text=(m.proof/('consumer-fix/'+mode+'.log')).read_text()
 receipts=re.findall(r'quick: stage=(\w+) workspace=.* packages=([^\n]+?) elapsed=([0-9.]+)s exit=([^\n]+)',text)
 selection=[json.loads(line) for line in text.splitlines() if line.startswith('{"base":')][0]
 packages=[package for w in selection['workspaces'] for package in w['packages']]
 expected=['umpire-temporal','umpire-temporal-api','umpire-temporal-guest','umpire-temporal-realization-macros','umpire-temporal-tests']
 facts={'mode':mode,'source':subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),'status':code,'input':str(path.relative_to(root)),'original_sha256':hashlib.sha256(old).hexdigest(),'edited_sha256':hashlib.sha256(new).hexdigest(),'selection':selection,'stage_receipts':receipts,'cache_hits':text.count('TLC cache hit '),'cache_misses':text.count('TLC cache miss '),'tlc_executions':text.count('TLC execute '),'native_interpreter':text.count('quick native/interpreter agreement '),'existing_phase_test_failed':'test phase_conversion_preserves_all_pairs_and_unknown_enum_precedence ... FAILED' in text}
 (m.proof/('consumer-fix/'+mode+'-facts.json')).write_text(json.dumps(facts,indent=2)+'\n');print(json.dumps(facts),flush=True)
 assert packages==expected,(packages,expected)
 if mode in ['prewarm','warm']:
  assert code==0 and facts['native_interpreter']>0 and facts['cache_hits']>0 and facts['cache_misses']==0 and facts['tlc_executions']==0
  assert [r[1] for r in receipts if r[0]=='test']==expected
  assert all(r[3]=='0' for r in receipts)
 else:
  assert code!=0 and facts['existing_phase_test_failed'] and facts['tlc_executions']==0
  assert receipts[-1][0]=='test' and receipts[-1][1]=='umpire-temporal' and receipts[-1][3]!='0'
finally:
 path.write_bytes(old)
 assert not subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)
