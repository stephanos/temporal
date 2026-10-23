#!/usr/bin/env python3
"""Ordered fn222 gates on one frozen Rust release candidate (3600 seconds total).

Inputs are complete qualification configurations, never edited installations.
The linked fn222-support receiver authenticates retained evidence independently.
A pending/failed stage stops the sequence and never invents later process exits.
"""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import selectors
import signal
import stat
import subprocess
import sys
import time

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
TAGS = ['disable_grpc_modules', 'gomad_wasm_slim', 'sqlite3_dotlk', 'test_dep']
CELLS = {'smoke': ('onebox/unchanged-workflow-type-smoke', './tests',
                   '^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$'),
         'frontend': ('onebox/frontend-system-info', './tests/gomadfunctional', '^TestFrontendSystemInfo$')}
GATE_NAMES = ['unchanged-source-preparation', 'measured-pass-replay-success',
              'measured-pass-replay-canary', 'storage-transport-inspection']
ENVIRONMENT = ['TEMPORAL_ROOT=/workspace', 'TMPDIR=/tmp', 'TEMPORAL_TEST_LOG_LEVEL=warn']


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False, allow_nan=False).encode() + b'\n'


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def read(path, maximum=32 << 20):
    with Path(path).open('rb') as stream:
        raw = stream.read(maximum + 1)
    if len(raw) > maximum:
        raise ValueError(f'input exceeds {maximum} bytes: {path}')
    return raw


def qualification_spec_witness(command):
    index = 3 if len(command) > 1 and command[1] == '--installation' else 1
    if len(command) <= index or command[index] != 'qualify':
        return None
    if len(command) != index + 3 or command[index + 1] != '--spec':
        raise ValueError('onebox measured command requires the exact qualify --spec FILE vector')
    text = command[index + 2]
    path = Path(text)
    if not path.is_absolute() or any(part in ('', '.', '..') for part in text.split('/')[1:]):
        raise ValueError('qualification spec requires a clean absolute path')
    if any(parent.is_symlink() for parent in path.parents):
        raise ValueError('qualification spec has a symlinked parent')
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, 'rb') as stream:
        if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
            raise ValueError('qualification spec requires a regular file')
        raw = stream.read((16 << 20) + 1)
    if not raw or len(raw) > 16 << 20:
        raise ValueError('qualification spec exceeds its nonempty 16MiB bound')
    return dict(path=text, sha256=digest(raw), bytes_base64=base64.b64encode(raw).decode('ascii'))


def write(path, value):
    with Path(path).open('xb') as stream:
        os.chmod(path, 0o600)
        stream.write(encode(value))
        stream.flush()
        os.fsync(stream.fileno())


def decoded(path):
    return json.loads(read(path))


def ordered(perform):
    for name in ['smoke-preparation', 'smoke-success', 'smoke-canary', 'inspection',
                 'frontend-preparation', 'frontend-success', 'frontend-canary', 'publication']:
        perform(name)


def measured_status(mode, cli_exit, report_exit, admission_exit):
    if (cli_exit, report_exit, admission_exit) != (1 if mode == 'canary' else 0, 0, 0):
        raise ValueError(f'{mode}: raw CLI/report/admission exits {cli_exit}/{report_exit}/{admission_exit}')


def complete(rows):
    expected = [(CELLS[cell][0], name) for cell in CELLS for name in GATE_NAMES]
    if [(row['cell'], row['name']) for row in rows] != expected or any(row['exit'] != 0 for row in rows):
        raise ValueError('support publication requires all eight ordered zero-exit gates')


def target_environment(cell):
    cwd = '/workspace/tests' if cell == 'smoke' else '/workspace/tests/gomadfunctional'
    return ['PWD=' + cwd, *ENVIRONMENT[:2], 'TZ=UTC', ENVIRONMENT[2]]


