#!/usr/bin/env python3
import json,os,pathlib,subprocess,sys,shutil,hashlib,concurrent.futures,fcntl,time,threading
binary,base,*mode=sys.argv[1:]
assert mode in ([], ['--inspect']), mode
root=pathlib.Path(base);root.mkdir(mode=0o700,parents=True,exist_ok=True)
receipt_number=0
receipt_lock=threading.Lock()
def run(*args,ok=True):
    global receipt_number
    if mode and args[0]=='fixture-open' and len(args)==2:
        args=('inspect','--json',args[1])
    p=subprocess.run([binary,*map(str,args)],capture_output=True,text=True)
    with receipt_lock:
        receipt_number+=1
        (root/f'command-{receipt_number}.json').write_text(json.dumps({'argv':[binary,*map(str,args)],'status':p.returncode,'stdout':p.stdout,'stderr':p.stderr},sort_keys=True))
    if ok: assert p.returncode==0,(args,p.returncode,p.stderr)
    else: assert p.returncode!=0,(args,p.stdout)
    return p
published=json.loads(run('fixture-publish','--root',root/'store','--seed','7').stdout)
artifact=pathlib.Path(published['artifact'])
run('fixture-open',artifact)
assert json.loads(run('fixture-publish','--root',root/'store','--seed','7').stdout)['artifact']==str(artifact)
original_manifest=(artifact/'manifest.json').read_bytes()
different=json.loads(run('fixture-publish','--root',root/'store','--seed','8').stdout)
assert different['artifact']!=str(artifact) and (artifact/'manifest.json').read_bytes()==original_manifest
run('fixture-open',different['artifact'])
def clone(name):
    path=root/name;shutil.copytree(artifact,path);return path
def dump(path,value): path.write_bytes(json.dumps(value,sort_keys=True,separators=(',',':')).encode())
def refused(name,mutate,reason):
    path=clone(name);mutate(path)
    result=run('fixture-open',path,ok=False)
    assert reason in result.stderr,(name,result.stderr)
    return name
cases=[]
run('fixture-open',clone('canonical-detached'))
def padded_envelope(p):
    path=p/'storage.json';path.write_bytes(b' '+path.read_bytes()+b'\n')
cases.append(refused('padded-envelope',padded_envelope,'noncanonical storage.json'))
cases.append(refused('digest',lambda p:(p/'output/stdout').write_bytes(b'x'*17),'mismatch'))
def record_binding(p):
    raw=b'{}';(p/'record.json').write_bytes(raw)
    manifest=json.loads((p/'manifest.json').read_bytes())
    manifest['files']['record.json']['bytes']=len(raw)
    manifest['files']['record.json']['sha256']=hashlib.sha256(raw).hexdigest()
    dump(p/'manifest.json',manifest)
cases.append(refused('record-payload-binding',record_binding,'canonical record payload identity'))
def target_kind(p):
    manifest=json.loads((p/'manifest.json').read_bytes())
    manifest['files']['target/module.wasm']['module']=False
    dump(p/'manifest.json',manifest)
cases.append(refused('target-kind-binding',target_kind,'module payload kind'))
cases.append(refused('size',lambda p:(p/'output/stdout').write_bytes(b'x'),'size mismatch'))
cases.append(refused('file-mode',lambda p:os.chmod(p/'output/stdout',0o644),'mode violation'))
cases.append(refused('directory-mode',lambda p:os.chmod(p/'output',0o755),'mode violation'))
cases.append(refused('root-mode',lambda p:os.chmod(p,0o755),'mode violation'))
cases.append(refused('manifest-mode',lambda p:os.chmod(p/'manifest.json',0o644),'mode violation'))
cases.append(refused('missing',lambda p:(p/'output/stdout').unlink(),'missing'))
cases.append(refused('unlisted',lambda p:(p/'extra').write_bytes(b'extra'),'unlisted'))
cases.append(refused('unlisted-directory',lambda p:(p/'extra').mkdir(),'unlisted'))
cases.append(refused('forbidden-link',lambda p:os.link(p/'output/stdout',root/'forbidden-link-outside'),'hard link'))
cases.append(refused('forbidden-module-link',lambda p:os.link(p/'target/module.wasm',root/'forbidden-module-link-outside'),'pool'))
def symlink(p):
    (p/'output/stdout').unlink();os.symlink('/etc/passwd',p/'output/stdout')
cases.append(refused('symlink-file',symlink,'type'))
def symlink_directory(p):
    shutil.rmtree(p/'output');os.symlink('/etc',p/'output')
cases.append(refused('symlink-directory',symlink_directory,'unlisted'))
def fifo(p):
    (p/'output/stdout').unlink();os.mkfifo(p/'output/stdout',0o600)
cases.append(refused('fifo',fifo,'type'))
def traversal(p):
    m=json.loads((p/'manifest.json').read_bytes());m['files']['../outside']=m['files'].pop('output/stdout');dump(p/'manifest.json',m)
cases.append(refused('traversal',traversal,'unsafe payload path'))
def manifest_bound(p):
    with (p/'manifest.json').open('wb') as f:f.truncate((16<<20)+1)
cases.append(refused('manifest-bound',manifest_bound,'exceeds bound'))
def old(p):
    dump(p/'storage.json',{'format':'gomad.artifact-storage/v1'});(p/'manifest.json').write_bytes(b'not-json')
