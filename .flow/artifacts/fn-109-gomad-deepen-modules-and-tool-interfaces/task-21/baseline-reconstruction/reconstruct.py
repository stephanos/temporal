#!/usr/bin/env python3
"""Reconstruct retained fn-108 source without loading Go packages or mutating Git."""
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import time
from datetime import datetime, timezone

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
BASE = '6782b55f49a0317b230e827ea2a63a37d116d502'
A108 = ROOT / '.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing'
A109 = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces'
commands = []
inputs = {}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def blob(data):
    return hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()


def read(path):
    data = path.read_bytes()
    inputs[str(path.relative_to(ROOT))] = sha(data)
    return data


def run(argv):
    start = datetime.now(timezone.utc).isoformat()
    clock = time.monotonic()
    result = subprocess.run(argv, cwd=ROOT, capture_output=True)
    commands.append({'argv': argv, 'cwd': str(ROOT), 'start_utc': start,
                     'elapsed_seconds': time.monotonic() - clock,
                     'exit_code': result.returncode, 'stdout_sha256': sha(result.stdout),
                     'stderr': result.stderr.decode()})
    assert result.returncode == 0, (argv, result.stderr)
    return result.stdout


def sections(data):
    lines = data.splitlines(keepends=True)
    starts = [i for i, line in enumerate(lines) if line.startswith(b'--- ')]
    result = []
    for n, start in enumerate(starts):
        end = starts[n + 1] if n + 1 < len(starts) else len(lines)
        while end > start and lines[end - 1].startswith((b'diff --git ', b'index ', b'new file mode ')):
            end -= 1
        assert lines[start + 1].startswith(b'+++ ')
        path = lines[start + 1][4:].split()[0].decode()
        assert path.startswith('b/tools/gomad3/') and '..' not in path.split('/')
        index = None
        if start >= 1 and lines[start - 1].startswith(b'index '):
            index = lines[start - 1].split()[1].decode().split('..')
        result.append({'path': path[2:], 'old': lines[start][4:].split()[0].decode(),
                       'lines': lines[start + 2:end], 'index': index,
                       'block_start_line': start + 1})
    return result


def apply_exact(old, section):
    source = old.splitlines(keepends=True)
    target = []
    cursor = 0
    hunks = 0
    lines = section['lines']
    i = 0
    while i < len(lines):
        header = re.fullmatch(rb'@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@[^\n]*\n', lines[i])
        assert header, (section['path'], lines[i])
        old_start, old_count, new_start, new_count = [int(v) if v is not None else 1 for v in header.groups()]
        old_pos = old_start - 1 if old_count else old_start
        new_pos = new_start - 1 if new_count else new_start
        assert old_pos >= cursor
        target.extend(source[cursor:old_pos])
        assert len(target) == new_pos, (section['path'], old_pos, new_pos)
        i += 1
        before, after = [], []
        while i < len(lines) and not lines[i].startswith(b'@@ '):
            line = lines[i]
            assert line[:1] in (b' ', b'+', b'-'), (section['path'], line)
            payload = line[1:]
            if i + 1 < len(lines) and lines[i + 1].startswith(b'\\ No newline at end of file'):
                payload = payload.removesuffix(b'\n')
                i += 1
            if line[:1] in (b' ', b'-'):
                before.append(payload)
            if line[:1] in (b' ', b'+'):
                after.append(payload)
            i += 1
        assert len(before) == old_count and len(after) == new_count
        assert source[old_pos:old_pos + old_count] == before, (section['path'], 'preimage mismatch')
        target.extend(after)
        cursor = old_pos + old_count
        hunks += 1
    target.extend(source[cursor:])
    return b''.join(target), hunks


