#!/usr/bin/env python3
"""Capture exact eight-ref/dirty-index custody; writes only custody evidence/recovery CAS."""
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import tarfile

HERE = Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
UMBRELLA = TEMPORAL.parent
STORE = UMBRELLA/'.recovery/fn258-task5-custody'
OBJECTS = STORE/'objects'
OVERLAY = HERE.parent/'disposition-overlay.json'
ENV = os.environ.copy()
for key in list(ENV):
    if key.startswith('GIT_'):
        ENV.pop(key)
ENV.update(GIT_OPTIONAL_LOCKS='0', BASH_ENV='')


def git(repo, *args, data=None, index=None, worktree=None):
    env = ENV.copy()
    if index:
        env['GIT_INDEX_FILE'] = str(index)
    if worktree:
        env['GIT_WORK_TREE'] = str(worktree)
    result = subprocess.run(['git','--no-optional-locks','--no-replace-objects','-c','core.fsmonitor=false','-c','core.hooksPath=/dev/null','-C',str(repo),*args],
                            input=data,env=env,capture_output=True)
    if result.returncode:
        raise RuntimeError(str(args)+': '+result.stderr.decode(errors='replace'))
    return result.stdout


def digest(data):
    return hashlib.sha256(data).hexdigest()


def cas(data):
    oid = digest(data)
    path = OBJECTS/oid
    if not path.exists():
        with path.open('xb') as stream:
            stream.write(data)
    assert digest(path.read_bytes()) == oid
    return {'sha256':oid,'size':len(data),'object':str(path)}


def payload(record):
    data = Path(record['object']).read_bytes()
    assert digest(data) == record['sha256'] and len(data) == record['size']
    return data


def names(raw):
    return [os.fsdecode(p) for p in raw.split(b'\0') if p]


def file_record(root, relative, store=False):
    assert relative and not Path(relative).is_absolute() and '..' not in Path(relative).parts
    path = root/relative
    try:
        info = path.lstat()
    except FileNotFoundError:
        return {'path':relative,'kind':'missing'}
    if stat.S_ISLNK(info.st_mode):
        data = os.fsencode(os.readlink(path));kind='symlink';mode='120000'
    elif stat.S_ISREG(info.st_mode):
        data = path.read_bytes();kind='file';mode='100755' if info.st_mode & 0o111 else '100644'
    else:
        return {'path':relative,'kind':'directory-or-gitlink','posix_mode':stat.S_IMODE(info.st_mode)}
    result = {'path':relative,'kind':kind,'mode':mode,'posix_mode':stat.S_IMODE(info.st_mode),'size':len(data),'sha256':digest(data)}
    if store:
        result['content'] = cas(data)
    return result


def status(root):
    return git(root,'status','--porcelain=v1','-z','--no-renames','--untracked-files=all')


def index_path(root):
    return Path(os.fsdecode(git(root,'rev-parse','--path-format=absolute','--git-path','index').strip()))


def fingerprint(root):
    raw_status = status(root)
    ignored = git(root,'ls-files','--others','--ignored','--exclude-standard','-z')
    assert not ignored, 'Unexpected ignored files require ownership review before custody admission'
    paths = sorted(set(names(git(root,'ls-files','-z'))+names(git(root,'ls-files','--others','--exclude-standard','-z'))))
    hasher = hashlib.sha256()
    for path in paths:
        hasher.update(json.dumps(file_record(root,path),sort_keys=True).encode()+b'\n')
    idx = index_path(root)
    return {'head':git(root,'rev-parse','HEAD').decode().strip(),'status_sha256':digest(raw_status),
            'index_raw_sha256':digest(idx.read_bytes()),'index_stages_sha256':digest(git(root,'ls-files','--stage','-z')),
            'tracked_and_untracked_path_count':len(paths),'full_worktree_bytes_modes_symlink_sha256':hasher.hexdigest(),
            'current_ignored_inventory_sha256':digest(ignored),'current_ignored_count':0}


