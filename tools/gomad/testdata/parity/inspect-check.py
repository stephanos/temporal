#!/usr/bin/env python3
"""Exercise public inspection and its authenticated artifact boundary."""
import argparse
import hashlib
import json
import copy
import shutil
import os
from pathlib import Path
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
p.add_argument('--details-input', type=Path, required=True)
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=True)
receipts = []


def run(*argv, status=0):
    result = subprocess.run([str(args.binary), *map(str, argv)], capture_output=True, text=True)
    receipts.append(dict(argv=[str(args.binary), *map(str, argv)], status=result.returncode,
                         stdout=result.stdout, stderr=result.stderr))
    (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    assert result.returncode == status, (argv, result.returncode, result.stderr)
    return result.stdout


published = json.loads(run('fixture-publish', '--root', args.work / 'store', '--seed', 7))
artifact = Path(published['artifact'])
report = json.loads(run('inspect', '--json', artifact))
assert report['schema'] == 'gomad.inspect/v9'
assert report['kind'] == 'artifact'
facts = report['artifact']
record_raw = (artifact / 'record.json').read_bytes()
assert facts['record'] == json.loads(record_raw)
assert facts['record_sha256'] == hashlib.sha256(record_raw).hexdigest()
assert facts['record']['seed'] == 7
assert facts['stored_bytes'] == published['stored_bytes']
assert facts['choices'] is None
relative = os.path.relpath(artifact, Path.cwd())
assert json.loads(run('inspect', '--json', relative))['path'] == str(artifact)
assert 'seed=7' in run('inspect', artifact)
assert json.loads(run('inspect', '--json=true', '--choices=false', artifact)) == report
run('inspect', '--choices', artifact, status=2)
assert 'no choice trace' in receipts[-1]['stderr']
for argv in [('inspect',), ('inspect', '--unknown', artifact),
             ('inspect', '--json', artifact, artifact), ('inspect', '--json=wrong', artifact)]:
    run(*argv, status=2)
corpus = Path(__file__).resolve().parent
bundle = json.loads(run('fixture-inspect-execution', '--root', args.work / 'choices', '--corpus', corpus))
assert len(bundle['records']) == 4, 'complete and actual overflow executions are required'
for entry in bundle['records']:
    choice_artifact = args.work / 'choices' / entry['artifact']
    inspected = json.loads(run('inspect', '--json', '--choices', choice_artifact))
    choice = inspected['artifact']['choices']
    assert choice['records'] == choice['branching_records'] == choice['runnable'] == 1
    assert choice['select_poll'] == choice['select_result'] == 0
    is_overflow = inspected['artifact']['record']['outcome']['termination'] == 'capacity'
    assert choice['exact_replay_available'] == (not is_overflow)
    if is_overflow:
        assert choice['terminal_state'] == 'overflow' and choice['limit'] == 160
        assert choice['replay_decisions'] == [] and choice['tape_sha256'] == ''
    else:
        complete_artifact = choice_artifact
        assert choice['replay_decisions'] == [dict(ordinal=0, kind='runnable', site_offset=42,
                                                   site_missing=False, alternatives=2, selected=choice['replay_decisions'][0]['selected'])]
    assert len(choice['sites']) == 1 and choice['sites'][0]['count'] == 1
    if not is_overflow:
        assert '"site_offset":42' in run('inspect', '--choices', choice_artifact)
    expected_fingerprint = hashlib.sha256(bytes.fromhex(inspected['artifact']['record']['identity']['target']) + b'\x01' + (42).to_bytes(8, 'big')).hexdigest()
    assert choice['sites'][0]['fingerprint'] == expected_fingerprint
choice_artifact = complete_artifact

source = json.loads(args.details_input.read_bytes())
details_bundle = json.loads(run('fixture-inspect-details', '--root', args.work / 'details', '--input', args.details_input))
details_artifact = args.work / 'details' / details_bundle['records'][0]['artifact']
facts = json.loads(run('inspect', '--json', details_artifact))['artifact']
for section in ('world', 'simulation', 'minimization'):
    assert facts[section] == source['details'][section], section
    assert f'"{section}":' in run('inspect', details_artifact)


def malformed(name, mutate, reason):
    value = copy.deepcopy(source)
    mutate(value)
    input_path = args.work / (name + '.json')
    input_path.write_text(json.dumps(value))
    root = args.work / name
    result = json.loads(run('fixture-inspect-details', '--root', root, '--input', input_path))
    run('inspect', '--json', root / result['records'][0]['artifact'], status=2)
    assert reason in receipts[-1]['stderr'], (name, receipts[-1])
    assert not receipts[-1]['stdout']


malformed('world-seed', lambda value: value['details']['world'].__setitem__('seed', 999), 'World seed mismatch')
malformed('world-time', lambda value: value['details']['world'].__setitem__('now_nanos', 999), 'World snapshot summary mismatch')
malformed('world-count', lambda value: value['details']['world'].__setitem__('transitions', 999), 'World snapshot summary mismatch')
malformed('simulation-digest', lambda value: value['details']['simulation']['record'].__setitem__('sha256', '0' * 64), 'payload identity mismatch')
malformed('simulation-bound', lambda value: value['details']['simulation']['plan'].__setitem__('limit', 1), 'caller bound')
malformed('minimization-budget', lambda value: value['details']['minimization'].__setitem__('attempts', 999), 'invalid minimization evidence')
malformed('minimization-replay', lambda value: value['details']['minimization']['predicate'].__setitem__('replay_match', False), 'invalid minimization evidence')
malformed('minimization-chain', lambda value: value['details']['minimization']['accepted'][0].__setitem__('before_sha256', '0' * 64), 'reduction chain')
malformed('minimization-final', lambda value: value['details']['minimization'].__setitem__('final_candidate_sha256', '0' * 64), 'final candidate mismatch')
malformed('minimization-duplicate', lambda value: value['details']['minimization']['accepted'][0]['removed'].append(copy.deepcopy(value['details']['minimization']['accepted'][0]['removed'][0])), 'minimization')
malformed('output-accounting', lambda value: value['details'].__setitem__('outputs', {'stdout': dict(full_sha256='0' * 64, total_bytes=99, retained_bytes=1, discarded_bytes=0, truncated=False)}), 'output stream accounting mismatch')

truncated = copy.deepcopy(source)
output = dict(full_sha256=hashlib.sha256(b'foundation-outputxyz').hexdigest(),
              total_bytes=20, retained_bytes=17, discarded_bytes=3, truncated=True)
truncated['details']['outputs'] = {'stdout': output}
truncated_input = args.work / 'truncated.json'
truncated_input.write_text(json.dumps(truncated))
truncated_root = args.work / 'truncated'
truncated_bundle = json.loads(run('fixture-inspect-details', '--root', truncated_root, '--input', truncated_input))
truncated_report = json.loads(run('inspect', '--json', truncated_root / truncated_bundle['records'][0]['artifact']))
assert truncated_report['artifact']['stdout'] == output


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def rewrite(path, alter):
    manifest = json.loads((path / 'manifest.json').read_bytes())
    details = json.loads((path / 'evidence/inspection.json').read_bytes())
    alter(manifest, details)
    raw = canonical(details)
    (path / 'evidence/inspection.json').write_bytes(raw)
    entry = dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
    manifest['files']['evidence/inspection.json'].update(entry)
    if 'evidence/inspection.json' in manifest['record']['streams']:
        manifest['record']['streams']['evidence/inspection.json'].update(entry)
    raw = canonical(manifest['record'])
    (path / 'record.json').write_bytes(raw)
    manifest['record_sha256'] = hashlib.sha256(raw).hexdigest()
    manifest['files']['record.json'].update(bytes=len(raw), sha256=manifest['record_sha256'])
    (path / 'manifest.json').write_bytes(canonical(manifest))


for name, alter, reason in [
    ('unbound-details', lambda manifest, details: manifest['record']['streams'].pop('evidence/inspection.json'), 'not bound'),
    ('unknown-detail', lambda manifest, details: details.__setitem__('unknown', 1), 'unknown field'),
    ('choice-count', lambda manifest, details: details['choices'].__setitem__('records', 99), 'choice trace bounds mismatch'),
    ('choice-tape', lambda manifest, details: details['choices'].__setitem__('tape_sha256', '0' * 64), 'choice tape identity mismatch'),
    ('choice-target', lambda manifest, details: details['choices']['identity'].__setitem__('TargetSHA256', [0] * 32), 'choice target identity mismatch'),
]:
    path = args.work / name
    shutil.copytree(choice_artifact, path)
    rewrite(path, alter)
    run('inspect', '--json', path, status=2)
    assert reason in receipts[-1]['stderr'], (name, receipts[-1])
    assert not receipts[-1]['stdout']
overflow = args.work / 'overflow'
shutil.copytree(choice_artifact, overflow)


def incomplete(manifest, details):
    details['choices']['terminal_state'] = 'overflow'
    details['choices']['tape_sha256'] = None


rewrite(overflow, incomplete)
run('inspect', '--json', '--choices', overflow, status=2)
assert 'choice terminal outcome mismatch' in receipts[-1]['stderr'] and not receipts[-1]['stdout']

malformed_trace = args.work / 'malformed-trace'
shutil.copytree(choice_artifact, malformed_trace)


def corrupt_trace(manifest, details):
    file = malformed_trace / 'evidence/choice'
    raw = bytearray(file.read_bytes())
    raw[7] = 1
    file.write_bytes(raw)
    digest = hashlib.sha256(raw).hexdigest()
    manifest['files']['evidence/choice']['sha256'] = digest
    manifest['record']['streams']['evidence/choice']['sha256'] = digest
    details['choices']['trace']['sha256'] = digest


rewrite(malformed_trace, corrupt_trace)
run('inspect', '--json', malformed_trace, status=2)
assert 'malformed_trace' in receipts[-1]['stderr'] and not receipts[-1]['stdout']
print('public inspect facts and grammar passed')
