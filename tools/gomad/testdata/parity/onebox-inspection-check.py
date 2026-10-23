#!/usr/bin/env python3
"""Retain the original onebox storage/transport inspection gate on a release CLI."""
import argparse
import base64
from datetime import datetime, timezone
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import stat
import subprocess
import sys
import time

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'rust/workload_tools'))
import producer
import workloads


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(path):
    require(stat.S_ISREG(path.lstat().st_mode), 'nonregular retained input: ' + str(path))
    value = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1 << 20), b''):
            value.update(block)
    return value.hexdigest()


def read(path, maximum):
    require(path.stat().st_size <= maximum, 'input exceeds bound: ' + str(path))
    digest(path)
    with path.open('rb') as source:
        raw = source.read(maximum + 1)
    require(len(raw) <= maximum, 'input grew beyond bound: ' + str(path))
    return raw


def inventory(root):
    result = {}
    for path in sorted(root.rglob('*')):
        mode = path.lstat().st_mode
        require(stat.S_ISDIR(mode) or stat.S_ISREG(mode), 'nonregular artifact member')
        if stat.S_ISREG(mode):
            result[str(path.relative_to(root))] = digest(path)
    require(result, 'empty retained inventory')
    return result


def authenticate_payloads(artifact):
    manifest = json.loads(read(artifact / 'manifest.json', 32 << 20))
    require(manifest['schema'] == 'gomad.artifact-manifest/v2', 'wrong artifact schema')
    require(manifest['record']['seed'] == 7, 'wrong inspection seed')
    require(manifest['record']['outcome']['class'] == 'target_failure', 'missing actual failure')
    for name, entry in manifest['files'].items():
        require(name and not Path(name).is_absolute() and all(part not in ('', '.', '..') for part in name.split('/')),
                'unclean retained payload path')
        path = artifact / name
        require(path.stat().st_size == entry['bytes'] and digest(path) == entry['sha256'],
                'retained payload identity changed: ' + name)
    for name, stream in manifest['record']['streams'].items():
        entry = manifest['files'][name]
        require(entry['bytes'] == stream['bytes'] and entry['sha256'] == stream['sha256'],
                'record stream identity changed')
    stdout_name = 'evidence/cluster/history/1/stdout'
    require(stdout_name in manifest['record']['streams'], 'onebox stdout is not record bound')
    stdout = read(artifact / stdout_name, 1 << 20)
    for marker in (b'workflow completed [step 0 attempt 2 step 1 attempt 2]',
                   b'Temporal inspection canary after workflow timers and activity retries completed'):
        require(marker in stdout, 'onebox canary preceded completed timers and retries')
    require('target/module.wasm' in manifest['files'], 'dormant retained module missing')
    inputs = json.loads(read(artifact / 'evidence/cluster-replay.json', 32 << 20))
    config = inputs['configuration']
    require(config['workload'] == {'kind': 'temporal-onebox-stopped-inspection'}, 'wrong inspection workload')
    require(config['nodes'] == ['history'] and len(config['incarnations']) == 1, 'wrong inspection topology')
    incarnation = config['incarnations'][0]
    require(incarnation['handle'] == {'node': 'history', 'incarnation': 1}, 'wrong retained incarnation')
    require(incarnation['arguments'] == ['gomad-target', '-test.run=^TestReplayInspectionWorkflowFailure$', '-test.v'],
            'changed original inspection selection')
    require(incarnation['model']['limit'] == 1 << 30, 'changed original incarnation trace bound')
    for limits in (config['limits'], incarnation['limits']):
        require(limits['memory_bytes'] == 4 << 30 and limits['fuel'] == 1 << 50
                and limits['output_bytes'] == 1 << 20 and limits['wall_millis'] == 1800000,
                'changed original inspection execution bounds')
    return manifest


