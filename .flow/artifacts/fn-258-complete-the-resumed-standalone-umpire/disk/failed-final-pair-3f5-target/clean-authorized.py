#!/usr/bin/env python3
"""Root-authorized cleanup of two exact idle failed-proof output paths only."""
import datetime,hashlib,importlib.util,json,os,pathlib,shutil,stat,subprocess
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('old_pair_audit',HERE/'retain-audit.py');audit=importlib.util.module_from_spec(spec);spec.loader.exec_module(audit)
ROOT=audit.ROOT;TARGET=audit.TARGET;PAIR=audit.PAIR;ART=audit.ART
LITERAL='/home/agent/.cache/codex-build/fn258-final-pair-3f5-054'
TEMP=(ROOT.parent/'.recovery/fn258-standalone-proof/tmp1tdtki_h.tar.gz')
EXPECTEDTEMP='/Users/stephan/Workspace/skunkworks/umpire/.recovery/fn258-standalone-proof/tmp1tdtki_h.tar.gz'
def save(n,x):(HERE/n).write_text(json.dumps(x,indent=2)+'\n')
def snapshot(root):
 rows=[]
 for parent,dirs,files in os.walk(root,followlinks=False):
  for name in sorted(dirs+files):
   p=pathlib.Path(parent)/name;s=p.lstat();r={'path':str(p.relative_to(root)),'mode':oct(stat.S_IMODE(s.st_mode)),'inode':s.st_ino,'device':s.st_dev,'uid':s.st_uid,'gid':s.st_gid}
   if p.is_symlink():r.update(kind='symlink',link=os.readlink(p))
   elif p.is_file():r.update(kind='file',size=s.st_size,sha256=audit.sha(p))
   elif p.is_dir():r['kind']='directory'
   else:raise AssertionError(('unexpected special',str(p)))
   rows.append(r)
 return sorted(rows,key=lambda r:r['path'])
def directory_identity(p):
 s=p.stat();return {'path':str(p),'inode':s.st_ino,'device':s.st_dev,'uid':s.st_uid,'mode':oct(stat.S_IMODE(s.st_mode))}
def assert_idle(label):
 x=audit.owners();save(label,x);assert not x['matches'],x
 return x
assert str(TARGET)==LITERAL and TARGET.resolve()==TARGET and not TARGET.is_symlink()
assert str(TEMP)==EXPECTEDTEMP and TEMP.resolve()==TEMP and TEMP.is_file() and not TEMP.is_symlink()
verified=json.loads((HERE/'independent-recovery-verification.json').read_text());assert verified['result']=='pass' and verified['objects']==2 and verified['archive_members']==1931
expected=json.loads((HERE/'target-inventory.json').read_text())['records'];assert audit.inventory()==expected,'target changed since retained inventory'
initial_ownership=assert_idle('cleanup-ownership-before.json')
# All old source/Git/index/ref/generated/launcher/receipt bytes outside the target must survive.
protected=snapshot(PAIR);save('cleanup-protected-source-before.json',{'root':str(PAIR),'records':protected})
protected_roots=[ROOT.parent/'umpire',pathlib.Path('/home/agent/.cache/codex-build/fn258-standalone-0547321'),audit.CAS,ROOT.parent/'.recovery/fn258-guest-proof/objects']
protected_roots += list(pathlib.Path('/home/agent/.cache/codex-build').glob('fn258-final-pair-cde*'))+list(pathlib.Path('/home/agent/.cache/codex-build/tmp').glob('fn258-final-pair-cde*'))
protected_ids=[directory_identity(p) for p in protected_roots if p.is_dir()]
# Seal all initial target bytes, not merely selected proof bytes.
full=snapshot(TARGET);save('cleanup-target-content-before.json',{'target':LITERAL,'records':full})
assert all(r['uid']==os.getuid() for r in full)
allocated_before=int(subprocess.check_output(['du','-s','-B1',LITERAL]).split()[0]);df_before={str(p):subprocess.check_output(['df','-B1',str(p)],text=True) for p in [TARGET,ROOT]}
# No retained manifest may use the incomplete temporary as recovery authority.
references=[];scanned=[]
for p in ART.rglob('*manifest*.json'):
 try:x=json.loads(p.read_text())
 except (ValueError,UnicodeError):continue
 scanned.append(str(p.relative_to(ART)))
 def walk(v):
  if isinstance(v,dict):
   if v.get('recovery_object')==EXPECTEDTEMP:references.append(str(p))
   for q in v.values():walk(q)
  elif isinstance(v,list):
   for q in v:walk(q)
 walk(x)
