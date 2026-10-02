import pathlib, subprocess, sys, json, hashlib, time
root=pathlib.Path.cwd(); a=pathlib.Path(__file__).resolve().parent
name=sys.argv[1]; cmd=sys.argv[2]
files=sorted(set(json.loads((a/'owned-files.json').read_text()))|set(json.loads((a/'parent-protected-source-check.json').read_text())['sources']))
def hashes(): return {f:hashlib.sha256((root/f).read_bytes()).hexdigest() if (root/f).exists() else None for f in files}
before=hashes(); started=time.time()
with (a/(name+'.log')).open('w') as log: result=subprocess.run(cmd,shell=True,executable='/bin/zsh',stdout=log,stderr=subprocess.STDOUT)
receipt={'command':cmd,'exit_code':result.returncode,'seconds':time.time()-started,'source_before':before,'source_after':hashes(),'toolchain_build_key':(root/'tools/gomad3/.toolchain/build-key').read_text().strip(),'log_sha256':hashlib.sha256((a/(name+'.log')).read_bytes()).hexdigest()}
(a/(name+'.receipt.json')).write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps({'name':name,'exit_code':result.returncode,'seconds':receipt['seconds'],'sources_unchanged':before==receipt['source_after']}))
print((a/(name+'.log')).read_text()[-3500:])
sys.exit(result.returncode)
