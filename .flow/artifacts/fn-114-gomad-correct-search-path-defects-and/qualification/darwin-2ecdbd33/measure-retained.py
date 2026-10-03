import datetime,hashlib,json,pathlib,os,subprocess
root=pathlib.Path('/Users/stephan/Workspace/temporal/gomad')
out=root/'.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/qualification/darwin-2ecdbd33'
store=root/'tools/gomad3/.toolchain/fn114-2ecdbd33-temporal-qualification'
meta=json.loads((out/'representative-final.json').read_text())
report=json.loads((out/'representative-qualification-set.json').read_text())
start=datetime.datetime.fromisoformat(meta['started_at'].replace('Z','+00:00')).timestamp()
campaigns=[p for p in (store/'v1').glob('campaign-*') if p.stat().st_mtime>=start]
old=[p for p in (store/'v1').glob('campaign-*') if p not in campaigns]
qualifications=[p for p in (store/'qualifications/v1').iterdir() if p.stat().st_mtime>=start]
assert len(campaigns)==112,(len(campaigns),len(old))
assert len(qualifications)==56,len(qualifications)
selected=campaigns+qualifications+[store/'targets']
paths=[]
for p in selected:
 paths.extend([x for x in p.rglob('*') if x.is_file()] if p.is_dir() else [p])
seen=set();distinct=0;every=0
for p in paths:
 s=p.stat();every+=s.st_size
 if (s.st_dev,s.st_ino) not in seen: seen.add((s.st_dev,s.st_ino));distinct+=s.st_size
pool=list((store/'targets').glob('sha256-*'));poolbytes=sum(p.stat().st_size for p in pool if p.is_file())
du=subprocess.check_output(['du','-skc',*[str(p) for p in selected]],text=True).splitlines()[-1]
on_disk_kib=int(du.split()[0])
manifestpaths=[p for p in paths if p.name=='manifest.json'];assert len(manifestpaths)==112
m=[json.loads(p.read_text()) for p in manifestpaths]
assert all(v['toolchain']['build_key']==meta['build_key'] for v in m)
summed_artifact_bytes=sum(sum(int(f['size']) for f in v['files']) + p.stat().st_size for p,v in zip(manifestpaths,m))
assert summed_artifact_bytes==int(report['artifact_bytes']),summed_artifact_bytes
result=dict(schema='gomad3.retained-measurement/v1',platform='darwin/arm64',build_key=meta['build_key'],report_sha256=hashlib.sha256((out/'representative-qualification-set.json').read_bytes()).hexdigest(),started_at=meta['started_at'],finished_at=meta['finished_at'],artifact_root=str(store),selection='112 campaigns and 56 qualification reports with filesystem mtimes at or after the successful retry start; all 14 shared target pool entries',campaigns=len(campaigns),excluded_earlier_campaigns=[p.name for p in old],artifacts=len(manifestpaths),qualified_workloads=report['completed'],replayed=report['replayed'],on_disk_kib=on_disk_kib,on_disk_bytes=on_disk_kib*1024,distinct_file_bytes=distinct,every_path_bytes=every,pool_entries=len(pool),pool_bytes=poolbytes,private_targets_comparison_bytes=every-poolbytes,sum_per_artifact_stored_bytes=summed_artifact_bytes)
(out/'retained-measurement.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
