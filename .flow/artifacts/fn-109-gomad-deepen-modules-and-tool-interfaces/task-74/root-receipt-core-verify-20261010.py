import collections
import functools
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
WORKER = PRIMARY / '.worktrees/fn-109-74-retention-candidate'
OUTPUT = WORKER / '.flow/tmp/fn10974-evidence'
BASE = '305182879f727f0e925483c41b8b8f0679770621'
SELECTED = 'tools/gomad3/runner/retention_characterization_test.go'


@functools.cache
def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def outcomes(name):
    terminals = {}
    for line in (OUTPUT / (name + '.log')).read_text().splitlines():
        event = json.loads(line)
        if event.get('Test') and event.get('Action') in ('pass', 'fail', 'skip'):
            assert event['Test'] not in terminals
            terminals[event['Test']] = event['Action']
    return terminals


receipts = {}
for path in sorted(OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json') or path.stem in ('final-terminal-inspection', 'final-terminal-report'):
        continue
    receipt = json.loads(path.read_text())
    if 'exit_code' not in receipt:
        continue
    binding_path = OUTPUT / (path.stem + '-binding.json')
    binding = json.loads(binding_path.read_text())
    assert sha(binding_path) == receipt['binding_sha256']
    assert sha(OUTPUT / (path.stem + '.log')) == receipt['log_sha256']
    assert receipt['argv'] == binding['argv'] and receipt['cwd'] == binding['cwd']
    assert binding['base'] == BASE
    assert isinstance(receipt['exit_code'], int) and receipt['elapsed_seconds'] > 0
    assert not receipt['timed_out'] and not receipt['remaining_group_members']
    assert receipt['source_after'] == binding['source_manifest']
    assert receipt['tools_after'] == binding['tools_manifest']
    settings = [dict(binding['actual_go_settings']), dict(receipt['actual_go_settings_after'])]
    for setting in settings:
        setting['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>', setting['GOGCCFLAGS'])
    assert settings[0] == settings[1]
    for name, identity in binding['tools_manifest'].items():
        if name == 'make_exported_lookup':
            continue
        target = Path(name)
        assert str(target.resolve()) == identity['resolved']
        assert (os.readlink(target) if target.is_symlink() else None) == identity['link']
        assert sha(target) == identity['sha256']
    for name, identity in binding['source_manifest'].items():
        target = Path(name)
        if name == str(WORKER / SELECTED) and path.stem.startswith('before-'):
            original = subprocess.check_output(['/usr/bin/git', 'show', BASE + ':' + SELECTED], cwd=WORKER)
            assert hashlib.sha256(original).hexdigest() == identity
        elif isinstance(identity, dict):
            assert target.is_symlink() and os.readlink(target) == identity['literal_link']
            assert str(target.resolve()) == identity['resolved']
            assert (sha(target) if target.is_file() else 'NON_FILE_TARGET') == identity['target_sha256']
        elif identity == 'NON_FILE_DIRECTORY_OR_GITLINK; not a consumed file':
            assert target.is_dir()
        elif identity == 'ABSENT_UNMATERIALIZED':
            assert not target.exists() and not target.is_symlink()
        else:
            assert sha(target) == identity, (path.name, name)
    receipts[path.stem] = {'exit_code': receipt['exit_code'], 'log_sha256': receipt['log_sha256']}

assert len(receipts) == 31
old = outcomes('before-ordinary')
new = outcomes('after-ordinary')
traced = outcomes('diagnostic-ordinary-trace')
assert old.keys() == new.keys() == traced.keys() and len(old) == 673
assert collections.Counter(old.values()) == {'pass': 498, 'fail': 163, 'skip': 12}
assert collections.Counter(new.values()) == {'pass': 515, 'fail': 146, 'skip': 12}
assert collections.Counter(traced.values()) == {'pass': 517, 'fail': 144, 'skip': 12}
changed = {name: [old[name], new[name]] for name in old if old[name] != new[name]}
assert len(changed) == 17 and all(value == ['fail', 'pass'] for value in changed.values())
table = outcomes('after-table')
assert len(table) == 19 and set(table.values()) == {'pass'}
failed_leaf = 'TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/failures/choice-exploration'
assert new[failed_leaf] == 'fail' and traced[failed_leaf] == 'pass'
assert 'sync artifact store: context deadline exceeded' in (OUTPUT / 'after-ordinary.log').read_text()
assert 'Error 137' in (OUTPUT / 'fast-lint.log').read_text()
trace = json.loads((OUTPUT / 'diagnostic-trace-output.json').read_text())
assert sha(trace['trace_path']) == trace['trace_sha256']
assert Path(trace['trace_path']).stat().st_size == trace['trace_size_bytes']
original = subprocess.check_output(['/usr/bin/git', 'show', BASE + ':' + SELECTED], cwd=WORKER)
current = (WORKER / SELECTED).read_bytes()
addition = b'configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)\n'
assert current.count(addition) == 2
assert current.replace(b'\t\t\t\t\t' + addition, b'', 1).replace(b'\t' + addition, b'', 1) == original
assert sha(WORKER / SELECTED) == 'c86e26d2c016af55427faea8927c9b224a89778b4b5d70d81dc4257a8163f168'
print(json.dumps({'receipt_count': len(receipts), 'excluded_final_observers': ['final-terminal-inspection', 'final-terminal-report'], 'receipts': receipts, 'all_bound_files_and_tools_rehashed': True, 'named_domain': 673, 'ordinary_before': dict(collections.Counter(old.values())), 'ordinary_after': dict(collections.Counter(new.values())), 'instrumented_diagnostic': dict(collections.Counter(traced.values())), 'selected_restorations': 17, 'whole_file_two_assignment_reconstruction': True, 'product_acceptance_pass': False, 'remaining': ['ordinary artifact-store deadline unexplained', 'fast lint analyzer killed137', 'original-base nested RED50', 'independent integrated source assessment']}, sort_keys=True))
