import os, pathlib, subprocess, json, hashlib, tarfile, io
root=pathlib.Path(__file__).resolve().parent
mise='/home/agent/.local/bin/mise'
sha='936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88'
def setup(name, config):
    d=root/name; d.mkdir(exist_ok=True)
    (d/'mise.toml').write_text(config)
    (d/'empty.toml').write_text('')
    e={k:os.environ[k] for k in ('HOME','PATH','HTTPS_PROXY','HTTP_PROXY','NO_PROXY','SSL_CERT_FILE','SSL_CERT_DIR') if k in os.environ}
    e.update({ 'MISE_DATA_DIR':str(d/'data'), 'MISE_CACHE_DIR':str(d/'cache'), 'MISE_STATE_DIR':str(d/'state'), 'MISE_CONFIG_DIR':str(d/'config'), 'MISE_GLOBAL_CONFIG_FILE':str(d/'empty.toml'), 'MISE_SYSTEM_CONFIG_FILE':str(d/'empty.toml'), 'MISE_CEILING_PATHS':str(root), 'MISE_TRUSTED_CONFIG_PATHS':str(d), 'MISE_YES':'1', 'MISE_AUTO_INSTALL':'false', 'MISE_EXEC_AUTO_INSTALL':'false', 'MISE_COLOR':'0', 'MISE_NOT_FOUND_AUTO_INSTALL':'false'})
    return d,e
def run(d,e,label,*args):
    p=subprocess.run([mise,*args],cwd=d,env=e,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=60)
    (d/(label+'.log')).write_bytes(p.stdout); (d/(label+'.status')).write_text(str(p.returncode)+'\n')
    print(d.name,label,p.returncode,p.stdout.decode(errors='replace')[-2500:])
    return p
config='''[tools."http:tlc"]
version = "1.7.4"
url = "https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar"
format = "raw"
bin = "tla2tools.jar"
checksum = "sha256:SHA"
'''
d,e=setup('http-good',config.replace('SHA',sha))
run(d,e,'config','config','ls')
run(d,e,'version','--version')
run(d,e,'install','install')
run(d,e,'where','where','http:tlc@1.7.4')
for p in (d/'data').rglob('tla2tools.jar'):
    print('jar',p.relative_to(root),p.stat().st_size,hashlib.sha256(p.read_bytes()).hexdigest())
d,e=setup('http-bad',config.replace('SHA','0'*64))
run(d,e,'install','install')
