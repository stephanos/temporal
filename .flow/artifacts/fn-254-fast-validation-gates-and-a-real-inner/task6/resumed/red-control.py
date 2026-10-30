exec(open('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task6/resumed/validate.py').read().split("assert run('quick-controls-current'")[0])
source=candidate/'model/umpire/dev/xtask/src/quick/selection.rs'; original=source.read_bytes();before=b'if path == &root.join("dev/fixtures/fixtures/sample_kettle/model.rs") {';after=b'if path == &root.join("dev/fixtures/fixtures/sample_kettle/disabled-model.rs") {';assert original.count(before)==1
try:
 source.write_bytes(original.replace(before,after,1))
 code=run('kettle-selection-red',['mise','exec','--','cargo','test','--locked','-p','xtask','--lib','quick::tests::shared_kettle_model_seeds_source_consumers_and_reverse_guest','--','--exact','--nocapture'],candidate/'model/umpire')
 log=(out/'kettle-selection-red.log').read_text();assert code!=0 and 'shared_kettle_model_seeds_source_consumers_and_reverse_guest ... FAILED' in log
finally: source.write_bytes(original)
assert run('kettle-selection-green',['mise','exec','--','cargo','test','--locked','-p','xtask','--lib','quick::tests::shared_kettle_model_seeds_source_consumers_and_reverse_guest','--','--exact','--nocapture'],candidate/'model/umpire')==0
assert not subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
