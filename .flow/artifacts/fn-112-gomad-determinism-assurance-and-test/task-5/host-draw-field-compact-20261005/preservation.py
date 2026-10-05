"""Save task-owned source snapshots and prove the bounded alpha-rename."""
import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
SCRATCH = ROOT / 'tools/gomad3/.toolchain/fn-110/host-draw-compact-20261005.fqsrOf'
STOCK = pathlib.Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin')
DESCRIPTOR = json.loads((ROOT / 'tools/gomad3/toolchain/version/version.json').read_text())
OVERLAY = 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go'
PATHS = DESCRIPTOR['patch_allowlist'] + [OVERLAY]


def source(path):
    return ROOT / path if path == OVERLAY else SCRATCH / 'go' / path


def sha(contents):
    return hashlib.sha256(contents).hexdigest()


if sys.argv[1] == 'save':
    for path in PATHS:
        destination = SCRATCH / 'before' / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(source(path).read_bytes())
    print('saved', len(PATHS), 'baseline source files in ignored task scratch')
else:
    files = {}
    expected_counts = {'src/runtime/runtime2.go': 1, 'src/runtime/proc.go': 3, OVERLAY: 3}
    for path in PATHS:
        before = (SCRATCH / 'before' / path).read_bytes()
        after = source(path).read_bytes()
        materialized = after if path == OVERLAY else (SCRATCH / 'final' / 'go' / path).read_bytes()
        count = before.count(b'gomadHostDrawScope')
        assert count == expected_counts.get(path, 0), (path, count)
        renamed = before.replace(b'gomadHostDrawScope', b'gomadHostDraw')
        normalized_before = subprocess.check_output([str(STOCK / 'gofmt')], input=renamed)
        normalized_after = subprocess.check_output([str(STOCK / 'gofmt')], input=after)
        normalized_materialized = subprocess.check_output([str(STOCK / 'gofmt')], input=materialized)
        assert normalized_before == normalized_after, path
        assert normalized_before == normalized_materialized, path
        if path != OVERLAY:
            assert normalized_after == materialized, ('materialized canonical formatting', path)
        assert before.count(b'previousHostDrawScope') == after.count(b'previousHostDrawScope'), path
        files[path] = {'before_sha256': sha(before), 'after_sha256': sha(after),
                       'alpha_gofmt_sha256': sha(normalized_after), 'renamed_references': count,
                       'raw_equal': before == after, 'alpha_gofmt_equal': True}
        files[path].update(materialized_sha256=sha(materialized), materialized_alpha_gofmt_equal=True)
    freeze = json.loads((OUT / 'freeze.json').read_text())
    tracked = subprocess.check_output(['git', 'ls-files', '-z'], cwd=ROOT).decode().split('\0')
    current = {name: sha((ROOT / name).read_bytes()) for name in tracked
               if name and not name.startswith('.flow/') and (ROOT / name).is_file()}
    changed = {name: {'before_sha256': freeze['sources'].get(name), 'after_sha256': current.get(name)}
               for name in set(freeze['sources']) | set(current)
               if freeze['sources'].get(name) != current.get(name)}
    user = {name: sha((ROOT / name).read_bytes()) for name in freeze['user_files']}
    assert user == freeze['user_files'], 'unrelated user files changed'
    runtime2 = (SCRATCH / 'final/go/src/runtime/runtime2.go').read_text()
    m_body = re.search(r'^type m struct \{\n(.*?)^\}', runtime2, re.MULTILINE | re.DOTALL).group(1)
    m_fields = re.findall(r'^\t(\w+)\s+([^\n]+)', m_body, re.MULTILINE)
    field_index = next(index for index, (name, _) in enumerate(m_fields) if name == 'gomadHostDraw')
    assert m_fields[field_index][1].strip() == 'bool'
    assert m_fields[field_index - 1][0] == 'spinning' and m_fields[field_index + 1][0] == 'blocked'
    m_position = {'field_index': field_index, 'type': 'bool',
                  'previous_field': m_fields[field_index - 1][0], 'next_field': m_fields[field_index + 1][0],
                  'complete_alpha_gofmt_equivalence': True}
    info = {'files': files, 'tracked_product_changes': changed,
            'user_hashes_preserved': user, 'patch_allowlist': DESCRIPTOR['patch_allowlist'],
            'overlay_allowlist': DESCRIPTOR['overlay_allowlist'], 'm_position': m_position}
    (OUT / 'preservation.json').write_text(json.dumps(info, indent=2) + '\n')
    (OUT / 'final-sources.json').write_text(json.dumps(current, indent=2) + '\n')
    print('alpha-rename + pinned gofmt equality:', len(files), 'files; renamed sites:', sum(expected_counts.values()))
    print('changed tracked product files:', '\n'.join(sorted(changed)))
