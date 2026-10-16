import sys, json
sys.path.insert(0, sys.argv[0].rsplit('/',1)[0])
from closure import *
pkgs = load(sys.argv[1]); g = graph(pkgs); r = roots(pkgs)
full = reach(g, r)
S="go.temporal.io/server/"
cuts = {
 "archivers": [(S+"common/archiver/provider", S+"common/archiver/gcloud"), (S+"common/archiver/provider", S+"common/archiver/s3store")],
 "sql-drivers": [(S+"tests/testcore", S+"common/persistence/sql/sqlplugin/mysql"), (S+"tests/testcore", S+"common/persistence/sql/sqlplugin/postgresql")],
 "es-aws-signing": [(S+"common/persistence/visibility/store/elasticsearch/client", "github.com/aws/")],
 "ringpop": [(S+"temporal", S+"common/membership/ringpop")],
 "cassandra": [(S+"common/persistence/client", S+"common/persistence/cassandra"), (S+"temporal", S+"common/persistence/cassandra"), (S+"common/persistence/persistence-tests", S+"common/persistence/cassandra")],
 "wci-component": [(S+"service/worker", "go.temporal.io/auto-scaled-workers/")],
}
def run(names):
    cs = [c for n in names for c in cuts[n]]
    def gg(n):
        return [m for m in g[n] if not any(base(n)==a and base(m).startswith(b) for a,b in cs)]
    seen=set(); st=list(r)
    while st:
        n=st.pop()
        if n in seen: continue
        seen.add(n); st.extend(m for m in gg(n) if m not in seen)
    gone = full - seen
    return gone
sel = sys.argv[2].split(",") if len(sys.argv)>2 else None
for n in (sel and [None] or list(cuts)+[None]):
    names = [n] if n else (sel or list(cuts))
    gone = run(names)
    b = sum(srcbytes(pkgs[x]) for x in gone)
    print(f"{'+'.join(names)}: {len(gone)} pkgs, {b} src bytes")
    if "-v" in sys.argv:
        for x in sorted(gone): print("   ", x)
