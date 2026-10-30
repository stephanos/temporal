"""Curate the single resumed proof only after terminal watchdog acceptance."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys

BASE = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
WATCHDOG = BASE / '.flow/tmp/fn25416-resume-20261028-watchdog-01'
STATIC = BASE / '.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task16/resume-20261028'
OLD = BASE / '.flow/tmp/fn25416-provider-validation'

def read(path):
    return json.loads(path.read_bytes())

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    result = read(WATCHDOG / 'watchdog-result.json')
    assert result['terminal'] and result['accepted'] and not result['rejected']
    assert result['exit_code'] == 0 and result['validation_accepted']
    launch = read(STATIC / 'resume.json')
    for row in launch['files']:
        assert digest(WATCHDOG / row['path']) == row['sha256'], row['path']
    validation = read(WATCHDOG / 'proof-output/validation.json')
    assert validation['accepted'] and validation['head'] == launch['source_head']
    assert validation['cleanup']['all_observed_owned_pids_absent']
    assert result['owned_cleanup']['all_observed_owned_pids_absent']
    assert read(WATCHDOG / 'proof-output/source-before.json') == read(WATCHDOG / 'proof-output/source-after.json')
    samples = [json.loads(line) for line in (WATCHDOG / 'space-samples.jsonl').read_text().splitlines()]
    assert samples and all(row['free'] >= 30 * 1024**3 for sample in samples for row in sample['spaces'])
    admission = read(WATCHDOG / 'admission.json')
    assert all(row['free'] >= 50 * 1024**3 for row in admission['admission_spaces'])
    focused = {
        'focused-repaired.log': ('f4614e854f3930b1a4817b6cc96bd84d0f2edd8b4b47b8666eafe145f8392c00', '267 tests run: 267 passed'),
        'system-gate-controls.log': ('f1201f59ac748558e275f115d6e5695a47c1b63caabd52b3a5b54cee002a4323', '16 tests run: 16 passed'),
        'path-list-green.log': ('b4052d943dfc38648183b0b2fe3df098dbc4738b218619f82225a85f2be9ac9d', '5 tests run: 5 passed'),
    }
    for name, (expected, sentinel) in focused.items():
        path = OLD / name
        assert digest(path) == expected, name
        assert sentinel in path.read_text(), name
    output = OLD / 'curated-task16'
    assert not output.exists(), 'curation output already exists'
    subprocess.run([sys.executable, str(OLD / 'curate.py'), str(WATCHDOG / 'proof-output')], check=True, cwd=BASE)
    extra = output / 'resumed-watchdog'
    extra.mkdir()
    for name in ['watchdog.py', 'test_watchdog.py', 'controls.json', 'controls.log', 'admission.json', 'live-handle.json', 'watchdog-result.json', 'space-samples.jsonl', 'proof.stdout.log', 'proof.stderr.log']:
        shutil.copy2(WATCHDOG / name, extra / name)
    shutil.copy2(Path(__file__), extra / 'curate_guarded.py')
    manifest = [{'path': str(p.relative_to(extra)), 'bytes': p.stat().st_size, 'sha256': digest(p)} for p in sorted(extra.iterdir()) if p.is_file()]
    (extra / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps({'output': str(output), 'watchdog_terminal_accepted': True, 'extra_files': len(manifest)}))

if __name__ == '__main__':
    main()
