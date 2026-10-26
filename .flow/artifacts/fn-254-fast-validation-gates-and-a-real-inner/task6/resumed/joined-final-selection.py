exec(open('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task6/resumed/joined-validate.py').read().split("assert not subprocess.check_output")[0])
revision=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip();assert revision.startswith('cdfd3720c8')
assert not subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True)
assert run('joined-final-fetch',['git','fetch','origin','umpire'],candidate)==0
assert run('joined-final-checkout',['git','checkout','--detach',revision],candidate)==0
source=candidate/'model/umpire/dev/fixtures/fixtures/sample_kettle/model.rs';original=source.read_bytes();edit=original.replace(b'//! Model authored with the attribute surface.',b'//! Model authored with the attribute surface (final selection content edit).',1);assert edit!=original
dirty=candidate/'.flow/artifacts/fn254-joined-dirty-evidence.json';assert not dirty.exists()
try:
 source.write_bytes(edit);dirty.write_text('{"proof":"final dirty evidence selection"}\n')
 assert run('joined-final-selection',['mise','exec','--','cargo','xtask','quick','--select-only'],candidate/'model/umpire')==0
 log=(out/'joined-final-selection.log').read_text();selection=next(json.loads(x) for x in log.splitlines() if x.startswith('{"base":'))
 assert [p for w in selection['workspaces'] for p in w['packages']]==['umpire-apalache','umpire-fixtures','umpire-kettle-guest','umpire-reference-kettle','umpire-suite','umpire-tla']
 assert any('ignored known non-model input:' in reason and str(dirty) in reason for reason in selection['reasons'])
 (out/'joined-final-selection-facts.json').write_text(json.dumps({'commit':revision,'selection':selection,'source_original_sha256':hashlib.sha256(original).hexdigest(),'source_edited_sha256':hashlib.sha256(edit).hexdigest(),'scope':'Final follow-up affects documentation fallback only; full shipping cold/warm performed on0c83ab3d1e, final-source selection retested here. No cache/domain/lock changes.'},indent=2)+'\n')
finally:
 source.write_bytes(original);dirty.unlink();status=subprocess.check_output(['git','status','--porcelain'],cwd=candidate,text=True);assert not status
 paths=list(json.loads((out/'joined-source.json').read_text())['files'])
 hashes={p:hashlib.sha256((candidate/p).read_bytes()).hexdigest() for p in paths};assert all(hashes[p]==hashlib.sha256((root/p).read_bytes()).hexdigest() for p in paths)
 (out/'joined-final-projection.json').write_text(json.dumps({'commit':revision,'status':status,'source_restored':source.read_bytes()==original,'files':hashes,'integrated_source_equal':True,'changed_since_shipping_source':[p for p,h in hashes.items() if json.loads((out/'joined-source.json').read_text())['files'][p]!=h],'free_bytes':shutil.disk_usage(root).free},indent=2)+'\n')
