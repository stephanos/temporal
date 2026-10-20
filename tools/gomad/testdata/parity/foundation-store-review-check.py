#!/usr/bin/env python3
"""Regression probes for ancestor durability and caller-bounded pool opening."""
import hashlib,json,os,pathlib,resource,subprocess,sys,time

binary,base,*selection=sys.argv[1:]
root=pathlib.Path(base);root.mkdir(parents=True,exist_ok=True,mode=0o700)
def run(*args,limit=False):
    def bounded(): resource.setrlimit(resource.RLIMIT_AS,(256<<20,256<<20))
    p=subprocess.run([binary,*map(str,args)],capture_output=True,text=True,
                     preexec_fn=bounded if limit else None)
    with (root/'commands.jsonl').open('a') as f:
        f.write(json.dumps(dict(argv=[binary,*map(str,args)],status=p.returncode,stdout=p.stdout,stderr=p.stderr))+'\n')
    return p
def publish(store,*args): return run('fixture-publish','--root',store,'--seed','7',*args)
def dump(path,value): path.write_bytes(json.dumps(value,sort_keys=True,separators=(',',':')).encode())
if not selection or selection==['pool']:
    store=root/'large-store';p=publish(store);assert p.returncode==0,p.stderr
    artifact=pathlib.Path(json.loads(p.stdout)['artifact'])
    size=1<<30;chunk=bytes(1<<20);digest=hashlib.sha256()
    for _ in range(size//len(chunk)): digest.update(chunk)
    target=digest.hexdigest();pool=store/'.targets'/target
    with pool.open('wb') as f:f.truncate(size)
    pool.chmod(0o600)
    module=artifact/'target/module.wasm';module.unlink();os.link(pool,module)
    record=json.loads((artifact/'record.json').read_bytes());record['identity']['target']=target
    identity=json.dumps(record['identity'],sort_keys=True,separators=(',',':')).encode()
    record['execution_identity']=hashlib.sha256(identity).hexdigest()
    dump(artifact/'record.json',record);raw=(artifact/'record.json').read_bytes()
    manifest=json.loads((artifact/'manifest.json').read_bytes())
    manifest['record']=record;manifest['record_sha256']=hashlib.sha256(raw).hexdigest()
    manifest['files']['record.json'].update(bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest())
    manifest['files']['target/module.wasm'].update(bytes=size,sha256=target)
    dump(artifact/'manifest.json',manifest)
    p=run('fixture-bound',artifact,limit=True)
    assert p.returncode==2 and 'caller bound' in p.stderr,(p.returncode,p.stderr)
if not selection or selection==['durability']:
    store=root/'recovery';p=publish(store,'--fault','crash:directory_created')
    assert p.returncode==86 and store.is_dir() and not p.stdout,(p.returncode,p.stderr)
    p=publish(store,'--fault','before_parent_sync')
    assert p.returncode==3 and 'before_parent_sync' in p.stderr and not p.stdout,(p.returncode,p.stdout,p.stderr)
if not selection or selection in (['durability'],['concurrent']):
    store=root/'concurrent'
    a=subprocess.Popen([binary,'fixture-publish','--root',str(store),'--seed','7','--fault','pause:directory_created'],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    try:
        deadline=time.monotonic()+10
        while not store.with_suffix('.paused').exists():
            assert a.poll() is None, a.communicate()
            assert time.monotonic()<deadline,'creator did not pause'
            time.sleep(.01)
        p=publish(store,'--fault','before_parent_sync')
        assert p.returncode==3 and not p.stdout,(p.returncode,p.stdout,p.stderr)
        p=publish(store,'--fault','audit:parent_durable')
        assert p.returncode==0 and store.with_suffix('.parent-synced').exists(),(p.returncode,p.stdout,p.stderr)
        assert run('fixture-open',json.loads(p.stdout)['artifact']).returncode==0
    finally:
        store.with_suffix('.release').touch(mode=0o600)
        out,err=a.communicate(timeout=10)
        assert a.returncode==0,(a.returncode,out,err)
print('store review regressions passed')
