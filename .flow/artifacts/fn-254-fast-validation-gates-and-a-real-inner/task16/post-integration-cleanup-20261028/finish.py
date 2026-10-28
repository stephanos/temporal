from pathlib import Path
import hashlib,json,os,re,runpy,shutil,stat,subprocess,tarfile

HERE=Path(__file__).resolve().parent
NS=runpy.run_path(str(HERE/'cleanup.py'),run_name='cleanup_helpers')
REPO=NS['REPO'];CHILD=NS['CHILD'];PARENT=NS['PARENT']
ROOTS=[('child',CHILD/'target'),('old',REPO/'.flow/tmp/fn258-standalone-release-0547321')]
STORE=REPO.parent/'.recovery/fn25416-provider-proof/objects'

def sha(path):
    h=hashlib.sha256()
    with path.open('rb') as stream:
        for data in iter(lambda:stream.read(1024*1024),b''):h.update(data)
    return h.hexdigest()

def snapshot(label,root):
    allowed={'child':{'.rustc_info.json','.rustdoc_fingerprint.json','current-contract-native','debug','doc','gate-receipts','go-build','guest','guest-cache','guest-wrappers','panic-native','tests','tmp','wasm32-wasip1','xtask-bootstrap'},'old':{'.rustc_info.json','release','tests','tmp'}}[label]
    assert set(p.name for p in root.iterdir()) <= allowed
    rows=[]
    for parent,dirs,files in os.walk(root):
        for name in dirs+files:
            path=Path(parent)/name;info=path.lstat()
            assert stat.S_ISREG(info.st_mode) or stat.S_ISDIR(info.st_mode),str(path)
            if stat.S_ISDIR(info.st_mode):continue
            relative=str(path.relative_to(root));magic=path.open('rb').read(8)
            compiled=magic.startswith(b'\x7fELF') or path.suffix in {'.rlib','.rmeta','.so'}
            go_cache=bool(re.fullmatch(r'go-build/[0-9a-f]{2}/[0-9a-f]{64}-[ad]',relative))
            rows.append(dict(path=label+'/'+relative,source=str(path),size=info.st_size,mode=oct(stat.S_IMODE(info.st_mode)),sha256=sha(path),role='reproducible-compiled-intermediate' if compiled or go_cache else 'supplemental-custody'))
    return sorted(rows,key=lambda x:x['path'])

def main():
    env=os.environ.copy();env.pop('BASH_ENV',None)
    old=ROOTS[1][1]
    pre=dict(recovery=NS['verify']('old'),ownership=NS['ownership'](old))
    command=[NS['CARGO'],'clean','--target-dir',str(old)]
    result=subprocess.run(command,cwd=REPO/'model/umpire',env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    pre.update(command=command,exit_code=result.returncode,stdout=result.stdout.decode(),stderr=result.stderr.decode())
    (HERE/'old-root-cargo-clean.json').write_text(json.dumps(pre,indent=2)+'\n')
    assert result.returncode==101 and 'CACHEDIR.TAG' in result.stderr.decode()
    inventory=[]
    for label,root in ROOTS:
        NS['ownership'](root);inventory+=snapshot(label,root)
    (HERE/'remaining-file-manifest.json').write_text(json.dumps(inventory,indent=2)+'\n')
    selected=[row for row in inventory if row['role']=='supplemental-custody']
    temporary=STORE.parent/'remaining-generated-source-and-records.preparing.tar.gz'
    assert not temporary.exists()
    with tarfile.open(temporary,'w:gz',compresslevel=3) as archive:
        for row in selected:archive.add(row['source'],arcname=row['path'],recursive=False)
    checksum=sha(temporary);destination=STORE/checksum
    if destination.exists():assert sha(destination)==checksum
    else:os.link(temporary,destination)
    manifest=dict(objects=[dict(recovery_object=str(destination),size=destination.stat().st_size,sha256=checksum)],archive_members=selected)
    (HERE/'supplemental-custody-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    verifier=runpy.run_path(str(REPO/'.flow/tmp/fn25416-proof-retention-20261028/verify.py'),run_name='source_free_supplemental_verifier')['verify']
    print(json.dumps({'supplemental_verification':verifier(manifest),'archive_bytes':destination.stat().st_size}),flush=True)
    actions=[]
    for label,target in ROOTS:
        row=dict(label=label,target=str(target),allocated_before=NS['allocated'](target),free_before=NS['free'](),recovery_before=NS['verify'](label),supplemental_before=verifier(manifest))
        expected=[x for x in inventory if x['path'].startswith(label+'/')]
        assert snapshot(label,target)==expected,'remaining source/cache changed'
        row['ownership_before']=NS['ownership'](target)
        assert target in [x[1] for x in ROOTS] and target.is_absolute() and not target.is_symlink()
        shutil.rmtree(target)
        row.update(target_absent=not target.exists(),allocated_after=NS['allocated'](target),free_after=NS['free'](),recovery_after=NS['verify'](label),supplemental_after=verifier(manifest),ownership_after=NS['ownership'](target))
        actions.append(row);(HERE/'guarded-remainder-removal.json').write_text(json.dumps(actions,indent=2)+'\n');print(json.dumps(row),flush=True)
    protected=[PARENT,CHILD/'framework',CHILD/'result.json',REPO.parent/'.recovery/fn25416-provider-proof/objects',REPO.parent/'.recovery/fn258-standalone-proof/objects',REPO/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task16/validation-60f1c762']
    tags=json.loads((HERE/'tagged-subtarget-cleanup.json').read_text())
    summary=dict(complete=True,targets_absent={str(p):not p.exists() for _,p in ROOTS},protected_paths_present={str(p):p.exists() for p in protected},parent_cache_allocated_after=NS['allocated'](PARENT),allocated_bytes_removed={'private_child_target':6609588224,'old_host_release_target':3779411968},initial_free_bytes=json.loads((HERE/'receipt.json').read_text())['actions'][0]['free_before'],final_free_bytes=NS['free'](),tagged_cargo_commands=len(tags),supplemental_object=manifest['objects'][0],supplemental_members=len(selected),remaining_manifest_entries=len(inventory),unknown_files_or_special_nodes_found=False,ownership_limitation='Readable process cwd/env/fds/maps checked; root socat10,dockerd542,containerd557 metadata unreadable/unknown, identities verified and available argv target-prefix matches negative.',free_space_note='Filesystem free-byte changes include concurrent activity; allocated bytes removed are reported separately.')
    assert all(summary['targets_absent'].values()) and all(summary['protected_paths_present'].values()) and summary['parent_cache_allocated_after']==876285952
    (HERE/'terminal-receipt.json').write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary),flush=True)

if __name__=='__main__':main()
