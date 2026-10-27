import datetime, hashlib, json, os, shutil, stat, subprocess, tarfile
from pathlib import Path

ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
TARGETS = [
 Path('/tmp/umpire-fn25417-standalone-dc7feca3/proof-Mxf6U4/target'),
 Path('/tmp/umpire-fn25417-standalone-dc7feca3/proof-SuPdUL/target'),
 Path('/tmp/umpire-fn25417-standalone-dc7feca3/proof-esl66Z/target'),
 Path('/tmp/umpire-fn25417-standalone-dc7feca3/proof-hlnVzv/target'),
 Path('/tmp/umpire-fn25417-standalone-8350ba8f/proof-zBSpnB/target'),
]
stamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
RECOVERY = ROOT.parent / '.recovery' / ('fn25417-disk-' + stamp)
RECOVERY.mkdir(mode=0o700, parents=True, exist_ok=False)
ARCHIVE = RECOVERY / 'retained-proof-output.tar.gz'
sha = lambda p: hashlib.file_digest(open(p, 'rb'), 'sha256').hexdigest()

def audit_live():
    found = []
    for proc in Path('/proc').iterdir():
        if not proc.name.isdigit(): continue
        try:
            cwd = os.readlink(proc / 'cwd')
            argv = (proc / 'cmdline').read_bytes().replace(b'\0', b' ').decode(errors='replace')
            env = (proc / 'environ').read_bytes().split(b'\0')
            selected = [v.decode(errors='replace') for v in env if v.startswith((b'CARGO_TARGET_DIR=', b'GOCACHE=', b'TMPDIR='))]
            if any(str(t) in s for t in TARGETS for s in [cwd, argv, *selected]):
                found.append({'pid':int(proc.name),'cwd':cwd,'argv':argv[:600],'cache_env':selected})
        except (OSError, ValueError): pass
    if found: raise RuntimeError({'active_target_users':found})


def keep(rel):
    parts = rel.parts
    if parts[0] in ('go-build', 'doc'): return False
    if any(p == 'incremental' and i and parts[i-1] in ('debug','release','guest') for i,p in enumerate(parts)): return False
    if rel.suffix in ('.rlib','.rmeta','.o','.a','.d','.bc','.ll','.rbc') and any(p in ('debug','release','guest') for p in parts): return False
    return True


def source_seals():
    seals = {}
    for target in TARGETS:
        framework = target.parent / 'framework'
        assert (framework / 'Cargo.toml').is_file(), str(framework)
        for p in framework.rglob('*'):
            if p.is_file() and not p.is_symlink(): seals[str(p)] = sha(p)
    return seals

audit_live()
for t in TARGETS:
    assert t.name == 'target' and not t.is_symlink() and t.is_dir(), str(t)
    assert (t / '.rustc_info.json').is_file() or not any(p.is_file() or p.is_symlink() for p in t.rglob('*')), str(t)
source_before = source_seals()
free_before = shutil.disk_usage(ROOT).free
entries = []
removed_bytes = 0
seen_inodes = set()
for i,t in enumerate(TARGETS):
    for folder,dirs,files in os.walk(t, followlinks=False):
        for name in files + [d for d in dirs if (Path(folder)/d).is_symlink()]:
            p = Path(folder)/name; info=p.lstat(); rel=p.relative_to(t)
            inode=(info.st_dev, info.st_ino)
            if inode not in seen_inodes:
                removed_bytes += info.st_blocks*512; seen_inodes.add(inode)
            if not keep(rel): continue
            row={'source':str(p),'archive':f'target-{i}/'+rel.as_posix(),'mode':stat.S_IMODE(info.st_mode),'bytes':info.st_size}
            if p.is_symlink(): row.update(kind='symlink',link=os.readlink(p))
            elif p.is_file(): row.update(kind='file',sha256=sha(p))
            else: raise RuntimeError(str(p))
            entries.append(row)
print(json.dumps({'phase':'preserve','targets':len(TARGETS),'retained_files':len(entries),'cache_allocated_bytes':removed_bytes,'recovery':str(RECOVERY)}),flush=True)
manifest={'targets':[str(t) for t in TARGETS],'classification':'Retain every non-compiler file, executable, wasm, manifest and observation; discard only Go build cache, generated rustdoc, incremental cache and known compiled dependency/object extensions in Cargo profile directories. All framework source trees remain in place.','entries':entries,'source_seals':source_before,'cache_allocated_bytes':removed_bytes,'free_before_bytes':free_before}
(RECOVERY/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
with tarfile.open(ARCHIVE,'w:gz',compresslevel=1) as tar:
    for row in entries: tar.add(row['source'],arcname=row['archive'],recursive=False)
print('Archive created; verifying every retained file',flush=True)
with tarfile.open(ARCHIVE,'r:gz') as tar:
    members={m.name:m for m in tar.getmembers()}
    assert len(members)==len(entries)
    for row in entries:
        m=members[row['archive']]
        assert stat.S_IMODE(m.mode)==row['mode']
        if row['kind']=='symlink': assert m.issym() and m.linkname==row['link']
        else:
            assert m.isfile() or m.islnk()
            stream=tar.extractfile(m)
            assert stream is not None and hashlib.file_digest(stream,'sha256').hexdigest()==row['sha256']
archive_sha=sha(ARCHIVE)
print(json.dumps({'phase':'verified','archive_bytes':ARCHIVE.stat().st_size,'archive_sha256':archive_sha}),flush=True)
assert source_before==source_seals()
audit_live()
results=[]
env=os.environ.copy();env.pop('BASH_ENV',None)
for t in TARGETS:
    audit_live()
    result=subprocess.run(['mise','exec','--','cargo','clean','--target-dir',str(t)],cwd=ROOT/'model/umpire',env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
    row={'target':str(t),'cargo_exit':result.returncode,'cargo_output':result.stdout}
    if t.exists():
        # Ownership, all non-compiler bytes and source are already sealed above.
        # Never add a cache tag merely to bypass Cargo's missing-tag refusal.
        audit_live()
        shutil.rmtree(t)
        row['exact_owned_cache_removal']=True
    assert not t.exists()
    results.append(row)
    print(json.dumps({'phase':'removed','target':str(t),'cargo_exit':result.returncode}),flush=True)
assert source_before==source_seals()
assert (ROOT/'.git').exists()
assert (ROOT/'.worktrees/fn25417-work-counts').exists()
assert Path('/home/agent/.cache/codex-build/cargo-target').exists()
assert Path('/home/agent/.cache/codex-build/fn25417-work-counts').exists()
summary={'recovery':str(RECOVERY),'archive':str(ARCHIVE),'archive_sha256':archive_sha,'manifest_sha256':sha(RECOVERY/'manifest.json'),'retained_files_verified':len(entries),'cache_allocated_bytes_removed':removed_bytes,'archive_bytes':ARCHIVE.stat().st_size,'net_reclaimed_estimate_bytes':removed_bytes-ARCHIVE.stat().st_size,'free_before_bytes':free_before,'free_after_bytes':shutil.disk_usage(ROOT).free,'framework_sources_unchanged':True,'protected_shared_and_next_task_caches_present':True,'results':results}
(RECOVERY/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
(ROOT/'.flow/tmp/disk-cleanup-20261027/result.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps(summary),flush=True)
