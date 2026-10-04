import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import subprocess
import sys
import tarfile
import time

REPO = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
ARTIFACTS = Path(__file__).resolve().parent
SCRATCH = Path('/tmp/d26-bridge-pin.39dePDRJ')
REVISION = '58b718565044ab3bc3385d3323ee908a6d54328e'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
ENV = ['env', 'GOWORK=off', 'GOTOOLCHAIN=local', 'GOMAXPROCS=2']


def capture(name, command, cwd):
    start = time.monotonic()
    result = subprocess.run(command, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    elapsed = time.monotonic() - start
    (ARTIFACTS / (name + '.log')).write_bytes(result.stdout)
    record = {'command': command, 'cwd': str(cwd), 'exit_code': result.returncode,
              'elapsed_seconds': elapsed, 'log_sha256': hashlib.sha256(result.stdout).hexdigest()}
    (ARTIFACTS / (name + '.json')).write_text(json.dumps(record, indent=2) + '\n')
    print(json.dumps(record))
    print(result.stdout.decode())
    return result


if sys.argv[1] == 'archive':
    assert SCRATCH.is_dir() and not list(SCRATCH.iterdir())
    archive = subprocess.check_output(['git', 'archive', '--format=tar', REVISION,
                                      'tools/gomad3', 'tools/gomad3sim'], cwd=REPO)
    members = []
    with tarfile.open(fileobj=io.BytesIO(archive), mode='r:') as source:
        for member in source.getmembers():
            path = PurePosixPath(member.name)
            assert not path.is_absolute() and '..' not in path.parts
            assert member.name == str(path) or member.name == str(path) + '/'
            assert member.isdir() or member.isfile(), (member.name, member.type)
            assert str(path) in ('tools', 'tools/gomad3', 'tools/gomad3sim') or str(path).startswith(('tools/gomad3/', 'tools/gomad3sim/'))
            members.append({'path': member.name, 'type': 'directory' if member.isdir() else 'regular', 'size': member.size})
        assert any(member['path'] == 'tools/gomad3/go.mod' for member in members)
        assert any(member['path'] == 'tools/gomad3sim/runtime_time_toolchain.go' for member in members)
    (SCRATCH / 'source.tar').write_bytes(archive)
    subprocess.run(['tar', '-xf', str(SCRATCH / 'source.tar'), '-C', str(SCRATCH)], check=True)
    bridge = 'tools/gomad3/target/internal/capabilitypolicy/simulation_bridge.go'
    tests = 'tools/gomad3/target/internal/capabilitypolicy/policy_test.go'
    for path in (bridge, tests):
        destination = SCRATCH / 'baseline' / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes((SCRATCH / path).read_bytes())
    hashes = {path: hashlib.sha256((SCRATCH / path).read_bytes()).hexdigest() for path in
              (bridge, tests, 'tools/gomad3/target/capability_test.go', 'tools/gomad3sim/runtime_time_toolchain.go', 'tools/gomad3/go.mod', 'tools/gomad3/go.sum')}
    ownership = capture('d26-ownership', ['git', 'show', 'ad90b462e0f947b88f0190c0b4d0f60940ff5aec', '--', 'tools/gomad3sim/runtime_time_toolchain.go'], REPO)
    predecessor = subprocess.check_output(['git', 'show', 'ad90b462^:tools/gomad3sim/runtime_time_toolchain.go'], cwd=REPO)
    d26 = subprocess.check_output(['git', 'show', 'ad90b462:tools/gomad3sim/runtime_time_toolchain.go'], cwd=REPO)
    assert hashlib.sha256(predecessor).hexdigest() == 'e6402e8fbfc848c7360d19a1b77de93e841d64870ab625433fac8a47de83d23d'
    assert d26 == (SCRATCH / 'tools/gomad3sim/runtime_time_toolchain.go').read_bytes()
    record = {'revision': REVISION, 'scratch': str(SCRATCH), 'archive_sha256': hashlib.sha256(archive).hexdigest(),
              'member_count': len(members), 'regular_members': sum(member['type'] == 'regular' for member in members),
              'members_checked': 'Only clean relative paths under tools/gomad3 and tools/gomad3sim; directories and regular members; no links or special entries.',
              'baseline_sha256': hashes, 'd26_commit': 'ad90b462e0f947b88f0190c0b4d0f60940ff5aec',
              'predecessor_runtime_time_sha256': hashlib.sha256(predecessor).hexdigest(),
              'd26_runtime_time_sha256': hashlib.sha256(d26).hexdigest(), 'head_runtime_equals_d26': True}
    (ARTIFACTS / 'source.json').write_text(json.dumps(record, indent=2) + '\n')
    print(json.dumps(record, indent=2))
elif sys.argv[1] in ('red', 'green'):
    capture(sys.argv[1], ENV + [GO, 'test', '-tags', 'test_dep', '-count=1', '-v', './target',
                               '-run', '^TestBuiltInSimulationLinknamesPinCurrentFirstPartySources$'], SCRATCH / 'tools/gomad3')
elif sys.argv[1] in ('controls-before', 'controls-after'):
    capture(sys.argv[1], ENV + [GO, 'test', '-tags', 'test_dep', '-count=1', '-v', './target/internal/capabilitypolicy',
                               '-run', '^TestBuiltInSimulation(LinknamesRequireExactFirstPartySource|TimeBridgeRequiresExactSource)$'], SCRATCH / 'tools/gomad3')
elif sys.argv[1] == 'finalize':
    paths = ['tools/gomad3/target/internal/capabilitypolicy/simulation_bridge.go',
             'tools/gomad3/target/internal/capabilitypolicy/policy_test.go']
    changed = []
    with tarfile.open(SCRATCH / 'source.tar', mode='r:') as archive:
        for member in archive.getmembers():
            if member.isfile() and archive.extractfile(member).read() != (SCRATCH / member.name).read_bytes():
                changed.append(member.name)
    assert sorted(changed) == sorted(paths), changed
    patch = b''
    for path in paths:
        result = subprocess.run(['diff', '-u', '--label', 'a/' + path, '--label', 'b/' + path,
                                 str(SCRATCH / 'baseline' / path), str(SCRATCH / path)], stdout=subprocess.PIPE)
        assert result.returncode == 1
        patch += result.stdout
    assert patch.count(b'@@ ') == 2
    (ARTIFACTS / 'bridge-pin-repair.patch').write_bytes(patch)
    check = capture('patch-dry-run', ['patch', '--dry-run', '-p1', '-i', str(ARTIFACTS / 'bridge-pin-repair.patch')], SCRATCH / 'baseline')
    assert check.returncode == 0
    formatting = capture('format-check', [str(Path(GO).with_name('gofmt')), '-l'] + paths, SCRATCH)
    assert formatting.returncode == 0 and formatting.stdout == b''
    identity = capture('toolchain-identity', ENV + [GO, 'version'], SCRATCH)
    assert identity.returncode == 0
    ancestor = capture('d26-ancestry', ['git', 'merge-base', '--is-ancestor', 'ad90b462e0f947b88f0190c0b4d0f60940ff5aec', REVISION], REPO)
    assert ancestor.returncode == 0
    source = json.loads((ARTIFACTS / 'source.json').read_text())
    final_hashes = {path: hashlib.sha256((SCRATCH / path).read_bytes()).hexdigest() for path in source['baseline_sha256']}
    for path in paths:
        assert (REPO / path).read_bytes() == (SCRATCH / 'baseline' / path).read_bytes()
    verification = {name: json.loads((ARTIFACTS / (name + '.json')).read_text()) for name in
                    ('red', 'controls-before', 'green', 'controls-after', 'patch-dry-run', 'format-check', 'toolchain-identity', 'd26-ancestry')}
    assert verification['red']['exit_code'] == verification['controls-before']['exit_code'] == 1
    assert verification['green']['exit_code'] == verification['controls-after']['exit_code'] == 0
    manifest = {'task': 'fn-105-gomad-follow-ups-deferred-scope.31', 'requirement': 'R26',
                'base_revision': REVISION, 'scratch': str(SCRATCH),
                'patch': str(ARTIFACTS / 'bridge-pin-repair.patch'), 'patch_sha256': hashlib.sha256(patch).hexdigest(),
                'changed_files': paths, 'baseline_sha256': source['baseline_sha256'], 'final_sha256': final_hashes,
                'all_other_archived_regular_sources_unchanged': True, 'shared_patch_targets_still_equal_baseline': True,
                'production_change': 'Only runtime_time_toolchain.go exact digest and ordered Current directive in builtInSimulationLinknames.',
                'predicate_unchanged': True, 'added_test': 'TestBuiltInSimulationTimeBridgeRequiresExactSource',
                'controls': ['wrong hash', 'predecessor hash', 'missing Current', 'reordered directives', 'malformed linkname',
                             'foreign source', 'foreign import', 'lookalike import', 'foreign test variant', 'wrong module', 'non-main module', 'replaced module'],
                'toolchain': GO, 'toolchain_sha256': hashlib.sha256(Path(GO).read_bytes()).hexdigest(),
                'environment': {'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOMAXPROCS': '2'},
                'verification': verification,
                'limitations': 'Host-side focused stock Go tests on linux/arm64 only. No patched runtime/native qualification, broad suite, review verdict, integration, Git/index mutation, or Flow status mutation.',
                'routing': 'Existing task-31/bridge-pin-repair-state.json judgment retained; no_key, explicit gpt-6.1-sol high retained; actual model unknown. No rejudgment.'}
    (ARTIFACTS / 'handover.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps({'patch_sha256': manifest['patch_sha256'], 'final_sha256': final_hashes, 'changed_files': changed}, indent=2))
else:
    raise SystemExit('unknown phase')
