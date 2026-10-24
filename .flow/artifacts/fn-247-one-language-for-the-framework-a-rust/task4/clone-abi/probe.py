#!/usr/bin/env python3
import base64, concurrent.futures, json, subprocess
from pathlib import Path
root=Path('/tmp/fn2474-clone-abi')
entries=json.load(open(root/'linux-tree.json'))['tree']
paths=['kernel/fork.c','arch/Kconfig']+[x['path'] for x in entries if x['path'].startswith('arch/') and x['path'].count('/')==2 and x['path'].endswith('/Kconfig')]
def fetch(path):
 name=path.replace('/','-')
 target=root/(name+'.json')
 cmd=['curl','-fsSL','-o',str(target),'https://api.github.com/repos/torvalds/linux/contents/'+path+'?ref=v6.12']
 result=subprocess.run(cmd,capture_output=True,text=True)
 lines=['COMMAND: '+repr(cmd),'command_rc='+str(result.returncode)]
 if result.returncode:
  lines.append(result.stderr)
  return path, '\n'.join(lines),None
 record=json.loads(target.read_text())
 content=base64.b64decode(record['content']).decode()
 (root/name).write_text(content)
 lines.append('SOURCE: '+record['html_url']+' blob_sha='+record['sha'])
 matches=[f'{i}:{line}' for i,line in enumerate(content.splitlines(),1) if 'CLONE_BACKWARDS' in line]
 lines.extend(matches or ['No CLONE_BACKWARDS selection/definition in this file.'])
 return path,'\n'.join(lines),matches
with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:results=list(pool.map(fetch,paths))
failures=[]
for path,log,matches in results:
 print(log,flush=True)
 if matches is None:failures.append(path)
print('fetch_failures='+repr(failures),flush=True)
raise SystemExit(bool(failures))
