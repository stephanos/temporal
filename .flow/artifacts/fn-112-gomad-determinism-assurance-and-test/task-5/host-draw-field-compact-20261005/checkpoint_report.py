"""Assemble source-bound checkpoint facts from retained terminal receipts."""
import hashlib
import json
import pathlib
import re

import capture

OUT = capture.OUT
ROOT = capture.ROOT
destination = OUT / 'checkpoint-report.json'
assert not destination.exists(), 'checkpoint report already exists'
baseline = json.loads((OUT / 'freeze.json').read_text())
final = json.loads((OUT / 'final-freeze.json').read_text())
assert capture.sources() == final['sources'], 'product source changed after final freeze'
assert {path: capture.digest(ROOT / path) for path in baseline['user_files']} == baseline['user_files']
assert {path: capture.digest(pathlib.Path(path)) for path in baseline['tools']} == baseline['tools']

measurements = {}
for phase in ['baseline', 'final']:
    measurements[phase] = {}
    for context in [1, 3]:
        path = OUT / f'{phase}-U{context}.patch'
        contents = path.read_bytes()
        measurements[phase][f'U{context}'] = {'bytes': len(contents), 'lines': contents.count(b'\n'),
                                             'sha256': hashlib.sha256(contents).hexdigest()}
assert (OUT / 'final-U1.patch').read_bytes() == (ROOT / 'tools/gomad3/toolchain/runtime/go1.27.1.patch').read_bytes()
assert measurements['baseline']['U1']['sha256'] == baseline['sources']['tools/gomad3/toolchain/runtime/go1.27.1.patch']

verbose = {}
for label in ['baseline-focused', 'final-focused', 'baseline-red-inventory', 'final-green-inventory']:
    contents = (OUT / (label + '.stdout')).read_text()
    verbose[label] = {result: len(re.findall(r'^\s*--- ' + result.upper() + r':', contents, re.MULTILINE))
                      for result in ['pass', 'fail', 'skip']}
    verbose[label]['top_level_pass'] = len(re.findall(r'^--- PASS:', contents, re.MULTILINE))

allowed = set(capture.recorded_environment({}))
environment_objects = []


def audit(value, path, pointer=''):
    if isinstance(value, dict):
        for key, item in value.items():
            if key == 'environment' and isinstance(item, dict):
                unexpected = sorted(set(item) - allowed)
                assert not unexpected, ('unexpected environment names', str(path), unexpected)
                environment_objects.append({'file': str(path.relative_to(OUT)), 'pointer': pointer + '/environment',
                                            'key_count': len(item), 'unexpected_key_count': 0})
            audit(item, path, pointer + '/' + key)
    elif isinstance(value, list):
        for index, item in enumerate(value):
            audit(item, path, pointer + '/' + str(index))


json_files = sorted(OUT.rglob('*.json'))
for path in json_files:
    audit(json.loads(path.read_text()), path)
secret_name_hits = 0
for path in list(OUT.rglob('*.stdout')) + list(OUT.rglob('*.stderr')):
    secret_name_hits += len(re.findall(r'\b(?:GH_TOKEN|GITHUB_TOKEN|OPENAI_API_KEY|AWS_SECRET_ACCESS_KEY)\s*[=:]', path.read_text(errors='replace')))
assert secret_name_hits == 0, 'credential-like assignments in raw logs require root inspection'

receipts = {}
for path in sorted(OUT.glob('*.json')):
    value = json.loads(path.read_text())
    if 'argv' in value and 'exit' in value:
        receipts[path.stem] = {'receipt': path.name, 'sha256': capture.digest(path),
                               'exit': value['exit'], 'elapsed_seconds': value['elapsed_seconds'],
                               'source_changes': value['source_changes'], 'argv': value['argv'], 'cwd': value['cwd']}

info = {'base_commit': baseline['base_commit'], 'measurements': measurements,
        'U1_saved_bytes': measurements['baseline']['U1']['bytes'] - measurements['final']['U1']['bytes'],
        'U3_saved_bytes': measurements['baseline']['U3']['bytes'] - measurements['final']['U3']['bytes'],
        'historical_R8_U3_bytes': 32652,
        'historical_R8_remaining_bytes': measurements['final']['U3']['bytes'] - 32652,
        'final_U1_equals_checked_patch': True, 'final_sources_equal_freeze': True,
        'source_count': len(final['sources']), 'changed_product_paths': final['changed_product_paths'],
        'user_and_pinned_tool_hashes_preserved': True, 'verbose_test_counts': verbose,
        'preservation_receipt': 'preservation.json',
        'fixture_sha256': capture.digest(ROOT / 'tools/gomad3/runner/testdata/diagnostic-identity-choices.json'),
        'fixture_equals_retained_emit': (ROOT / 'tools/gomad3/runner/testdata/diagnostic-identity-choices.json').read_bytes() == (OUT / 'final-identity-emit.stdout').read_bytes(),
        'environment_audit': {'json_files': len(json_files), 'objects': environment_objects,
                              'allowed_key_names': sorted(allowed), 'unexpected_key_count': 0,
                              'credential_assignment_log_hits': secret_name_hits},
        'tool_helpers': {name: capture.digest(OUT / name) for name in
                         ['capture.py', 'compact_checkpoint_test.go', 'test-overlay.json', 'preservation.py',
                          'observations.py', 'compare_receipts.py', 'final_freeze.py', 'identity-audit/derive.mjs']},
        'lint_tools': {str(path): capture.digest(ROOT / path) for path in
                       [pathlib.Path('.bin/golangci-lint-v2.13.0'), pathlib.Path('.bin/errortype'),
                        pathlib.Path('/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'),
                        pathlib.Path('/tmp/fn109-lint-tools.ZdNe1t50/errortype')]},
        'receipts': receipts}
assert info['fixture_equals_retained_emit']
destination.write_text(json.dumps(info, indent=2) + '\n')
print('checkpoint source/tool/user/fixture closure verified;', len(receipts), 'terminal receipts')
print('environment whitelist:', len(allowed), 'allowed keys;', len(environment_objects),
      'objects;', sum(item['key_count'] for item in environment_objects), 'keys; unexpected0; credential-log-assignments0')
print('verbose test counts:', json.dumps(verbose, sort_keys=True))