cases.append(refused('old-lineage',old,'unsupported artifact storage format'))
rootlink=root/'root-link';os.symlink(artifact,rootlink)
assert 'type' in run('fixture-open',rootlink,ok=False).stderr
pinned=clone('pinned');run('fixture-pinned',pinned)
substituted=clone('substituted')
assert 'changed while opening' in run('fixture-root-substitution',substituted,ok=False).stderr
assert 'changed while opening' in run('fixture-payload-substitution',clone('payload-substitution'),ok=False).stderr
assert 'changed while reading' in run('fixture-payload-mutation',clone('payload-mutation'),ok=False).stderr
cases.extend(['payload-preopen-substitution','payload-during-read-mutation'])
assert 'caller bound' in run('fixture-bound',artifact,ok=False).stderr
budget=run('fixture-publish','--root',root/'budget','--seed','7','--budget','1',ok=False)
assert budget.returncode==3 and 'budget exceeded' in budget.stderr
retention=root/'retained-budget'
bound=str(published['stored_bytes']*2-1)
run('fixture-publish','--root',retention,'--seed','7','--budget',bound)
assert 'budget exceeded' in run('fixture-publish','--root',retention,'--seed','8','--budget',bound,ok=False).stderr
for point in ['stage_created','payloads_durable','envelope_durable','manifest_durable','staging_durable','before_rename','after_rename','store_durable']:
    store=root/('crash-'+point)
    result=run('fixture-publish','--root',store,'--seed','7','--fault','crash:'+point,ok=False)
    assert result.returncode==86 and not result.stdout,(point,result.returncode,result.stdout)
    for candidate in store.iterdir():
        if not candidate.name.startswith('.'):run('fixture-open',candidate)
    recovered=json.loads(run('fixture-publish','--root',store,'--seed','7').stdout)
    run('fixture-open',recovered['artifact'])
    cases.append('crash-'+point)
for point in ['before_reuse_sync','after_reuse_sync']:
    result=run('fixture-publish','--root',root/'store','--seed','7','--fault','crash:'+point,ok=False)
    assert result.returncode==86 and not result.stdout
    run('fixture-open',artifact)
    cases.append('crash-'+point)
for fault,reason in [('store_sync_error','sync artifact store'),('rename_unsupported','exclusive artifact publication refused')]:
    fault_store=root/fault
    failed=run('fixture-publish','--root',fault_store,'--seed','7','--fault',fault,ok=False)
    assert failed.returncode==3 and not failed.stdout and reason in failed.stderr
    recovered=json.loads(run('fixture-publish','--root',fault_store,'--seed','7').stdout)
    run('fixture-open',recovered['artifact'])
    cases.append(fault)
pool=root/'pool'
a=json.loads(run('fixture-publish','--root',root/'pool-store-a','--seed','7','--pool',pool).stdout)['artifact']
b=json.loads(run('fixture-publish','--root',root/'pool-store-b','--seed','8','--pool',pool).stdout)['artifact']
assert os.stat(pathlib.Path(a)/'target/module.wasm').st_ino==os.stat(pathlib.Path(b)/'target/module.wasm').st_ino
run('fixture-open',a);run('fixture-open',b)
substitution_store=root/'pool-substitute'
substitution_artifact=json.loads(run('fixture-publish','--root',substitution_store,'--seed','9','--pool',pool).stdout)['artifact']
local_pool=substitution_store/'.targets'
os.rename(local_pool,substitution_store/'.targets-original')
local_pool.mkdir(mode=0o700)
for source in (substitution_store/'.targets-original').iterdir():
    if source.name!='.lock':shutil.copyfile(source,local_pool/source.name);os.chmod(local_pool/source.name,0o600)
assert 'pool inode mismatch' in run('fixture-open',substitution_artifact,ok=False).stderr
shutil.rmtree(local_pool)
os.symlink(substitution_store/'.targets-original',local_pool)
assert 'type' in run('fixture-open',substitution_artifact,ok=False).stderr
same_bytes=clone('same-bytes-module')
os.link(same_bytes/'target/module.wasm',root/'same-bytes-external')
assert 'pool inode mismatch' in run('fixture-open',same_bytes,'--pool',pool,ok=False).stderr
concurrent_store=root/'concurrent'
with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
    publications=list(executor.map(lambda _:json.loads(run('fixture-publish','--root',concurrent_store,'--seed','7').stdout)['artifact'],range(8)))
assert len(set(publications))==1
run('fixture-open',publications[0])
lockstore=root/'lock-store';run('fixture-publish','--root',lockstore,'--seed','7')
with (lockstore/'.lock').open('rb') as lock:
    fcntl.flock(lock,fcntl.LOCK_EX)
    waiter=subprocess.Popen([binary,'fixture-publish','--root',str(lockstore),'--seed','8'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    time.sleep(.1)
    assert waiter.poll() is None,'publisher ignored real OS lock'
    fcntl.flock(lock,fcntl.LOCK_UN)
    out,err=waiter.communicate(timeout=10)
    assert waiter.returncode==0,err
cases.extend(['root-link','pinned-path-replacement','root-substitution','caller-bound','budget','retained-budget','non-equivalent-collision','pool-links','pool-root-inode-substitution','pool-root-symlink','same-bytes-module-pool-mismatch','concurrent-collision','lock-contention'])
review=subprocess.run([sys.executable,str(pathlib.Path(__file__).with_name('foundation-store-review-check.py')),binary,str(root/'review-regressions')],capture_output=True,text=True)
(root/'review-regressions.log').write_text(review.stdout+review.stderr)
assert review.returncode==0,review.stderr
cases.extend(['sparse-pool-caller-bound','interrupted-ancestor-recovery-sync','concurrent-ancestor-sync'])
cases.extend('immutable-'+scope+'-'+point+'-recovery' for scope in ('pool','layout')
             for point in ('file_created','file_partial','before_file_sync','file_durable','file_published'))
print(json.dumps({'passed':cases,'count':len(cases)},sort_keys=True))
