import datetime
import hashlib
import json
import pathlib
import re
import subprocess

root = pathlib.Path('tools/gomad3')
report = json.loads(pathlib.Path('.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/current-impact.json').read_text())
version = json.loads((root / 'toolchain/version/version.json').read_text())
manifest = json.loads((root / 'deterministicio/boundary/manifest.json').read_text())
pack_files = sorted((root / 'internal/compatibilitypack/packs').glob('*.json'))
packs = [json.loads(path.read_text()) for path in pack_files]
adapters = sorted((root / 'deterministicio').glob('*_adapter.go'))
patch = (root / 'toolchain/runtime/go1.27.1.patch').read_bytes()
overlay = sorted(path for path in (root / 'toolchain/runtime/overlay').rglob('*') if path.is_file())
prior = json.loads(pathlib.Path('.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-baseline.json').read_text())
counts = {
 'runtime_patch': {'lines': len(patch.splitlines()), 'bytes': len(patch), 'files': len(re.findall(rb'^--- a/', patch, re.MULTILINE))},
 'runtime_overlay': {'files': len(overlay), 'lines': sum(len(path.read_bytes().splitlines()) for path in overlay)},
 'dependency_adapters': {'modules': len(version['adapters']), 'sha256_anchor_literals': sum(len(re.findall(r'"sha256:[0-9a-f]{64}"', path.read_text())) for path in adapters)},
 'interception_fingerprints': {'intercepts': len(manifest['intercepts']), 'declarations_including_platform_overrides': sum(1 + len(entry.get('platform_overrides', {})) for entry in manifest['intercepts'])},
 'compatibility_packs': {'packs': len(packs), 'rules': sum(len(pack['rules']) for pack in packs), 'unique_module_version_pins': len({(module['path'], module['version']) for pack in packs for module in pack['activation'] + [rule['module'] for rule in pack['rules']]})},
 'clock_inventory': {'references': sum(pin['class'] == 'clock_inventory_reference' for pin in report['pins'])}
}
steps = {
 'runtime_patch': ['gomadtool patch-materialize', 'hand edit upstream source in scratch tree', 'gomadtool patch-regenerate', 'hand edit descriptor allowlists when paths change', 'make generate validate', 'make upgrade-dossier'],
 'runtime_overlay': ['hand edit overlay sources for the new Go runtime', 'hand edit descriptor overlay allowlist when paths change', 'make generate validate', 'make upgrade-dossier'],
 'dependency_adapters': ['hand edit toolchain/version/version.json exact version and sum', 'hand edit adapter rewrite anchors and source SHA-256 literals', 'hand edit prepared source-set hashes for each qualified platform', 'gomadtool version-generate', 'run adapter qualification tests on both hosts', 'hand edit and regenerate affected compatibility packs'],
 'interception_fingerprints': ['gomadtool boundary-generate', 'review generated fingerprint diff', 'approve exact boundary diff through upgrade-dossier'],
 'compatibility_packs': ['gomadtool compatibility-pack discover', 'gomadtool compatibility-pack review', 'person reviews source changes and exact approval digest', 'gomadtool compatibility-pack generate --approve-review', 'gomadtool compatibility-pack check'],
 'clock_inventory': ['run toolchain clock-inventory test on both hosts', 'hand edit reviewedHostClockReferences after reviewing each changed caller', 'rerun toolchain clock-inventory test on both hosts']
}
paths = [root / 'toolchain/runtime/go1.27.1.patch', root / 'toolchain/version/version.json', root / 'deterministicio/boundary/manifest.json', root / 'toolchain/clock_inventory_test.go'] + pack_files + adapters
result = {
 'schema': 'gomad3.pin-maintenance-baseline/v1',
 'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(),
 'recorded_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
 'measurement_command': 'python3 .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/measure-baseline.py',
 'counts': counts,
 'pre_tooling_bump_steps': steps,
 'manual_step_unit': 'one command invocation or one hand edit of a checked-in file; conditional path changes and per-platform qualification are named explicitly',
 'fn110_baseline': {'path': '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-baseline.json', 'head': prior['head'], 'patch': prior['patch'], 'overlay': prior['overlay']},
 'source_sha256': {path.as_posix(): hashlib.sha256(path.read_bytes()).hexdigest() for path in paths},
 'resolution': 'No module resolution or downloads required: exact module versions and go.sum identities bind the upstream source inventories; missing sums produce unknown, not unaffected.'
}
output = pathlib.Path('.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/baseline.json')
output.write_text(json.dumps(result, sort_keys=True, separators=(',', ':')) + '\n')
print(json.dumps(counts, sort_keys=True))
