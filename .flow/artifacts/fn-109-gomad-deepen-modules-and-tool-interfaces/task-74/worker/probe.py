import sys
sys.dont_write_bytecode = True
import run as r

admission = r.PRIMARY/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/deadline-observation-admission-20261010.md'
assert r.sha(admission) == 'a6ba203a35db81e2e1d96879be57de114612c57a2b4d89a15da2ecc0bec43ccc'
assert r.run('diagnostic-precondition',['/usr/bin/python3',str(r.PACKET/'inspect-lane.py')],r.ROOT,inputs=[admission]) == 0
trace = r.OUTPUT/'deadline-observation-20261010.trace'
assert not trace.exists()
rc = r.run('diagnostic-ordinary-trace',['go','test','-tags','test_dep','-count=1','-json','-trace',str(trace),'./runner'],r.ROOT/'tools/gomad3',inputs=[admission])
r.save('diagnostic-trace-output.json',{'trace_path':str(trace),'trace_sha256':r.sha(trace),'trace_size_bytes':trace.stat().st_size,'ordinary_exit_code':rc,'admission_sha256':r.sha(admission),'limitation':'Instrumented diagnostic only; does not replace failed original ordinary receipt or certify uninstrumented acceptance.'})
