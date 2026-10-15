import json, sys, os, re, collections
def load(path):
    dec = json.JSONDecoder(); data = open(path).read(); i = 0; pkgs = {}
    while i < len(data):
        while i < len(data) and data[i].isspace(): i += 1
        if i >= len(data): break
        obj, i = dec.raw_decode(data, i); pkgs[obj["ImportPath"]] = obj
    return pkgs
def base(p): return p.split(" [")[0]
def module(p):
    o = p.get("Module") or {}
    return o.get("Path", "std" if p.get("Standard") else "?")
def srcbytes(p):
    d = p.get("Dir", ""); n = 0
    for f in (p.get("GoFiles") or []) + (p.get("CgoFiles") or []) + (p.get("SFiles") or []):
        try: n += os.path.getsize(os.path.join(d, f))
        except OSError: pass
    return n
def graph(pkgs):
    g = {}
    for k, p in pkgs.items():
        imps = list(p.get("Imports") or [])
        # resolve via ImportMap? Imports already resolved paths in -deps -test output
        g[k] = [x for x in imps if x in pkgs]
    return g
def reach(g, roots, banned=lambda n: False):
    seen = set(); st = [r for r in roots if not banned(r)]
    while st:
        n = st.pop()
        if n in seen: continue
        seen.add(n)
        for m in g.get(n, []):
            if m not in seen and not banned(m): st.append(m)
    return seen
def roots(pkgs):
    return [k for k, p in pkgs.items() if not p.get("DepOnly")]
