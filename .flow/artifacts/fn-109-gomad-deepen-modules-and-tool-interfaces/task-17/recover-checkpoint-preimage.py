#!/usr/bin/env python3
"""Recover the exact Runner selector before task 18's filesystem additions."""
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SOURCE = 'tools/gomad3/runner/internal/execution/simulation_root_integration_test.go'
CURRENT = '02544185057eff83b65b288b6062c374376b3a10137dc526405c222e052ec682'
EXPECTED = 'bc4c86b6210232026f0eb94edbee209fa9817a62b66d58a2da7b627e9bbbf833'


def main():
    data = (ROOT / SOURCE).read_bytes()
    assert hashlib.sha256(data).hexdigest() == CURRENT
    names = ['PartialIOAndLifetime', 'DirectoryAndChdir', 'AccessAndBounds',
             'ReplayRejectsWriteBeforeMutation']
    for name in names:
        line = ('\t\t"TestProcessFilesystemHandle' + name + '",\n').encode()
        assert data.count(line) == 1
        data = data.replace(line, b'')
    assert hashlib.sha256(data).hexdigest() == EXPECTED
    scratch = Path(subprocess.check_output([
        'mktemp', '-d', '/tmp/fn109-task17-preimage.XXXXXXXX']).decode().strip())
    destination = scratch / SOURCE
    destination.parent.mkdir(parents=True)
    destination.write_bytes(data)
    report = dict(task=17, path=SOURCE, current_sha256=CURRENT,
                  recovered_sha256=EXPECTED, scratch=str(scratch),
                  removed_task18_selectors=['TestProcessFilesystemHandle' + name for name in names],
                  provenance='Reverse only four task-18 selector additions; complete corrected task-17 evidence hash matches.',
                  shared_source_writes=False, git_mutations=False,
                  native_qualification=False)
    (OUT / 'checkpoint-preimage.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
