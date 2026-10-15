import sys, os, re
sys.path.insert(0, sys.argv[0].rsplit('/',1)[0])
from closure import *
pkgs = load(sys.argv[1]); gone = [l.strip() for l in open(sys.argv[2]) if l.strip()]
for n in gone:
    p = pkgs[n]; d = p["Dir"]
    hits = []
    for f in p.get("GoFiles") or []:
        src = open(os.path.join(d,f)).read()
        if re.search(r'^func init\(\)', src, re.M): hits.append(f+":init")
        for m in re.findall(r'(\w+\.Register\w*)\(', src): hits.append(f+":"+m)
    print(n, "|", module(p), "|", ",".join(sorted(set(hits)))[:300])
