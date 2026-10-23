#!/usr/bin/env python3
"""Execute roots 4-6 using the unchanged intentional IO failure guest."""
import copy
import json
from pathlib import Path
import runpy
import shutil

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal, encode, digest = (common[key] for key in ('Lane', 'equal', 'encode', 'digest'))


def reseal(root, name, raw):
    (root / name).write_bytes(raw)
    manifest = json.loads((root / 'manifest.json').read_bytes())
    manifest['files'][name].update(bytes=len(raw), sha256=digest(raw))
    manifest['record']['streams'][name].update(bytes=len(raw), sha256=digest(raw))
    record = encode(manifest['record'])
    (root / 'record.json').write_bytes(record)
    manifest['files']['record.json'].update(bytes=len(record), sha256=digest(record))
    manifest['record_sha256'] = digest(record)
    (root / 'manifest.json').write_bytes(encode(manifest))


def identity_cases(provenance):
    for name in ('compiler', 'helper', 'engine', 'model', 'profile', 'imports', 'memory', 'limits', 'inputs', 'argv'):
        value = copy.deepcopy(provenance)
        if name in ('compiler', 'helper', 'model'):
            value['inputs']['compiler_sha256' if name == 'compiler' else 'helper'] = (
                'f' if name == 'model' else '0') * 64
        elif name == 'engine':
            value['configuration']['engine'] = 'changed'
        elif name == 'profile':
            value['configuration']['profile'] = 'changed'
        elif name == 'imports':
            value['module_metadata']['imports'] = None
        elif name == 'memory':
            value['module_metadata']['initial_memory_pages'] += 1
        elif name == 'limits':
            value['configuration']['limits']['fuel'] += 1
        elif name == 'inputs':
            value['configuration']['namespace']['stdin'] = list(b'changed')
        else:
            value['configuration']['arguments'] = ['changed']
        yield name, value


