#!/usr/bin/env python3
"""Receiver-owned Rust qualification orchestration; never imports Go qualification producers."""
import argparse
import fnmatch
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
from contextlib import contextmanager
import signal
import subprocess
import sys
import tempfile
import time

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
FROZEN = REPO / '.flow/artifacts/wasm-qualification-driver/parallel'
DAG = FROZEN / 'dag.tsv'


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


scheduler = load_module('frozen_qualification_scheduler', FROZEN / 'scheduler.py')


class Refusal(ValueError):
    pass


class Unexecuted(Refusal):
    pass


def require(condition, reason):
    if not condition:
        raise Refusal(reason)


def unique(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, 'duplicate JSON key: ' + key)
        result[key] = value
    return result


def read(path):
    return json.loads(Path(path).read_bytes(), object_pairs_hook=unique,
                      parse_constant=lambda value: (_ for _ in ()).throw(Refusal('nonfinite JSON: ' + value)))


def write(path, value):
    with Path(path).open('x') as handle:
        json.dump(value, handle, sort_keys=True, separators=(',', ':'), allow_nan=False)
        handle.write('\n')


def sha(path):
    with Path(path).open('rb') as handle:
        return hashlib.file_digest(handle, 'sha256').hexdigest()


def check_files(files):
    require(bool(files), 'empty frozen input set')
    for name, digest in files.items():
        path = Path(name)
        require(path.is_absolute() and path.is_file() and not path.is_symlink(), 'not a regular frozen input: ' + name)
        require(sha(path) == digest, 'frozen input changed: ' + name)


def mapping():
    rows = read(HERE / 'semantic-map.json')
    labels = [row['label'] for row in rows]
    require(len(labels) == len(set(labels)) == 96, 'missing/duplicate semantic stage')
    require(set(labels) == set(scheduler.load_dag(DAG)), 'semantic stage/DAG mismatch')
    return {row['label']: row for row in rows}


def stage_commands(mode, config):
    require(mode in ('serial', 'parallel'), 'unknown mode')
    return {label: [sys.executable, str(HERE / 'driver.py'), 'stage', '--config', str(config), label]
            for label in mapping()}


def coverage(expected, observed):
    require(len(expected) == len(set(expected)) and expected, 'empty/duplicate required subgates')
    names = [row['id'] for row in observed]
    require(len(names) == len(set(names)), 'duplicate observed subgate')
    require(set(names) == set(expected), 'missing/unexpected subgates: ' + repr(set(expected) ^ set(names)))
    for row in observed:
        require(row.get('status') == 'PASS' and row.get('executed') is True and row.get('exit') == 0
                and not row.get('synthetic', False), 'unexecuted/refused/synthetic subgate: ' + row['id'])


def deadline(limit):
    require(isinstance(limit, (int, float)) and math.isfinite(limit) and 0 < limit <= 3600,
            'deadline must be positive and at most 3600 seconds')
    return limit


def verdict(status, sent, finished, limit):
    deadline(limit)
    return 3 if status == 0 and finished - sent > limit else status


def bind_identity(expected_role, expected, role, observed):
    require(expected_role == role == 'release', 'qualification requires admitted release role')
    require(bool(expected) and expected == observed, 'candidate identity mismatch')
    require(all(isinstance(v, str) and len(v) == 64
                and all(c in '0123456789abcdef' for c in v) for v in expected.values()), 'malformed identity')


def completed(code, accepted):
    require(code in accepted, f'command refused/failed: exit {code}, expected {accepted}')


def clean_env(config):
    env = {key: value for key, value in os.environ.items()
           if not key.startswith(('GOMAD', 'QUALDRV')) and key not in
           ('RUSTFLAGS', 'CARGO_ENCODED_RUSTFLAGS', 'GOROOT', 'GOEXPERIMENT', 'GOOS', 'GOARCH')}
    env.update(config.get('environment', {}))
    env.update(config.get('temporary_paths', {}))
    env.update(LC_ALL='C', GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CARGO_INCREMENTAL='0',
               RUSTC_WRAPPER='', RUSTC_WORKSPACE_WRAPPER='')
    return env