def target_args(cell, mode):
    return ['-test.v', '-test.count=1', '-test.timeout=72h',
            '-test.run=' + ('[' if mode == 'canary' else CELLS[cell][2]),
            '-persistenceType=sql', '-persistenceDriver=sqlite']


def qualification_spec(cell, mode, root, repo, source):
    return dict(Manifest=dict(Provisional=False, ID=CELLS[cell][0].split('/')[-1], Workload='onebox', Cell=mode,
                IntegratedSourceSHA256='sha256:' + digest(read(source)) if source.exists() else '',
                IntegratedSourceManifestPath=str(source), Seed=7, Repeat=2),
        ArtifactRoot=str(root), Campaign=dict(Target=dict(Backend='wasm', Kind='go-test',
            Source=CELLS[cell][1], WorkingDir=str(repo), Args=target_args(cell, mode), BuildTags=TAGS),
            ExecutionTimeout=900_000_000_000, OverallTimeout=1_800_000_000_000,
            TerminateGrace=1_000_000_000, OnFailure='all', FailureBudget=1,
            OutputLimit=256 << 20, WorldTransitionLimit=32 << 20, ChoiceTraceLimit=64 << 20,
            SuccessBytesLimit=2 << 30, Environment=target_environment(cell), Artifacts=str(root / 'artifacts')))


def kill_session(session):
    stopped = set()
    while True:
        members = set()
        for token in subprocess.check_output(['/bin/ps', '-axo', 'pid='], timeout=5).split():
            pid = int(token)
            try:
                if os.getsid(pid) == session:
                    members.add(pid)
                    os.kill(pid, signal.SIGSTOP)
            except ProcessLookupError:
                pass
        if members <= stopped:
            break
        stopped.update(members)
    for pid in sorted(stopped, key=lambda value: value == session):
        try:
            if os.getsid(pid) == session:
                os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass


