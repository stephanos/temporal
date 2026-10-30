import os,pathlib,subprocess,json,hashlib,stat
repo=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
s=repo/'.flow/tmp/fn25416-apalache-probe'
b=pathlib.Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn25416-apalache-probe')
b=b/'clean-context'
b.mkdir()
config='''[tools."http:apalache"]
version = "0.62.4"
url = "https://github.com/apalache-mc/apalache/releases/download/v0.62.4/apalache-0.62.4.tgz"
format = "tar.gz"
strip_components = 1
bin_path = "bin"
checksum = "sha256:b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4"
'''
(b/'mise.toml').write_text('[tools]\n')
(b/'mise.apalache.toml').write_text(config)
(b/'empty.toml').write_text('')
for p in ['mise.toml','mise.apalache.toml','empty.toml']:
 (s/p).write_bytes((b/p).read_bytes())
e={k:os.environ[k] for k in ('HOME','PATH','HTTPS_PROXY','HTTP_PROXY','NO_PROXY','SSL_CERT_FILE','SSL_CERT_DIR') if k in os.environ}
e.update({ 'MISE_DATA_DIR':str(b/'data'),'MISE_CACHE_DIR':str(b/'cache'),'MISE_STATE_DIR':str(b/'state'),'MISE_CONFIG_DIR':str(b/'config'),'MISE_GLOBAL_CONFIG_FILE':str(b/'empty.toml'),'MISE_SYSTEM_CONFIG_FILE':str(b/'empty.toml'),'MISE_CEILING_PATHS':str(b.parent),'MISE_TRUSTED_CONFIG_PATHS':str(b),'MISE_YES':'1','MISE_AUTO_INSTALL':'false','MISE_EXEC_AUTO_INSTALL':'false','MISE_COLOR':'0','MISE_NOT_FOUND_AUTO_INSTALL':'false'})
(s/'resolver-environment.json').write_text(json.dumps({k:v for k,v in e.items() if k.startswith('MISE_') or k=='HOME'},indent=2)+'\n')
commands=[]
def run(label,args,cwd=b,env=e,timeout=180):
 commands.append({'label':label,'argv':args,'cwd':str(cwd),'timeout':timeout})
 (s/'commands.json').write_text(json.dumps(commands,indent=2)+'\n')
 try:
  p=subprocess.run(args,cwd=cwd,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=timeout)
 except subprocess.TimeoutExpired as ex:
  (s/(label+'.log')).write_bytes(ex.stdout or b'');(s/(label+'.status')).write_text('timeout\n');raise
 (s/(label+'.log')).write_bytes(p.stdout);(s/(label+'.status')).write_text(str(p.returncode)+'\n')
 print(label,p.returncode,p.stdout.decode(errors='replace')[-1500:],flush=True)
 return p
mise='/home/agent/.local/bin/mise'
run('mise-version',[mise,'--version'])
run('default-tools',[mise,'ls','--json'])
run('opt-in-config',[mise,'--env','apalache','config','ls'])
p=run('install',[mise,'--env','apalache','install','http:apalache@0.62.4'])
if p.returncode: raise SystemExit(p.returncode)
p=run('where',[mise,'--env','apalache','where','http:apalache@0.62.4'])
if p.returncode: raise SystemExit(p.returncode)
installed=pathlib.Path(p.stdout.decode().strip()); jar=installed/'lib/apalache.jar'
assert installed.is_absolute() and installed.is_relative_to(b/'data/installs')
metadata=[]
for path in [b/'data',*list(installed.relative_to(b/'data').parents)[::-1]]: pass
for path in [b/'data',b/'data/installs',*list(jar.parents)[::-1],jar]:
 if not path.is_relative_to(b/'data'): continue
 st=path.lstat(); assert not stat.S_ISLNK(st.st_mode),path
 metadata.append({'path':str(path),'mode':oct(st.st_mode),'type':'file' if stat.S_ISREG(st.st_mode) else 'directory'})
assert stat.S_ISREG(jar.lstat().st_mode) and jar.resolve()==jar
h=hashlib.file_digest(jar.open('rb'),'sha256').hexdigest()
assert h=='e5223915507b65cc5f1638bd037ede60f8a4fc1644aff46355779f9b05df9262'
(s/'admission.json').write_text(json.dumps({'installed':str(installed),'jar':str(jar),'jar_bytes':jar.stat().st_size,'jar_sha256':h,'path_components':metadata},indent=2)+'\n')
archives=[]
for path in (b/'cache').rglob('*'):
 if path.is_file() and path.stat().st_size==192038424:
  ah=hashlib.file_digest(path.open('rb'),'sha256').hexdigest(); archives.append({'path':str(path),'bytes':path.stat().st_size,'sha256':ah}); assert ah=='b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4'
(s/'archives.json').write_text(json.dumps(archives,indent=2)+'\n')
assert archives,'archive not retained by mise; investigate before runtime'
j=pathlib.Path('/home/agent/.local/share/mise/installs/java/temurin-27/bin/java')
je=e.copy()
for k in ['JAVA_TOOL_OPTIONS','JDK_JAVA_OPTIONS','_JAVA_OPTIONS','CLASSPATH','TLA_LIBRARY']: je.pop(k,None)
run('java-version',[str(j),'-version'],env=je)
work=b/'runtime';work.mkdir()
p=run('apalache-version',[str(j),'-Xmx512m',f'-Duser.home={work}',f'-Djava.io.tmpdir={work}','-jar',str(jar),'version'],cwd=work,env=je,timeout=30)
assert p.returncode==0 and '0.62.4' in p.stdout.decode()
print('PASS exact real AP clean install and non-solver version admission',flush=True)
