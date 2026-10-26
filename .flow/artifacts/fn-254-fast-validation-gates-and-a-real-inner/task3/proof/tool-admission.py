import hashlib,json,os,pathlib,shutil,subprocess,tomllib
W=pathlib.Path(__file__).resolve().parents[5];P=pathlib.Path(__file__).resolve().parent
S=W/'.flow/tmp/fn2543-tool-admission';(S/'independent').mkdir(parents=True,exist_ok=True)
for f in ['mise.toml','mise.lock']:shutil.copyfile(W/'model/umpire'/f,S/'independent'/f)
E=dict(os.environ);E.pop('BASH_ENV',None);E.update(MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false')
rows=[]
for name,cwd in [('root',W),('framework',W/'model/umpire'),('framework-independent',S/'independent')]:
 env=E.copy()
 if name.endswith('independent'):env.update(MISE_CEILING_PATHS=str(S),MISE_TRUSTED_CONFIG_PATHS=str(S/'independent'))
 for command in [['mise','ls','--installed','--json','github:nextest-rs/nextest'],['mise','exec','--','cargo','nextest','--version']]:
  r=subprocess.run(command,cwd=cwd,env=env,capture_output=True,text=True);assert r.returncode==0,(r.stdout,r.stderr)
  rows.append(dict(name=name,cwd=str(cwd),argv=command,exit=r.returncode,stdout=r.stdout,stderr=r.stderr))
versions=[r for r in rows if r['argv'][-1]=='--version'];assert all('cargo-nextest 0.9.149' in r['stdout'] for r in versions)
locks={str(p.relative_to(W)):{'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'value':tomllib.loads(p.read_text())} for p in [W/'mise.lock',W/'model/umpire/mise.lock']}
(P/'tool-admission.json').write_text(json.dumps(dict(commands=rows,locks=locks,installation='mise lock github:nextest-rs/nextest in each public config; mise install github:nextest-rs/nextest under normal root config; official precompiled Github release, no Cargo install or global Mise upgrade',guest_parser='guest.rs Tools deserializes rust/go/protoc as String and ignores unknown nextest field; existing guest_pipeline controls executed separately'),indent=2)+'\n')
print('public and independent framework tool admission passed')
