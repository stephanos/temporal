#!/usr/bin/env python3
import sys, pathlib, shutil, json, subprocess, os
root=pathlib.Path(__file__).resolve().parent
args=sys.argv[1:]
retained=[]
for i,arg in enumerate(args):
 p=pathlib.Path(arg)
 if p.is_file() and p.suffix in ('.o','.a'):
  out=root/'inputs'/str(i)/p.name
  out.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(p,out);retained.append(str(out))
 else: retained.append(arg)
(root/'link-args.json').write_text(json.dumps(retained,indent=2))
(root/'link-cwd.txt').write_text(os.getcwd())
sys.exit(subprocess.call(['/usr/bin/cc',*args]))
