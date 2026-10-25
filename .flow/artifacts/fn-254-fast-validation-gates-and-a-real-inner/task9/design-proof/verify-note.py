from pathlib import Path
import re,json,os,subprocess
root=Path.cwd(); note=root/'model/design/framework-gate.md'; text=note.read_text()
# This source-checkout verifier includes tracked plus nonignored current candidates.
# Ignored generated caches are not declared source; tracked ignored files remain checked.
def candidate_files(directory):
 raw=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z','--',directory],cwd=root)
 paths=sorted({root/os.fsdecode(name) for name in raw.split(b'\0') if name})
 for path in paths:assert path.is_file(),f'missing candidate input: {path}'
 return paths
scripts=sorted(str(p.relative_to(root/'model/scripts')) for p in candidate_files('model/scripts'))
script_section=text.split('## Every current script\n',1)[1].split('## Every current xtask source module\n',1)[0]
script_rows=re.findall(r'^\| `([^`]+)` \| [FTM] \|',script_section,re.M)
assert sorted(script_rows)==scripts,(script_rows,scripts)
modules=sorted(p.name for p in candidate_files('model/dev/xtask/src') if p.suffix=='.rs')
module_section=text.split('## Every current xtask source module\n',1)[1].split('## Caches, pins and generators\n',1)[0]
assert sorted(re.findall(r'^\| `([^`]+)` \|',module_section,re.M))==modules
make=(root/'Makefile').read_text().splitlines()
stages=[]
for n,line in enumerate(make,1):
 if match:=re.search(r'\$\(call RUST_MODEL_STEP,([^\)]+)\)',line):stages.append((match[1],n))
for stage,n in stages:assert f'| `{stage}`, {n} |' in text,(stage,n)
assert len(stages)==19
joins={}
for p in candidate_files('model/umpire'):
 if not p.is_file():continue
 try:lines=p.read_text().splitlines()
 except UnicodeError:continue
 ns=[n for n,line in enumerate(lines,1) if 'join("../../../..")' in line]
 if ns:joins[str(p.relative_to(root/'model/umpire'))]=ns
for file,ns in joins.items():assert f'| `{file}` | '+', '.join(map(str,ns))+' |' in text,file
assert sum(map(len,joins.values()))==45
# All precise source citations must point to real files and existing lines.
citations=[]
for ref in re.findall(r'`([^`\n]+)`',text):
 m=re.fullmatch(r'([^ :]+):([0-9][0-9,\-]*)',ref)
 if not m:continue
 p=root/m[1]
 assert p.is_file(),ref
 numbers=[int(n) for n in re.findall(r'\d+',m[2])]
 assert min(numbers)>0 and max(numbers)<=len(p.read_text().splitlines()),ref
 citations.append(ref)
links=re.findall(r'\[[^\]]+\]\(([^)]+)\)',text)
for link in links:assert (note.parent/link.split('#')[0]).is_file(),link
assert 'dev/gate/policy/framework-boundary.toml' not in text
assert 'intermediate\nroot gate must run fmt/clippy/docs/tests against both workspaces' in text
assert not re.search(r'\b(TBD|TODO)\b',text)
print(json.dumps({'candidate_scope':'tracked plus nonignored current source files','scripts':len(scripts),'xtask_source_files':len(modules),'timed_stages':len(stages),'repository_root_joins':sum(map(len,joins.values())),'source_citations_checked':len(citations),'markdown_links_checked':len(links),'result':'pass'},sort_keys=True))