class Commands:
    def __init__(self, root, seconds=3600):
        self.root = root
        self.deadline = time.monotonic() + seconds
        self.receipts = []
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith('GOMAD_') and key != 'GOROOT'}
        self.environment.update({key: value for key, value in os.environ.items() if key in
            ['GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', 'GOMAD_WASM_TEMPORAL_BUILD_CACHE',
             'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES']})
        self.environment.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2')

    def run(self, name, command, mandatory=False, seconds=3600, private=None):
        command = list(map(str, command))
        specification = qualification_spec_witness(command)
        executable = Path(command[0]).resolve()
        executable_hash = digest(read(executable, 512 << 20))
        remaining = min(seconds, self.deadline - time.monotonic())
        if remaining <= 0:
            raise ValueError('onebox 3600-second outer bound exhausted')
        environment = dict(self.environment)
        if mandatory:
            environment['GOMAD_WASM_QUALIFY_WORKLOADS'] = '1'
        paths = {stream: self.root / (name + '.' + stream) for stream in ['stdout', 'stderr']}
        started = time.monotonic_ns()
        started_nanos = time.time_ns()
        with paths['stdout'].open('xb') as stdout, paths['stderr'].open('xb') as stderr:
            for path in paths.values():
                path.chmod(0o600)
            launch = CORPUS.parents[1] / 'rust/driver/launch.py'
            read_gate, write_gate = os.pipe()
            blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
            try:
                process = subprocess.Popen([sys.executable, str(launch), str(read_gate), *command],
                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=environment,
                    start_new_session=True, pass_fds=(read_gate,))
                registration = dict(pid=process.pid, stage_pid=os.getpid(), active=True,
                                    argv=command, start_ns=started_nanos)
                process_path = self.root / (name + '.process.json')
                write(process_path, registration)
                os.write(write_gate, b'1')
            finally:
                os.close(read_gate)
                os.close(write_gate)
                signal.pthread_sigmask(signal.SIG_SETMASK, blocked)
            streams = selectors.DefaultSelector()
            streams.register(process.stdout, selectors.EVENT_READ, (stdout, 'stdout'))
            streams.register(process.stderr, selectors.EVENT_READ, (stderr, 'stderr'))
            sizes = dict(stdout=0, stderr=0)
            failure = None
            deadline = time.monotonic() + remaining
            try:
                while streams.get_map():
                    if time.monotonic() >= deadline:
                        failure = 'bounded command timeout'
                        break
                    for key, _ in streams.select(min(0.1, max(0, deadline - time.monotonic()))):
                        raw = os.read(key.fileobj.fileno(), 65536)
                        if not raw:
                            streams.unregister(key.fileobj)
                            continue
                        destination, stream = key.data
                        sizes[stream] += len(raw)
                        if sizes[stream] > 64 << 20:
                            failure = 'command output exceeds 64MiB'
                            break
                        destination.write(raw)
                    if failure:
                        break
            finally:
                streams.close()
                if failure:
                    kill_session(process.pid)
                try:
                    process.wait(timeout=max(0.001, deadline - time.monotonic()))
                except subprocess.TimeoutExpired:
                    failure = 'bounded command timeout'
                    kill_session(process.pid)
                    process.wait()
                process.stdout.close()
                process.stderr.close()
        registration.update(active=False, exit=process.returncode, observation='completed', end_ns=time.time_ns())
        replacement = process_path.with_suffix('.complete')
        write(replacement, registration)
        os.replace(replacement, process_path)
        row = dict(schema='gomad.rust-onebox-command/v1', command=command,
                   executable=dict(path=str(executable), sha256=executable_hash),
                   operator_process_id=process.pid, exit=process.returncode, cwd=os.getcwd(),
                   started_nanos=started_nanos, ended_nanos=time.time_ns(),
                   elapsed_nanos=time.monotonic_ns() - started, mandatory_workloads=mandatory,
                   environment={key: environment.get(key) for key in ['PATH', 'GOMAD_WASM_INSTALLATION',
                     'GOMAD_STOCK_GO', 'GOMAD_WASM_RUNTIME_ROOT', 'GOMAD_WASM_QUALIFICATION_GUEST_PROFILE',
                     'GOMAD_WASM_TEMPORAL_BUILD_CACHE', 'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES',
                     'GOMAD_WASM_QUALIFY_WORKLOADS']},
                   environment_sha256=digest(encode(environment).removesuffix(b'\n')))
        if specification is not None:
            row['qualification_spec'] = specification
            try:
                if qualification_spec_witness(command) != specification:
                    raise ValueError('qualification spec bytes changed during measured process')
            except (ValueError, OSError) as error:
                failure = 'qualification spec witness refused: ' + str(error)
        for stream, path in paths.items():
            raw = read(path, 64 << 20)
            row[stream] = dict(path=str(path), sha256=digest(raw), bytes=len(raw))
        if private is not None and row['exit'] in (0, 1) and failure is None:
            try:
                row['private_observations'] = private[0].observation_module().from_command(
                    private[1], process.pid, command, read(paths['stdout'], 64 << 20))
            except (ValueError, OSError, KeyError, TypeError) as error:
                failure = 'private preparation capture refused: ' + str(error)
                row['private_observations_error'] = str(error)
        receipt = self.root / (name + '.command.json')
        write(receipt, row)
        self.receipts.append(dict(path=str(receipt), sha256=digest(read(receipt))))
        if failure:
            raise ValueError(f'{name}: {failure}; raw exit {row["exit"]}')
        return row


def require_zero(row):
    if row['exit'] != 0:
        raise ValueError(f'command exit {row["exit"]}: {row["command"]}')
    return row


