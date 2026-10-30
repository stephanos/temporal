#!/usr/bin/env python3
"""Record the existing documented paired TLC setup before required validation."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time

HERE = Path(__file__).resolve().parent
PAIR = Path('/home/agent/.cache/codex-build/tmp/fn258-final-pair-3f5-054')
EXPECTED = {'temporal': '3f5f613578f2857b738a7792fad08f12b3ea0a41',
            'umpire': '05473210b48c3db862f22051eb7154ce98258f98'}


def git(repo, *args):
    return subprocess.check_output(['git', '--no-optional-locks', '-C', str(repo), *args], text=True).strip()


for name, revision in EXPECTED.items():
    assert git(PAIR / name, 'rev-parse', 'HEAD') == revision
    assert not git(PAIR / name, 'status', '--porcelain=v1', '--untracked-files=all')
script = PAIR / 'umpire/scripts/fetch-tlc.sh'
expected = re.search(r'^expected=([0-9a-f]{64})$', script.read_text(), re.M)[1]
jar = PAIR / 'umpire/build/tlc/tla2tools-1.7.4.jar'
row = {'source_pair': EXPECTED, 'cwd': str(PAIR / 'temporal'),
       'argv': ['/bin/bash', 'model/scripts/fetch-tlc.sh'],
       'documented_prerequisite': 'Consumer model/README.md TLC instructions',
       'source_script_sha256': hashlib.sha256(script.read_bytes()).hexdigest(),
       'jar_present_before': jar.exists(), 'expected_jar_sha256': expected,
       'scope': 'Explicit tool setup outside gate timing; no cold paired auto-provisioning claim.'}
environment = os.environ.copy()
environment.pop('BASH_ENV', None)
log = HERE / 'tlc-setup.log'
receipt = HERE / 'tlc-setup.json'
assert not log.exists() and not receipt.exists()
started = time.monotonic()
with log.open('wb') as stream:
    result = subprocess.run(row['argv'], cwd=PAIR / 'temporal', env=environment,
                            stdout=stream, stderr=subprocess.STDOUT)
row.update(exit_code=result.returncode, seconds=time.monotonic() - started,
           log=log.name, log_sha256=hashlib.sha256(log.read_bytes()).hexdigest())
if jar.is_file():
    row['actual_jar_sha256'] = hashlib.sha256(jar.read_bytes()).hexdigest()
row['source_status_after'] = {name: git(PAIR / name, 'status', '--porcelain=v1', '--untracked-files=all')
                              for name in EXPECTED}
receipt.write_text(json.dumps(row, indent=2) + '\n')
assert result.returncode == 0 and row.get('actual_jar_sha256') == expected
assert not any(row['source_status_after'].values())
print(json.dumps(row, indent=2))