def main():
    lane = Lane(__doc__)
    installation = lane.installation('observed', cooperative=False)
    source = common['MODULE'] / 'internal/backends/wasm/testdata/io_failure_module'
    target = ['go-test', './io_failure', '--', '-test.run=^TestDeterministicIOFailure$', '-test.v']
    flags = ['--build-tag=gomad_fixture', '--build-tag=test_dep']
    root, output = lane.explore('intentional-failure', target, source,
        flags, installation=installation, status=(1,), on_failure='first')
    checked = lane.authenticate(root)
    equal((lane.report(output)['failures'], len(checked['executions'])), (1, 1), 'intentional_failure.count')
    artifacts = lane.artifacts(root)
    equal(len(artifacts), 1, 'intentional_failure.artifacts')
    artifact = artifacts[0]
    equal(b'deterministic failure after reading "pending"' in (artifact / 'output/stdout').read_bytes(),
          True, 'intentional_failure.original_fixture')
    record = common['seed']['authenticate_artifact'](artifact)
    equal(record['profile'], 'gomad.wasi-stock-exploratory/v1', 'observed.profile')
    equal('native' in ''.join(record['streams']), False, 'observed.no_native_transcript')
    lane.replay('observed-repeat', artifact, exact=False)
    observed = lane.args.work / 'replay-observed'
    result = lane.run('observed-output', ['replay', '--observed', observed, artifact], (1,))
    equal(b'reproduced=true' in result.stdout, True, 'observed_output.match')
    equal((observed / 'stdout').read_bytes(), (artifact / 'output/stdout').read_bytes(), 'observed_output.stdout')
    equal((observed / 'stderr').read_bytes(), (artifact / 'output/stderr').read_bytes(), 'observed_output.stderr')
    lane.roots.append(dict(root=4, original='TestIntegratedIntentionalFailureObservedReplay'))

    provenance = json.loads((artifact / 'target/provenance.json').read_bytes())
    mutations = []
    for name, value in identity_cases(provenance):
        altered = lane.args.work / ('identity-' + name)
        shutil.copytree(artifact, altered)
        reseal(altered, 'target/provenance.json', encode(value))
        common['seed']['authenticate_artifact'](altered)
        started = lane.args.work / (name + '-guest-started')
        result = lane.run('refuse-' + name, ['replay', altered], (2,),
                          extra={'GOMAD_FIXTURE_GUEST_START_FILE': str(started)})
        equal(b'incompatible replay artifact:' in result.stderr, True, name + '.classified_preflight')
        equal(started.exists(), False, name + '.zero_guest_starts')
        mutations.append(name)
    for name in ('target/module.wasm', 'target/provenance.json', 'evidence/transcript', 'output/stdout'):
        label = name.replace('/', '-')
        altered = lane.args.work / ('corrupt-' + label)
        shutil.copytree(artifact, altered)
        data = bytearray((altered / name).read_bytes())
        equal(bool(data), True, name + '.nonempty')
        data[0] ^= 1
        (altered / name).write_bytes(data)
        started = lane.args.work / (label + '-guest-started')
        result = lane.run('corrupt-' + label, ['replay', altered], (3,),
                          extra={'GOMAD_FIXTURE_GUEST_START_FILE': str(started)})
        equal(b'runner_failure: artifact integrity:' in result.stderr, True, name + '.integrity_classification')
        equal(b'unsupported artifact storage' in result.stderr, False, name + '.not_lineage')
        equal(started.exists(), False, name + '.zero_guest_starts')
    missing = lane.args.work / 'missing-provider.json'
    selected = json.loads(installation.read_bytes())
    selected['helper_sha256'] = '0' * 64
    missing.write_bytes(encode(selected))
    # Replay selects its linked provider from the artifact. A foreign provider
    # binding must refuse; it cannot select the removed native fallback.
    altered = lane.args.work / 'missing-provider'
    shutil.copytree(artifact, altered)
    inputs = json.loads((altered / 'evidence/replay.json').read_bytes())
    inputs['helper_sha256'] = ''
    reseal(altered, 'evidence/replay.json', encode(inputs))
    started = lane.args.work / 'missing-provider-started'
    lane.run('missing-provider', ['replay', altered], (2,),
             extra={'GOMAD_FIXTURE_GUEST_START_FILE': str(started)})
    equal(started.exists(), False, 'missing_provider.zero_guest_starts')
    lane.roots.append(dict(root=5, original='TestReplayRejectsChangedArtifactIdentityBeforeGuest',
                           resealed=mutations, corruptions=4,
                           merged_model_binding='inputs.helper seals the actual linked model implementation'))

    execution_log = lane.args.work / 'observed-executions.jsonl'
    partial, output = lane.explore('observed-cancel', target, source, flags, seeds='7-8',
        installation=installation, status=(3,),
        extra={'GOMAD_FIXTURE_BACKEND_CANCEL': 'observed-seed-8',
               'GOMAD_FIXTURE_BACKEND_EXECUTIONS': str(execution_log)})
    equal(json.loads(output.stdout.splitlines()[-1])['classification'], 'cancelled', 'cancel.classification')
    equal(execution_log.read_text().splitlines(), ['0'], 'cancel.seed_8_held_before_guest')
    checked = lane.authenticate(partial, complete=False)
    equal([(row['ordinal'], row['seed']) for row in checked['executions']], [(0, 7)], 'cancel.first_only')
    equal(checked['executions'][0]['record']['outcome']['class'], 'target_failure', 'cancel.one_failure')
    resumed = lane.run('observed-resume', ['resume', '--json', partial], (1,),
        extra={'GOMAD_FIXTURE_BACKEND_EXECUTIONS': str(execution_log)})
    equal(execution_log.read_text().splitlines(), ['0', '1'], 'resume.exactly_once_committed_guests')
    report = lane.report(resumed)
    equal((report['attempted'], report['failures'], report['stop_reason']),
          (2, 2, 'seeds_exhausted'), 'observed_resume.result')
    checked = lane.authenticate(partial)
    equal([(row['ordinal'], row['seed']) for row in checked['executions']], [(0, 7), (1, 8)],
          'observed_resume.backend_ordinals')
    for execution in checked['executions']:
        equal(execution['record']['profile'], record['profile'], 'observed_resume.backend_profile')
        for name in ('target', 'implementation'):
            equal(execution['record']['identity'][name], record['identity'][name],
                  'observed_resume.backend_' + name)
    lane.roots.append(dict(root=6, original='TestObservedCampaignResumeRetainsBackendAndOrdinals'))
    lane.finish()


if __name__ == '__main__':
    main()
