#!/usr/bin/env python3
"""Execute the three transferred retained-inspection roots on detached own-lineage artifacts."""
import argparse
import base64
import gzip
import hashlib
import json
import os
from pathlib import Path
import signal
import struct
import subprocess
import time


class Reader:
    def __init__(self, raw):
        self.raw = raw
        self.offset = 0

    def take(self, count):
        assert 0 <= count <= len(self.raw) - self.offset, 'truncated inspection frame'
        value = self.raw[self.offset:self.offset + count]
        self.offset += count
        return value

    def number(self, width):
        return int.from_bytes(self.take(width), 'little')

    def handle(self):
        size = self.number(4)
        assert 0 < size <= 1024
        return self.take(size).decode(), self.number(8)

    def done(self):
        assert self.offset == len(self.raw), 'inspection trailing bytes'


def boundaries(raw):
    source = Reader(raw)
    assert source.take(8) == b'GMWIB\x01\0\0'
    count = source.number(4)
    assert 0 < count <= 65536
    result = []
    for _ in range(count):
        frame = Reader(source.take(source.number(4)))
        node, incarnation = frame.handle()
        row = dict(node=node, incarnation=incarnation)
        row.update(zip(('decision', 'trace', 'order', 'time', 'world_bytes', 'lifecycle_bytes'),
                       (frame.number(8) for _ in range(6))))
        cuts = frame.number(4)
        assert 0 < cuts <= 65536
        row['cuts'] = []
        for _ in range(cuts):
            instance, serial = frame.handle()
            cut = dict(instance=instance, incarnation=serial)
            cut.update(zip(('choice_records', 'choice_bytes', 'wasi_bytes', 'network_bytes',
                            'storage_bytes', 'stdout_bytes', 'stderr_bytes'),
                           (frame.number(8) for _ in range(7))))
            frame.take(frame.number(4))
            frame.take(32 * 3 + 128)
            row['cuts'].append(cut)
        frame.done()
        result.append(row)
    source.done()
    return result


def network_cuts(cuts, lengths, target):
    assert len(cuts) == len(lengths) == 2
    assert {(cut['instance'], cut['incarnation']) for cut in cuts} == set(lengths)
    for cut in cuts:
        handle = cut['instance'], cut['incarnation']
        assert cut['network_bytes'] == (0 if handle == target else lengths[handle]), cut


def inventory(root):
    return {str(path.relative_to(root)): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in root.rglob('*') if path.is_file()}


