"""Bounded byte-exact source module derivation outside the frozen candidate."""
import hashlib
import json
import os
from pathlib import Path
import re
import stat

import workloads

MAX_FILES = 50000
MAX_BYTES = 512 << 20
EXCLUDED_PREFIXES = tuple(Path(name) for name in (
    '.flow', '.git', '.tmp', '.turbo', '.worktrees', '.bin', '.stamp', '.gomad',
    '.toolchain', 'dist', 'evidence', 'output', 'outputs', 'build', 'target',
    'tools/gomad/rust/target', 'tools/gomad/scripts/__pycache__'))


def digest(body):
    return hashlib.sha256(body).hexdigest()


def require(condition, message):
    if not condition:
        raise ValueError('source custody: '+message)


def canonical_path(path):
    path = Path(path)
    require(path.is_absolute() and path.resolve(strict=True) == path, 'noncanonical or linked path')
    return path


def regular(path, mode=None):
    path = canonical_path(path)
    info = path.lstat()
    require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1, 'nonregular or linked source')
    require(info.st_size <= MAX_BYTES, 'input byte bound exceeded')
    if mode is not None:
        require(info.st_uid == os.geteuid(), 'derived file owner changed')
        require(stat.S_IMODE(info.st_mode) == mode, 'derived file mode changed')
    def identity(value):
        return (value.st_dev, value.st_ino, value.st_mode, value.st_nlink, value.st_uid,
                value.st_gid, value.st_size, value.st_mtime_ns, value.st_ctime_ns)

    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, 'rb') as stream:
        before = os.fstat(stream.fileno())
        require(identity(before) == identity(info), 'source changed before descriptor read')
        body = stream.read(MAX_BYTES+1)
        after = os.fstat(stream.fileno())
    require(len(body) <= MAX_BYTES, 'input byte bound exceeded')
    require(identity(before) == identity(after) == identity(path.lstat())
            and len(body) == before.st_size, 'source changed during descriptor read')
    canonical_path(path)
    return body


def recipe(root, repo, config_files, workload, source_manifest):
    root, repo = canonical_path(root), canonical_path(repo)
    root_info = root.lstat()
    require(stat.S_ISDIR(root_info.st_mode) and stat.S_IMODE(root_info.st_mode) == 0o700,
            'producer root must be a private directory')
    require(root_info.st_uid == os.geteuid(), 'producer root owner changed')
    require(root != repo and not root.is_relative_to(repo) and not repo.is_relative_to(root), 'source and producer roots overlap')
    manifest = canonical_path(source_manifest)
    raw = regular(manifest)
    require(config_files.get(str(manifest)) == digest(raw), 'unsealed source manifest')
    entries = {}
    for line in raw.decode('utf-8').splitlines():
        match = re.fullmatch(r'([0-9a-f]{64})  (/.+)', line)
        require(match is not None, 'invalid frozen source manifest')
        pin, name = match.groups()
        original = Path(name)
        require(original.is_relative_to(repo) and str(original) == name and '..' not in original.parts,
                'foreign or unclean frozen source')
        relative = original.relative_to(repo)
        require(name not in entries, 'duplicate frozen source')
        entries[name] = (relative, pin)
    require(entries, 'empty frozen source manifest')
    copied = {}
    original_pins = {}
    copied_bytes = 0
    for name, (relative, pin) in entries.items():
        if any(relative.is_relative_to(prefix) for prefix in EXCLUDED_PREFIXES):
            continue
        require(len(copied) < MAX_FILES, 'copied file bound exceeded')
        copied_bytes += Path(name).lstat().st_size
        require(copied_bytes <= MAX_BYTES, 'copied byte bound exceeded')
        body = regular(Path(name))
        require(config_files.get(name) == pin == digest(body), 'original source changed or unsealed')
        copied[str(relative)] = (name, body)
    require('go.mod' in copied and 'go.sum' in copied, 'original module files missing')
    require(not re.search(rb'(?m)^\s*replace\s', copied['go.mod'][1]), 'ambient module replacement refused')
    base = workloads.WASM/'testdata'
    names = {'replication': ['temporal_storage/recovery_test.go', 'temporal_broker/replication_test.go'],
             'recovery': ['temporal_storage/recovery_test.go'],
             'retry': sorted(relative[len(str(base/'network_streams'))+1:] for relative in copied
                             if Path(relative).parent == base/'network_streams' and relative.endswith('.go'))}.get(workload)
    require(names is not None and names, 'unknown or empty wrapper workload')
    if workload == 'retry':
        names = ['network_streams/'+name for name in names]
    originals = [str(base/name) for name in names]
    require(all(name in copied for name in originals), 'wrapper missing from frozen module')
    key_names = [str(base/'network_streams/temporal_retry_test.go')] if workload == 'retry' else originals
    require(all(name in copied for name in key_names), 'wrapper key missing')
    key = digest(b''.join(copied[name][1] for name in key_names))[:16]
    relative_stage = '.tmp/fn232-temporal-'+workload+'-'+key
    for name in originals:
        original, body = copied[name]
        relative = relative_stage+'/'+Path(name).name
        require(relative not in copied, 'wrapper collision')
        copied[relative] = (original, body)
        original_pins[original] = digest(body)
    require(len(copied) <= MAX_FILES, 'copied file bound exceeded')
    size = sum(len(body) for _, body in copied.values())
    require(size <= MAX_BYTES, 'copied byte bound exceeded')
    module = root/'source-module'
    table = {str(module/relative): {'original': original, 'sha256': digest(body), 'bytes': len(body)}
             for relative, (original, body) in sorted(copied.items())}
    record = dict(schema='gomad.source-custody/v1', root=str(root), repo=str(repo),
                  source_manifest=str(manifest), source_manifest_sha256=digest(raw), workload=workload,
                  source='./'+relative_stage, working_dir=str(module), files=table,
                  original_wrappers=original_pins, copied_files=len(copied), copied_bytes=size,
                  max_files=MAX_FILES, max_bytes=MAX_BYTES)
    return record, copied


