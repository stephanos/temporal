#!/usr/bin/env python3
"""Every fixture entry point must refuse in an ordinary linked host build."""
import json
from pathlib import Path
import subprocess
import sys

binary, receipts = sys.argv[1:]
output = Path(receipts)
output.mkdir(mode=0o700, parents=True, exist_ok=False)
commands = ['fixture-record', 'fixture-publish', 'fixture-open', 'fixture-bundle',
            'fixture-execution-bundle', 'fixture-watchdog-cancel', 'fixture-bound',
            'fixture-pinned', 'fixture-root-substitution', 'fixture-payload-substitution',
            'fixture-payload-mutation', 'fixture-inspect-execution', 'fixture-inspect-details',
            'fixture-minimize-controller']
for command in commands:
    result = subprocess.run([binary, command], capture_output=True, text=True, timeout=5)
    (output / (command + '.json')).write_text(json.dumps({
        'argv': [binary, command], 'status': result.returncode,
        'stdout': result.stdout, 'stderr': result.stderr}, sort_keys=True))
    assert result.returncode == 2 and 'unsupported command' in result.stderr and not result.stdout, command
print(f'release fixture entry points absent: {len(commands)}/{len(commands)}')
for fault in ['crash:directory_created','pause:directory_created','audit:parent_durable','before_parent_sync',
              *[prefix+scope+'_'+point for scope in ('pool','layout')
                for prefix,point in [('crash:','file_created'),('crash:','file_partial'),
                                     ('crash:','before_file_sync'),('','file_sync_error')]]]:
    args=[binary,'fixture-publish','--root',str(output/'forbidden-store'),'--fault',fault]
    result=subprocess.run(args,capture_output=True,text=True,timeout=5)
    (output/('fault-'+fault.replace(':','-')+'.json')).write_text(json.dumps(dict(argv=args,status=result.returncode,stdout=result.stdout,stderr=result.stderr),sort_keys=True))
    assert result.returncode==2 and 'unsupported command' in result.stderr and not result.stdout,fault
assert not (output/'forbidden-store').exists()
