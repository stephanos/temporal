exec(open(__file__.replace('probe-more.py','probe.py')).read().split("config='''")[0])
config='''[tools."github:tlaplus/tlaplus"]
version = "1.7.4"
asset_pattern = "tla2tools.jar"
format = "raw"
bin = "tla2tools.jar"
checksum = "sha256:SHA"
'''
for name,h in [('github-good',sha),('github-bad','0'*64)]:
    d,e=setup(name,config.replace('SHA',h))
    run(d,e,'config','config','ls')
    run(d,e,'install','install')
archive=root/'apalache-layout-fixture.tgz'
with tarfile.open(archive,'w:gz') as t:
    for name,data,mode in [('apalache-0.62.4/lib/apalache.jar',b'fixture-jar',0o644),('apalache-0.62.4/bin/apalache-mc',b'#!/bin/sh\nexit 0\n',0o755)]:
        i=tarfile.TarInfo(name); i.size=len(data); i.mode=mode; t.addfile(i,io.BytesIO(data))
config=f'''[tools."http:apalache-fixture"]
version = "0.62.4"
url = "{archive.as_uri()}"
format = "tar.gz"
strip_components = 1
bin_path = "bin"
checksum = "sha256:{hashlib.sha256(archive.read_bytes()).hexdigest()}"
'''
d,e=setup('archive-layout',config)
run(d,e,'install','install')
run(d,e,'where','where','http:apalache-fixture@0.62.4')
print('archive JAR', (d/'data/installs/http-apalache-fixture/0.62.4/lib/apalache.jar').read_bytes())
d=root/'http-good'; config=(d/'mise.toml').read_text(); d,e=setup('http-good',config)
run(d,e,'lock','lock','--platform','linux-arm64')
jar=d/'data/installs/http-tlc/1.7.4/tla2tools.jar'
original=jar.read_bytes(); (d/'original-jar.sha256').write_text(hashlib.sha256(original).hexdigest()+'\n')
jar.write_bytes(b'CORRUPTED BY ISOLATED RESEARCH PROBE\n')
run(d,e,'locked-corrupt','install','--locked')
(d/'after-locked.sha256').write_text(hashlib.sha256(jar.read_bytes()).hexdigest()+'\n')
print('corrupt remains',jar.read_bytes()==b'CORRUPTED BY ISOLATED RESEARCH PROBE\n')
jar.write_bytes(original)
