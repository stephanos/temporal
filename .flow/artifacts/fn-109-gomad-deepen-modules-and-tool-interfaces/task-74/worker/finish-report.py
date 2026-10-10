import datetime
import os
import signal
import subprocess
import sys
import time
sys.dont_write_bytecode = True
import run as r

inputs = sorted(r.OUTPUT.glob('*.log'))+sorted(r.OUTPUT.glob('*.json'))
argv = ['/usr/bin/python3',str(r.PACKET/'inspect-terminal-report.py')]
source_before, tools_before = r.sources(inputs),r.tools()
env = r.environment()
binding = {'observation_kind':'non-Go terminal observer; no Go-env execution','base':r.BASE,'head':r.git('rev-parse','HEAD').strip(),'argv':argv,'cwd':str(r.ROOT),'source_manifest':source_before,'tools_manifest':tools_before,'wrapper_sha256':r.sha(__file__),'environment_value_hashes':{key:r.hashlib.sha256(value.encode()).hexdigest() for key,value in env.items()},'wall_bound_seconds':600,'limitations':'Shared external gate may remain live. This observer proves owned termination only; no global lane emptiness is assumed.'}
binding_sha = r.save('final-terminal-report-binding.json',binding)
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
log = r.OUTPUT/'final-terminal-report.log'
timed_out = False
with log.open('xb') as stream:
    process = subprocess.Popen(argv,cwd=r.ROOT,env=env,stdout=stream,stderr=subprocess.STDOUT,start_new_session=True)
    try:
        rc = process.wait(timeout=600)
    except subprocess.TimeoutExpired:
        timed_out = True
        os.killpg(process.pid,signal.SIGTERM)
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid,signal.SIGKILL)
            process.wait()
        rc = 124
members = r.group_members(process.pid)
source_after, tools_after = r.sources(inputs),r.tools()
receipt = {'observation_kind':'non-Go terminal observer; no Go-env execution','binding_sha256':binding_sha,'argv':argv,'cwd':str(r.ROOT),'start_utc':start,'end_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'elapsed_seconds':time.monotonic()-clock,'exit_code':rc,'child_returncode':process.returncode,'timed_out':timed_out,'process_group':process.pid,'remaining_group_members':members,'all_commands_terminal':not members,'source_before_after_equal':source_before==source_after,'tools_before_after_equal':tools_before==tools_after,'source_after':source_after,'tools_after':tools_after,'log_sha256':r.sha(log),'go_settings_probe':'NOT_EXECUTED: explicit no-additional-Go boundary'}
r.save('final-terminal-report.json',receipt)
print(r.json.dumps({'exit_code':rc,'log_sha256':receipt['log_sha256'],'all_commands_terminal':not members,'go_executed':False},sort_keys=True))
assert not timed_out and not members and source_before == source_after and tools_before == tools_after