assert not references
incomplete={'path':EXPECTEDTEMP,'size':TEMP.stat().st_size,'sha256':audit.sha(TEMP),'manifest_recovery_references':references,'manifest_files_checked':scanned}
assert incomplete['size']==338104923
save('cleanup-incomplete-temporary-before.json',incomplete)
assert_idle('cleanup-ownership-immediate.json')
commands=[]
# Only genuine Cargo targets: never manufacture a tag at orchestration root.
tags=json.loads((HERE/'cargo-target-tags.json').read_text())['actual_tags']
assert not (TARGET/'CACHEDIR.TAG').exists()
for row in tags:
 assert row['valid']
 tag=TARGET/row['path'];child=tag.parent
 assert TARGET in child.parents and tag.read_bytes().startswith(b'Signature: 8a477f597d28d172789f06886806bc55')
 argv=['mise','exec','--','cargo','clean','--manifest-path',str(PAIR/'umpire/Cargo.toml'),'--target-dir',str(child)]
 env=os.environ.copy();env.update(MISE_TRUSTED_CONFIG_PATHS=str(PAIR/'umpire'),MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_NOT_FOUND_AUTO_INSTALL='false')
 result=subprocess.run(argv,cwd=PAIR/'umpire',env=env,capture_output=True,text=True)
 commands.append({'argv':argv,'exit':result.returncode,'stdout':result.stdout,'stderr':result.stderr});save('cleanup-cargo-commands.json',commands)
 assert result.returncode==0,commands[-1]
# Record and hash all surviving owned output before guarded exact orchestration removal.
remainder=snapshot(TARGET);save('cleanup-remainder-before-removal.json',{'target':LITERAL,'records':remainder})
assert all(r['uid']==os.getuid() for r in remainder)
assert_idle('cleanup-ownership-before-remainder.json')
assert str(TARGET)==LITERAL and TARGET.resolve()==TARGET and not TARGET.is_symlink()
assert audit.inventory() and not any(r['kind']=='special' or r.get('mountpoint') for r in audit.inventory())
shutil.rmtree(TARGET)
# This exact failed temporary is the only extra path authorized for removal.
assert str(TEMP)==EXPECTEDTEMP and TEMP.is_file() and not TEMP.is_symlink() and audit.sha(TEMP)==incomplete['sha256']
previous_target=audit.TARGET;audit.TARGET=TEMP
try:temp_ownership=audit.owners()
finally:audit.TARGET=previous_target
save('cleanup-incomplete-temporary-ownership.json',temp_ownership);assert not temp_ownership['matches']
TEMP.unlink()
assert not TARGET.exists() and not TARGET.is_symlink() and not TEMP.exists()
assert snapshot(PAIR)==protected,'protected source/index/ref/receipt bytes changed'
for expected_id in protected_ids:assert directory_identity(pathlib.Path(expected_id['path']))==expected_id
save('cleanup-protected-roots.json',protected_ids)
# All proof authority remains recoverable after original-target deletion.
proc=subprocess.run(['python3','-B',str(HERE/'verify-recovery.py')],capture_output=True,text=True);assert proc.returncode==0,proc.stderr
receipt={'observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'authorization':'Root explicitly released exact idle FAILED old-pair target and exact incomplete temporary in current task; no source retirement.','removed_target':LITERAL,'removed_incomplete_temporary':EXPECTEDTEMP,'target_absent':True,'temporary_absent':True,'allocated_target_bytes_before':allocated_before,'incomplete_temporary_bytes_before':incomplete['size'],'cargo_tagged_children_cleaned':len(commands),'orchestration_remainder_reason':'Root has no real CACHEDIR.TAG; after six real tagged Cargo cleans, literal-path guarded owned remainder removal used full content inventory and immediate live checks. No fabricated tag.','df_before':df_before,'df_after':{str(ROOT):subprocess.check_output(['df','-B1',str(ROOT)],text=True),str(TARGET.parent):subprocess.check_output(['df','-B1',str(TARGET.parent)],text=True)},'source_pair_and_all_source_index_ref_launcher_receipt_bytes_unchanged':True,'protected_directory_identities_unchanged':True,'independent_recovery_after_cleanup':json.loads((HERE/'independent-recovery-verification.json').read_text()),'unreadable_system_fields':initial_ownership['unreadable'],'no_live_readable_target_references_before_mutation':True,'no_other_cache_source_ref_flow_main_mutation':True,'target_content_inventory_sha256':audit.sha(HERE/'cleanup-target-content-before.json'),'protected_inventory_sha256':audit.sha(HERE/'cleanup-protected-source-before.json')}
save('cleanup-receipt.json',receipt);print(json.dumps({'target_absent':True,'temporary_absent':True,'allocated_target_bytes_released':allocated_before,'temporary_bytes_released':incomplete['size'],'cargo_cleans':len(commands),'recovery_after':receipt['independent_recovery_after_cleanup']}))