def snapshot(root, candidate):
    raw_status = status(root)
    changed = [os.fsdecode(row[3:]) for row in raw_status.split(b'\0') if row]
    idx = index_path(root)
    shared = list(idx.parent.glob('sharedindex.*'))
    assert not shared, 'Split index dependencies require explicit custody support'
    stages = git(root,'ls-files','--stage','-z')
    head_entries = {}
    for row in git(root,'ls-tree','-rz','HEAD').split(b'\0'):
        if row:
            meta,path=row.split(b'\t',1);mode,kind,oid=meta.split();head_entries[path]=(mode,oid)
    extra = {}
    unmerged = []
    for row in stages.split(b'\0'):
        if not row:
            continue
        meta,path=row.split(b'\t',1);mode,oid,stage=meta.split()
        if stage != b'0':
            unmerged.append({'path':os.fsdecode(path),'mode':mode.decode(),'blob':oid.decode(),'stage':int(stage)})
        if mode != b'160000' and (stage != b'0' or head_entries.get(path) != (mode,oid)):
            extra[oid.decode()] = cas(git(root,'cat-file','blob',oid.decode()))
    return {'candidate':candidate,'root':str(root),'head':git(root,'rev-parse','HEAD').decode().strip(),
            'status_z':cas(raw_status),'raw_index':cas(idx.read_bytes()),'index_stages_z':cas(stages),
            'unmerged_stages':unmerged,'supplemental_index_blobs':extra,
            'staged_binary_patch':cas(git(root,'diff','--cached','--binary','--full-index','--no-ext-diff','--no-textconv','--no-renames','HEAD','--')),
            'unstaged_binary_patch':cas(git(root,'diff','--binary','--full-index','--no-ext-diff','--no-textconv','--no-renames','--')),
            'ignored_inventory_z':cas(git(root,'ls-files','--others','--ignored','--exclude-standard','-z')),
            'raw_working_files':[file_record(root,path,True) for path in sorted(changed)],
            'historical_ignored_qualification':'Current ignored inventory only; historical absent checkout/ignored bytes remain unknown.'}


def tar_snapshot(snap):
    output=io.BytesIO()
    with tarfile.open(fileobj=output,mode='w:gz',format=tarfile.PAX_FORMAT) as archive:
        content=json.dumps(snap,indent=2).encode()+b'\n';entry=tarfile.TarInfo('custody-manifest.json');entry.size=len(content);archive.addfile(entry,io.BytesIO(content))
        for record in snap['raw_working_files']:
            if record['kind']=='missing':
                continue
            assert record['kind'] in {'file','symlink'}
            data=payload(record['content']);entry=tarfile.TarInfo('worktree/'+record['path']);entry.mode=record['posix_mode']
            if record['kind']=='symlink':
                entry.type=tarfile.SYMTYPE;entry.linkname=os.fsdecode(data);archive.addfile(entry)
            else:
                entry.size=len(data);archive.addfile(entry,io.BytesIO(data))
        records={'index/raw-index':snap['raw_index'],'index/stages.z':snap['index_stages_z'],
                 'inventory/status.z':snap['status_z'],'inventory/current-ignored.z':snap['ignored_inventory_z'],
                 'patches/staged.binary.patch':snap['staged_binary_patch'],'patches/unstaged.binary.patch':snap['unstaged_binary_patch']}
        records.update({'index/extra-blobs/'+oid:record for oid,record in snap['supplemental_index_blobs'].items()})
        for name,record in records.items():
            data=payload(record);entry=tarfile.TarInfo(name);entry.size=len(data);archive.addfile(entry,io.BytesIO(data))
    return cas(output.getvalue())


