exec(open(__file__.replace('probe-optional.py','probe.py')).read().split("config='''")[0])
d,e=setup('opt-in','[tools]\n')
(d/'mise.apalache.toml').write_text('''[tools."http:apalache"]
version = "0.62.4"
url = "file:///definitely-absent-apalache.tgz"
checksum = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
''')
run(d,e,'default-ls','ls','--json')
run(d,e,'default-exec','exec','--','/bin/true')
run(d,e,'opt-in-ls','--env','apalache','ls','--json')
run(d,e,'opt-in-exec','--env','apalache','exec','--','/bin/true')
for case in ['http-good','github-good','archive-layout']:
    for p in (root/case/'data/installs').glob('*/*'):
        if p.name[0].isdigit(): print('install-root',p,'symlink',p.is_symlink())
