import json,os,pathlib,subprocess,time,sys
W=pathlib.Path(__file__).resolve().parents[5];P=pathlib.Path(__file__).resolve().parent
E=dict(os.environ)
for k in list(E):
 if k=='BASH_ENV' or k.startswith(('CARGO_PROFILE_','RUSTFLAGS','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','CARGO_ENCODED_RUSTFLAGS')):E.pop(k)
E.update(CARGO_TARGET_DIR='/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2543-nextest/framework',CARGO_BUILD_JOBS='2',RUST_TEST_THREADS='2',CARGO_INCREMENTAL='0',UMPIRE_BACKENDS='require',UMPIRE_APALACHE='require',INSTA_UPDATE='always',TRYBUILD='overwrite')
filter='(package(=umpire-tlc) & (test(=controls::real_tlc_healthy_batches_use_at_most_two_model_launches) | test(=controls::real_tlc_failed_batches_preserve_independent_traces))) | (package(=umpire-tla) & test(=vector::real_vector_native_and_tlc_states_and_properties_agree)) | package(=umpire-testing) | (package(=umpire-fixtures) & test(=parameterized_world_tests::actual_tlc_replays_the_false_claim_in_the_same_selected_world)) | (package(=umpire-apalache) & test(=controls::real_later_start_fault_preserves_partial_binding)) | package(=umpire-dsl-macros)'
name='representative-temporal' if len(sys.argv)>1 and sys.argv[1]=='temporal' else 'representative'
cwd=W/'model/umpire'
if name.endswith('temporal'):
 cwd=W/'model'; E['CARGO_TARGET_DIR']=E['CARGO_TARGET_DIR'].removesuffix('framework')+'temporal'
 filter='(package(=umpire-temporal-tests) & (test(=activity_tests::actual_tlc_rejected_outcome_counterexample_replays_in_the_activity_model) | test(=activity_tests::actual_tlc_agrees_and_replays_retry_payload_counterexamples))) | (package(=temporal-xtask) & test(test_run::tests::))'
args=['mise','exec','--','cargo','nextest','run','--workspace','--locked','--fail-fast','--run-ignored','default','--test-threads','8','--status-level','all','--final-status-level','all','--success-output','never','--failure-output','immediate-final','-E',filter]
start=time.monotonic();events=[];peak=0;owners_peak=0;rss_peak=0;last=None;samples=0
with (P/(name+'.out')).open('wb') as out,(P/(name+'.err')).open('wb') as err:
 p=subprocess.Popen(args,cwd=cwd,env=E,stdout=out,stderr=err)
 while p.poll() is None:
  raw=subprocess.check_output(['ps','-eo','pid,ppid,rss,comm,args'],text=True)
  processes={}
  for line in raw.splitlines()[1:]:
   v=line.strip().split(None,4)
   if len(v)==5:
    pid,ppid,rss,comm,cmd=v;processes[int(pid)]=(int(ppid),int(rss),comm,cmd)
  children={p.pid}
  for _ in range(15):children|={pid for pid,v in processes.items() if v[0] in children}
  java=[];owners=set()
  for pid in children:
   if pid not in processes or processes[pid][2]!='java':continue
   q=pid;owner=None
   for _ in range(15):
    if q not in processes:break
    if '/debug/deps/' in processes[q][3]:owner=q
    q=processes[q][0]
   owners.add(owner);java.append({'pid':pid,'rss_kib':processes[pid][1],'owner':owner,'argv':processes[pid][3]})
  host_java=[pid for pid,v in processes.items() if v[2]=='java']
  peak=max(peak,len(java));owners_peak=max(owners_peak,len(owners));rss_peak=max(rss_peak,sum(x['rss_kib'] for x in java));samples+=1
  state=tuple((x['pid'],x['owner']) for x in java)
  if state!=last:events.append({'seconds':time.monotonic()-start,'java':java,'host_java_pids':host_java});last=state
  time.sleep(.05)
 code=p.wait()
r={'argv':args,'cwd':str(cwd),'env':{k:E[k] for k in ['CARGO_BUILD_JOBS','RUST_TEST_THREADS','CARGO_INCREMENTAL','UMPIRE_BACKENDS','UMPIRE_APALACHE','INSTA_UPDATE','TRYBUILD']},'exit':code,'seconds':time.monotonic()-start,'java_peak':peak,'jvm_test_owner_peak':owners_peak,'aggregate_jvm_rss_peak_kib':rss_peak,'sampling_interval_seconds':.05,'samples':samples,'events':events,'host_budget':'Task11 yielded exclusive JVM lease; one workspace invocation at a time. Group max1 allows two nested TLC recovery JVMs, each default Xmx512m. Sampling is observed peak, source/config establish bounds.'}
(P/(name+'.json')).write_text(json.dumps(r,indent=2)+'\n');print({k:r[k] for k in ['exit','seconds','java_peak','jvm_test_owner_peak','aggregate_jvm_rss_peak_kib']});assert code==0;assert peak>0;assert peak<=2;assert owners_peak==1