def restore_snapshot(snap, archive_record, bare):
    git(bare,'update-ref','--no-deref','HEAD',snap['head'])  # Isolated recovery HEAD only.
    restore=STORE/('restore-'+snap['candidate'])
    version=2
    while restore.exists():
        restore=STORE/('restore-'+snap['candidate']+'-v'+str(version));version+=1
    restore.mkdir()
    # Extract only our normalized members, never follow archive links.
    with tarfile.open(fileobj=io.BytesIO(payload(archive_record)),mode='r:gz') as archive:
        for member in archive.getmembers():
            relative=Path(member.name)
            assert not relative.is_absolute() and '..' not in relative.parts
            destination=restore/relative;destination.parent.mkdir(parents=True,exist_ok=True)
            if member.issym():
                destination.symlink_to(member.linkname)
            else:
                assert member.isfile();destination.write_bytes(archive.extractfile(member).read());destination.chmod(member.mode)
    tree=restore/'worktree';tree.mkdir(exist_ok=True)
    for record in snap['raw_working_files']:
        actual=file_record(tree,record['path'])
        assert actual == {key:value for key,value in record.items() if key != 'content'}, (record['path'],actual)
    for oid,record in snap['supplemental_index_blobs'].items():
        assert git(bare,'hash-object','-w','--stdin',data=payload(record)).decode().strip()==oid
    idx=restore/'index/raw-index'
    assert idx.read_bytes()==payload(snap['raw_index'])
    verification_index=restore/'index/verification-index'
    verification_index.write_bytes(idx.read_bytes())
    assert git(bare,'ls-files','--stage','-z',index=verification_index,worktree=tree)==payload(snap['index_stages_z'])
    changed=[record['path'] for record in snap['raw_working_files']]
    assert git(bare,'status','--porcelain=v1','-z','--no-renames','--untracked-files=all','--',*changed,index=verification_index,worktree=tree)==payload(snap['status_z'])
    assert git(bare,'diff','--cached','--binary','--full-index','--no-ext-diff','--no-textconv','--no-renames',snap['head'],'--',index=verification_index,worktree=tree)==payload(snap['staged_binary_patch'])
    assert git(bare,'diff','--binary','--full-index','--no-ext-diff','--no-textconv','--no-renames','--',*changed,index=verification_index,worktree=tree)==payload(snap['unstaged_binary_patch'])
    assert git(bare,'ls-files','--stage','-z',index=verification_index,worktree=tree)==payload(snap['index_stages_z'])
    assert idx.read_bytes()==payload(snap['raw_index'])
    return {'candidate':snap['candidate'],'result':'pass','raw_dirty_files_verified':len(changed),
            'index_records_restored_exactly':len([row for row in payload(snap['index_stages_z']).split(b'\0') if row]),
            'unmerged_stage_records_verified':len(snap['unmerged_stages']),
            'staged_and_unstaged_patch_bytes_equal':True,'verification_index_stat_cache_may_refresh':True,'restored_raw_index_byte_identical':True,
            'restoration':str(restore),'qualification':'Only dirty/untracked paths materialized; all other committed bytes recover from the independently recovered source bundle.'}


