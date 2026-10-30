from pathlib import Path
import copy, hashlib, json, os, stat, tarfile, tempfile

HERE = Path(__file__).resolve().parent
REPO = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
PROOF = Path('/tmp/umpire-fn25416-final-60f1c762a618-4090033/proof-5XlzyE')
RUN = REPO/'.flow/tmp/fn25416-resume-20261028-watchdog-01/proof-output'
STORE = REPO.parent/'.recovery/fn25416-provider-proof/objects'
LIB = Path('/home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/lib/rustlib/aarch64-unknown-linux-gnu/lib/libstd-f9f1a1f02cfcb825.so')

def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024*1024), b''): h.update(chunk)
    return h.hexdigest()

def regular(path):
    for ancestor in [path, *path.parents]:
        assert not ancestor.is_symlink(), str(ancestor)
    assert stat.S_ISREG(path.lstat().st_mode), str(path)

def idle():
    needles = [os.fsencode(PROOF), os.fsencode(REPO/'.flow/tmp/fn25416-provider-validation/target')]
    owners = []
    for process in Path('/proc').iterdir():
        if not process.name.isdigit() or int(process.name) == os.getpid(): continue
        try:
            cwd = os.fsencode(os.readlink(process/'cwd'))
            env = (process/'environ').read_bytes()
            if any(n in cwd or n in env for n in needles): owners.append(int(process.name))
        except (FileNotFoundError, ProcessLookupError, PermissionError): pass
    assert not owners, owners
    return {'matching_owned_cache_processes': owners}

def allocated(root):
    seen = set(); total = 0
    for parent, directories, files in os.walk(root):
        for name in directories+files:
            info = os.lstat(os.path.join(parent, name)); key = info.st_dev, info.st_ino
            if key not in seen: seen.add(key); total += info.st_blocks*512
    return total

def main():
    assert not (HERE/'manifest.json').exists()
    ownership = idle()
    inventory = json.loads((RUN/'child-nextest-all.stdout.log').read_text())
    result = json.loads((PROOF/'result.json').read_text())
    assert result['complete'] and result['source_unchanged'] and result['child_exit'] == 0
    selected = {}
    def add(path, label, role, expected=None):
        regular(path)
        assert not Path(label).is_absolute() and '..' not in Path(label).parts
        checksum = digest(path)
        if expected is not None: assert checksum == expected, label
        before = path.stat()
        item = dict(path=label, source=str(path), role=role, size=before.st_size,
                    mode=oct(stat.S_IMODE(before.st_mode)), sha256=checksum)
        assert label not in selected or selected[label] == item
        selected[label] = item
    binaries = [Path(s['binary-path']) for s in inventory['rust-suites'].values()]
    assert len(binaries) == len(set(binaries)) == 69
    for path in binaries:
        assert path.is_relative_to(PROOF/'target') and path.open('rb').read(4) == b'\x7fELF'
        add(path, str(path.relative_to(PROOF)), 'actual nextest producer ELF')
    for section in ['binaries', 'artifacts']:
        for name, row in result[section].items():
            path = Path(row['path']); assert path.is_relative_to(PROOF)
            if path.is_dir():
                expected = {os.fsdecode(bytes.fromhex(x['path_hex'])):x for x in row['files']}
                actual = [f for f in path.rglob('*') if not f.is_dir()]
                assert {str(f.relative_to(path)) for f in actual} == set(expected)
                for f in actual:
                    add(f, str(f.relative_to(PROOF)), section+' '+name, expected[str(f.relative_to(path))]['sha256'])
            else: add(path, str(path.relative_to(PROOF)), section+' '+name, row['sha256'])
    add(LIB, 'external-runtime/rust-1.99.1/aarch64-unknown-linux-gnu/'+LIB.name, 'pinned Rust dynamic runtime')
    for name in ['result.json','inputs.json','tools.json','controls.json','setup.jsonl','stages.jsonl']:
        add(PROOF/name, 'metadata/proof/'+name, 'canonical child proof metadata')
    for name in ['child-nextest-all.stdout.log','child-nextest-all.json','native-binaries.json','native-bootstrap.json','scheduled-standalone.json','validation.json','harness.json','owned-cleanup.json','source-before.json','source-after.json']:
        add(RUN/name, 'metadata/harness/'+name, 'accepted harness metadata')
    for path in (PROOF/'target/gate-receipts').rglob('*'):
        if path.is_file(): add(path, 'metadata/gate-receipts/'+str(path.relative_to(PROOF/'target/gate-receipts')), 'child stage producer receipts')
    add(PROOF/'target/.rustc_info.json', 'metadata/target-rustc-info.json', 'actual compiler identity')
    for path in [HERE/'prepare.py', HERE/'verify.py']:
        add(path, 'metadata/retention/'+path.name, 'retention implementation')
    members = [selected[k] for k in sorted(selected)]
    STORE.mkdir(parents=True, exist_ok=True)
    temporary = STORE.parent/'provider-proof-payloads.preparing.tar.gz'
    assert not temporary.exists()
    with tarfile.open(temporary, 'w:gz', compresslevel=3) as archive:
        for row in members:
            source = Path(row['source']); regular(source)
            archive.add(source, arcname=row['path'], recursive=False)
    checksum = digest(temporary); destination = STORE/checksum
    if destination.exists(): assert digest(destination) == checksum
    else: os.link(temporary, destination)
    # The preparation archive is deliberately retained; no cleanup is authorized.
    manifest = dict(schema_version=1, kind='accepted-proof-payload-retention-only',
        source_head='60f1c762a618d32478b58ae4e79c151f4355aeb3', proof_root=str(PROOF),
        inventory_count=69, nextest_test_identities=inventory['test-count'], ownership=ownership,
        original_allocated_bytes={'private_child_target':allocated(PROOF/'target'),
            'source_parent_target':allocated(REPO/'.flow/tmp/fn25416-provider-validation/target')},
        objects=[dict(recovery_object=str(destination), sha256=checksum, size=destination.stat().st_size)],
        archive_members=members, authoritative_versioned_evidence=str(REPO/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task16/validation-60f1c762'),
        cleanup_performed=False, task_complete=False)
    (HERE/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    print(json.dumps({'manifest':str(HERE/'manifest.json'),'members':len(members),'archive_bytes':destination.stat().st_size}),flush=True)

if __name__ == '__main__': main()
