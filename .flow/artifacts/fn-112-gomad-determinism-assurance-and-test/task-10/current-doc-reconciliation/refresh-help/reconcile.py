import hashlib
import json
import pathlib
import shlex
import subprocess

OUT = pathlib.Path(__file__).resolve().parent
ROOT = OUT.parents[5]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(name):
    return json.loads((OUT / name).read_bytes())


original = read('evidence.json')
admission = read('admission.json')
before = read('inputs-before.json')
after = read('inputs-after.json')
log = (OUT / 'build.stderr').read_text().splitlines()
work = pathlib.Path(log[0].removeprefix('WORK='))
private = pathlib.Path(admission['private_directory'])
assert work.is_relative_to(private)
assert ROOT == pathlib.Path('/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/task11210')
current = ROOT / 'tools/gomad3'
source = {}
generated = {}
packages = []
missing = []
for line in log:
    if line.startswith('cd '):
        current = pathlib.Path(shlex.split(line.replace('$WORK', str(work)))[1])
        continue
    if '/compile ' not in line:
        continue
    args = shlex.split(line.replace('$WORK', str(work)))
    if not args[0].endswith('/compile'):
        continue
    packages.append({'package': args[args.index('-p') + 1], 'compiler': args[0]})
    for arg in args:
        if not arg.endswith('.go'):
            continue
        path = pathlib.Path(arg)
        if not path.is_absolute():
            path = current / path
        key = str(path)
        if key in before:
            source[key] = {'before_sha256': before[key], 'after_sha256': after.get(key)}
        elif path.is_relative_to(private) and path.is_file():
            generated[key] = {'generated_build_output_postimage_sha256': sha(path), 'bytes': path.stat().st_size}
        else:
            missing.append(key)
tools = {}
for path, digest in before.items():
    if '/pkg/tool/linux_arm64/' in path or path in [admission['go_executable'], '/usr/bin/python3', '/usr/bin/git']:
        tools[path] = {'before_sha256': digest, 'after_sha256': after.get(path)}
retained = {}
for path in OUT.iterdir():
    if path.is_file():
        retained[path.name] = sha(path)
sealed = {}
for name in ['check_source.py', 'current-bindings.json', 'handover.md', 'evidence.json']:
    sealed[name] = sha(OUT.parent / name)
expected_sealed = {
    'check_source.py': '6513536206150a745b26a639228581d227557d7199f9edb3083aad8beb1b33a7',
    'current-bindings.json': 'a280ad8e185eb0517dbf37107f52950ec0265f2e1249d46dd1d0947bb258228b',
    'handover.md': 'e7fe47063837f28e2032dac178332c142601edbea3d0ef1461482b628c33e0a0',
    'evidence.json': '4d47f440c9ecaf79137f51bab0ba94f967f815937d7d404779cee5876c67c857',
}
binary_before = read('binary-before.json')
binary_after = read('binary-after.json')
for name in ['build', 'refresh-help']:
    record = read(name + '.json')
    for stream in ['stdout', 'stderr']:
        assert sha(OUT / (name + '.' + stream)) == record[stream + '_sha256']
result = {
    'head': original['head_after'], 'commits': [], 'task_status': 'in_progress',
    'tests': ['single stock go build -trimpath -o private/gomadtool ./cmd/gomadtool (exit 0)', 'private/gomadtool compatibility-pack refresh -h (exit 2)', 'capture.py (exit 1; generated $WORK path parser failed)', 'reconcile.py (read-only postprocessing; no command rerun)'],
    'build': read('build.json'), 'refresh_help': read('refresh-help.json'),
    'input_manifest_count': len(before), 'preexisting_inputs_unchanged': before == after,
    'compiled_package_count': len(packages), 'compiled_packages': packages,
    'prebound_consumed_go_source_count': len(source), 'prebound_consumed_go_sources': source,
    'generated_go_postimage_count': len(generated), 'generated_go_postimages': generated,
    'unresolved_go_inputs': missing, 'prebound_tools': tools,
    'binary_before': binary_before, 'binary_after': binary_after,
    'aggregate_input_count': original['aggregate_input_count'],
    'aggregate_manifest_sha256': original['aggregate_manifest_sha256'],
    'aggregate_mismatches': original['aggregate_mismatches'],
    'sealed_packet_hashes': sealed, 'sealed_packet_matches_review': sealed == expected_sealed,
    'retained_artifact_sha256': retained,
    'limitations': [
        'Generated cgo Go files have postimage hashes only; they did not exist before the build.',
        'Host C compiler, system headers and libc compilation/link inputs were not prebound; the retained -x log identifies commands, not a complete reproducible compilation environment.',
        'Pre/post inventories bind Go module sources, transitive Go host packages and stock Go tool executables, not every external compilation input.',
        'The fresh observation covers this current refresh -h route only. Both historical hash-only refresh gaps remain open in their sealed receipts.',
        'No patched runtime, imported-dependency/native qualification, determinism bound, global test/lint acceptance, formal review, or task completion is established.',
    ],
}
result['checks_pass'] = bool(
    not missing and len(generated) == 14 and len(source) == 1662
    and len(packages) == 301 and before == after
    and all(record['before_sha256'] == record['after_sha256'] for record in source.values())
    and all(record['before_sha256'] == record['after_sha256'] for record in tools.values())
    and binary_before == binary_after and sha(pathlib.Path(binary_after['path'])) == binary_after['sha256']
    and result['sealed_packet_matches_review'] and not result['aggregate_mismatches']
    and result['build']['exit_code'] == 0 and result['refresh_help']['exit_code'] == 2
    and result['build']['terminal'] and result['refresh_help']['terminal']
    and not result['build']['timed_out'] and not result['refresh_help']['timed_out']
    and result['refresh_help']['stdout_bytes'] == 0
    and original['refresh_flags'] == original['expected_refresh_flags']
    and not subprocess.check_output(['/usr/bin/git', 'diff', '--name-only', original['head_after']], cwd=ROOT))
with (OUT / 'reconciled-evidence.json').open('x') as stream:
    json.dump(result, stream, indent=2, sort_keys=True)
    stream.write('\n')
print(json.dumps({'checks_pass': result['checks_pass'], 'prebound_consumed_go_sources': len(source), 'generated_go_postimages': len(generated), 'compiled_packages': len(packages), 'unresolved_go_inputs': missing, 'sealed_packet_matches_review': result['sealed_packet_matches_review']}, sort_keys=True))
raise SystemExit(0 if result['checks_pass'] else 1)
