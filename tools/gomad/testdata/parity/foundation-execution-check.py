#!/usr/bin/env python3
import json,pathlib,subprocess,sys
binary,corpus,root=sys.argv[1:]
prompt=subprocess.run([binary,'fixture-watchdog-cancel','--corpus',corpus],capture_output=True,text=True,timeout=5)
assert prompt.returncode==0,(prompt.returncode,prompt.stderr)
result=subprocess.run([binary,'fixture-execution-bundle','--root',root,'--corpus',corpus],capture_output=True,text=True,timeout=60)
assert result.returncode==0,(result.returncode,result.stderr)
bundle=json.loads(result.stdout)
assert bundle['schema']=='gomad.foundation-result/v1'
assert len(bundle['records'])==12
print('execution classes, modeled streams, concurrent ledger and prompt watchdog cancellation passed')
