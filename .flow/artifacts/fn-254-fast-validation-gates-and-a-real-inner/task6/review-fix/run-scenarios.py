import pathlib,json,hashlib,re,subprocess,sys
from importlib.machinery import SourceFileLoader
m=SourceFileLoader('proof','/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task6/run-proof.py').load_module()
root=m.candidate; mode=sys.argv[1]
if mode=='warm':
 path=root/'model/temporal/feature/activity/model.rs';old=path.read_bytes();new=old.replace(b'//! Bounded activity retries, optional deadlines and cooperative cancellation.',b'//! Bounded activity retries, optional deadlines and cooperative cancellation (timing proof).',1)
 assert old!=new
elif mode=='edited-test':
 path=root/'model/temporal/feature/activity/tests.rs';old=path.read_bytes();new=old+b'\n#[test]\n#[allow(clippy::print_stderr)]\nfn aaaa_quick_review_failure() {\n    eprintln!("QUICK_REVIEW_EDITED_AUTHORED_TEST_EXECUTED");\n    std::process::exit(73);\n}\n'
else: raise SystemExit(mode)
try:
 path.write_bytes(new)
 if mode=='edited-test':
  code=m.run('review-fix/edited-test-selection',['mise','exec','--','cargo','run','--locked','--manifest-path','model/Cargo.toml','-p','temporal-xtask','--bin','temporal-xtask','--','quick','--select-only'])
  assert code==0
 extra={'RUST_TEST_THREADS':'1'} if mode=='edited-test' else {}
 code=m.run('review-fix/'+mode,['make','model-quick'],extra=extra)
 text=(m.proof/('review-fix/'+mode+'.log')).read_text()
 receipts=re.findall(r'quick: stage=(\w+) workspace=.* packages=([^\n]+?) elapsed=([0-9.]+)s exit=([^\n]+)',text)
 selection=[json.loads(line) for line in text.splitlines() if line.startswith('{"base":')][0]
 facts={'mode':mode,'source':subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),'status':code,'changed_input':str(path.relative_to(root)),'original_sha256':hashlib.sha256(old).hexdigest(),'edited_sha256':hashlib.sha256(new).hexdigest(),'selection':selection,'stage_receipts':receipts,'cache_hits':text.count('TLC cache hit '),'cache_misses':text.count('TLC cache miss '),'tlc_executions':text.count('TLC execute '),'native_interpreter':text.count('quick native/interpreter agreement '),'edited_test_body_ran':'QUICK_REVIEW_EDITED_AUTHORED_TEST_EXECUTED' in text,'test_binary_exit73':'exit status: 73' in text}
 assert {r[0] for r in receipts}=={'fmt','clippy','test'}
 if mode=='warm':
  assert code==0 and facts['native_interpreter']>0 and facts['cache_hits']>0 and facts['cache_misses']==0 and facts['tlc_executions']==0
  assert all(r[3]=='0' for r in receipts)
 else:
  assert code!=0 and facts['edited_test_body_ran'] and facts['test_binary_exit73'] and facts['tlc_executions']==0
  assert receipts[-1][0]=='test' and receipts[-1][3]=='73'
  commands=[line for line in text.splitlines() if 'stage=test ' in line and ' command=' in line]
  assert len(commands)==1 and '"--test"' not in commands[0] and '"--exact"' not in commands[0]
 (m.proof/('review-fix/'+mode+'-facts.json')).write_text(json.dumps(facts,indent=2)+'\n')
 print(json.dumps(facts),flush=True)
finally:
 path.write_bytes(old)
 assert not subprocess.check_output(['git','status','--porcelain'],cwd=root,text=True)
