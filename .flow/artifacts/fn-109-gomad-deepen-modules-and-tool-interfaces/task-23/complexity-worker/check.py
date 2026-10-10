import collections
import hashlib
import json
import re
import subprocess
import sys
sys.dont_write_bytecode = True
import run as r

BASE = (r.ROOT / '.flow/tmp/base_commit').read_text().strip()
MAIN = 'cmd/tools/lintcode/main.go'
TEST = 'cmd/tools/lintcode/main_test.go'

def original(path):
    return subprocess.check_output(['/usr/bin/git', 'show', BASE + ':' + path], cwd=r.ROOT, text=True)

def function(text, name):
    pattern = r'(?m)^func (?:\([^\n]+\) )?' + re.escape(name) + r'\('
    start = re.search(pattern, text).start()
    end = text.index('\n}\n', start) + len('\n}\n')
    return text[start:end]

def dedent(text, levels):
    return '\n'.join(line[levels:] if line.startswith('\t' * levels) else line for line in text.split('\n'))

base, current = original(MAIN), (r.ROOT / MAIN).read_text()
assert function(base, 'regularHostDirectory') == function(current, 'regularHostDirectory')
load = function(base, 'loadOwnership')
lookup = load[load.index('\t\tobject :='):load.index('\t\tentries, err :=')]
want_lookup = dedent(lookup, 1).replace('return policy, ', 'return nil, ') + '\treturn literalPaths(name, values.Values[0])\n'
assert function(current, 'classificationPaths').split('{\n', 1)[1].removesuffix('}\n') == want_lookup
registration = load[load.index('\t\t\tfor _, entry :=', load.index('case "hostSourcePackages"')):load.index('\t\tdefault:')]
want_registration = dedent(registration, 2).replace('policy.hostPackages', 'p.hostPackages').replace('return policy, ', 'return ') + '\treturn nil\n'
assert function(current, 'registerHostPackages').split('{\n', 1)[1].removesuffix('}\n') == want_registration
classify = function(base, 'classify')
validation = classify[classify.index('\t\tif !p.hostPackages'):classify.index('\tdefault:')]
want_validation = dedent(validation, 1).replace('return entry, ', 'return ') + '\treturn nil\n'
assert function(current, 'validateGomadHostSource').split('{\n', 1)[1].removesuffix('}\n') == want_validation
coverage = function(base, 'coveredPackages')
decoder = coverage[coverage.index('\tcovered :='):coverage.index('\tfor _, entry := range sources {', coverage.index('\tcovered :='))]
assert function(current, 'packageSourceCoverage').split('{\n', 1)[1].removesuffix('}\n') == decoder + '\treturn covered, nil\n'

reconstructed = current
for name in ('classificationPaths', 'registerHostPackages', 'validateGomadHostSource', 'packageSourceCoverage'):
    reconstructed = reconstructed.replace(function(current, name) + '\n', '', 1)
reconstructed = reconstructed.replace('\t\tentries, err := classificationPaths(file, name)\n', lookup + '\t\tentries, err := literalPaths(name, values.Values[0])\n', 1)
reconstructed = reconstructed.replace('\t\t\tif err := policy.registerHostPackages(root, entries); err != nil {\n\t\t\t\treturn policy, err\n\t\t\t}\n', registration, 1)
reconstructed = reconstructed.replace('\t\tif err := p.validateGomadHostSource(path, relative); err != nil {\n\t\t\treturn entry, err\n\t\t}\n', validation, 1)
reconstructed = reconstructed.replace('\tcovered, err := packageSourceCoverage(data)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n', decoder, 1)
assert reconstructed == base
tests = (r.ROOT / TEST).read_text()
new_tests = tests[tests.index('func TestLintRuntimeHostRegistration('):tests.index('func TestLintRuntimeHostPackages(')]
assert tests.replace(new_tests, '', 1).replace('\t"io"\n', '', 1) == original(TEST)