def inspect(root, record):
    module = root/'source-module'
    canonical_path(module)
    expected_files = set(record['files'])
    expected_dirs = {str(module)}
    for name in expected_files:
        expected_dirs.update(str(path) for path in Path(name).parents if path.is_relative_to(module))
    actual_files, actual_dirs = set(), {str(module)}
    require(module.stat().st_uid == os.geteuid(), 'derived directory owner changed')
    require(stat.S_IMODE(module.lstat().st_mode) == 0o700, 'derived directory mode changed')
    pending = [module]
    while pending:
        directory = pending.pop()
        with os.scandir(directory) as children:
            for child in children:
                path = Path(child.path)
                info = path.lstat()
                require(not stat.S_ISLNK(info.st_mode), 'derived symlink')
                if stat.S_ISDIR(info.st_mode):
                    require(str(path) in expected_dirs, 'unexpected derived directory')
                    require(info.st_uid == os.geteuid(), 'derived directory owner changed')
                    require(stat.S_IMODE(info.st_mode) == 0o700, 'derived directory mode changed')
                    actual_dirs.add(str(path))
                    pending.append(path)
                else:
                    require(str(path) in expected_files, 'unexpected derived source')
                    body = regular(path, 0o600)
                    require(digest(body) == record['files'][str(path)]['sha256'], 'derived source changed')
                    actual_files.add(str(path))
    require(actual_files == expected_files and actual_dirs == expected_dirs, 'derived inventory changed')


def verify(root, repo, config_files, workload, target_source, target_working_dir, *, frozen_source_manifest=None):
    """Reconstruct the complete derivation from sealed originals without mutation."""
    root = canonical_path(root)
    raw = regular(root/'source-custody.json', 0o600)
    observed = json.loads(raw)
    require(isinstance(observed, dict) and isinstance(observed.get('source_manifest'), str), 'invalid custody receipt')
    if frozen_source_manifest is not None:
        require(str(canonical_path(frozen_source_manifest)) == observed['source_manifest'], 'foreign source manifest')
    expected, _ = recipe(root, repo, config_files, workload, observed['source_manifest'])
    require(observed == expected and raw == workloads.canonical(expected)+b'\n', 'custody recipe changed')
    require(target_source == expected['source'] and str(target_working_dir) == expected['working_dir'], 'target custody differs')
    inspect(root, expected)
    return {name: value['sha256'] for name, value in expected['files'].items()}


def stage(root, repo, config_files, workload, source_manifest):
    root = canonical_path(root)
    record, copied = recipe(root, repo, config_files, workload, source_manifest)
    receipt, module = root/'source-custody.json', root/'source-module'
    if receipt.exists() or receipt.is_symlink():
        verify(root, repo, config_files, workload, record['source'], record['working_dir'],
               frozen_source_manifest=source_manifest)
    else:
        require(not module.exists() and not module.is_symlink(), 'source module collision')
        module.mkdir(mode=0o700)
        for relative, (_, body) in sorted(copied.items()):
            target = module/relative
            missing = []
            parent = target.parent
            while not parent.exists():
                missing.append(parent)
                parent = parent.parent
            for parent in reversed(missing):
                parent.mkdir(mode=0o700)
            with target.open('xb') as stream:
                stream.write(body)
            target.chmod(0o600)
        with receipt.open('xb') as stream:
            stream.write(workloads.canonical(record)+b'\n')
        receipt.chmod(0o600)
        verify(root, repo, config_files, workload, record['source'], record['working_dir'],
               frozen_source_manifest=source_manifest)
    return record['source'], record['working_dir'], record['original_wrappers']
