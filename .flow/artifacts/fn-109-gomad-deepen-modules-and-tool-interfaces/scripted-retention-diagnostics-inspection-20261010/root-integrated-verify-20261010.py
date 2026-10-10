import hashlib
import json
import os
from pathlib import Path
import subprocess

PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
WORKER = PRIMARY / '.worktrees/fn-109-75-six-attachments-candidate'
ARTIFACT = Path('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-retention-diagnostics-inspection-20261010')
RAW = WORKER / '.flow/tmp/fn10975-evidence'
SEALS = {}


def read(path):
    data = Path(path).read_bytes()
    SEALS[str(path)] = hashlib.sha256(data).hexdigest()
    return data


def sha(path):
    return hashlib.sha256(read(path)).hexdigest()


def git(directory, *args):
    return subprocess.check_output(['/usr/bin/git', *args], cwd=directory)


def product_rows(directory):
    result = {}
    for row in git(directory, 'ls-files', '-s', '-z').decode().split('\0'):
        if not row:
            continue
        metadata, name = row.split('\t', 1)
        mode, blob, stage = metadata.split()
        assert stage == '0'
        if not name.startswith(('.flow/', '.turbo/')) and name != 'MILESTONES.md':
            result[name] = (mode, blob)
    return result


read(__file__)
left, right = product_rows(PRIMARY), product_rows(WORKER)
assert left == right
identities = {}
for name, (mode, blob) in sorted(left.items()):
    p, w = PRIMARY / name, WORKER / name
    if mode == '160000':
        identities[name] = {'mode': mode, 'gitlink': blob}
    elif mode == '120000':
        assert p.is_symlink() and w.is_symlink() and os.readlink(p) == os.readlink(w)
        assert sha(p) == sha(w)
        identities[name] = {'mode': mode, 'literal_link': os.readlink(p), 'target_sha256': sha(p)}
    else:
        assert p.is_file() and w.is_file() and sha(p) == sha(w), name
        identities[name] = {'mode': mode, 'sha256': sha(p)}

index = json.loads(read(WORKER / ARTIFACT / 'worker/evidence.json'))
assert len(index['raw_receipts']) == 40
receipts = {}
for path in sorted(RAW.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    receipt = json.loads(read(path))
    if 'process_group' not in receipt:
        continue
    name = path.stem
    binding_path, log_path = RAW / (name + '-binding.json'), RAW / (name + '.log')
    binding = json.loads(read(binding_path))
    assert sha(binding_path) == receipt['binding_sha256'] and sha(log_path) == receipt['log_sha256'], name
    assert receipt['argv'] == binding['argv'] and receipt['cwd'] == binding['cwd']
    assert receipt['exit_code'] == receipt['child_returncode']
    assert receipt['source_before_after_equal'] and receipt['tools_before_after_equal']
    assert receipt['all_commands_terminal'] and not receipt['remaining_group_members'] and not receipt['timed_out']
    if name in index['raw_receipts']:
        entry = index['raw_receipts'][name]
        assert entry['receipt_sha256'] == sha(path)
        assert entry['binding_sha256'] == sha(binding_path) and entry['log_sha256'] == sha(log_path)
        for key in ('argv', 'cwd', 'exit_code', 'elapsed_seconds', 'start_utc', 'end_utc'):
            assert entry[key] == receipt[key]
    receipts[name] = {'receipt_sha256': sha(path), 'binding_sha256': sha(binding_path),
                      'log_sha256': sha(log_path), 'numeric_exit': receipt['exit_code']}
assert len(receipts) == 42

seal = json.loads(read(RAW / 'handover-seal.log'))
assert len(seal['packet_files']) == 18 and seal['all_owned_commands_terminal']
assert seal['observer_exit_code'] == 0 and seal['current_gate_lane']['gate_lane_empty']
for name, expected in seal['packet_files'].items():
    path = Path(name)
    assert sha(path) == sha(PRIMARY / path.relative_to(WORKER)) == expected

core = json.loads(read(PRIMARY / ARTIFACT / 'root-source-ordinary-final-verification-20261010.json'))
assert core['product_acceptance_pass'] is False
for name, expected in core['input_seals'].items():
    assert sha(name) == expected, name

domain = json.loads(read(RAW / 'current-validation-domain.log'))
assert receipts['validate']['numeric_exit'] == 0
assert domain['materialized_domain_files'] == 132 and domain['discovered_top_level_tests'] == 113
assert len(domain['domain_files']) == 132
for name, expected in domain['domain_files'].items():
    assert sha(PRIMARY / name) == sha(WORKER / name) == expected
top_tests = sorted(name for name in domain['domain_files'] if Path(name).parent == Path('tests') and name.endswith('_test.go'))
assert top_tests == sorted(str(path.relative_to(PRIMARY)) for path in (PRIMARY / 'tests').glob('*_test.go'))
for name, goos, goarch in (('vet-linux-amd64', 'linux', 'amd64'), ('vet-darwin-arm64', 'darwin', 'arm64')):
    binding = json.loads(read(RAW / (name + '-binding.json')))
    settings = binding['actual_go_settings']
    assert (settings['GOOS'], settings['GOARCH'], settings['GOHOSTOS'], settings['GOHOSTARCH']) == (goos, goarch, 'linux', 'arm64')
    assert receipts[name]['numeric_exit'] == 0

print(json.dumps({
    'primary_head': git(PRIMARY, 'rev-parse', 'HEAD').decode().strip(),
    'worker_head': git(WORKER, 'rev-parse', 'HEAD').decode().strip(),
    'tracked_product_identities_equal': len(identities),
    'product_identity_map_sha256': hashlib.sha256(json.dumps(identities, sort_keys=True).encode()).hexdigest(),
    'special_modes': {name: identity for name, identity in identities.items() if identity['mode'] in ('120000', '160000')},
    'packet_files_equal': 18,
    'terminal_receipts': receipts,
    'core_input_seals_reverified': len(core['input_seals']),
    'validation_domain_files': 132,
    'discovered_top_level_tests': len(top_tests),
    'input_seals': SEALS,
    'product_acceptance_pass': False,
    'limits': 'Non-Go reconciliation only. Flow/Turbo artifacts and MILESTONES are excluded from product equality; existing user edits are separately protected. Symlink targets are compared by literal link and content, not relocated absolute resolution. Gitlinks compare index identities only. Retained worker receipts prove their recorded commands, not a fresh primary execution. Historical empty groups are point observations. Ordinary failures, RED50/RED6, unreached integrated errortype, task74 historical deadline and native acceptance remain open.',
}, sort_keys=True, indent=2))
