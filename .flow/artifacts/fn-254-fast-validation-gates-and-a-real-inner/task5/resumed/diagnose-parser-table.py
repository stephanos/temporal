from pathlib import Path
import subprocess,os,json,time,hashlib
r=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal');o=r/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task5/resumed';candidate=r/'.flow/tmp/fn2546-resumed/candidate';path=candidate/'model/umpire/author/dsl/macros/tests/parser_refusals/mod.rs';original=path.read_bytes();s=original.decode();s=s.replace('''    for case in cases::CASES {
        let observed = expand(case).map_err(|error| format!("{}: {error}", case.name))?;''','''    let mut failures = Vec::new();
    for case in cases::CASES {
        let observed = match expand(case) {
            Ok(observed) => observed,
            Err(error) => { failures.push(format!("{}: {error}", case.name)); continue; }
        };''');s=s.replace('''        assert_eq!(observed, expected, "{}", case.name);
    }
    Ok(())''','''        if observed != expected {
            failures.push(format!("{}: observed {observed:?}; expected {expected:?}", case.name));
        }
    }
    if failures.is_empty() { Ok(()) } else { Err(failures.join("\\n")) }''',1);assert s.encode()!=original
env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(r/'.flow/tmp/fn2546-resumed/target'),TMPDIR='/tmp/umpire-fn2545-parser-fixture-controls',CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',CARGO_PROFILE_DEV_DEBUG='0',CARGO_PROFILE_TEST_DEBUG='0',RUST_TEST_THREADS='2',UMPIRE_BACKENDS='skip',UMPIRE_APALACHE='skip',MISE_TRUSTED_CONFIG_PATHS=str(candidate))
cmd=['mise','exec','--','cargo','test','--locked','-p','umpire-dsl-macros','--test','fixtures','parser_refusals::','--','--nocapture'];start=time.monotonic()
try:
 path.write_text(s)
 with (o/'parser-table-all-mismatches.log').open('w') as f:p=subprocess.run(cmd,cwd=candidate/'model/umpire',env=env,stdout=f,stderr=subprocess.STDOUT)
 (o/'parser-table-all-mismatches.json').write_text(json.dumps({'command':cmd,'exit_code':p.returncode,'elapsed_seconds':time.monotonic()-start,'scope':'Temporary isolated-candidate instrumentation collects all mismatches without accepting any; root source untouched.'},indent=2)+'\n');print('diagnostic exit',p.returncode)
finally:path.write_bytes(original);assert hashlib.sha256(path.read_bytes()).hexdigest()==hashlib.sha256(original).hexdigest()