def main():
    assert shutil.disk_usage(UMBRELLA).free > 30*1024**3, 'Below30GiB host: root coordination required'
    assert not (HERE/'custody-manifest.json').exists(), 'Preserve existing custody; explicit new version required'
    OBJECTS.mkdir(parents=True,exist_ok=True)
    ledger=json.loads(OVERLAY.read_bytes())
    rows=[row for row in ledger['entries'] if row['unresolved_product_owner_freeze'] or (row['planned_receiver'] and row['planned_receiver']['planned_ref'].endswith(('039','075')))]
    assert len(rows)==8
    refs={row['ref']:row['source_recovery']['required_original_or_observed_object'] for row in rows}
    for ref,oid in refs.items():
        assert git(TEMPORAL,'rev-parse',ref).decode().strip()==oid
    roots={row['planned_receiver']['planned_ref'].rsplit('-',1)[-1]:Path(row['worktrees_observed_now'][0]['worktree']) for row in rows if row['planned_receiver']}
    before={name:fingerprint(root)for name,root in roots.items()}
    shallow=Path(os.fsdecode(git(TEMPORAL,'rev-parse','--path-format=absolute','--git-path','shallow').strip()))
    shallow_data=shallow.read_bytes() if shallow.exists() else b''
    shallow_record=cas(shallow_data)
    print('Source fingerprints captured; creating exact8-ref available-history bundle',flush=True)
    bundle=None
    for item in OBJECTS.iterdir():
        if not item.is_file():
            continue
        with item.open('rb') as stream:
            header=stream.read(32)
        if header.startswith((b'# v2 git bundle',b'# v3 git bundle')):
            heads=dict((ref,oid) for oid,ref in (line.split()for line in git(TEMPORAL,'bundle','list-heads',str(item)).decode().splitlines()))
            if heads==refs:
                bundle={'sha256':item.name,'size':item.stat().st_size,'object':str(item)}
                assert digest(item.read_bytes())==item.name
                break
    if bundle is None:
        temporary=STORE/'source-exact-eight.bundle'
        assert not temporary.exists()
        git(TEMPORAL,'bundle','create',str(temporary),*refs)
        bundle=cas(temporary.read_bytes());temporary.unlink()  # Own duplicate staging output only.
    bare=STORE/'recovered-source.git'
    if not bare.exists():
        subprocess.run(['git','init','--bare',str(bare)],env=ENV,check=True,capture_output=True)
        if shallow_data:
            (bare/'shallow').write_bytes(shallow_data)
        git(bare,'bundle','unbundle',bundle['object'])
        for ref,oid in refs.items():
            git(bare,'update-ref',ref,oid)
    else:
        assert (bare/'shallow').read_bytes()==shallow_data
        for ref,oid in refs.items():
            assert git(bare,'rev-parse',ref).decode().strip()==oid
    source_graph=git(TEMPORAL,'rev-list',*refs)
    recovered_graph=git(bare,'rev-list',*refs)
    assert source_graph==recovered_graph
    fsck=git(bare,'fsck','--full','--strict')
    assert not (bare/'objects/info/alternates').exists()
    print('Independent available-history bundle recovery verified; capturing dirty/index archives',flush=True)
    snapshots=[snapshot(root,name)for name,root in roots.items()]
    archives={snap['candidate']:tar_snapshot(snap)for snap in snapshots}
    restorations=[restore_snapshot(snap,archives[snap['candidate']],bare)for snap in snapshots]
    after={name:fingerprint(root)for name,root in roots.items()}
    assert before==after, 'Original worktree/index/dirty bytes changed during custody'
    assert (shallow.read_bytes()if shallow.exists()else b'')==shallow_data
    for ref,oid in refs.items():
        assert git(TEMPORAL,'rev-parse',ref).decode().strip()==oid
    result={'result':'pass','overlay_sha256':digest(OVERLAY.read_bytes()),'capture_script_sha256':digest(Path(__file__).read_bytes()),
            'cas_store':str(STORE),'source_refs':refs,'source_bundle':bundle,'source_shallow_boundaries':shallow_record,
            'available_history_commits':len(source_graph.splitlines()),'available_history_graph_sha256':digest(source_graph),
            'source_history_scope':'Exactly supplied available shallow history. Earlier unavailable parents are not recovered; restore captured shallow bytes before unbundling.',
            'bundle_recovery':{'bare_repository':str(bare),'objects_independent_of_original':True,'graph_equal':True,'strict_fsck_stdout':fsck.decode(errors='replace')},
            'snapshots':snapshots,'archives':archives,'independent_dirty_restorations':restorations,
            'originals_before':before,'originals_after':after,'originals_unchanged':True,'eight_source_refs_unchanged':True,
            'historical_ignored_state':'Unresolved for six absent historical checkouts; current ignored inventories of039/075 are empty, not retrospective proof.',
            'initial_refused_attempts':['First status comparison refused against an unborn isolated bare HEAD; now explicitly selects captured source HEAD.', 'Second check detected Git stat-cache refresh in the isolated restored index after successful stage/status/patch comparisons. Verification now uses a disposable index copy and leaves archived raw index byte-identical. Original indexes were unchanged; source bundle/pack reused.'],
            'receiving_refs_created':0,'source_refs_or_index_mutated':False,'product_or_tracker_mutations':False}
    (HERE/'custody-manifest.json').write_text(json.dumps(result,indent=2)+'\n')
    (STORE/'manifest.json').write_text(json.dumps({'canonical_manifest':str(HERE/'custody-manifest.json'),'manifest_sha256':digest((HERE/'custody-manifest.json').read_bytes()),'source_bundle':bundle,'archives':archives},indent=2)+'\n')
    print(json.dumps({'result':'pass','source_refs':8,'available_source_commits':len(source_graph.splitlines()),'bundle_bytes':bundle['size'],'restorations':restorations,'originals_unchanged':True}),flush=True)


if __name__=='__main__':
    main()
