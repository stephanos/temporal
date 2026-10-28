#!/usr/bin/env python3
"""Retain terminal Cargo release proof, without touching any paired build target."""
import datetime,gzip,hashlib,json,pathlib,re,shutil,subprocess,tarfile,tempfile
HERE=pathlib.Path(__file__).resolve().parent
NAMED=HERE.parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
TARGET=ROOT/'.flow/tmp/fn258-standalone-release-0547321'
CAS=ROOT.parent/'.recovery/fn258-standalone-proof/objects'
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
 return h.hexdigest()
def write(name,x):(HERE/name).write_text(json.dumps(x,indent=2)+'\n')
HEAD=re.compile(r'^\s+(Running .+ \(([^\n]+)\)|Doc-tests ([^\n]+))$',re.M)
RESULT=re.compile(r'^test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored; (\d+) measured; (\d+) filtered out;',re.M)
def parse(text):
 heads=list(HEAD.finditer(text)); native=[];docs=[];excluded=[]
 for i,m in enumerate(heads):
  section=text[m.end():heads[i+1].start() if i+1<len(heads) else len(text)]
  results=[dict(zip(['passed','failed','ignored','measured','filtered_out'],map(int,r))) for r in RESULT.findall(section)]
  assert results,('missing result',m[1])
  record={'header':m[1],'header_line':text[:m.start()].count('\n')+1,'result_lines':[text[:m.end()].count('\n')+section[:r.start()].count('\n')+1 for r in RESULT.finditer(section)]}
  if m[2]:
   assert pathlib.Path(m[2]).parent==TARGET/'release/deps'
   first=re.search(r'^running (\d+) tests?$',section,re.M);assert first
   result=results[-1];assert int(first[1])==sum(result[k] for k in ['passed','failed','ignored','measured'])
   assert result['failed']==result['measured']==result['filtered_out']==0
   record.update(binary=m[2],counts=result,declared_tests=int(first[1]));native.append(record)
   excluded.extend({'enclosing_producer':m[2],'nested_control_summary':r} for r in results[:-1])
  else:
   record.update(crate=m[3],results=results);docs.append(record)
 def totals(rs):
  x={k:sum(r[k] for r in rs) for k in ['passed','failed','ignored','measured','filtered_out']};x['discovered']=x['passed']+x['failed']+x['ignored']+x['measured'];return x
 return {'native_producers':native,'doctest_producers':docs,'excluded_nested_summaries':excluded,'native_counts':totals([r['counts'] for r in native]),'doctest_counts':totals([r for d in docs for r in d['results']])}
receipt=json.loads((NAMED/'release.json').read_bytes());log=(NAMED/'release.log').read_text()
assert receipt['exit_code']==0 and receipt['source_status_after']=='' and receipt['source']=='05473210b48c3db862f22051eb7154ce98258f98'
assert sha(NAMED/'release.log')==receipt['log_sha256']
seals={name:sha(NAMED/name) for name in ['release.log','release.json']}
counts=parse(log);assert counts['native_counts']['passed']==908 and counts['native_counts']['ignored']==2 and len(counts['native_producers'])==69
assert counts['doctest_counts']['passed']==4 and counts['doctest_counts']['ignored']==1 and len(counts['excluded_nested_summaries'])==9
# Refuse missing top-level result; extra nested control summaries cannot inflate totals.
last=log.rfind('test result: ok.');bad=log[:last]+log[last:].replace('test result: ok.','test result: MISSING.',1)
try:parse(bad)
except AssertionError:missing_refused=True
else:missing_refused=False
assert missing_refused
m=HEAD.search(log);inject=m.end();altered=log[:inject]+log[inject:].replace('running 0 tests','running 0 tests\n\ntest result: ok. 999 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out;',1)
assert parse(altered)['native_counts']==counts['native_counts']
write('count-audit.json',dict(counts,ignored_test_lines=[l for l in log.splitlines() if re.match(r'^test .* \.\.\. ignored',l)],method='Cargo Running header segments: first declared count must equal final summary; earlier native summaries are nested controls. All doctest blocks retained, including merged run-pass/compile-fail blocks.',controls={'missing_result_refused':True,'injected_nested_999_not_added':True}))
records=[]
def retain(path,dest,role,copy=False):
 digest=sha(path);obj=CAS/digest
 if obj.exists():assert sha(obj)==digest
 else:
  with tempfile.NamedTemporaryFile(dir=CAS,delete=False) as f:
   tmp=pathlib.Path(f.name)
   with path.open('rb') as source:shutil.copyfileobj(source,f)
  assert sha(tmp)==digest;tmp.replace(obj)
 r={'source':str(path),'destination':dest,'role':role,'sha256':digest,'size':path.stat().st_size,'mode':oct(path.stat().st_mode&0o7777),'recovery_object':str(obj)};records.append(r)
 if copy:shutil.copyfile(path,HERE/dest)
 return r
