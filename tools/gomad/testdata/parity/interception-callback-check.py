#!/usr/bin/env python3
"""Refuse a real launcher's stdout after its marker; authenticate partial evidence.

Uses the compiled-Go callback input collected by interception-prepare.py.
Consumer failure is external to the guest and is not claimed to replay exactly.
"""
import argparse
import os
from pathlib import Path
import selectors
import signal
import subprocess
import time

from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

loader = SourceFileLoader('interception_callback_check', str(Path(__file__).with_name('interception-check.py')))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)

MARKER = b'gomad callback marker\n'


def validate_partial(record, delivered, retained):
    check.equal((record['outcome']['class'], record['outcome']['termination'], check.mapped_status(record['outcome'])),
                ('runner_failure', 'infrastructure', 3), 'callback.infrastructure_status')
    if not delivered or not retained.startswith(delivered) or len(retained) <= len(delivered):
        raise ValueError('callback requires delivered prefix plus retained failed delivery bytes')
    if not any(word in record['outcome']['reason'].lower() for word in ['pipe', 'output']):
        raise ValueError('callback failure reason must identify the output consumer')


def refuse_stdout(runner, argv, extra):
    argv = list(map(str, argv))
    process = subprocess.Popen(argv, env=dict(runner.environment, **extra), stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, start_new_session=True, bufsize=0)
    deadline = time.monotonic() + runner.args.timeout
    delivered, stderr = bytearray(), bytearray()
    expired = False
    try:
        with selectors.DefaultSelector() as selector:
            selector.register(process.stdout, selectors.EVENT_READ)
            while not delivered.endswith(b'\n'):
                remaining = deadline - time.monotonic()
                if remaining <= 0 or not selector.select(remaining):
                    raise subprocess.TimeoutExpired(argv, runner.args.timeout)
                byte = os.read(process.stdout.fileno(), 1)
                if not byte:
                    raise ValueError('compiled Go guest exited before callback marker')
                delivered.extend(byte)
                if len(delivered) > 1024:
                    raise ValueError('compiled Go callback marker exceeds bound')
        check.equal(bytes(delivered), MARKER, 'callback.actual_marker')
        process.stdout.close()
        process.stdout = None
        _, raw_stderr = process.communicate(timeout=max(0, deadline - time.monotonic()))
        stderr.extend(raw_stderr)
    except BaseException as error:
        expired = isinstance(error, subprocess.TimeoutExpired)
        if process.poll() is None:
            os.killpg(process.pid, signal.SIGTERM)
        try:
            _, raw_stderr = process.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            _, raw_stderr = process.communicate()
        stderr.extend(raw_stderr)
        raise
    finally:
        runner.save('closed-stdout', argv, process.returncode, bytes(delivered), bytes(stderr), expired)
    check.equal(process.returncode, 3, 'callback.process_status')
    return bytes(delivered), bytes(stderr)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'installation', 'inputs', 'work']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--timeout', type=int, default=600)
    args = parser.parse_args()
    if not 1 <= args.timeout <= 600:
        parser.error('timeout must preserve the original overall bound, 1..600 seconds')
    for name in ['binary', 'installation', 'inputs', 'work']:
        setattr(args, name, getattr(args, name).resolve())
    case = check.document(args.inputs)['callback']
    check.equal(case['args'], ['stream'], 'callback.registered_fixture_mode')
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    runner = check.Runner(args)
    root = args.work / 'artifacts'
    delivered, stderr = refuse_stdout(runner, [args.binary, '--installation', args.installation, 'run-guest',
                                              '--artifacts', root, '--seeds', '1,3', case['module'], *case['args']],
                                      dict(GOMAD_PREPARATION_RECEIPT=case['receipt'],
                                           GOMAD_PREPARATION_RECEIPT_SHA256=case['receipt_sha256']))
    preparation = check.load('interception_callback_preparation', check.HERE / 'interception-prepare.py')
    admitted = preparation.retained_input(root)
    check.equal(Path(admitted['module']).read_bytes(), Path(case['module']).read_bytes(), 'callback.actual_module')
    executed, prepared = check.campaign_roles([(path, check.campaign.canonical_file(path / 'plan.json'))
                                               for path in (root / 'campaigns').glob('campaign-*')])
    observed = check.campaign.authenticate_campaign(executed)
    check.equal((observed['plan']['selection'], len(observed['executions'])), ('1,3', 1), 'callback.later_unexecuted')
    execution = observed['executions'][0]
    check.equal((execution['ordinal'], execution['seed']), (0, 1), 'callback.first_attempt')
    artifact = executed / execution['artifact']
    record = check.campaign.authenticate_artifact(artifact)
    raw_module = Path(case['module']).read_bytes()
    check.bind_module_input(record, check.campaign.canonical_file(artifact / 'manifest.json'),
                            check.parity.digest(raw_module), len(raw_module))
    sealed = check.authenticate_origin(artifact, record, observed['plan']['origin'])
    for campaign_root in [prepared, executed]:
        plan = check.campaign.canonical_file(campaign_root / 'plan.json')
        payload = campaign_root / plan['prepared_artifact']
        prepared_record = check.campaign.authenticate_artifact(payload)
        check.equal(check.authenticate_origin(payload, prepared_record, plan['origin']), sealed, 'callback.same_origin')
    retained = (artifact / 'output/stdout').read_bytes()
    validate_partial(record, delivered, retained)
    runner.run('inspect-partial', [args.binary, 'inspect', '--json', artifact], 0)
    _, unavailable = runner.run('inspect-partial-choices', [args.binary, 'inspect', '--json', '--choices', artifact], 2)
    check.equal(b'artifact has no choice trace' in unavailable, True, 'callback.no_complete_choice_trace')
    check.equal(stderr.splitlines()[-1], b'gomad run-guest: no replayable artifact was retained',
                'callback.no_invented_replay')
    _, refused = runner.run('verify-partial', [args.binary, 'replay', '--verify-only', artifact], 2)
    check.equal(b'incompatible replay artifact: WASM artifact exact choice identity mismatch' in refused,
                True, 'callback.partial_replay_preflight_refusal')
    (args.work / 'observation.json').write_bytes(check.parity.canonical(dict(
        schema='gomad.interception-callback-observation/v1', status=3, artifact=str(artifact),
        delivered_bytes=len(delivered), delivered_sha256=check.parity.digest(delivered),
        retained_bytes=len(retained), retained_sha256=check.parity.digest(retained),
        module_sha256=record['identity']['target'], record_sha256=check.parity.digest(check.parity.canonical(record)),
        origin=sealed, attempted=1, unexecuted=1, replayable=False, exact_replay_claimed=False)) + b'\n')
    print('compiled Go closed-consumer partial evidence authenticated')


if __name__ == '__main__':
    main()