def validate_debug(root, report):
    require(report['status'] == 'inspected_prefix' and report['prefix_verified'] is True
            and report['witness_match'] is True and report['truncated'] is False, 'inspection prefix/witness mismatch')
    require(report['resolved']['instance'] == 'history' and report['resolved']['incarnation'] == 1,
            'inspection selected the wrong incarnation')
    output = root / 'inspection'
    require(Path(report['output_path']).resolve() == output / 'debug.json', 'foreign diagnostic output')
    debug = json.loads(read(output / 'debug.json', 8 << 20))
    heap = read(output / 'heap.pprof', 4 << 20)
    require(debug['stacks'] and heap and debug['probes'], 'empty stacks/heap/probes')
    # Bound decompression independently of the compressed diagnostic file.
    import io
    with gzip.GzipFile(fileobj=io.BytesIO(heap)) as source:
        require(bool(source.read(1)), 'empty heap profile')
    require({'temporal.shard.snapshot', 'temporal.mutable_state.snapshot'} <=
            {item['name'] for item in debug['probes']}, 'missing original snapshot probe owners')
    for name in ('temporal.mutable_states', 'temporal.shards'):
        rows = [item for item in debug['dumps'] if item['name'] == name]
        require(len(rows) == 1 and rows[0]['status'] == 'ok', 'missing successful snapshot dump: ' + name)
        body = base64.b64decode(rows[0]['data_base64'], validate=True)
        require(0 < len(body) <= 4 << 20 and b'snapshot=' in body, 'snapshot dump lacks useful content')
        if name == 'temporal.mutable_states':
            require(b'values may precede stop' in body, 'missing snapshot stop caveat')
    return inventory(output)


class Gate:
    def __init__(self, root, binary, repo):
        self.root, self.binary, self.repo = root, binary, repo
        self.commands = []
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith('GOMAD_') and key not in ('GOROOT', 'GOFLAGS', 'GOEXPERIMENT')}

    def run(self, name, command, expected, timeout, json_output=False, detached=False):
        command = list(map(str, command))
        environment = dict(self.environment)
        if detached:
            environment.update(PATH='', GOMAD_WASM_INSTALLATION=str(self.root / 'installation-unavailable.json'),
                               GOMAD_COMPILER=str(self.root / 'compiler-unavailable'),
                               GOMAD_RUNTIME_ROOT=str(self.root / 'runtime-unavailable'))
        stdout, stderr = self.root / (name + '.stdout'), self.root / (name + '.stderr')
        started = datetime.now(timezone.utc).isoformat()
        clock = time.monotonic()
        timed_out = False
        with stdout.open('xb') as out, stderr.open('xb') as err:
            process = subprocess.Popen(command, stdout=out, stderr=err, cwd=self.repo,
                                       env=environment, start_new_session=True)
            try:
                process.wait(timeout=timeout)
            except subprocess.TimeoutExpired:
                timed_out = True
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
        receipt = dict(name=name, command=command, exit=process.returncode, timed_out=timed_out,
                       started=started, ended=datetime.now(timezone.utc).isoformat(),
                       elapsed_seconds=time.monotonic() - clock, timeout_seconds=timeout,
                       environment_intent=dict(mandatory_workloads=False, detached=detached),
                       stdout=str(stdout), stderr=str(stderr), stdout_sha256=digest(stdout), stderr_sha256=digest(stderr))
        producer.write(self.root / (name + '.command.json'), receipt)
        self.commands.append(receipt)
        require(not timed_out and process.returncode == expected,
                f'{name} exit {process.returncode}, expected {expected}; logs: {stdout}, {stderr}')
        if json_output:
            return json.loads(read(stdout, 32 << 20))
        return read(stdout, 32 << 20)

    def replay(self, name, artifact):
        stdout = self.run(name, [self.binary, 'replay', artifact], 1, 1800, detached=True)
        require(stdout.strip() == b'gomad: reproduced=true diagnostic=false result=target_failure choice-replay=exact',
                'detached replay did not match the complete retained failure')


def explore_command(binary, installation, root, repo, stage, scenario):
    command = [binary, '--installation', installation, 'explore', '--json', '--seeds', '7', '--parallel', '1',
               '--execution-timeout', '30m', '--overall-timeout', '45m', '--terminate-grace', '1s',
               '--on-failure', 'first', '--failure-budget', '1', '--output-limit', str(1 << 20),
               '--world-transition-limit', str(1 << 20), '--choice-bytes', '0', '--scenario', scenario,
               '--artifacts', root / 'artifacts']
    for entry in ('GOMAD_DEBUG_CANARY=1', 'TEMPORAL_ROOT=/workspace', 'TMPDIR=/tmp', 'TEMPORAL_TEST_LOG_LEVEL=warn'):
        command += ['--env', entry]
    for tag in workloads.TAGS + ['gomad']:
        command += ['--build-tag', tag]
    return command + ['--working-dir', repo, 'go-test', './' + str(stage.relative_to(repo)), '--',
                      '-test.run=^TestReplayInspectionWorkflowFailure$', '-test.v']