def outcomes(name):
    events = [json.loads(line) for line in (r.OUTPUT / (name + '.log')).read_text().splitlines()]
    return {event['Test']: event['Action'] for event in events if event.get('Test') and event['Action'] in ('pass', 'fail', 'skip')}

before, after = outcomes('before-helper-tests'), outcomes('final-helper-tests')
assert all(after[name] == status for name, status in before.items())
assert set(before.values()) == {'pass', 'skip'} and set(after.values()) == {'pass', 'skip'}
controls = outcomes('before-characterization')
assert len(controls) == 13 and set(controls.values()) == {'pass'}
assert all(after[name] == 'pass' for name in controls)
assert set(after) - set(before) == set(controls)

def blocks(path):
    lines = path.read_text().splitlines()
    return ['\n'.join(lines[index:index + 3]) for index, line in enumerate(lines) if re.match(r'^tools/gomad3/[^:]+:\d+:\d+: .+$', line)]

prior = r.ROOT.parent / 'fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-aggregate-lint.log'
retained, fast, nested = blocks(prior), blocks(r.OUTPUT / 'final-fast-lint.log'), blocks(r.OUTPUT / 'final-gomad-lint.log')
assert retained == fast == nested and len(fast) == 50
assert hashlib.sha256('\n'.join(fast).encode()).hexdigest() == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
assert collections.Counter(re.search(r'\(([^()]*)\)$', block.splitlines()[0])[1] for block in fast) == {'forbidigo': 8, 'staticcheck': 42}
assert '0 issues.' in (r.OUTPUT / 'final-helper-lint.log').read_text()
assert (r.OUTPUT / 'final-gofmt.log').read_bytes() == b''
first = json.loads((r.OUTPUT / 'before-helper-lint-binding.json').read_text())['sources']
last = json.loads((r.OUTPUT / 'final-mixedbrain-lint-binding.json').read_text())['sources']
product_paths = [name for name in first if not name.startswith('/') and first[name] != last[name]]
assert product_paths == [MAIN, TEST]
assert {name for name in first if not name.startswith('/')} == {name for name in last if not name.startswith('/')}
receipts = {}
runner = (r.PACKET / 'run.py').read_text()
initial_runner = runner.replace("    for path in sorted(PACKET.glob('*.py')):\n        result[str(path)] = digest(path)\n", "    result[str(Path(__file__).resolve())] = digest(__file__)\n", 1).replace("r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)'", "r'go-build[0-9]+'", 1)
runner_preimages = {hashlib.sha256(text.encode()).hexdigest(): text for text in (initial_runner, runner)}
for path in sorted(r.OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    receipt = json.loads(path.read_text())
    if 'exit_code' not in receipt:
        continue
    assert receipt['source_before_after_equal'] and receipt['tools_before_after_equal'] and receipt['settings_before_after_equal_normalized'] and receipt['all_commands_terminal'] and not receipt['timed_out']
    binding = json.loads((r.OUTPUT / receipt['binding']).read_text())
    assert r.normalized(binding['actual_go_settings']) == r.normalized(receipt['actual_go_settings_after'])
    assert binding['sources'][str(r.PACKET / 'run.py')] in runner_preimages
    receipts[path.stem] = {key: receipt[key] for key in ('exit_code', 'elapsed_seconds', 'log_sha256')}
for sha, text in runner_preimages.items():
    (r.OUTPUT / ('run-preimage-' + sha + '.py')).write_text(text)
print(json.dumps({'base_commit': BASE, 'exact_production_reconstruction': True, 'all_original_test_bytes_preserved': True, 'regularHostDirectory_unchanged': True, 'protected_product_inputs_unchanged': len(first) - len(product_paths) - 1, 'baseline_test_counts': dict(collections.Counter(before.values())), 'final_test_counts': dict(collections.Counter(after.values())), 'pre_change_characterization_counts': dict(collections.Counter(controls.values())), 'retained_nested_lint_blocks': len(fast), 'nested_lint_blocks_sha256': hashlib.sha256('\n'.join(fast).encode()).hexdigest(), 'receipts': receipts}, indent=2))
