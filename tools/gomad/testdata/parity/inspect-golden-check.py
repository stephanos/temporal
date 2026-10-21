#!/usr/bin/env python3
"""Authenticate raw evidence before comparing the exact inspect view twice."""
import argparse
import importlib.machinery
import importlib.util
import json
import os
from pathlib import Path
import subprocess

corpus = Path(__file__).resolve().parent
loader = importlib.machinery.SourceFileLoader('parity', str(corpus.parents[1] / 'scripts/behavioral-parity'))
spec = importlib.util.spec_from_loader(loader.name, loader)
parity = importlib.util.module_from_spec(spec)
loader.exec_module(parity)
p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
p.add_argument('--details-input', type=Path, required=True)
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def command(*argv):
    result = subprocess.run(list(map(str, argv)), capture_output=True, timeout=120)
    receipts.append(dict(argv=list(map(str, argv)), status=result.returncode,
                         stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_bytes(parity.canonical(receipts))
    assert result.returncode == 0, (argv, result.returncode, result.stderr)
    return result.stdout


def payload(value, path):
    parity.fields(value, dict(schema=str, file=str, sha256=str, bytes=int, limit=int), path)


def optional_facts(value):
    if value['world'] is not None:
        world = value['world']
        parity.fields(world, dict(profile=str, implementation_sha256=str, seed=int,
                                  now_nanos=int, transitions=int, terminal_kind=str,
                                  terminal_detail=str, history=dict), '$inspect.artifact.world')
        payload(world['history'], '$inspect.artifact.world.history')
    if value['simulation'] is not None:
        simulation = value['simulation']
        parity.fields(simulation, dict(profile=str, controller_sha256=str, execution_sha256=str,
                                       candidate_sha256=str, outcome_sha256=str,
                                       failure_sha256=(str, type(None)), plan=dict, record=dict), '$inspect.artifact.simulation')
        for name in ('plan', 'record'):
            payload(simulation[name], '$inspect.artifact.simulation.' + name)
    if value['minimization'] is not None:
        minimization = value['minimization']
        parity.fields(minimization, dict(schema=str, implementation_sha256=str, parent_record_hash=str,
                                         parent_failure_signature=str, original_candidate_sha256=str,
                                         final_candidate_sha256=str, attempt_budget=int, attempts=int,
                                         original_forced_decisions=int, final_forced_decisions=int,
                                         accepted=list, predicate=dict), '$inspect.artifact.minimization')
        parity.fields(minimization['predicate'], dict(failure_signature=str, domain=str, reason=str,
                                                     termination=str, replay_match=bool, choice_replay=str,
                                                     simulation_replay=str), '$inspect.artifact.minimization.predicate')
        for reduction in minimization['accepted']:
            parity.fields(reduction, dict(kind=str, before_sha256=str, after_sha256=str, removed=list), '$inspect.artifact.minimization.accepted')
            for decision in reduction['removed']:
                parity.fields(decision, dict(dimension=str, ordinal=int, identity=str), '$inspect.artifact.minimization.accepted.removed')


def inspect(root, item, show_choices):
    artifact = root / item['artifact']
    argv = [args.binary, 'inspect', '--json']
    if show_choices:
        argv.append('--choices')
    report = parity.decode(command(*argv, artifact))
    parity.fields(report, dict(schema=str, kind=str, path=str, artifact=dict), '$inspect')
    assert report['schema'] == 'gomad.inspect/v9' and report['kind'] == 'artifact'
    assert report['path'] == str(artifact)
    value = report['artifact']
    parity.fields(value, dict(record=dict, record_sha256=str, failure_signature=str,
                             stored_bytes=int, files=dict, choices=(dict, type(None)),
                             stdout=dict, stderr=dict,
                             world=(dict, type(None)), simulation=(dict, type(None)),
                             minimization=(dict, type(None))), '$inspect.artifact')
    manifest_raw = (artifact / 'manifest.json').read_bytes()
    manifest = parity.decode(manifest_raw)
    assert parity.canonical(manifest) == manifest_raw
    assert value['record'] == manifest['record']
    assert value['files'] == manifest['files']
    assert value['record_sha256'] == manifest['record_sha256'] == item['record_sha256']
    assert value['failure_signature'] == item['failure_signature']
    assert value['stored_bytes'] == len(manifest_raw) + len((artifact / 'storage.json').read_bytes()) + sum(file['bytes'] for file in manifest['files'].values())
    parity.record_contract(value['record'], '$inspect.artifact.record')
    optional_facts(value)
    for name in ('stdout', 'stderr'):
        parity.fields(value[name], dict(full_sha256=str, total_bytes=int, retained_bytes=int,
                                       discarded_bytes=int, truncated=bool), '$inspect.artifact.' + name)
    if value['choices'] is not None:
        choice = value['choices']
        parity.fields(choice, dict(schema=str, profile=str, implementation_sha256=str,
                                  limit=int, payload_bytes=int, sha256=str, terminal_state=str,
                                  exact_replay_available=bool, records=int, decisions=int,
                                  branching_records=int, runnable=int, select_poll=int,
                                  select_result=int, sites=list, replay_decisions=list,
                                  tape_sha256=str), '$inspect.artifact.choices')
        for site in choice['sites']:
            parity.fields(site, dict(fingerprint=str, kind=str, count=int, maximum_alternatives=int, symbol=dict), '$inspect.artifact.choices.sites')
            assert site['symbol'] == {'unavailable': 'no_symbols'}
        for decision in choice['replay_decisions']:
            parity.fields(decision, dict(ordinal=int, kind=str, site_offset=int, site_missing=bool,
                                        alternatives=int, selected=int, symbol=dict), '$inspect.artifact.choices.replay_decisions')
            assert decision['symbol'] == {'unavailable': 'no_symbols'}
    if 'evidence/inspection.json' in value['files']:
        details = parity.decode((artifact / 'evidence/inspection.json').read_bytes())
        for section in ('world', 'simulation', 'minimization'):
            assert value[section] == details[section], section
    # This is the only diagnostic normalization; all hashes and record bytes stay exact.
    report['path'] = '<output>/' + str(artifact.relative_to(args.work)).split('/', 1)[1]
    return report


views = []
for repetition in range(2):
    run_root = args.work / str(repetition)
    run_root.mkdir(mode=0o700)
    view = []
    for name, verb in [('base', 'fixture-execution-bundle'), ('choices', 'fixture-inspect-execution'),
                       ('details', 'fixture-inspect-details')]:
        root = run_root / name
        argv = [args.binary, verb, '--root', root]
        argv += ['--input', args.details_input] if name == 'details' else ['--corpus', corpus]
        result = parity.decode(command(*argv))
        parity.authenticate(root, result)
        view.append(dict(name=name, reports=[inspect(root, item, name == 'choices') for item in result['records']],
                         raw_records=[(root / item['record_file']).read_text() for item in result['records']],
                         raw_ledger=(root / result['ledger_file']).read_text()))
    views.append(view)
parity.check_equal(views[0], views[1])
destination = corpus / 'goldens-inspect' / 'inspect.json'
encoded = parity.canonical(views[0]) + b'\n'
if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_bytes(encoded)
else:
    assert destination.is_file() and destination.read_bytes() == encoded, f'{destination}: golden mismatch'
print('inspect exact view, canonical records and raw ledger integrity agree across fresh repetitions')