class Lane:
    def __init__(self, args):
        self.args = args
        self.rows = []
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith('GOMAD_')}
        self.environment.update(GOMAD_COMPILER='/unavailable/live-inspection/compiler',
                                GOMAD_RUNTIME_ROOT='/unavailable/live-inspection/runtime')

    def run(self, name, artifact, flags, statuses=(0,), cancel=False):
        command = [str(self.args.binary), 'replay', *flags, str(artifact)]
        started = time.monotonic()
        process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   env=self.environment, start_new_session=True)
        early_cancel = False
        try:
            if cancel:
                try:
                    stdout, stderr = process.communicate(timeout=1)
                    early_cancel = True
                except subprocess.TimeoutExpired:
                    process.send_signal(signal.SIGINT)
                    stdout, stderr = process.communicate(timeout=9)
            else:
                stdout, stderr = process.communicate(timeout=180)
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
                process.communicate()
        elapsed = time.monotonic() - started
        # A linked CLI has no helper process; a surviving process group is a leaked worker.
        try:
            os.killpg(process.pid, 0)
        except ProcessLookupError:
            pass
        else:
            raise AssertionError(('inspection process group survived', process.pid))
        row = dict(name=name, argv=command, status=process.returncode, elapsed_seconds=elapsed,
                   expected_statuses=list(statuses), cancelled=cancel, early_cancel=early_cancel)
        self.rows.append(row)
        (self.args.work / (name + '.stdout')).write_bytes(stdout)
        (self.args.work / (name + '.stderr')).write_bytes(stderr)
        (self.args.work / 'commands.json').write_text(json.dumps(self.rows, indent=2) + '\n')
        if cancel:
            assert not early_cancel, ('timeout dump returned before cancellation', process.returncode, stderr)
            assert process.returncode != 0, 'cancelled inspection succeeded'
        else:
            assert process.returncode in statuses, (name, process.returncode, stderr)
        return json.loads(stdout) if stdout else None, elapsed

    def inspect(self, name, artifact, last, count, target=False):
        destination = self.args.work / name
        flags = [f'--stop=choice={last["decision"]}', '--dump=stacks', '--dump=heap',
                 '--dump=fixture.snapshot', '--dump=fixture.panic', '--debug-output', str(destination)]
        if target:
            flags += [f'--instance={last["node"]}', f'--incarnation={last["incarnation"]}']
        report, _ = self.run(name, artifact, flags)
        assert report['status'] == 'inspected_prefix' and report['prefix_verified'] and report['witness_match']
        assert report['resolved']['choice'] == last['decision']
        assert report['resolved']['incarnation'] == last['incarnation']
        assert len(report['prefix_cuts']) == count
        debug = json.loads((destination / 'debug.json').read_bytes())
        dumps = {dump['name']: dump for dump in debug['dumps']}
        assert dumps['fixture.snapshot']['status'] == 'ok'
        assert dumps['fixture.panic']['status'] == 'panic'
        assert debug['stacks'] and gzip.decompress((destination / 'heap.pprof').read_bytes())
        return report

    def searches(self, name, artifact, last, first, target=False):
        owner = [f'--instance={last["node"]}', f'--incarnation={last["incarnation"]}'] if target else []
        common = [*owner, '--search-timeout=30s']
        scan = [*common, '--check=fixture.before_three', f'--choice-range={first}:{last["decision"]}',
                '--search-mode=scan', '--search-attempts=1' if target else '--search-attempts=3']
        initial = None
        for repeat in range(1 if target else 2):
            result, _ = self.run(f'{name}-scan-{repeat}', artifact, scan, (0, 1))
            evaluations = result['search']['evaluations']
            assert evaluations and all(item['prefix_verified'] and item['outcome'] in ('pass', 'fail')
                                       for item in evaluations)
            if target:
                assert len(evaluations) == 1
            if initial is not None:
                assert evaluations == initial
            initial = evaluations
        point = [*common, f'--choice-range={last["decision"]}:{last["decision"]}', '--search-attempts=1']
        if not target:
            result, _ = self.run(name + '-bisect', artifact,
                                 [*point, '--check=fixture.before_three', '--search-mode=bisect'], (1,))
            assert result['search']['status'] == 'already_failing_at_first'
            assert result['search']['conditional_on_monotonicity']
            result, _ = self.run(name + '-nonmonotonic', artifact,
                                 [*point, '--check=fixture.recovering', '--search-mode=bisect'], (2,))
            assert result is None
        result, _ = self.run(name + '-panic', artifact,
                             [*point, '--check=fixture.panic_check', '--search-mode=scan'], (1,))
        search = result['search']
        assert search['status'] == 'invariant_evaluation_failure' and search.get('first_failure') is None
        assert len(search['evaluations']) == 1
        assert search['evaluations'][0]['detail'] == 'panic' and search['evaluations'][0]['outcome'] == 'error'
        timeout = [flag for flag in point if not flag.startswith('--search-timeout=')]
        result, elapsed = self.run(name + '-timeout', artifact,
                                   [*timeout, '--search-timeout=5s', '--check=fixture.timeout_check'], (1,))
        search = result['search']
        assert elapsed < 10 and search['status'] == 'incomplete' and search.get('first_failure') is None
        assert len(search['evaluations']) == 1 and search['evaluations'][0]['outcome'] == 'timeout'
        if not target:
            assert search.get('remaining') is not None
        _, elapsed = self.run(name + '-cancel', artifact,
                               [*owner, f'--stop=choice={last["decision"]}', '--dump=fixture.timeout'], cancel=True)
        assert elapsed < 10


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--work', type=Path, required=True)
    for case in ('standalone', 'network', 'restart'):
        parser.add_argument('--' + case + '-artifact', type=Path, required=True)
    parser.add_argument('--unavailable', type=Path, action='append', required=True,
                        help='detached original source/compiler/runtime/cache paths (each must be absent)')
    args = parser.parse_args()
    assert len(args.unavailable) >= 4 and all(not path.exists() for path in args.unavailable)
    args.binary = args.binary.resolve()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    lane = Lane(args)
    for name in ('standalone', 'network', 'restart'):
        artifact = getattr(args, name + '_artifact').resolve()
        before = inventory(artifact)
        manifest = json.loads((artifact / 'manifest.json').read_bytes())
        assert manifest['record']['seed'] == 7
        limits = manifest['record']['limits']
        assert limits['fuel'] == 1 << 60 and limits['memory_bytes'] == 256 << 20
        assert limits['wall_millis'] == 60000 and limits['output_bytes'] == 1 << 20
        metadata = json.loads((artifact / 'evidence/inspection').read_bytes())
        assert metadata['complete']
        rows = boundaries(base64.b64decode(metadata['boundary_frames'], validate=True))
        target = ('a', 2) if name == 'restart' else ('b', 1) if name == 'network' else None
        selected = [row for row in rows if target is None or (row['node'], row['incarnation']) == target]
        assert selected
        last = selected[-1]
        report = lane.inspect(name, artifact, last, 3 if name == 'restart' else 2 if name == 'network' else 1,
                              target is not None)
        if name == 'network':
            lengths = {}
            for item in metadata['incarnations']:
                assert item['streams']['network_encoding'] == 'gomad.cluster-network-binary/v1'
                raw = base64.b64decode(item['streams']['network'], validate=True)
                assert raw
                lengths[(item['handle']['node'], item['handle']['incarnation'])] = len(raw)
            network_cuts(last['cuts'], lengths, target)
            network_cuts(report['prefix_cuts'], lengths, target)
        else:
            if name == 'standalone':
                lane.inspect(name + '-repeat', artifact, last, 1)
                closed = metadata['incarnations'][0]['registry']['closed_order']
                first = next(row['decision'] for row in rows if row['order'] >= closed)
            else:
                assert all(cut['instance'] != 'never' for cut in report['prefix_cuts'])
                refused, _ = lane.run('restart-absent-target', artifact,
                                     [f'--stop=choice={last["decision"]}'], (2,))
                assert refused is None
                first = last['decision']
            lane.searches(name, artifact, last, first, name == 'restart')
        assert inventory(artifact) == before, (name, 'retained artifact mutated')
    (args.work / 'result.json').write_text(json.dumps(dict(
        schema='gomad.transferred-live-inspection/v1', roots=[
            'TestInspectionRetainedArtifactWithoutSourceOrCompiler',
            'TestInspectionRetainedClusterNetworkStoppedPrefix',
            'TestInspectionRetainedClusterTargetsRestartedIncarnation'], commands=len(lane.rows)), indent=2) + '\n')


if __name__ == '__main__':
    main()