def save_process(path, record):
    descriptor, temporary = tempfile.mkstemp(prefix=path.name + '.', dir=path.parent)
    try:
        with os.fdopen(descriptor, 'w') as handle:
            json.dump(record, handle, sort_keys=True, separators=(',', ':'), allow_nan=False)
            handle.write('\n')
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


class MeasuredPopen(subprocess.Popen):
    """Popen owns lifecycle; its single reap also retains this exact child's usage."""
    def __init__(self, *args, **kwargs):
        require(sys.platform.startswith('linux') and hasattr(os, 'wait4') and hasattr(os, 'WNOWAIT'),
                'Linux per-child wait4 resource measurement unavailable')
        self.child_resources = None
        self.started_monotonic_nanos = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
        self.reaped_monotonic_nanos = None
        self.parent_pid = os.getpid()
        self._measured_status = None
        super().__init__(*args, **kwargs)

    def _try_wait(self, wait_flags):
        # Popen holds _waitpid_lock. Probe without reaping, then retain status
        # and rusage before delivering a pending termination/interrupt handler.
        # Cached status lets interrupted Popen.wait finish without a second reap.
        blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
        try:
            if self._measured_status is not None:
                return self._measured_status
            ready = os.waitid(os.P_PID, self.pid, os.WEXITED | os.WNOHANG | os.WNOWAIT)
            if ready is not None:
                require(ready.si_pid == self.pid, 'resource probe returned another child')
                waited, status, usage = os.wait4(self.pid, os.WNOHANG)
                reaped_monotonic_nanos = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
                require(waited == self.pid, 'resource measurement lost its exact child')
                self._measured_status = (waited, status)
                self.reaped_monotonic_nanos = reaped_monotonic_nanos
                require(all(math.isfinite(value) and value >= 0 for value in (usage.ru_utime, usage.ru_stime))
                        and isinstance(usage.ru_maxrss, int) and usage.ru_maxrss >= 0,
                        'invalid per-child resource measurement')
                self.child_resources = dict(schema='gomad.driver-child-resources/v1', measurement='wait4',
                    scope='whole-command-child',
                    pid=self.pid, waited_pid=waited, wait_status=status, exit_code=os.waitstatus_to_exitcode(status),
                    user_nanos=round(usage.ru_utime * 1_000_000_000),
                    system_nanos=round(usage.ru_stime * 1_000_000_000), peak_rss_bytes=usage.ru_maxrss * 1024,
                    user_seconds=usage.ru_utime, system_seconds=usage.ru_stime,
                    ru_maxrss=usage.ru_maxrss, ru_maxrss_unit='KiB',
                    cpu_conversion='round(seconds*1e9)', rss_conversion='ru_maxrss*1024')
                return self._measured_status
        finally:
            signal.pthread_sigmask(signal.SIG_SETMASK, blocked)
        # Popen's no-timeout path otherwise loops immediately after a WNOHANG
        # probe. Sleep only affects readiness polling, never resource accounting.
        if not wait_flags & os.WNOHANG:
            time.sleep(0.01)
        return 0, 0


