import json,os,shutil,subprocess,tempfile,time,tomllib
from pathlib import Path
checkout=Path.cwd().resolve();framework=checkout/'model/umpire';proof=checkout/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task10/revision1/proof'
target=Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn25410-gate/parent');binary=target/'debug/xtask'
env=os.environ.copy();env.pop('BASH_ENV',None);env.update(CARGO_TARGET_DIR=str(target),CARGO_BUILD_JOBS='2',RUST_TEST_THREADS='2',CARGO_INCREMENTAL='0')
commands=[]
def run(label,root,expected,reason=None):
 e=env|{'UMPIRE_TOOL_ROOT':str(root),'UMPIRE_QUICK_CACHE':'1','UMPIRE_CACHE_DISABLE':'0'};argv=[str(binary),'gate','--stage','source-boundary'];start=time.monotonic();p=subprocess.run(argv,cwd=checkout,env=e,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 commands.append(dict(label=label,argv=argv,cwd=str(checkout),tool_root=str(root),exit=p.returncode,expected=expected,seconds=time.monotonic()-start,output=p.stdout.decode(errors='replace')))
 (proof/'compiled-boundary-controls.json').write_text(json.dumps(dict(scope='real compiled scanner on synthetic declared inputs; not a standalone full gate',commands=commands),indent=2)+'\n')
 assert p.returncode==expected,(label,p.stdout)
 if reason:assert reason.encode() in p.stdout,(label,p.stdout)
run('actual-framework-current-inputs',framework,0)
with tempfile.TemporaryDirectory(prefix='fn25410-boundary-') as scratch:
 base=Path(scratch);root=base/'framework';root.mkdir();policy=root/'dev/gate/policy/framework-boundary.txt';policy.parent.mkdir(parents=True);shutil.copyfile(framework/'dev/gate/policy/framework-boundary.txt',policy)
 for name in ['mise.toml','rust-toolchain.toml']:shutil.copyfile(framework/name,root/name)
 (root/'Cargo.toml').write_text('[package]\nname="boundary-control"\nversion="0.1.0"\nedition="2024"\n[workspace]\n');(root/'src').mkdir();(root/'src/lib.rs').write_text('');(root/'Cargo.lock').write_text('version = 4\n[[package]]\nname="boundary-control"\nversion="0.1.0"\n')
 data=tomllib.loads(policy.read_text())
 for n,text in enumerate(data['controls']):
  (root/'README.md').write_text(text+'\n');run('organization-or-source-negative-'+str(n),root,1,'framework forbidden name')
 for n,text in enumerate(data['accepted_controls']):
  (root/'README.md').write_text(text+'\n');run('organization-positive-'+str(n),root,0)
 (root/'README.md').unlink();(base/'outside.txt').write_text('outside\n')
 for n,text in enumerate([r'''const QUOTE: char = '"';''',r'''const QUOTE: u8 = b'"';''',r'''const QUOTE: char = '\"';''',r'''const QUOTE: u8 = b'\"';''',"fn f<'a>(s: &'a str) {}"]):
  (root/'source.rs').write_text(text+'\nconst TEXT: &str = include_str!("../outside.txt");\n');run('character-or-lifetime-escape-'+str(n),root,1,'framework source-path containment')
print('real compiled boundary: current tree,17 negatives,2 positives,5 escaping include controls passed')
