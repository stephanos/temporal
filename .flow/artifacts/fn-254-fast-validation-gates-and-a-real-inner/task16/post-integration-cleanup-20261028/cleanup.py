from pathlib import Path
import datetime, json, os, runpy, subprocess

REPO=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
HERE=Path(__file__).resolve().parent
CHILD=Path('/tmp/umpire-fn25416-final-60f1c762a618-4090033/proof-5XlzyE')
PARENT=REPO/'.flow/tmp/fn25416-provider-validation/target'
CARGO='/home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/bin/cargo'

def allocated(root):
    if not root.exists(): return 0
    seen=set(); total=0
    for parent, directories, files in os.walk(root):
        for name in directories+files:
            info=os.lstat(os.path.join(parent,name)); key=info.st_dev,info.st_ino
            if key not in seen: seen.add(key); total+=info.st_blocks*512
    return total

def free():
    return {str(p):os.statvfs(p).f_bavail*os.statvfs(p).f_frsize for p in [Path('/tmp'),REPO]}

def ownership(target):
    prefix=os.fsencode(target); owners=[]; inaccessible=[]
    for proc in Path('/proc').iterdir():
        if not proc.name.isdigit() or int(proc.name)==os.getpid(): continue
        try:
            matches=[]
            for name in ['cwd','exe']:
                try:
                    value=os.fsencode(os.readlink(proc/name))
                    if value==prefix or value.startswith(prefix+b'/'): matches.append(name)
                except FileNotFoundError: pass
            try:
                if prefix in (proc/'environ').read_bytes(): matches.append('environment')
            except FileNotFoundError: pass
            try:
                for fd in (proc/'fd').iterdir():
                    try:
                        value=os.fsencode(os.readlink(fd))
                        if value==prefix or value.startswith(prefix+b'/'): matches.append('fd:'+fd.name)
                    except FileNotFoundError: pass
            except FileNotFoundError: pass
            try:
                if prefix in (proc/'maps').read_bytes(): matches.append('memory-map')
            except FileNotFoundError: pass
            if matches: owners.append({'pid':int(proc.name),'matches':matches})
        except (FileNotFoundError,ProcessLookupError): pass
        except PermissionError:
            identity=dict(pid=int(proc.name),uid=proc.stat().st_uid,comm=(proc/'comm').read_text().strip())
            assert identity['uid']==0 and {10:'socat',542:'dockerd',557:'containerd'}.get(identity['pid'])==identity['comm'],identity
            assert prefix not in (proc/'cmdline').read_bytes(),identity
            identity.update(available_argv_target_prefix_match=False,unreadable_metadata='unknown foreign infrastructure')
            inaccessible.append(identity)
    assert not owners, owners
    return dict(owners=owners,inaccessible_foreign_root_infrastructure=inaccessible,checks=['cwd','exe','environment','open-fds-including-locks','memory-maps'])

def verify(which):
    if which=='child':
        location=REPO/'.flow/tmp/fn25416-proof-retention-20261028'
        ns=runpy.run_path(str(location/'verify.py'),run_name='source_free_verify')
        return ns['verify'](json.loads((location/'manifest.json').read_text()))
    location=REPO/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/standalone-named-0547321/release-retention'
    ns=runpy.run_path(str(location/'verify-recovery.py'),run_name='source_free_verify')
    return ns['verify'](json.loads((location/'recovery-manifest.json').read_text()),json.loads((location/'archive-manifest.json').read_text()))

def main():
    receipt=dict(kind='authorized-post-integration-exact-cache-cleanup',task='fn-254-fast-validation-gates-and-a-real-inner.16',
        source_closure='36f2cd46c5d30748de93c987237a2b1926375a7d',integrated_main='1b20fbe795',actions=[])
    assert PARENT.is_dir() and (CHILD/'framework').is_dir()
    parent_before=allocated(PARENT)
    for label,target in [('child',CHILD/'target'),('old-release',REPO/'.flow/tmp/fn258-standalone-release-0547321')]:
        row=dict(label=label,target=str(target),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),allocated_before=allocated(target),free_before=free())
        row['recovery_before']=verify('child' if label=='child' else 'old')
        row['ownership_before']=ownership(target)
        command=[CARGO,'clean','--target-dir',str(target)]
        env=os.environ.copy();env.pop('BASH_ENV',None)
        row.update(command=command,cwd=str(REPO/'model/umpire'))
        completed=subprocess.run(command,cwd=REPO/'model/umpire',env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        (HERE/(label+'.stdout.log')).write_bytes(completed.stdout)
        (HERE/(label+'.stderr.log')).write_bytes(completed.stderr)
        row.update(exit_code=completed.returncode,target_absent=not target.exists(),allocated_after=allocated(target),free_after=free())
        row['recovery_after']=verify('child' if label=='child' else 'old')
        row['ownership_after']=ownership(target)
        row['allocated_bytes_removed']=row['allocated_before']-row['allocated_after']
        receipt['actions'].append(row)
        (HERE/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
        print(json.dumps(row),flush=True)
        assert completed.returncode==0
        if target.exists():
            print('REMAINDERS_REQUIRE_INSPECTION',str(target),flush=True)
            break
    receipt['protected_paths_present']={str(p):p.exists() for p in [PARENT,CHILD/'framework',CHILD/'result.json',REPO.parent/'.recovery/fn25416-provider-proof/objects',REPO.parent/'.recovery/fn258-standalone-proof/objects']}
    receipt['parent_allocated_before']=parent_before;receipt['parent_allocated_after']=allocated(PARENT)
    receipt['complete']=len(receipt['actions'])==2 and all(x['target_absent'] for x in receipt['actions']) and all(receipt['protected_paths_present'].values())
    receipt['free_space_note']='Observed filesystem changes include concurrent activity; allocated target bytes are reported separately.'
    (HERE/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')

if __name__=='__main__':main()