def command(config, root, name, argv, accepted=(0,), timeout=None, private_observations=None, exact_environment=None):
    require(argv and all(isinstance(arg, str) for arg in argv), 'invalid argv')
    require(private_observations is None or private_observations == config, 'foreign private observation context')
    require(exact_environment is None or isinstance(exact_environment, dict)
            and all(isinstance(key, str) and isinstance(value, str) for key, value in exact_environment.items()), 'invalid exact command environment')
    effective_environment = clean_env(config) if exact_environment is None else dict(exact_environment)
    start = time.time_ns()
    observation, proc = 'completed', None
    private_seal = None
    process_path = root / (name + '.process.json')
    require(not process_path.exists(), 'command process registration already exists')
    with (root / (name + '.stdout')).open('xb') as stdout, (root / (name + '.stderr')).open('xb') as stderr:
        try:
            read_gate, write_gate = os.pipe()
            blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
            try:
                launch = [sys.executable, str(HERE / 'launch.py'), str(read_gate), *argv]
                proc = MeasuredPopen(launch, cwd=config['repo'], env=effective_environment, stdout=stdout, stderr=stderr,
                                        start_new_session=True, pass_fds=(read_gate,))
                process = dict(pid=proc.pid, stage_pid=os.getpid(), parent_pid=proc.parent_pid, active=True, argv=argv, start_ns=start,
                               clock='CLOCK_MONOTONIC', started_monotonic_nanos=proc.started_monotonic_nanos)
                save_process(process_path, process)
                os.write(write_gate, b'1')
            finally:
                os.close(read_gate)
                os.close(write_gate)
                signal.pthread_sigmask(signal.SIG_SETMASK, blocked)
            code = proc.wait(timeout=timeout)
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            else:
                process['cleanup_signal'] = int(signal.SIGKILL)
                raise Refusal('producer exited with remaining process-group members')
            if private_observations is not None and code in accepted:
                import native_observations
                private_seal = native_observations.from_command(private_observations, proc.pid, argv,
                                                                (root / (name + '.stdout')).read_bytes())
        except BaseException:
            observation = 'inconclusive'
            if proc is not None:
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                proc.wait()
            raise
        finally:
            if proc is not None:
                cleanup_error = None
                try:
                    if wait_command_processes(proc.pid) and observation == 'completed':
                        raise Refusal('producer exited with remaining process-session members')
                except (Refusal, OSError, ValueError) as error:
                    cleanup_error = error
                    observation = 'inconclusive'
                process.update(active=cleanup_error is not None, exit=proc.returncode, observation=observation, end_ns=time.time_ns(),
                               resources=proc.child_resources, reaped_monotonic_nanos=proc.reaped_monotonic_nanos)
                save_process(process_path, process)
                receipt = dict(argv=argv, cwd=config['repo'], pid=proc.pid, exit=proc.returncode,
                      environment=config.get('environment', {}), temporary_paths=config.get('temporary_paths', {}),
                      effective_environment=None if exact_environment is None else effective_environment,
                      environment_mode='inherited-clean' if exact_environment is None else 'exact',
                      parent_pid=proc.parent_pid, clock='CLOCK_MONOTONIC',
                      started_monotonic_nanos=proc.started_monotonic_nanos, reaped_monotonic_nanos=proc.reaped_monotonic_nanos,
                      resources=proc.child_resources, observation=observation, start_ns=start, end_ns=time.time_ns())
                if private_seal is not None:
                    receipt['private_observations'] = private_seal
                write(root / (name + '.command.json'), receipt)
                if cleanup_error is not None:
                    raise cleanup_error
    completed(code, accepted)
    return (root / (name + '.stdout')).read_bytes()


def process_states():
    require(sys.platform.startswith('linux') and Path('/proc').is_dir(),
            'process group quiescence requires Linux procfs')
    for process in Path('/proc').iterdir():
        if not process.name.isdecimal():
            continue
        try:
            fields = (process/'stat').read_text().rsplit(') ', 1)[1].split()
        except (FileNotFoundError, ProcessLookupError):
            continue
        except (OSError, IndexError) as error:
            raise Refusal('cannot observe process group quiescence') from error
        require(len(fields) >= 4, 'invalid process group observation')
        yield fields[0], int(fields[2]), int(fields[3])


def process_group_live(group):
    return any(observed == group and state not in ('Z', 'X')
               for state, observed, _ in process_states())


