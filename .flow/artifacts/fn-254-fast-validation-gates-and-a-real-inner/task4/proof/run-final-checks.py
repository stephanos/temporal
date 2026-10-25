import json,os,pathlib,subprocess,time,sys
p=pathlib.Path(__file__).parent
packages=json.loads((p/'affected-packages.json').read_text());selection=[v for name in packages for v in ['-p',name]]
checks=[
 ('tlc-tests',['cargo','test','--locked','-p','umpire-tlc']),
 ('affected-doctests',['cargo','test','--locked','--doc',*selection]),
 ('kettle-documented',['cargo','test','--locked','-p','umpire-reference-kettle','--test','main','probes::']),
 ('dsl-documented',['cargo','test','--locked','-p','umpire-dsl','--test','main','readme::']),
 ('expansion-documented',['cargo','test','--locked','-p','umpire-catalog-syntax','--test','main','expansion::executable_expansion_and_source_coverage']),
 ('snapshot-ownership',['cargo','test','--locked','-p','umpire-temporal-tests','--test','snapshot_ownership']),
 ('strategy-compile',['cargo','test','--locked','-p','umpire-tlc','--features','strategy-equivalence','--no-run','--message-format=json']),
 ('strategy-unit',['cargo','test','--locked','-p','umpire-tlc','--features','strategy-equivalence','--lib']),
 ('strategy-focused',['cargo','test','--locked','-p','umpire-tlc','--features','strategy-equivalence','--test','main','controls::real_tlc_failed_batches_preserve_independent_traces','--','--exact']),
 ('affected-clippy',['cargo','clippy','--locked',*selection,'--all-targets','--all-features','--','-D','warnings']),
 ('affected-fmt',['cargo','fmt','--check',*selection]),
 ('final-compile',['cargo','test','--locked','--no-run',*selection,'--message-format=json']),
]
results=[]
for name,cmd in checks:
 print('RUNNING '+name+' '+str(cmd),flush=True);start=time.monotonic()
 with (p/(name+'.log')).open('w') as out:
  rc=subprocess.run(['mise','exec','--',*cmd],stdout=out,stderr=subprocess.STDOUT).returncode
 row={'name':name,'command':['mise','exec','--',*cmd],'exit':rc,'seconds':round(time.monotonic()-start,3),'log':name+'.log','env':{k:os.environ.get(k) for k in ['CARGO_TARGET_DIR','CARGO_BUILD_JOBS','CARGO_INCREMENTAL','CARGO_PROFILE_TEST_DEBUG','CARGO_PROFILE_DEV_DEBUG','RUSTFLAGS','CARGO_ENCODED_RUSTFLAGS','RUST_TEST_THREADS','UMPIRE_BACKENDS','UMPIRE_APALACHE','INSTA_UPDATE']}}
 results.append(row);(p/'final-check-results.json').write_text(json.dumps(results,indent=2)+'\n');print('FINISHED '+json.dumps(row),flush=True)
 if rc:sys.exit(rc)
