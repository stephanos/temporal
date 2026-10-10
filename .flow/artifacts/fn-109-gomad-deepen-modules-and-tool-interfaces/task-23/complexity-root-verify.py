import collections
import hashlib
import json
from pathlib import Path
import re
import subprocess

PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
CANDIDATE = PRIMARY / '.worktrees/fn-109-23-lint-complexity-candidate'
RAW = CANDIDATE / '.flow/tmp/fn10923-complexity'
PACKET = CANDIDATE / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23/complexity-worker'
BASE = '671750104b609e4ddba31a096ae0031c4380b5ae'
PRODUCT = {
    'cmd/tools/lintcode/main.go': 'cb96fa8a2a8b73ed1d50c8ee201a5048fe20c349c9aebe76342ed3737365c986',
    'cmd/tools/lintcode/main_test.go': 'f7ad16372cae34ee431d0f0b07dd9d84cf54cd1d2b2e02e2fb2a4030a7738627',
}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def outcomes(name):
    events = [json.loads(line) for line in (RAW / (name + '.log')).read_text().splitlines()]
    terminal = [event for event in events if event.get('Test') and event['Action'] in ('pass', 'fail', 'skip')]
    result = {event['Test']: event['Action'] for event in terminal}
    assert len(result) == len(terminal)
    return result


def blocks(path):
    lines = path.read_text().splitlines()
    return ['\n'.join(lines[index:index + 3]) for index, line in enumerate(lines)
            if re.match(r'^tools/gomad3/[^:]+:\d+:\d+: .+$', line)]


for name, wanted in PRODUCT.items():
    assert digest(CANDIDATE / name) == wanted, name
    original = subprocess.check_output(['/usr/bin/git', 'show', BASE + ':' + name], cwd=PRIMARY)
    assert (PRIMARY / name).read_bytes() in (original, (CANDIDATE / name).read_bytes()), name

receipts = {}
raw_hashes = {path.name: digest(path) for path in sorted(RAW.iterdir()) if path.is_file()}
for path in sorted(RAW.glob('*.json')):
    receipt = json.loads(path.read_text())
    if 'exit_code' not in receipt:
        continue
    binding = json.loads((RAW / receipt['binding']).read_text())
    assert digest(RAW / receipt['log']) == receipt['log_sha256'], path.name
    assert binding['argv'] == receipt['argv'] and binding['cwd'] == receipt['cwd']
    assert receipt['source_before_after_equal'] and binding['sources'] == receipt['sources_after']
    assert receipt['tools_before_after_equal'] and binding['tools'] == receipt['tools_after']
    assert receipt['all_commands_terminal'] and not receipt['remaining_process_group'] and not receipt['timed_out']
    assert isinstance(receipt['exit_code'], int) and receipt['elapsed_seconds'] >= 0
    for tool, identity in receipt['tools_after'].items():
        actual = Path(tool)
        assert str(actual.resolve()) == identity['realpath']
        assert digest(actual.resolve()) == identity['sha256'], tool
    for setting in ('GOOS', 'GOARCH', 'GOVERSION', 'GOTOOLCHAIN', 'GOENV', 'GOWORK', 'GOFLAGS', 'GOMOD', 'GOCACHE', 'GOMODCACHE', 'GOTMPDIR'):
        assert binding['actual_go_settings'][setting] == receipt['actual_go_settings_after'][setting]
    for source, value in receipt['sources_after'].items():
        if source.endswith('/run.py'):
            assert digest(RAW / ('run-preimage-' + value + '.py')) == value
        if path.stem.startswith('final-') and source in PRODUCT:
            assert value == PRODUCT[source], (path.name, source)
    receipts[path.stem] = {key: receipt[key] for key in ('exit_code', 'elapsed_seconds', 'log_sha256')}

before, final, additions = outcomes('before-helper-tests'), outcomes('final-helper-tests'), outcomes('before-characterization')
assert all(final[name] == action for name, action in before.items())
assert set(final) - set(before) == set(additions)
assert len(additions) == 13 and set(additions.values()) == {'pass'}
assert collections.Counter(before.values()) == {'pass': 85, 'skip': 1}
assert collections.Counter(final.values()) == {'pass': 98, 'skip': 1}
assert before['TestLintPolicyRealGolangci'] == final['TestLintPolicyRealGolangci'] == 'skip'
assert 'set LINT_POLICY_GOLANGCI to the pinned golangci-lint v2.13.0 binary to run actual policy fixtures' in (RAW / 'final-helper-tests.log').read_text()
assert 'LINT_POLICY_GOLANGCI' not in json.loads((RAW / 'final-helper-tests-binding.json').read_text())['environment']
expected_exits = {
    'before-helper-lint': 1, 'after-helper-lint': 1, 'final-helper-lint': 0,
    'final-helper-tests': 0, 'final-helper-vet': 0, 'final-helper-errortype': 0,
    'final-gofmt': 0, 'final-ownership': 0, 'final-architecture': 0,
    'final-validate': 0, 'final-fast-lint': 2, 'final-gomad-lint': 2,
    'final-mixedbrain-lint': 0,
}
assert all(receipts[name]['exit_code'] == value for name, value in expected_exits.items())
retained = blocks(PRIMARY / '.worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-aggregate-lint.log')
assert retained == blocks(RAW / 'final-fast-lint.log') == blocks(RAW / 'final-gomad-lint.log')
block_sha = hashlib.sha256('\n'.join(retained).encode()).hexdigest()
assert len(retained) == 50 and block_sha == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
assert (RAW / 'final-helper-lint.log').read_text().strip() == '0 issues.'
assert (RAW / 'final-gofmt.log').read_bytes() == b''
preservation = json.loads((RAW / 'preservation-check.log').read_text())
assert preservation['exact_production_reconstruction'] and preservation['all_original_test_bytes_preserved']
assert preservation['regularHostDirectory_unchanged'] and preservation['base_commit'] == BASE
assert preservation['nested_lint_blocks_sha256'] == block_sha
print(json.dumps({'candidate': str(CANDIDATE), 'base': BASE, 'product': PRODUCT,
                  'receipts': receipts, 'raw_files': raw_hashes,
                  'original_outcomes_unchanged': len(before), 'additive_pass_outcomes': len(additions),
                  'actual_tool_fixture_skip': 'LINT_POLICY_GOLANGCI unset in complete helper suite',
                  'nested_lint_blocks_sha256': block_sha,
                  'limits': 'Retained execution evidence; current tool seals do not repair missing historical ps or inherited nonselected-environment capture. No native or aggregate PASS.'}, indent=2))