class Onebox:
    def __init__(self, args):
        self.args = args
        if os.environ.get('GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', '') not in ('', 'slim'):
            raise ValueError('fn222 fixes the qualification guest profile to slim')
        self.root = args.root
        self.commands = Commands(self.root)
        self.private = None
        directory = args.repo / 'tools/gomad/rust/workload_tools'
        if (directory / 'producer.py').is_file():
            sys.path.insert(0, str(directory))
            import producer
            producer.owned_root(self.root)
        if getattr(args, 'private_observation_config', None) is not None:
            directory = args.repo / 'tools/gomad/rust/workload_tools'
            sys.path.insert(0, str(directory))
            import producer
            context = producer.private_configuration(args.private_observation_config, args.binary,
                                                       args.repo, self.root, args.source_manifest)
            producer.private_admission(context, self.root)
            self.private = (producer, context)
        if self.private is not None:
            producer, _ = self.private
            import workloads
            profile = os.environ.get('GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', '')
            workloads.guest_build_tags(profile)
            if args.cache is None:
                raise ValueError('private onebox context requires its frozen --cache')
            override = os.environ.get('GOMAD_WASM_TEMPORAL_BUILD_CACHE', '')
            requested_cache = producer.qualification_build_cache(self.root, override or str(args.cache))
            if override and requested_cache != args.cache:
                raise ValueError('private compiler-cache override differs from frozen --cache')
            minimum = os.environ.get('GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES', '')
            memory = producer.memory_admission(sys.platform, Path('/proc/meminfo'), minimum,
                                               retained_raw=self.root / 'host-memory.raw')
            write(self.root / 'qualification-controls.json', dict(guest_profile=profile,
                requested_build_cache=str(requested_cache), build_cache=str(requested_cache.resolve()),
                memory_minimum=minimum, memory_platform=sys.platform, memory_path='/proc/meminfo', memory=memory))
        for cell, package in [('smoke', './tests'), ('frontend', './tests/gomadfunctional')]:
            if getattr(args, cell + '_config', None) is None:
                path = self.root / (cell + '-policy.json')
                require_zero(self.commands.run(cell + '-policy', [args.support, 'temporal-policy',
                             args.repo, package, 'qualification', path]))
                setattr(args, cell + '_config', path)
        if getattr(args, 'inspection_config', None) is None:
            args.inspection_config = getattr(args, 'captured_config', None)
        if args.inspection_config is None:
            path = self.root / 'inspection-policy.json'
            require_zero(self.commands.run('inspection-policy', [args.support, 'temporal-policy',
                         args.repo, '.', 'inspection', path]))
            args.inspection_config = path
        self.gates = {cell: [] for cell in CELLS}
        self.results = []
        self.process_receipts = []
        self.preparations = {}
        self.installations = {}
        self.source = getattr(args, 'source_manifest', None) or self.root / 'integrated-source.sha256'
        files = subprocess.check_output(['git', '-C', str(args.repo), 'ls-files', '-z']).split(b'\0')
        self.inputs = {str(args.repo / os.fsdecode(name)): digest(read(args.repo / os.fsdecode(name), 512 << 20))
                       for name in sorted(files) if name}
        self.inputs.update({str(path): digest(read(path, 512 << 20)) for path in
                           [args.binary, args.support, args.compiler, Path(sys.executable).resolve(), args.smoke_config, args.frontend_config,
                            args.inspection_config, Path(__file__).resolve(), args.inspection]})
        if getattr(args, 'source_manifest', None) is None:
            with self.source.open('xb') as stream:
                stream.write(''.join(f'{sha}  {path}\n' for path, sha in sorted(self.inputs.items())).encode())
            self.source.chmod(0o600)
        self.inputs[str(self.source)] = digest(read(self.source))
        if self.private is not None:
            self.inputs[str(args.private_observation_config)] = digest(read(args.private_observation_config))
        write(self.root / 'frozen-inputs.json', self.inputs)

    def frozen(self):
        if time.monotonic() >= self.commands.deadline:
            raise ValueError('onebox 3600-second outer bound exhausted')
        for path, expected in self.inputs.items():
            if digest(read(path, 512 << 20)) != expected:
                raise ValueError(f'frozen source/candidate changed: {path}')

    def gate(self, cell, name, command, commands, **fields):
        log = self.root / f'{cell}-{name}.gate.json'
        write(log, dict(schema='gomad.rust-onebox-stage/v1', command=command, exit=0, commands=commands))
        self.gates[cell].append(dict(cell=CELLS[cell][0], name=name, command=list(map(str, command)), exit=0,
                                     log=str(log), log_sha256=digest(read(log)), **fields))

    def prepare(self, cell):
        args = self.args
        preparation = self.root / (cell + '-preparation')
        preparation.mkdir(mode=0o700)
        installation = preparation / 'installation.json'
        installed = require_zero(self.commands.run(cell + '-install', [args.binary, 'install',
            '--compiler', args.compiler, '--runtime-root', args.runtime_root,
            '--cache', getattr(args, 'cache', None) or self.root / 'preparation-cache',
            '--compile-cache', self.root / 'compile-cache',
            '--profile', 'gomad.wasi-cooperative/v1', '--config', getattr(args, cell + '_config'),
            '--memory-bytes', str(4 << 30), '--fuel', str(1 << 50), '--output', installation]))
        self.installations[cell] = installation
        command = [args.binary, '--installation', installation, 'plan', '--json', '--output', preparation / 'plan.json',
                   '--seeds=7', '--parallel=1', '--choices', '--coverage=choice', '--choice-bytes=67108864',
                   '--execution-timeout=15m', '--overall-timeout=30m', '--terminate-grace=1s',
                   '--on-failure=all', '--failure-budget=1', '--output-limit=268435456',
                   '--world-transition-limit=33554432', '--keep-successes=all', '--success-limit=1',
                   '--success-bytes=2147483648', '--artifacts', preparation / 'artifacts', '--working-dir', args.repo]
        for tag in TAGS:
            command.extend(['--build-tag', tag])
        for entry in target_environment(cell):
            command.extend(['--env', entry])
        command.extend(['go-test', CELLS[cell][1], '--', *target_args(cell, 'success')])
        planned = require_zero(self.commands.run(cell + '-plan', command, seconds=600, private=self.private))
        response = decoded(planned['stdout']['path'])
        if response['schema'] != 'gomad.plan-result/v1' or response['guest_executions'] != 0:
            raise ValueError('gate 1 requires zero-execution seed preparation')
        admitted = require_zero(self.commands.run(cell + '-prepare-admission', [args.support, 'prepare-check',
                      response['preparation_receipt'], response['preparation_receipt_sha256']]))
        self.preparations[cell] = response
        self.gate(cell, GATE_NAMES[0], admitted['command'], [installed, planned, admitted],
                  preparation_module_sha256=response['module_sha256'],
                  preparation_receipt=response['preparation_receipt'],
                  preparation_receipt_sha256=response['preparation_receipt_sha256'])

    def measured(self, cell, mode):
        args = self.args
        root = self.root / (cell + '-results') / mode
        specification = self.root / f'{cell}-{mode}.spec.json'
        write(specification, qualification_spec(cell, mode, root, args.repo, self.source))
        cli = self.commands.run(cell + '-' + mode, [args.binary, '--installation', self.installations[cell],
                               'qualify', '--spec', specification], mandatory=cell == 'smoke', private=self.private)
        if cli['exit'] != (1 if mode == 'canary' else 0):
            raise ValueError(f'{cell}/{mode}: unexpected raw CLI exit {cli["exit"]}')
        response = decoded(cli['stdout']['path'])
        result = Path(response['result_path'])
        if not result.is_absolute() or not result.is_relative_to(root):
            raise ValueError('result outside isolated support-cell root')
        report = self.commands.run(cell + '-' + mode + '-report', [args.binary, 'report', '--result', result])
        admission = self.commands.run(cell + '-' + mode + '-admission', [args.support, 'check-result', result])
        measured_status(mode, cli['exit'], report['exit'], admission['exit'])
        checked = decoded(admission['stdout']['path'])
        if checked['module_sha256'] != self.preparations[cell]['module_sha256']:
            raise ValueError('measured module differs from gate-1 preparation')
        self.results.append(str(result))
        self.process_receipts.append(dict(result=str(result), **self.commands.receipts[-3]))
        self.gate(cell, GATE_NAMES[1 if mode == 'success' else 2], admission['command'], [cli, report, admission])

    def perform(self, name):
        self.frozen()
        if name == 'inspection':
            args = self.args
            row = require_zero(self.commands.run('inspection', [sys.executable, args.inspection,
                '--binary', args.binary, '--repo', args.repo, '--root', self.root / 'inspection',
                '--compiler', args.compiler, '--runtime-root', args.runtime_root,
                '--cache', getattr(args, 'cache', None) or self.root / 'inspection-cache', '--captured-config', args.inspection_config,
                *(['--private-observation-config', args.private_observation_config]
                  if self.private is not None else [])]))
            self.inspection = row
            receipt = Path(decoded(row['stdout']['path'])['receipt'])
            self.inspection_fields = dict(inspection_receipt=str(receipt),
                                          inspection_receipt_sha256=digest(read(receipt)))
            self.gate('smoke', GATE_NAMES[3], row['command'], [row], **self.inspection_fields)
        elif name == 'publication':
            self.gate('frontend', GATE_NAMES[3], self.inspection['command'], [self.inspection], **self.inspection_fields)
            if self.private is not None:
                producer, context = self.private
                for reference in self.commands.receipts:
                    receipt = decoded(reference['path'])
                    if 'private_observations' in receipt:
                        producer.observation_module().verify_snapshot(context, receipt['operator_process_id'],
                                                                      receipt['private_observations'])
            rows = [gate for cell in CELLS for gate in self.gates[cell]]
            complete(rows)
            request = self.root / 'support-request.json'
            write(request, dict(executable=dict(path=str(self.args.binary), sha256=self.inputs[str(self.args.binary)]),
                                results=self.results, process_receipts=self.process_receipts, gates=rows))
            require_zero(self.commands.run('support-publication', [self.args.support, 'publish', request,
                                                                  self.root / 'support-list']))
            require_zero(self.commands.run('support-validation', [self.args.support, 'validate', self.root / 'support-list.json']))
        else:
            cell, mode = name.split('-')
            self.prepare(cell) if mode == 'preparation' else self.measured(cell, mode)
        self.frozen()


