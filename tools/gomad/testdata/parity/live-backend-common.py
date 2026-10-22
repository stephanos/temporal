"""Shared execution and authentication for transferred live backend roots."""
import argparse
import json
import os
from pathlib import Path
import runpy
import signal
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
MODULE = CORPUS.parents[1]
REPOSITORY = MODULE.parents[1]
seed = runpy.run_path(str(CORPUS / 'campaign-check.py'))
rounds = runpy.run_path(str(CORPUS / 'round-check.py'))
encode = seed['parity'].canonical
equal = seed['equal']
digest = seed['parity'].digest


class Lane:
    def __init__(self, description):
        parser = argparse.ArgumentParser(description=description)
        for name in ('binary', 'installation', 'work'):
            parser.add_argument('--' + name, required=True, type=Path)
        self.args = parser.parse_args()
        for name in ('binary', 'installation', 'work'):
            setattr(self.args, name, getattr(self.args, name).resolve())
        self.args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith('GOMAD_') and key != 'GOROOT'}
        self.commands = []
        self.roots = []
        self.binary_sha256 = digest(self.args.binary.read_bytes())

    def run(self, name, arguments, statuses=(0,), installation=None, extra=None):
        command = [self.args.binary, '--installation', installation or self.args.installation, *arguments]
        child = subprocess.Popen(list(map(str, command)), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                 env=dict(self.environment, **(extra or {})), start_new_session=True)
        timed_out = False
        try:
            stdout, stderr = child.communicate(timeout=1200)
        except subprocess.TimeoutExpired:
            timed_out = True
            try:
                os.killpg(child.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            stdout, stderr = child.communicate()
        result = subprocess.CompletedProcess(command, child.returncode, stdout, stderr)
        for stream in ('stdout', 'stderr'):
            (self.args.work / (name + '.' + stream)).write_bytes(getattr(result, stream))
        self.commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                                  expected_statuses=list(statuses), environment_overrides=extra or {}, timed_out=timed_out,
                                  stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (self.args.work / 'commands.json').write_bytes(encode(self.commands))
        equal(result.returncode in statuses, True,
              name + ': ' + result.stdout.decode(errors='replace') + result.stderr.decode(errors='replace'))
        return result

    def installation(self, name, cooperative=True, fuel=4000000000, read_step=None):
        value = json.loads(self.args.installation.read_bytes())
        value['fuel'], value['memory_bytes'] = fuel, 256 << 20
        value['profile'] = ('gomad.wasi-cooperative/v1' if cooperative
                            else 'gomad.wasi-stock-exploratory/v1')
        value['model'].update(mode=1, epoch_nanos=946684800000000000,
                              read_step_nanos=(0 if cooperative else 1000) if read_step is None else read_step,
                              now=1 if cooperative and read_step != 1000 else 0)
        value['model']['limits'].update(calls=10000, transcript_bytes=8 << 20, pending_events=4096)
        value['namespace']['limits'].update(files=1000, descriptors=64, filesystem_bytes=8 << 20,
                                            output_bytes=1 << 20)
        value['namespace']['epoch_nanos'] = 946684800000000000
        value['cache_root'] = str(self.args.work / (name + '-preparation'))
        value['overlay_module_root'] = str(MODULE)
        path = self.args.work / (name + '-installation.json')
        path.write_bytes(encode(value))
        path.chmod(0o600)
        return path

    def explore(self, name, target, working_dir, flags=(), status=(0, 1), installation=None, extra=None,
                seeds='7', parallel=1, execution='60s', overall='180s', choices=False,
                strategy='seed', keep='none', output=1 << 20, success_limit=16, success_bytes=256 << 20, on_failure='all'):
        destination = self.args.work / name
        command = ['explore', '--json', '--seeds=' + seeds, '--parallel=' + str(parallel),
                   '--strategy=' + strategy, '--on-failure=' + on_failure, '--failure-budget=1',
                   '--execution-timeout=' + execution, '--overall-timeout=' + overall,
                   '--terminate-grace=100ms', '--output-limit=' + str(output),
                   '--world-transition-limit=1048576', '--coverage=' + ('choice' if choices else 'none'),
                   '--keep-successes=' + keep, '--artifacts', destination,
                   '--working-dir', working_dir]
        if choices:
            command += ['--choices', '--choice-bytes=1048576']
        if keep != 'none':
            command += ['--success-limit=' + str(success_limit), '--success-bytes=' + str(success_bytes)]
        command += [*flags, *target]
        result = self.run(name, command, status, installation, extra)
        roots = list((destination / 'campaigns').glob('campaign-*'))
        equal(len(roots), 1, name + '.campaign_count')
        return roots[0], result

    def authenticate(self, root, choice=False, complete=True):
        return (rounds['authenticate'](root) if choice else seed['authenticate_campaign'](root, complete=complete))

    def report(self, result):
        reports = [json.loads(line) for line in result.stdout.splitlines()]
        return next(row for row in reversed(reports) if row.get('type') in ('result', 'complete'))

    def artifacts(self, root, choice=False):
        checked = self.authenticate(root, choice)
        executions = ([execution for event in checked['events'] for execution in event['executions']]
                      if choice else checked['executions'])
        return list(dict.fromkeys(root / row['artifact'] for row in executions if row.get('artifact')))

    def replay(self, name, artifact, exact=True, failure=True):
        before = rounds['snapshot'](artifact)
        result = self.run(name, ['replay', artifact], (1 if failure else 0,))
        equal(b'reproduced=true' in result.stdout, True, name + '.match')
        equal(b'choice-replay=exact' in result.stdout, exact, name + '.exact')
        if not exact:
            equal(b'choice-replay=none' in result.stdout, True, name + '.observed_repetition')
        equal(rounds['snapshot'](artifact), before, name + '.immutable')
        return result

    def guest(self, name, constant, replacement=None):
        fixture = CORPUS / 'live_backend' / {'explorationGuest': 'exploration',
                                             'minimizationGuest': 'minimization'}[constant]
        source = (fixture / 'main.go').read_text()
        if replacement:
            old, new = replacement
            equal(source.count(old), 1, name + '.original_replacement')
            source = source.replace(old, new, 1)
        root = self.args.work / (name + '-source')
        root.mkdir(mode=0o700)
        (root / 'go.mod').write_bytes((fixture / 'go.mod').read_bytes())
        (root / 'main.go').write_text(source)
        return root

    def finish(self):
        equal(digest(self.args.binary.read_bytes()), self.binary_sha256, 'binary.unchanged')
        (self.args.work / 'roots.json').write_bytes(encode(dict(
            schema='gomad.live-backend-corpus/v1', binary_sha256=self.binary_sha256,
            driver_sha256=digest(Path(sys.argv[0]).read_bytes()),
            shared_driver_sha256=digest(Path(__file__).read_bytes()),
            fixture_inputs={str(path.relative_to(CORPUS)): digest(path.read_bytes())
                            for path in sorted((CORPUS / 'live_backend').rglob('*')) if path.is_file()},
            roots=self.roots)))
        print('executed roots: ' + ', '.join(str(row['root']) for row in self.roots))