retain(NAMED/'release.log','command.log','canonical terminal command log',True)
retain(NAMED/'release.json','command.json','canonical terminal command receipt',True)
retain(TARGET/'.rustc_info.json','rustc-info.json','actual target compiler identity',True)
source=pathlib.Path(receipt['cwd']);source_identity={'head':subprocess.check_output(['git','-C',str(source),'rev-parse','HEAD'],text=True).strip(),'tree':subprocess.check_output(['git','-C',str(source),'rev-parse','HEAD^{tree}'],text=True).strip(),'status':subprocess.check_output(['git','-C',str(source),'status','--porcelain'],text=True)}
assert source_identity['head']==receipt['source'] and source_identity['tree']==receipt['source_tree'] and not source_identity['status']
for name in ['Cargo.toml','Cargo.lock','rust-toolchain.toml','.cargo/config.toml']:
 path=source/name
 if path.is_file():retain(path,'source-'+name.replace('/','_'),'source manifest/profile/toolchain input',True)
files=[pathlib.Path(r['binary']) for r in counts['native_producers']];assert len(set(files))==69
# Dynamic dependency inspection is read-only; preserve required locally linked shared libraries too.
ldd=[];extra=set()
for path in files:
 proc=subprocess.run(['ldd',str(path)],capture_output=True,text=True)
 ldd.append({'binary':str(path),'exit':proc.returncode,'stdout':proc.stdout,'stderr':proc.stderr})
 for dep in re.findall(r'=> (/[^ ]+)',proc.stdout):
  d=pathlib.Path(dep)
  if TARGET in d.parents:extra.add(d)
write('dynamic-dependencies.json',ldd)
files+=sorted(extra-set(files));members=[]
with tempfile.NamedTemporaryFile(dir=CAS.parent,suffix='.tar.gz',delete=False) as f:archive=pathlib.Path(f.name)
with archive.open('wb') as raw:
 with gzip.GzipFile(filename='',fileobj=raw,mode='wb',mtime=0,compresslevel=1) as compressed:
  with tarfile.open(fileobj=compressed,mode='w|') as out:
   for path in files:
    assert path.is_file() and not path.is_symlink()
    info=out.gettarinfo(str(path),arcname=str(path.relative_to(TARGET)));info.mode&=0o7777;info.uid=info.gid=info.mtime=0;info.uname=info.gname=''
    with path.open('rb') as data:out.addfile(info,data)
    members.append({'path':info.name,'sha256':sha(path),'size':info.size,'mode':oct(info.mode)})
archive_record=retain(archive,'release-proof-binaries.tar.gz','all69 actual Cargo release producer binaries plus target-local dynamic dependencies');archive.unlink()
write('archive-manifest.json',{'object':archive_record,'members':members})
write('recovery-manifest.json',{'schema_version':1,'source':receipt['source'],'tree':receipt['source_tree'],'objects':records,'store':str(CAS)})
write('summary.json',{'observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source_identity':source_identity,'command':receipt['argv'],'exit':0,'seconds':receipt['seconds'],'profile':'release [optimized + debuginfo], actual Finished line137; nested UI/trybuild dev profiles are controls, not top-level profile','compiler':'actual target.rustc_info: rustc1.99.1(58cf8f9c3 2026-10-13),aarch64-unknown-linux-gnu,LLVM23.1.1','backend_mode':receipt['backend_mode'],'counts':{'native':counts['native_counts'],'doctests':counts['doctest_counts'],'total_discovered':915,'total_passed':912,'total_ignored':3,'excluded_nested_summaries':9},'coverage_limits':['Cargo release workspace tests do not run real Go-host/CGO-disabled guest parity or build a WASI guest. No such output claimed. Required054 gate and final paired gates own that parity proof.','Default backend mode means real model-TLC/Apalache coverage not established by this command; backend/control unit tests may run fake children. No native gate stage authority claimed.','Doctest temporary executable files are not reported by Cargo and not retained as named binaries; terminal per-block results retained.','System dynamic libraries are recorded with ldd, not copied as owned proof outputs.'], 'named_binary_count':69,'archive_members':len(members),'raw_member_bytes':sum(r['size'] for r in members),'compressed_bytes':archive_record['size'],'canonical_original_seals':seals,'no_build_cleanup_source_flow_or_pair_target_mutation':True})
assert all(sha(NAMED/name)==digest for name,digest in seals.items())
print(json.dumps({'records':len(records),'binaries':69,'members':len(members),'compressed_bytes':archive_record['size']}))