def process_session_groups(sessions):
    groups = {}
    for state, group, session in process_states():
        if session in sessions and state not in ('Z', 'X'):
            groups.setdefault(session, set()).add(group)
    return groups


def wait_process_group(group):
    deadline = time.monotonic()+5
    while process_group_live(group):
        require(time.monotonic() < deadline, 'process group did not quiesce after SIGKILL')
        time.sleep(0.01)


def wait_process_session(session):
    deadline = time.monotonic()+5
    observed = False
    while groups := process_session_groups({session}).get(session, set()):
        observed = True
        for group in groups:
            try:
                os.killpg(group, signal.SIGKILL)
            except ProcessLookupError:
                pass
        require(time.monotonic() < deadline, 'process session did not quiesce after SIGKILL')
        time.sleep(0.01)
    return observed


def wait_command_processes(pid):
    errors, remaining_session = [], False
    try:
        wait_process_group(pid)
    except (Refusal, OSError, ValueError) as error:
        errors.append(str(error))
    try:
        remaining_session = wait_process_session(pid)
    except (Refusal, OSError, ValueError) as error:
        errors.append(str(error))
    require(not errors, 'command process cleanup incomplete: '+'; '.join(errors))
    return remaining_session


class StageExecutor(scheduler.RealExecutor):
    def __init__(self, runner, state_dir, env):
        super().__init__(runner, state_dir, env)
        self.stage_pids = {}
        self.command_root = Path(state_dir).parent

    def start(self, stage, lane):
        super().start(stage, lane)
        self.stage_pids[stage.label] = self.procs[stage.label][0].pid

    def cleanup_commands(self, stage_pids):
        attempted, errors, owners = set(), [], set(stage_pids)
        for _ in range(32):
            records = []
            for path in self.command_root.rglob('*.process.json'):
                try:
                    records.append((path, read(path)))
                except (Refusal, OSError, ValueError, KeyError) as error:
                    errors.append(str(path)+': '+str(error))
            while True:
                children = {record['pid'] for _, record in records if record.get('stage_pid') in owners}
                if children <= owners:
                    break
                owners.update(children)
            registrations = [(path, record) for path, record in records
                             if record.get('stage_pid') in owners and record.get('active')
                             and (path, record['pid']) not in attempted]
            if not registrations:
                break
            sessions = {record['pid'] for _, record in registrations}
            groups = set(sessions)
            try:
                for owned_groups in process_session_groups(sessions).values():
                    groups.update(owned_groups)
            except (Refusal, OSError, ValueError) as error:
                errors.append(str(error))
            for group in groups:
                try:
                    os.killpg(group, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                except (OSError, ValueError) as error:
                    errors.append(str(group)+': '+str(error))
            for path, record in registrations:
                attempted.add((path, record['pid']))
                try:
                    wait_command_processes(record['pid'])
                    record.update(active=False, observation='inconclusive', cleanup_signal=int(signal.SIGKILL))
                    save_process(path, record)
                except (Refusal, OSError, ValueError, KeyError) as error:
                    errors.append(str(path)+': '+str(error))
        else:
            errors.append('command registrations did not quiesce during cleanup')
        require(not errors, 'command cleanup incomplete: '+'; '.join(errors))
        return bool(attempted)

    def wait_any(self):
        label, code, ended = super().wait_any()
        if self.cleanup_commands({self.stage_pids[label]}) and code == 0:
            code = 1
        return label, code, ended

    def terminate(self):
        pids = set(self.stage_pids.values()) | {proc.pid for proc, _, _ in self.procs.values()}
        errors = []
        try:
            super().terminate()
        except (Refusal, OSError, ValueError, subprocess.SubprocessError) as error:
            errors.append(str(error))
        for proc, _, _ in self.procs.values():
            try:
                proc.wait()
            except (OSError, subprocess.SubprocessError) as error:
                errors.append(str(error))
        try:
            self.cleanup_commands(pids)
        except (Refusal, OSError, ValueError, KeyError) as error:
            errors.append(str(error))
        require(not errors, 'stage cleanup incomplete: '+'; '.join(errors))


@contextmanager
def busy_host(label):
    workers = []
    try:
        if label in ('busy-boundaries', 'busy-grouped'):
            for _ in range(12):
                workers.append(subprocess.Popen([sys.executable, '-c', 'while True: pass'],
                                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL))
        yield
    finally:
        for worker in workers:
            worker.terminate()
        cleanup_deadline = time.monotonic() + 5
        for worker in workers:
            try:
                worker.wait(timeout=max(0, cleanup_deadline-time.monotonic()))
            except subprocess.TimeoutExpired:
                worker.kill()
                worker.wait()


def guard(config):
    require(sha(config['_config_path']) == config['_config_sha256'], 'driver config changed')
    check_files(config['files'])
    repo = Path(config['repo'])
    head = subprocess.check_output(['git', '-C', str(repo), 'rev-parse', 'HEAD'], text=True).strip()
    require(head == config['head'], 'frozen HEAD changed')
    tracked = subprocess.check_output(['git', '-C', str(repo), 'ls-files', '-z']).decode().split('\0')
    wanted = {str(repo / path) for path in tracked if path and not path.startswith('.flow/')
              and (repo / path).is_file() and not (repo / path).is_symlink()}
    require(wanted <= set(config['files']), 'incomplete non-Flow source inventory')
    required_driver = [HERE / 'driver.py', HERE / 'adapters.py', HERE / 'semantic-map.json', HERE / 'required-final.json', HERE / 'sqlite.py', HERE / 'launch.py', HERE / 'host_admission.py', HERE / 'live.py', HERE / 'final.py', HERE / 'reach.py', HERE / 'receiving.py', HERE / 'runtime_gate.py', HERE / 'receiving_cost.py', HERE / 'native_observations.py',
                       HERE.parent / 'workload_tools/workloads.py',
                       FROZEN / 'scheduler.py', DAG, FROZEN / 'lib.sh',
                       FROZEN.parent / 'q1-serial/run-qualification.sh', FROZEN / 'required-roots.py']
    require(all(str(path) in config['files'] for path in required_driver), 'unsealed driver/reference input')
    for path, target in config.get('symlinks', {}).items():
        require(Path(path).is_symlink() and os.readlink(path) == target, 'frozen symlink changed: ' + path)
    symlinks = {str(repo / path): os.readlink(repo / path) for path in tracked if path and (repo / path).is_symlink()}
    require(symlinks == config['symlinks'], 'frozen source symlink inventory changed')
    raw_status = subprocess.check_output(['git', '-C', str(repo), 'status', '--porcelain', '--untracked-files=all', '-z'])
    allowed = []
    registry = Path(config['root']) / 'scheduler' / 'registry'
    for path in registry.glob('*.json'):
        row = read(path)
        if 'end_ns' not in row:
            allowed.extend(row.get('scratch', []))
    for row in raw_status.decode().split('\0'):
        if not row:
            continue
        name = row[3:]
        if name.startswith('.flow/'):
            continue
        require(row.startswith('?? ') and any(fnmatch.fnmatchcase(name, pattern) or
                fnmatch.fnmatchcase(name, pattern + '/*') for pattern in allowed), 'dirty frozen source: ' + row)
    require(config['source_manifest'] in config['files'] and sha(config['source_manifest']) == config['source_sha256'],
            'frozen source manifest changed')


def configured(path):
    config = read(path)
    require(config['schema'] == 'gomad.rust-qualification-driver/v1', 'unknown driver config')
    require(Path(config['repo']).resolve() == REPO, 'wrong candidate checkout')
    require(set(config['stages']) == set(mapping()), 'missing/unexpected configured stages')
    require(all(entry.get('adapter') != 'pending' for entry in config['stages'].values()),
            'unbound original stage endpoints cannot start qualification')
    require(config.get('final_reader') and config.get('final_sections'), 'missing actual final readers')
    require(config.get('synthetic') is not True, 'synthetic fixture cannot qualify')
    require(set(config['identity']) == {'host', 'helper'}, 'frozen host/helper identities required')
    require(all(isinstance(config[key], (int, float)) and math.isfinite(config[key]) and config[key] > 0
                for key in ('cores', 'mem_gib')), 'invalid host resource caps')
    require(config['cores'] <= (os.cpu_count() or 1), 'core cap exceeds host')
    require(isinstance(config['sent_epoch'], (int, float)) and math.isfinite(config['sent_epoch'])
            and config['sent_epoch'] > 0 and config['sent_epoch'] <= time.time(), 'invalid send epoch')
    deadline(config['deadline_s'])
    from host_admission import STAT, bindings
    bindings(config)
    require(str(STAT) in config['files'], 'unsealed native filesystem probe')
    check_files(config['files'])
    return config


def subgates(label):
    row = mapping()[label]
    return [name + (f'#{i}' if row['repetitions'] > 1 else '')
            for name in row['required_subgates'] for i in range(row['repetitions'])]


def stage(config, label):
    from adapters import execute
    root = Path(config['root']) / 'stages' / label
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    started = time.time_ns()
    status, reason, observed = 1, '', []
    try:
        from host_admission import admit
        admit(config, root / 'host-admission.json', initial=False)
        guard(config)
        observed = execute(config, root, label)
        coverage(subgates(label), observed)
        guard(config)
        status = 0
    except KeyboardInterrupt:
        status, reason = 130, 'interrupted'
    except Unexecuted as error:
        status, reason = 4, 'UNEXECUTED: ' + str(error)
    except (Refusal, OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        reason = str(error)
    write(root / 'receipt.json', dict(schema='gomad.rust-driver-stage/v1', stage=label, exit=status,
          status='PASS' if status == 0 else 'UNEXECUTED' if status == 4 else 'FAIL', reason=reason,
          start_ns=started, end_ns=time.time_ns(), source=config['source_sha256'],
          identity=config['identity'], subgates=observed, config_sha256=config['_config_sha256']))
    return status


def final_checks(config):
    guard(config)
    receipts = {}
    for label in mapping():
        path = Path(config['root']) / 'stages' / label / 'receipt.json'
        require(path.is_file(), 'UNEXECUTED stage: ' + label)
        row = read(path)
        require(row['stage'] == label and row['exit'] == 0 and row['status'] == 'PASS', 'nonpassing stage: ' + label)
        require(row['source'] == config['source_sha256'] and row['identity'] == config['identity']
                and row['config_sha256'] == config['_config_sha256'], 'stale stage identity: ' + label)
        coverage(subgates(label), row['subgates'])
        receipts[label] = sha(path)
    reader = config.get('final_reader')
    if not reader:
        raise Unexecuted('complete transferred-live/stress/R4/ringpop/test-wasm final reader is not integrated')
    require(reader[0] in config['files'], 'unsealed final reader')
    raw = command(config, Path(config['root']), 'final-reader', reader + ['--root', config['root']])
    report = json.loads(raw)
    require(report['schema'] == 'gomad.rust-driver-final-check/v1' and report['identity'] == config['identity']
            and report['source_sha256'] == config['source_sha256'], 'final reader binding mismatch')
    coverage(read(HERE / 'required-final.json')['subgates'], report['subgates'])
    guard(config)
    return receipts


def run(config, path, mode):
    root = Path(config['root'])
    root.mkdir(parents=True, exist_ok=False)
    (root / 'scheduler').mkdir()
    status, reason, receipts = 2, '', {}
    executor = None
    try:
        from host_admission import admit
        admit(config, root / 'host-admission.json', initial=True)
        guard(config)
        # Host evidence is retained even on admission refusal; no platform is substituted.
        write(root / 'host.json', dict(uname=list(os.uname()), cpu_count=os.cpu_count(),
                                      mem_available_gib=scheduler.mem_available_gib(), sent_epoch=config['sent_epoch']))
        from adapters import admission
        admission(config, root)
        stages = scheduler.load_dag(DAG)
        if mode == 'serial':
            # Serial reference uses the exact parallel argv, retaining all original resources as evidence.
            previous = None
            for stage_row in stages.values():
                stage_row.cls = 'solo'
                if previous and previous not in stage_row.deps:
                    stage_row.deps.append(previous)
                previous = stage_row.label
        policy = scheduler.Policy(config['cores'], config['mem_gib'], False,
                                  config['sent_epoch'] + config['deadline_s'], False,
                                  live_mem=scheduler.mem_available_gib)
        runner = [sys.executable, str(HERE / 'driver.py'), 'stage', '--config', str(path),
                  '--config-sha256', config['_config_sha256']]
        executor = StageExecutor(runner, str(root / 'scheduler'), dict(os.environ))
        result = scheduler.run(stages, policy, executor)
        write(root / 'scheduler' / 'schedule.json', result)
        status = result['status']
        if status in (0, 3):
            receipts = final_checks(config)
    except KeyboardInterrupt:
        status, reason = 130, 'interrupted'
    except (Refusal, OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        status, reason = 2, str(error)
    finally:
        cleanup_errors = []
        if executor is not None:
            try:
                executor.terminate()
            except (Refusal, OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
                cleanup_errors.append(str(error))
            for proc, log, _ in executor.procs.values():
                try:
                    proc.wait()
                except (OSError, subprocess.SubprocessError) as error:
                    cleanup_errors.append(str(error))
                finally:
                    try:
                        log.close()
                    except OSError as error:
                        cleanup_errors.append(str(error))
        if cleanup_errors:
            status = 2
            reason = '; '.join(filter(None, [reason, *cleanup_errors]))
        finished = time.time()
        status = verdict(status, config['sent_epoch'], finished, config['deadline_s'])
        write(root / 'outcome.json', dict(schema='gomad.rust-driver-outcome/v1', mode=mode, exit=status,
              status='PASS' if status == 0 else 'DEADLINE' if status == 3 else 'INCOMPLETE', reason=reason,
              sent_epoch=config['sent_epoch'], finished_epoch=finished, wall_seconds=finished-config['sent_epoch'],
              deadline_seconds=config['deadline_s'], identity=config['identity'], receipts=receipts))
    return status


def main():
    def interrupted(signum, frame):
        raise KeyboardInterrupt(signal.Signals(signum).name)
    for signum in (signal.SIGTERM, signal.SIGHUP, signal.SIGINT):
        signal.signal(signum, interrupted)
    signal.pthread_sigmask(signal.SIG_UNBLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('serial', 'parallel', 'stage', 'plan'))
    parser.add_argument('--config', type=Path)
    parser.add_argument('--config-sha256')
    parser.add_argument('label', nargs='?')
    args = parser.parse_args()
    try:
        if args.mode == 'plan':
            print(json.dumps(dict(stages=mapping(), serial=stage_commands('serial', '/config'),
                                  parallel=stage_commands('parallel', '/config')), indent=2))
            return 0
        require(args.config is not None, '--config is required')
        config = configured(args.config)
        config['_config_sha256'] = args.config_sha256 or sha(args.config)
        config['_config_path'] = str(args.config.resolve())
        require(sha(args.config) == config['_config_sha256'], 'driver config changed')
        if args.mode == 'stage':
            require(args.label in mapping(), 'unknown stage')
            return stage(config, args.label)
        return run(config, args.config.resolve(), args.mode)
    except (Refusal, OSError, ValueError, KeyError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.modules['driver'] = sys.modules[__name__]
    raise SystemExit(main())