def main(arguments=None):
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'support', 'repo', 'root', 'compiler', 'runtime-root']:
        parser.add_argument('--' + name, type=Path, required=True)
    for name in ['smoke-config', 'frontend-config', 'inspection-config', 'captured-config']:
        parser.add_argument('--' + name, type=Path)
    parser.add_argument('--source-manifest', type=Path)
    parser.add_argument('--private-observation-config', type=Path)
    parser.add_argument('--cache', type=Path)
    parser.add_argument('--prepare-only', action='store_true')
    parser.add_argument('--inspection', type=Path, default=CORPUS / 'onebox-inspection-check.py')
    args = parser.parse_args(arguments)
    for key, value in vars(args).items():
        if isinstance(value, Path):
            setattr(args, key, value.resolve())
    args.root.mkdir(mode=0o700, parents=True, exist_ok=False)
    run = Onebox.__new__(Onebox)
    run.gates = {cell: [] for cell in CELLS}
    run.commands = Commands(args.root)
    try:
        run.__init__(args)
        if args.prepare_only:
            run.perform('smoke-preparation')
            print(json.dumps(dict(row='smoke', preparation=run.preparations['smoke'])))
            return
        ordered(run.perform)
    except (ValueError, OSError, KeyError, subprocess.SubprocessError) as error:
        write(args.root / 'pending.json', dict(schema='gomad.rust-onebox-pending/v1', error=str(error),
              gates=run.gates, commands=run.commands.receipts))
        raise
    print(json.dumps(dict(support_list=str(args.root / 'support-list.json'))))


if __name__ == '__main__':
    main()