def main():
    started = datetime.now(timezone.utc).isoformat()
    scratch = Path(run(['mktemp', '-d', '/tmp/fn109-baseline-reconstruction.XXXXXXXX']).decode().strip())
    archive = run(['git', 'archive', '--format=tar', BASE, 'tools/gomad3'])
    tree = run(['git', 'ls-tree', '-r', BASE, '--', 'tools/gomad3']).decode()
    entries = {}
    for line in tree.splitlines():
        meta, path = line.split('\t')
        mode, kind, identity = meta.split()
        assert kind == 'blob' and mode in ('100644', '100755'), (path, mode)
        entries[path] = {'git_blob': identity, 'mode': mode}
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        for member in tar.getmembers():
            assert (member.name == 'tools' or member.name.startswith('tools/')) and '..' not in Path(member.name).parts
            assert member.isdir() or member.isfile()
        tar.extractall(scratch, filter='data')
    for path, meta in entries.items():
        data = (scratch / path).read_bytes()
        assert blob(data) == meta['git_blob'], path
        meta['sha256'] = sha(data)
    protected = read(A108 / 'final-protected-paths.txt').decode()
    expected_modified = set(re.findall(r'^ M (tools/gomad3/\S+)$', protected, re.M))
    expected_new = set(re.findall(r'^\?\? (tools/gomad3/\S+)$', protected, re.M))
    tracked = sections(read(A108 / 'final-working-tree.diff'))
    assert len(tracked) == 24 and {s['path'] for s in tracked} == expected_modified
    changed = []
    for section in tracked:
        path = section['path']
        old = (scratch / path).read_bytes()
        assert section['old'] == 'a/' + path and section['index']
        assert blob(old).startswith(section['index'][0]), path
        new, hunks = apply_exact(old, section)
        assert blob(new).startswith(section['index'][1]), path
        (scratch / path).write_bytes(new)
        changed.append({'path': path, 'input': 'final-working-tree.diff',
                        'block_start_line': section['block_start_line'], 'hunks': hunks,
                        'before_sha256': sha(old), 'after_sha256': sha(new),
                        'before_git_blob': blob(old), 'after_git_blob': blob(new),
                        'retained_index_prefixes': section['index']})
    selected = {
        'task5.diff': ['runner/completion.go', 'runner/completion_test.go', 'runner/completion_characterization_test.go'],
        'task6.diff': ['runner/retention.go', 'runner/retention_test.go', 'runner/retention_characterization_test.go'],
        'task4-working-tree.diff': ['upgrade/upgrade_unix_test.go'],
    }
    overlaps = []
    new_files = []
    for filename, relative_paths in selected.items():
        blocks = sections(read(A108 / filename))
        wanted = {'tools/gomad3/' + path for path in relative_paths}
        additions = [s for s in blocks if s['path'] in wanted]
        assert len(additions) == len(wanted) and {s['path'] for s in additions} == wanted
        overlaps.append({'input': filename, 'selected_new_paths': sorted(wanted),
                         'excluded_paths': sorted(s['path'] for s in blocks if s['path'] not in wanted)})
        for section in additions:
            assert section['old'] == '/dev/null' and section['path'] not in entries
            target = scratch / section['path']
            assert not target.exists()
            data, hunks = apply_exact(b'', section)
            if section['index']:
                assert blob(data).startswith(section['index'][1])
            target.write_bytes(data)
            new_files.append({'path': section['path'], 'input': filename,
                              'block_start_line': section['block_start_line'], 'hunks': hunks,
                              'sha256': sha(data), 'git_blob': blob(data),
                              'retained_index_prefixes': section['index']})
    assert len(new_files) == 7 and {f['path'] for f in new_files} == expected_new
    size_data = read(A108 / 'final-size-files.txt').decode()
    inventory = {}
    for line in size_data.split('# files: class physical code codebytes path\n')[1].splitlines():
        fields = line.split()
        if len(fields) == 5 and fields[4].startswith('tools/gomad3/'):
            inventory[fields[4]] = {'class': fields[0], 'physical': int(fields[1])}
    files = {str(p.relative_to(scratch)): p for p in (scratch / 'tools/gomad3').rglob('*') if p.is_file()}
    assert set(files) == set(entries) | expected_new == set(inventory)
    assert len(files) == 670
    for path, file in files.items():
        assert file.read_bytes().count(b'\n') == inventory[path]['physical'], path
        if path not in expected_modified | expected_new:
            assert sha(file.read_bytes()) == entries[path]['sha256']
    crosschecks = []
    for task, manifest_name in [('task-3', 'source-pre.json'), ('task-6', 'preimages.json')]:
        manifest = json.loads(read(A109 / task / manifest_name))
        records = [{'path': p, 'sha256': h} for p, h in manifest.items()] if isinstance(manifest, dict) else manifest
        for record in records:
            path = record['path']
            preimage = A109 / task / 'preimages' / path
            data = read(preimage)
            assert sha(data) == record['sha256'], (task, path)
            baseline_sha = sha(files[path].read_bytes()) if path in files else None
            crosschecks.append({'task': task, 'path': path, 'retained_sha256': sha(data),
                                'reconstructed_sha256': baseline_sha, 'equal': baseline_sha == sha(data),
                                'role': 'crosscheck only; later fn-109 preimages may include predecessor changes'})
    for path in [A108 / 'final.md', A108 / 'task6-review.md', A108 / 'task6-evidence.md',
                 A108 / 'final-gates/results.txt', A109 / 'task1-evidence.txt',
                 A109 / 'task-21/bounded-campaign-measurement-source-scout.md',
                 ROOT / '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md']:
        read(path)
    source_lines = ''.join(f'{sha(files[path].read_bytes())}  {path}\n' for path in sorted(files))
    input_lines = ''.join(f'{identity}  {path}\n' for path, identity in sorted(inputs.items()))
    (OUT / 'source.sha256').write_text(source_lines)
    (OUT / 'input.sha256').write_text(input_lines)
    evidence = {
        'status': 'verified_reconstruction_inputs_only', 'started_utc': started,
        'ended_utc': datetime.now(timezone.utc).isoformat(), 'base_revision': BASE,
        'baseline_kind': 'actual first fn-109 task: commit plus dirty fn-108.2-.6',
        'scratch_path': str(scratch), 'scope': 'tools/gomad3 only',
        'source_manifest_sha256': sha(source_lines.encode()), 'input_manifest_sha256': sha(input_lines.encode()),
        'archive_sha256': sha(archive), 'tracked_base_files': len(entries),
        'reconstructed_files': len(files), 'tracked_modifications': changed, 'untracked_additions': new_files,
        'selected_blocks_and_excluded_overlaps': overlaps, 'base_files': entries,
        'inventory_physical_lines_verified': 670, 'retained_preimage_crosschecks': crosschecks,
        'inputs': inputs, 'commands': commands, 'commits': [], 'tests': [], 'prs': [],
        'task21_started_or_claimed': False, 'acceptance_claim': False,
        'native_qualification': 'not run', 'go_package_loading': 'not run',
        'routing': {'requested': 'gpt-6.1-sol/high', 'actual_metadata': 'unknown',
                    'tier': 'session; judged once unavailable(no_key)'},
        'provenance_limits': [
            'Historical gate tree fingerprint a875ec2570434ad6 has no retained fingerprint command; not equated to this manifest.',
            'Seven new-file diff blocks generally lack historical full file hashes; their bytes are verified by exact hunk counts, final physical-line inventory, and applicable later preimage matches.',
            'This reconstructs the nested tools/gomad3 module, not an invented full-repository historical snapshot identity.',
            'No measurement, build, test, generation, native qualification or task-21 acceptance was performed.',
        ],
    }
    (OUT / 'evidence.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps({'scratch_path': str(scratch), 'files': len(files),
                      'modified': len(changed), 'new': len(new_files),
                      'source_manifest_sha256': evidence['source_manifest_sha256'],
                      'input_manifest_sha256': evidence['input_manifest_sha256'],
                      'crosscheck_matches': sum(c['equal'] for c in crosschecks),
                      'crosscheck_total': len(crosschecks)}))


if __name__ == '__main__':
    main()
