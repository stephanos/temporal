#!/usr/bin/env python3
"""Recover task-13 checkpoint bytes against its retained full-file hashes."""
import hashlib
import io
import json
from pathlib import Path
import re
import runpy
import subprocess
import tarfile

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
BASE = '0dd05b313acd0986312da7fd3159520e6a21f1bf'
HISTORY = Path('/tmp/gomad-task17.WFwzYI/baseline.diff')
HELPER = OUT.parent/'task-21/baseline-reconstruction/reconstruct.py'
EXPECTED_HISTORY = 'cff064cdefea694335315a0065811b3928cdbfc4de590395b2d0f18423886f68'
EXPECTED_HELPER = 'f19edebc9708e3d6716682e58d91b09aa9cc30dcd9d7d6d666f7cfbba6e2796c'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def main():
    historical = HISTORY.read_bytes()
    assert sha(historical) == EXPECTED_HISTORY
    assert sha(HELPER.read_bytes()) == EXPECTED_HELPER
    exact = runpy.run_path(str(HELPER), run_name='checkpoint_helper')
    wanted = ['tools/gomad3/runner/internal/execution/simulation_time.go',
              'tools/gomad3/toolchain/version/version.json']
    blocks = re.split(rb'(?m)(?=^diff --git )', historical)
    module_only = b''.join(block for block in blocks if any(
        block.startswith(('diff --git a/'+path+' b/'+path+'\n').encode())
        for path in wanted))
    sections = {row['path']: row for row in exact['sections'](module_only)}
    assert set(sections) == set(wanted)
    expected = json.loads((OUT/'evidence.json').read_text())['source_sha256']
    recovered = {}
    origins = {}
    time_path = 'tools/gomad3/runner/internal/execution/simulation_time.go'
    section = dict(sections[time_path])
    selected = []
    for line in section['lines']:
        if line.startswith(b'@@ ') and int(re.match(rb'@@ -(\d+)', line)[1]) >= 196:
            break
        selected.append(line)
    section['lines'] = selected
    recovered[time_path], hunks = exact['apply_exact'](git('show', BASE+':'+time_path), section)
    assert hunks == 3
    origins[time_path] = 'base plus first three exact historical codec-removal hunks; no lifecycle hunks'
    version_path = 'tools/gomad3/toolchain/version/version.json'
    section = dict(sections[version_path])
    start = next(i for i, line in enumerate(section['lines']) if line.startswith(b'@@ -174,6 '))
    selected = section['lines'][start:]
    selected[0] = selected[0].replace(b'+180,9', b'+174,9')
    section['lines'] = selected
    recovered[version_path], hunks = exact['apply_exact'](git('show', BASE+':'+version_path), section)
    assert hunks == 1
    origins[version_path] = 'base plus exact historical time-wire allowlist hunk; excludes later domain entries'
    sources = {}
    for path, identity in expected.items():
        data = recovered[path] if path in recovered else (ROOT/path).read_bytes()
        assert sha(data) == identity, (path, sha(data), identity)
        sources[path] = data
        origins.setdefault(path, 'current bytes match retained task-13 SHA-256')
    assert len(sources) == 22
    scratch = Path(subprocess.check_output(['mktemp', '-d', '/tmp/fn109-task13-checkpoint.XXXXXXXX']).decode().strip())
    archive = git('archive', '--format=tar', BASE, 'tools/gomad3')
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        for member in tar.getmembers():
            assert member.name == 'tools' or member.name.startswith('tools/gomad3')
            assert '..' not in Path(member.name).parts
            assert member.isdir() or member.isfile()
        tar.extractall(scratch, filter='data')
    for path, data in sources.items():
        destination = scratch/path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    final_files = sorted(path for path in (scratch/'tools/gomad3').rglob('*') if path.is_file())
    rows = [{'path': str(path.relative_to(scratch)), 'sha256': sha(path.read_bytes())} for path in final_files]
    for path, identity in expected.items():
        assert sha((scratch/path).read_bytes()) == identity
    report = dict(base_revision=BASE, historical_diff_sha256=EXPECTED_HISTORY,
                  helper_sha256=EXPECTED_HELPER, scratch=str(scratch),
                  retained_task13_hashes_matched=22, files=len(rows),
                  origins=origins, sources=rows, git_mutations=False,
                  shared_source_writes=False, native_qualification=False)
    (OUT/'checkpoint-reconstruction.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps({key: report[key] for key in ['scratch', 'files', 'retained_task13_hashes_matched', 'git_mutations', 'shared_source_writes']}))


if __name__ == '__main__':
    main()
