import sys
sys.path.insert(0, sys.argv[0].rsplit('/',1)[0])
from closure import *
pkgs = load(sys.argv[1]); g = graph(pkgs); r = roots(pkgs)
full = reach(g, r)
cands = {
 "elasticsearch": ["go.temporal.io/server/common/persistence/visibility/store/elasticsearch","github.com/olivere/elastic"],
 "es-client-only": ["github.com/olivere/elastic"],
 "aws-sdk": ["github.com/aws/"],
 "s3-archiver": ["go.temporal.io/server/common/archiver/s3store"],
 "gcloud-archiver": ["go.temporal.io/server/common/archiver/gcloud"],
 "cassandra": ["go.temporal.io/server/common/persistence/cassandra","github.com/gocql/gocql","go.temporal.io/server/common/persistence/nosql"],
 "mysql": ["go.temporal.io/server/common/persistence/sql/sqlplugin/mysql","github.com/go-sql-driver/mysql"],
 "postgresql": ["go.temporal.io/server/common/persistence/sql/sqlplugin/postgresql","github.com/jackc/pgx","github.com/lib/pq"],
 "k8s": ["k8s.io/"],
 "gcp": ["cloud.google.com/","google.golang.org/api"],
 "auto-scaled-workers": ["go.temporal.io/auto-scaled-workers"],
 "bedrock/ecs/lambda": ["github.com/aws/aws-sdk-go-v2/service/bedrock","github.com/aws/aws-sdk-go-v2/service/ecs","github.com/aws/aws-sdk-go-v2/service/lambda"],
 "gcp-run": ["cloud.google.com/go/run"],
 "ringpop": ["github.com/temporalio/ringpop-go","github.com/temporalio/tchannel-go","go.temporal.io/server/common/membership/ringpop"],
 "prometheus": ["github.com/prometheus/"],
}
only = sys.argv[2:] 
for name, prefs in cands.items():
    if only and name not in only: continue
    ban = lambda n: any(base(n).startswith(p) for p in prefs)
    after = reach(g, r, ban)
    gone = full - after
    b = sum(srcbytes(pkgs[n]) for n in gone)
    seams = sorted({(base(a), base(m)) for a in after for m in g[a] if ban(m)})
    print(f"## {name}: eliminates {len(gone)} pkgs, {b} src bytes; seam edges {len(seams)}")
    for a,m in seams[:40]: print("   ", a, "->", m)
    if "-v" in only:
        for n in sorted(gone): print("      gone", n)
