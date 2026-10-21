#!/usr/bin/env python3
"""Public authenticated inspect regressions for review RID 434a336b."""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import resource
import shutil
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
corpus = Path(__file__).resolve().parent
receipts, failures = [], []


def run(name, argv, status=0, reason=None, bounded=False):
    def limit():
        resource.setrlimit(resource.RLIMIT_AS, (256 << 20, 256 << 20))
    result = subprocess.run([str(args.binary), *map(str, argv)], capture_output=True,
                            text=True, preexec_fn=limit if bounded else None, timeout=60)
    receipts.append(dict(name=name, argv=[str(args.binary), *map(str, argv)],
                         status=result.returncode, expected=status, bounded=bounded,
                         stdout=result.stdout, stderr=result.stderr))
    (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    if result.returncode != status or reason and reason not in result.stderr or status and result.stdout:
        failures.append(name)
    return result


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def seal(path, manifest, details=None):
    if details is not None:
        raw = canonical(details)
        (path / 'evidence/inspection.json').write_bytes(raw)
        entry = dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        manifest['files']['evidence/inspection.json'].update(entry)
        manifest['record']['streams']['evidence/inspection.json'].update(entry)
    raw = canonical(manifest['record'])
    (path / 'record.json').write_bytes(raw)
    manifest['record_sha256'] = hashlib.sha256(raw).hexdigest()
    manifest['files']['record.json'].update(bytes=len(raw), sha256=manifest['record_sha256'])
    (path / 'manifest.json').write_bytes(canonical(manifest))


bundle = json.loads(run('publish-details', ['fixture-inspect-details', '--root', args.work / 'source',
    '--input', corpus / 'inspect-details.json']).stdout)
source = args.work / 'source' / bundle['records'][0]['artifact']


def altered(name, mutate, reason, original=source, status=2):
    path = args.work / name
    shutil.copytree(original, path)
    manifest = json.loads((path / 'manifest.json').read_bytes())
    details = json.loads((path / 'evidence/inspection.json').read_bytes())
    mutate(manifest, details)
    seal(path, manifest, details)
    run(name, ['inspect', '--json', '--choices=false', path], status, reason)
    return path


def minimize(name, change, reason='minimization', status=2):
    return altered(name, lambda manifest, details: change(details['minimization']), reason, status=status)


minimize('count-contradiction', lambda value: value.update(final_forced_decisions=1))
minimize('unknown-kind', lambda value: value['accepted'][0].update(kind='unknown'))
minimize('unknown-dimension', lambda value: value['accepted'][0]['removed'][0].update(dimension='unknown'))
minimize('suffix-ordinal', lambda value: value['accepted'][0]['removed'][0].update(ordinal=20))
minimize('ineffective-reduction', lambda value: value.update(final_candidate_sha256=value['original_candidate_sha256']) or value['accepted'][0].update(after_sha256=value['original_candidate_sha256']))
minimize('duplicate-coordinate', lambda value: value['accepted'][0]['removed'][1].update(ordinal=0))
minimize('empty-reductions', lambda value: value.update(accepted=[], final_candidate_sha256=value['original_candidate_sha256']))
minimize('zero-attempts', lambda value: value.update(attempts=0, accepted=[], final_candidate_sha256=value['original_candidate_sha256'], final_forced_decisions=2))
for kind, dimensions in [('schedule_range', ['runtime', 'scenario', 'network', 'storage', 'crash']),
                         ('fault_entries', ['fault'])]:
    for dimension in dimensions:
        def change(value):
            value['accepted'][0]['kind'] = kind
            for removed in value['accepted'][0]['removed']:
                removed['dimension'] = dimension
        minimize('supported-' + kind + '-' + dimension, change, reason=None, status=0)
minimize('fault-kind-runtime', lambda value: value['accepted'][0].update(kind='fault_entries'))
minimize('suffix-kind-scenario', lambda value: value['accepted'][0]['removed'][0].update(dimension='scenario'))


def split_reductions(value):
    first = value['accepted'][0]
    second = copy.deepcopy(first)
    middle = hashlib.sha256(b'typed intermediate candidate').hexdigest()
    first.update(after_sha256=middle, removed=[first['removed'][1]])
    second.update(before_sha256=middle, removed=[second['removed'][0]])
    value.update(accepted=[first, second], attempts=2)


minimize('successive-suffixes', split_reductions, reason=None, status=0)


def repeated_coordinate(value):
    split_reductions(value)
    value['accepted'][1]['removed'][0]['ordinal'] = 1


minimize('repeated-coordinate-across-reductions', repeated_coordinate)


def repeated_candidate(value):
    split_reductions(value)
    value['accepted'][1]['after_sha256'] = value['original_candidate_sha256']
    value['final_candidate_sha256'] = value['original_candidate_sha256']


minimize('repeated-candidate', repeated_candidate)

choices = json.loads(run('publish-choices', ['fixture-inspect-execution', '--root', args.work / 'choices',
    '--corpus', corpus]).stdout)
complete = args.work / 'choices' / choices['records'][0]['artifact']
overflow = args.work / 'choices' / choices['records'][2]['artifact']


def terminal(state):
    def change(manifest, details):
        details['choices'].update(terminal_state=state, tape_sha256=None)
    return change


altered('overflow-success', terminal('overflow'), 'choice terminal outcome', complete)
altered('interrupted-success', terminal('interrupted'), 'choice terminal outcome', complete)
altered('complete-without-tape', terminal('complete'), 'choice tape', complete)
altered('complete-capacity', terminal('complete'), 'choice terminal outcome', overflow)
altered('overflow-wrong-capacity', lambda manifest, details: manifest['record']['outcome'].update(reason='WASI capacity: unrelated'), 'choice terminal outcome', overflow)


def interrupted(manifest, details):
    terminal('interrupted')(manifest, details)
    manifest['record']['outcome'].update({'class': 'runner_failure', 'termination': 'watchdog_timeout',
                                         'exit_code': None, 'reason': 'execution watchdog timeout'})


altered('interrupted-watchdog', interrupted, None, complete, status=0)
altered('complete-watchdog', lambda manifest, details: manifest['record']['outcome'].update(
    {'class': 'runner_failure', 'termination': 'watchdog_timeout', 'exit_code': None, 'reason': 'execution watchdog timeout'}),
    'choice terminal outcome', complete)
altered('observed-profile-complete-tape', lambda manifest, details: manifest['record'].update(
    profile='gomad.wasi-stock-exploratory/v1'), None, complete, status=0)


def cancelled(manifest, details):
    terminal('interrupted')(manifest, details)
    manifest['record']['outcome'].update({'class': 'runner_failure', 'termination': 'cancelled',
                                         'exit_code': None, 'reason': 'runner_cancelled'})


altered('interrupted-cancelled', cancelled, None, complete, status=0)

# A large valid but undisplayed file must authenticate without a file-sized allocation.
large = args.work / 'streaming-undisplayed'
shutil.copytree(source, large)
size = 512 << 20
file = large / 'evidence/large'
with file.open('wb') as handle:
    handle.truncate(size)
file.chmod(0o600)
digest = hashlib.sha256()
block = bytes(1 << 20)
for _ in range(size // len(block)):
    digest.update(block)
manifest = json.loads((large / 'manifest.json').read_bytes())
manifest['files']['evidence/large'] = dict(bytes=size, sha256=digest.hexdigest(), module=False)
seal(large, manifest)
run('streaming-undisplayed', ['inspect', '--json', large], bounded=True)
with file.open('r+b') as handle:
    handle.write(b'x')
run('streaming-undisplayed-corrupt', ['inspect', '--json', large], 2, 'identity mismatch', bounded=True)
# Metadata-consistent sparse oversized details must refuse on the bound, not allocation.
oversized = args.work / 'oversized-details'
shutil.copytree(source, oversized)
manifest = json.loads((oversized / 'manifest.json').read_bytes())
with (oversized / 'evidence/inspection.json').open('wb') as handle:
    handle.truncate(size)
for entry in [manifest['files']['evidence/inspection.json'], manifest['record']['streams']['evidence/inspection.json']]:
    entry.update(bytes=size, sha256=digest.hexdigest())
seal(oversized, manifest)
run('oversized-details', ['inspect', '--json', oversized], 2, 'caller bound', bounded=True)
# A payload descriptor bound must precede blanket authentication too.
descriptor = args.work / 'oversized-descriptor'
shutil.copytree(source, descriptor)
with (descriptor / 'evidence/large').open('wb') as handle:
    handle.truncate(size)
(descriptor / 'evidence/large').chmod(0o600)
manifest = json.loads((descriptor / 'manifest.json').read_bytes())
manifest['files']['evidence/large'] = dict(bytes=size, sha256=digest.hexdigest(), module=False)
details = json.loads((descriptor / 'evidence/inspection.json').read_bytes())
details['simulation']['plan'].update(file='evidence/large', bytes=size, sha256=digest.hexdigest(), limit=1)
seal(descriptor, manifest, details)
run('oversized-descriptor', ['inspect', '--json', descriptor], 2, 'caller bound', bounded=True)
assert not failures, failures
print('review regressions: bounded public authentication, minimization accounting and terminal consistency passed')
