from pathlib import Path
import hashlib,re,struct,subprocess
BASELINE = '81660e841106a59ed26f7cb227462d1a85a93a9f'
base=Path('tools/gomad/internal')
def h(b):return hashlib.sha256(b).hexdigest()
def digest(b):return 'sha256:'+h(b)
def read(p,old=False):return subprocess.check_output(['git','show',BASELINE+':'+str(base/p)]) if old else (base/p).read_bytes()
def inv(path,names,prefix=b'',old=False):return prefix+b''.join((n+':'+h(read(Path(path)/n,old))+'\n').encode() for n in names)
def strings(s):return re.findall(r'"([^"]+)"',s)
def calc(old=False):
 def rd(p):return read(Path(p),old)
 def source(p):return rd(p).decode()
 def names(p):return strings(re.search(r'for _, name := range \[\]string\{(.*?)\}',source(p),re.S).group(1))
 network=digest(inv('model/network',names('model/network/identity.go'),old=old))
 world=digest(inv('model/world',names('model/world/implementation.go'),b'gomad-world-implementation/v1\n',old))
 clock=digest(inv('model/clock',names('model/clock/simulation_broker.go'),b'gomad-simulation-clock-implementation/v1\n',old))
 choice=digest(rd('model/choice/interrupted.go'))
 def field(b):return struct.pack('>Q',len(b))+b
 volume_names=strings(re.search(r'var implementationPaths = \[\]string\{(.*?)\}',source('model/volume/identity.go'),re.S).group(1))
 volume=digest(b'gomad.host-volume-implementation/v1\x00'+b''.join(field(n.encode())+field(rd('model/volume/'+n)) for n in sorted(volume_names)))
 strict=b'gomad.strict-json-implementation/v1\x00'+b''.join(field(v.encode()) for v in ['github.com/go-json-experiment/json','v0.0.0-20260820222146-c27c302e5fc3','h1:UADEEmDKgfXbtnGJZ97beY5XLo9ZechG1nlU4KnRrkE='])+b''.join(field(n.encode())+field(rd('model/strictjson/'+n)) for n in ['decode.go','identity.go'])
 strict=digest(strict)
 softpath='evidence/record/softassert' if old else 'model/softassert'
 soft=digest(inv(softpath,['identity.go','softassert.go'],b'gomad-soft-assertion/v1\n',old)+('strict-json:'+strict+'\n').encode())
 wasipath='backends/wasm/wasi' if old else 'model/wasi'
 wasi=inv(wasipath,names(wasipath+'/identity.go'),old=old)
 for label,value in [('volume-projection',volume),('network-projection',network),('interrupted-replay',choice),('broker-clock',clock),('broker-world',world),('strict-json',strict),('soft-assertion',soft)]:wasi+=(label+':'+value+'\n').encode()
 return digest(wasi)
old=calc(True);assert old=='sha256:5fddc30fd325f3168d32a0084a3e76e3a10b341da7aee86ddcd9f3b2a8b91bc3',old
new=calc();print('historical verified:',old);print('retained reference:',new)
