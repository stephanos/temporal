import json,os,subprocess,time
from pathlib import Path
root=Path.cwd().resolve()
proof=root/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task10/revision1/proof'
env=os.environ.copy()
for k in ['BASH_ENV','CARGO_PROFILE_DEV_DEBUG','CARGO_PROFILE_TEST_DEBUG','RUSTFLAGS','CARGO_ENCODED_RUSTFLAGS','CARGO_BUILD_TARGET','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER']:
 env.pop(k,None)
env.update(CARGO_TARGET_DIR='/home/agent/.cache/codex-build/umpire-milestones-20261024/fn25410-gate/parent',CARGO_BUILD_JOBS='2',RUST_TEST_THREADS='2',CARGO_INCREMENTAL='0')
checks=[]
for label,cwd,argv in [
 ('framework-proto-controls',root,['bash','model/umpire/scripts/tests/proto-go.sh']),
 ('outer-proto-controls',root,['bash','model/scripts/tests/proto-go.sh']),
 ('framework-timer-controls',root,['bash','model/umpire/scripts/test-time-gate.sh']),
 ('framework-proto-tools-controls',root,['bash','model/umpire/scripts/tests/proto-tools.sh']),
 ('framework-bootstrap-controls',root,['bash','model/umpire/scripts/tests/bootstrap.sh']),
 ('framework-build-guest-controls',root,['bash','model/umpire/scripts/tests/build-guests-controls.sh']),
 ('root-timer-controls',root,['bash','model/scripts/test-time-gate.sh']),
 ('framework-native-tests',root/'model/umpire',['mise','exec','--','cargo','test','--locked','-p','xtask','--lib','--test','main']),
 ('outer-native-tests',root/'model',['mise','exec','--','cargo','test','--locked','-p','temporal-xtask']),
 ('framework-native-clippy',root/'model/umpire',['mise','exec','--','cargo','clippy','--locked','-p','xtask','--all-targets','--','-D','warnings']),
 ('outer-native-clippy',root/'model',['mise','exec','--','cargo','clippy','--locked','-p','temporal-xtask','--all-targets','--','-D','warnings']),
 ('framework-fmt',root/'model/umpire',['mise','exec','--','cargo','fmt','--all','--check']),
 ('outer-fmt',root/'model',['mise','exec','--','cargo','fmt','--all','--check']),
]:
 start=time.monotonic()
 with (proof/(label+'.log')).open('wb') as f:
  p=subprocess.run(argv,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT)
 checks.append(dict(label=label,cwd=str(cwd),argv=argv,exit=p.returncode,seconds=time.monotonic()-start))
 (proof/'focused-commands.json').write_text(json.dumps(dict(commands=checks,env={k:env[k] for k in ['CARGO_TARGET_DIR','CARGO_BUILD_JOBS','RUST_TEST_THREADS','CARGO_INCREMENTAL']}),indent=2)+'\n')
 print(label,p.returncode,flush=True)
 if p.returncode: raise SystemExit(p.returncode)
