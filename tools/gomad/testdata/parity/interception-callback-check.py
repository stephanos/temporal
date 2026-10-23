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
import stat
import time

from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

loader = SourceFileLoader('interception_callback_check', str(Path(__file__).with_name('interception-check.py')))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)

MARKER = b'gomad callback marker\n'


def private_json(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > 8 << 20:
        raise ValueError('unsafe callback process input: ' + str(path))
    return check.document(path)


class Runner(check.Runner):
    def __init__(self, args, pins=None, read_only=False, started_ns=None):
        super().__init__(args)
        self.read_only = read_only
        self.pids = set()
        pins = pins or {}
        if read_only:
            self.admission = private_json(args.work / 'admission.json')
            check.equal(self.admission['files'], pins, 'callback.same_static_inputs')
            self.environment = self.admission['environment']
        else:
            started = time.monotonic_ns() if started_ns is None else started_ns
            self.admission = dict(schema='gomad.interception-callback-admission/v1', pid=os.getpid(),
                cwd=str(Path.cwd()), started_ns=started, deadline_ns=started+args.timeout*1_000_000_000,
                environment=self.environment, files=pins)
            (args.work / 'admission.json').write_bytes(check.parity.canonical(self.admission))
        check.equal(self.admission['schema'], 'gomad.interception-callback-admission/v1', 'callback.admission_schema')
        if not (type(self.admission['pid']) is int and self.admission['pid'] > 0
                and type(self.admission['started_ns']) is int and type(self.admission['deadline_ns']) is int
                and 0 < self.admission['deadline_ns']-self.admission['started_ns'] <= 600_000_000_000):
            raise ValueError('callback original overall admission changed')
        if any(key.startswith('GOMAD_') for key in self.environment):
            raise ValueError('callback inherited uncontrolled Gomad environment')
        self.deadline = self.admission['deadline_ns'] / 1e9
        self.check_inputs()

    def check_inputs(self):
        for name, digest in self.admission['files'].items():
            path = Path(name)
            if not path.is_absolute() or path.is_symlink() or not path.is_file():
                raise ValueError('callback static input missing or unsafe: ' + name)
            check.equal(check.parity.digest(path.read_bytes()), digest, 'callback.current_static_input.' + name)

    def remaining(self):
        value = self.deadline-time.monotonic()
        if value <= 0:
            raise ValueError('callback original overall deadline expired before spawn; not retried')
        return value

    def save_process(self, name, argv, process, started, environment, stdout, stderr, expired=False, close=None, observation='completed'):
        self.save(name, argv, process.returncode, stdout, stderr, expired)
        row = dict(schema='gomad.interception-callback-process/v1', argv=argv, cwd=self.admission['cwd'],
            environment=environment, pid=process.pid, parent_pid=os.getpid(), status=process.returncode,
            observation=observation, clock='CLOCK_MONOTONIC', started_ns=started, ended_ns=time.monotonic_ns(),
            deadline_ns=self.admission['deadline_ns'], timed_out=expired, reaped=process.poll() is not None,
            stdout_sha256=check.parity.digest(stdout), stderr_sha256=check.parity.digest(stderr), close=close)
        path = self.args.work / (name + '.process.json')
        with path.open('xb') as output:
            output.write(check.parity.canonical(row))
        return row

    def read_process(self, name, argv, status, extra=None, closed=False):
        row = private_json(self.args.work / (name + '.process.json'))
        check.equal(row['schema'], 'gomad.interception-callback-process/v1', 'callback.process_schema')
        check.equal(row['argv'], list(map(str, argv)), 'callback.process_argv')
        check.equal(row['cwd'], self.admission['cwd'], 'callback.process_cwd')
        check.equal(row['environment'], dict(self.environment, **(extra or {})), 'callback.process_environment')
        if not (type(row['pid']) is int and row['pid'] > 0 and row['pid'] not in self.pids
                and type(row['parent_pid']) is int and row['parent_pid'] == self.admission['pid']
                and type(row['status']) is int and row['status'] == status and row['reaped'] is True
                and row['observation'] == 'completed' and row['timed_out'] is False and row['clock'] == 'CLOCK_MONOTONIC'
                and row['deadline_ns'] == self.admission['deadline_ns']
                and type(row['started_ns']) is int and type(row['ended_ns']) is int
                and self.admission['started_ns'] <= row['started_ns'] < row['ended_ns'] <= row['deadline_ns']):
            raise ValueError('callback actual process identity, lifecycle or whole deadline changed')
        stdout, stderr = ((self.args.work / (name + '.' + stream)).read_bytes() for stream in ('stdout', 'stderr'))
        check.equal([check.parity.digest(stdout), check.parity.digest(stderr)],
                    [row['stdout_sha256'], row['stderr_sha256']], 'callback.raw_process_capture')
        if closed:
            close = row['close']
            if not (type(close['fd']) is int and close['fd'] >= 0 and close['closed'] is True and close['child_running'] is True
                    and row['started_ns'] <= close['marker_observed_ns'] <= close['closed_ns'] <= row['ended_ns']):
                raise ValueError('callback marker/live child/closed stdout descriptor witness changed')
            check.equal(stdout, MARKER, 'callback.exact_delivered_marker')
        else:
            check.equal(row['close'], None, 'callback.reader_verb_has_no_close_witness')
        self.pids.add(row['pid'])
        return stdout, stderr

    def run(self, name, argv, status, extra=None):
        argv = list(map(str, argv))
        if self.read_only:
            return self.read_process(name, argv, status, extra)
        remaining = self.remaining()
        started = time.monotonic_ns()
        environment = dict(self.environment, **(extra or {}))
        process = subprocess.Popen(argv, cwd=self.admission['cwd'], env=environment,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
        expired = False
        stdout, stderr = b'', b''
        observation = 'completed'
        try:
            stdout, stderr = process.communicate(timeout=remaining)
        except BaseException as error:
            expired = isinstance(error, subprocess.TimeoutExpired)
            observation = 'inconclusive'
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGTERM)
            try:
                stdout, stderr = process.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                stdout, stderr = process.communicate()
            raise
        finally:
            self.save_process(name, argv, process, started, environment, stdout, stderr, expired,
                              observation=observation)
        return self.read_process(name, argv, status, extra)


def bind_launch(launch, process, receipt_sha256):
    check.equal(launch['process_id'], process['pid'], 'callback.actual_launcher_pid')
    check.equal(launch['parent_process_id'], process['parent_pid'], 'callback.actual_launcher_parent_pid')
    check.equal(launch['preparation_receipt_sha256'], receipt_sha256, 'callback.actual_preparation_receipt')


def validate_partial(record, delivered, retained):
    check.equal((record['outcome']['class'], record['outcome']['termination'], check.mapped_status(record['outcome'])),
                ('runner_failure', 'infrastructure', 3), 'callback.infrastructure_status')
    if not delivered or not retained.startswith(delivered) or len(retained) <= len(delivered):
        raise ValueError('callback requires delivered prefix plus retained failed delivery bytes')
    if not any(word in record['outcome']['reason'].lower() for word in ['pipe', 'output']):
        raise ValueError('callback failure reason must identify the output consumer')


def refuse_stdout(runner, argv, extra):
    argv = list(map(str, argv))
    if runner.read_only:
        return runner.read_process('closed-stdout', argv, 3, extra, closed=True)
    runner.remaining()
    started = time.monotonic_ns()
    environment = dict(runner.environment, **extra)
    process = subprocess.Popen(argv, cwd=runner.admission['cwd'], env=environment, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, start_new_session=True, bufsize=0)
    deadline = runner.deadline
    close = None
    delivered, stderr = bytearray(), bytearray()
    expired = False
    observation = 'completed'
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
        observed = time.monotonic_ns()
        descriptor = process.stdout.fileno()
        running = process.poll() is None
        process.stdout.close()
        close = dict(fd=descriptor, marker_observed_ns=observed, closed_ns=time.monotonic_ns(),
                     closed=process.stdout.closed, child_running=running)
        process.stdout = None
        _, raw_stderr = process.communicate(timeout=max(0, deadline - time.monotonic()))
        stderr.extend(raw_stderr)
    except BaseException as error:
        observation = 'inconclusive'
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
        runner.save_process('closed-stdout', argv, process, started, environment, bytes(delivered), bytes(stderr), expired, close, observation)
    return runner.read_process('closed-stdout', argv, 3, extra, closed=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'installation', 'inputs', 'work']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--timeout', type=int, default=600)
    parser.add_argument('--read-only', action='store_true')
    args = parser.parse_args()
    started_ns = time.monotonic_ns()
    if not 1 <= args.timeout <= 600:
        parser.error('timeout must preserve the original overall bound, 1..600 seconds')
    for name in ['binary', 'installation', 'inputs', 'work']:
        setattr(args, name, getattr(args, name).resolve())
    case = check.document(args.inputs)['callback']
    check.equal(case['args'], ['stream'], 'callback.registered_fixture_mode')
    if not args.read_only:
        args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    if args.work.resolve() != args.work or stat.S_IMODE(args.work.stat().st_mode) != 0o700:
        raise ValueError('callback requires its private canonical proof directory')
    sources = [args.binary,args.installation,args.inputs,Path(case['module']),Path(case['receipt'])]
    sources += list(check.HERE.glob('*.py')) + [Path(check.parity.__file__).resolve()]
    pins = {str(path):check.parity.digest(path.read_bytes()) for path in sources}
    runner = Runner(args, pins, args.read_only, started_ns)
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
    process = private_json(args.work / 'closed-stdout.process.json')
    bind_launch(sealed['launch'], process, case['receipt_sha256'])
    provenance = check.campaign.canonical_file(artifact / 'target/provenance.json')
    check.equal((provenance['inputs']['kind'],provenance['inputs']['source'],provenance['inputs']['arguments']),
                ('go-run','./cmd/gomad/testdata/seedfree',['stream']), 'callback.original_compiled_go_recipe')
    for campaign_root in [prepared, executed]:
        plan = check.campaign.canonical_file(campaign_root / 'plan.json')
        payload = campaign_root / plan['prepared_artifact']
        prepared_record = check.campaign.authenticate_artifact(payload)
        check.equal(check.authenticate_origin(payload, prepared_record, plan['origin']), sealed, 'callback.same_origin')
    retained = (artifact / 'output/stdout').read_bytes()
    validate_partial(record, delivered, retained)
    inspected, _ = runner.run('inspect-partial', [args.binary, 'inspect', '--json', artifact], 0)
    receiving = check.load('callback_independent_inspect', check.HERE / 'interception-receive.py')
    receiving.authenticate_inspect(inspected, artifact)
    _, unavailable = runner.run('inspect-partial-choices', [args.binary, 'inspect', '--json', '--choices', artifact], 2)
    check.equal(b'artifact has no choice trace' in unavailable, True, 'callback.no_complete_choice_trace')
    check.equal(stderr.splitlines()[-1], b'gomad run-guest: no replayable artifact was retained',
                'callback.no_invented_replay')
    _, refused = runner.run('verify-partial', [args.binary, 'replay', '--verify-only', artifact], 2)
    check.equal(b'incompatible replay artifact: WASM artifact exact choice identity mismatch' in refused,
                True, 'callback.partial_replay_preflight_refusal')
    observation = dict(
        schema='gomad.interception-callback-observation/v1', status=3, artifact=str(artifact),
        delivered_bytes=len(delivered), delivered_sha256=check.parity.digest(delivered),
        retained_bytes=len(retained), retained_sha256=check.parity.digest(retained),
        module_sha256=record['identity']['target'], record_sha256=check.parity.digest(check.parity.canonical(record)),
        origin=sealed, attempted=1, unexecuted=1, replayable=False, exact_replay_claimed=False)
    runner.check_inputs()
    if not args.read_only:
        runner.remaining()
        (args.work / 'observation.json').write_bytes(check.parity.canonical(observation) + b'\n')
    print('compiled Go closed-consumer partial evidence authenticated')


if __name__ == '__main__':
    main()