def copy_alias(source, destination):
    def copy_file(src, dst):
        require(stat.S_ISREG(Path(src).lstat().st_mode), 'nonregular private tool input')
        try:
            os.link(src, dst)
        except OSError:
            shutil.copy2(src, dst)
        return dst
    shutil.copytree(source, destination, copy_function=copy_file)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'repo', 'root', 'compiler', 'runtime-root', 'cache', 'captured-config'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    binary, repo, root = args.binary.resolve(strict=True), args.repo.resolve(strict=True), args.root.resolve()
    root.mkdir(parents=True, mode=0o700, exist_ok=False)
    gate = Gate(root, binary, repo)
    binary_digest = digest(binary)
    compiler = args.compiler.resolve(strict=True)
    require(compiler.name == 'go' and compiler.parent.name == 'bin', 'stock compiler distribution bin/go required')
    private = root / 'private-inputs'
    private.mkdir(mode=0o700)
    copy_alias(compiler.parent.parent, private / 'compiler')
    copy_alias(args.runtime_root.resolve(strict=True), private / 'runtime')
    stage = repo / 'service/history' / ('.gomad-inspection-onebox-' + hashlib.sha256(str(root).encode()).hexdigest()[:16])
    require(not stage.exists(), 'staged source already exists')
    producer.write(root / 'unchanged-source-inputs.json', workloads.stage_inspection(repo, stage, 'onebox'))
    captured = producer.captured(repo, args.captured_config.resolve(strict=True))
    policy, scenario, installation = root / 'policy.json', root / 'scenario.json', root / 'installation.json'
    producer.write(policy, workloads.inspection_policy('onebox', captured))
    producer.write(scenario, workloads.inspection_scenario(['-test.run=^TestReplayInspectionWorkflowFailure$', '-test.v']))
    gate.run('install', [binary, 'install', '--compiler', private / 'compiler/bin/go', '--runtime-root', private / 'runtime',
                        '--cache', args.cache.resolve(), '--output', installation, '--profile', 'gomad.wasi-cooperative/v1',
                        '--config', policy, '--fuel', str(1 << 50), '--memory-bytes', str(4 << 30)], 0, 600)
    response = gate.run('explore', explore_command(binary, installation, root, repo, stage, scenario), 1, 2700, True)
    require((response['attempted'], response['failures'], response['retained_evidence']) == (1, 1, 1),
            'onebox must retain exactly one actual failure')
    manifests = list((root / 'artifacts').rglob('manifest.json'))
    require(len(manifests) == 1, 'ambiguous or missing retained failure')
    artifact = manifests[0].parent
    before = inventory(artifact)
    manifest = authenticate_payloads(artifact)
    detached = stage.with_name(stage.name + '-unavailable')
    stage.rename(detached)
    private.rename(root / 'private-inputs-unavailable')
    require(not stage.exists() and not private.exists(), 'preparation inputs remain available')
    gate.replay('detached-replay', artifact)
    report = gate.run('stopped-inspection', [binary, 'replay', '--stop=failure', '--instance=history', '--incarnation=1',
        '--dump=stacks', '--dump=heap', '--dump=temporal.mutable_states', '--dump=temporal.shards', '--probe=temporal.*.snapshot',
        '--debug-bytes', str(8 << 20), '--dump-bytes', str(4 << 20), '--debug-output', root / 'inspection', artifact], 0, 1800, True, True)
    diagnostics = validate_debug(root, report)
    require(inventory(artifact) == before and authenticate_payloads(artifact) == manifest, 'inspection modified retained evidence')
    gate.replay('reopened-replay', artifact)
    require(inventory(artifact) == before and authenticate_payloads(artifact) == manifest, 'reopen modified retained evidence')
    require(digest(binary) == binary_digest, 'release executable changed during gate')
    receipt = root / 'inspection-receipt.json'
    producer.write(receipt, dict(schema='gomad.rust-temporal-inspection/v1', owner='onebox', artifact=str(artifact),
        binary_sha256=binary_digest, source_detached=str(detached), private_inputs_detached=str(root / 'private-inputs-unavailable'),
        record_sha256=manifest['record_sha256'], report=report, artifact_sha256=before, diagnostic_sha256=diagnostics,
        commands={row['name']: dict(path=str(root / (row['name'] + '.command.json')),
                  sha256=digest(root / (row['name'] + '.command.json')), exit=row['exit']) for row in gate.commands},
        subgates=[dict(id='TestTemporalInspectionRetainedFailure/onebox', status='PASS', executed=True, exit=0)]))
    print(json.dumps(dict(receipt=str(receipt))))


if __name__ == '__main__':
    main()
